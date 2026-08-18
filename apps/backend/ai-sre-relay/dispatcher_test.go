package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type handlerFunc func(context.Context, Alert) error

func (f handlerFunc) Handle(ctx context.Context, a Alert) error { return f(ctx, a) }

func TestDispatcherProcessesAlert(t *testing.T) {
	got := make(chan Alert, 1)
	h := handlerFunc(func(_ context.Context, a Alert) error { got <- a; return nil })
	d := newDispatcher(h, 1, 4, time.Second, &counters{}, silentLogger())

	if !d.enqueue(Alert{Fingerprint: "f1"}) {
		t.Fatal("enqueue returned false on empty queue")
	}
	select {
	case a := <-got:
		if a.Fingerprint != "f1" {
			t.Fatalf("processed %s, want f1", a.Fingerprint)
		}
	case <-time.After(time.Second):
		t.Fatal("alert not processed")
	}
	d.shutdown(context.Background())
}

func TestDispatcherRecoversFromPanic(t *testing.T) {
	processed := make(chan string, 1)
	h := handlerFunc(func(_ context.Context, a Alert) error {
		if a.Fingerprint == "boom" {
			panic("induced")
		}
		processed <- a.Fingerprint
		return nil
	})
	d := newDispatcher(h, 1, 4, time.Second, &counters{}, silentLogger())
	d.enqueue(Alert{Fingerprint: "boom"})
	d.enqueue(Alert{Fingerprint: "ok"})

	// The single worker must survive the panic and still process the next alert.
	select {
	case fp := <-processed:
		if fp != "ok" {
			t.Fatalf("processed %s, want ok", fp)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not survive panic")
	}
	d.shutdown(context.Background())
}

func TestDispatcherQueueFullDrops(t *testing.T) {
	release := make(chan struct{})
	var handled sync.Map
	h := handlerFunc(func(_ context.Context, a Alert) error {
		handled.Store(a.Fingerprint, true)
		<-release // hold the single worker busy
		return nil
	})
	c := &counters{}
	d := newDispatcher(h, 1, 1, time.Second, c, silentLogger())

	d.enqueue(Alert{Fingerprint: "a"})       // taken by the worker (now blocked)
	time.Sleep(20 * time.Millisecond)        // ensure worker picked up 'a'
	if !d.enqueue(Alert{Fingerprint: "b"}) { // fills the size-1 queue
		t.Fatal("second enqueue should fit the queue")
	}
	if d.enqueue(Alert{Fingerprint: "c"}) { // queue full + worker busy -> refuse
		t.Fatal("third enqueue should be refused when the queue is full")
	}
	// The refusal has to reach Prometheus: it is the only signal that
	// coverage was lost, and the pipeline counters never move for an alert
	// that was turned away before reaching a worker.
	if got := c.alertsRejected.Load(); got != 1 {
		t.Fatalf("alertsRejected = %d, want 1", got)
	}
	// A refused fingerprint must not stay marked in-flight, or the retry that
	// the 503 provokes would be silently coalesced and never investigated.
	d.mu.Lock()
	_, stillHeld := d.inflight["c"]
	d.mu.Unlock()
	if stillHeld {
		t.Fatal("refused fingerprint left marked in-flight; its retry would be coalesced away")
	}

	close(release)
	d.shutdown(context.Background())

	// Overflow must cost only the refused alert; queued work still runs.
	if _, ok := handled.Load("b"); !ok {
		t.Fatal("queued alert was lost when a later alert overflowed the queue")
	}
	if _, ok := handled.Load("c"); ok {
		t.Fatal("refused alert was investigated anyway")
	}
}

// The refusal must survive re-offer: once capacity exists, the same alert the
// queue turned away has to be accepted, otherwise the sender's retry is
// pointless and the 503 buys nothing.
func TestDispatcherAcceptsRefusedAlertOnRetry(t *testing.T) {
	release := make(chan struct{})
	investigated := make(chan string, 4)
	h := handlerFunc(func(_ context.Context, a Alert) error {
		<-release
		investigated <- a.Fingerprint
		return nil
	})
	c := &counters{}
	d := newDispatcher(h, 1, 1, time.Second, c, silentLogger())

	d.enqueue(Alert{Fingerprint: "a"})
	time.Sleep(20 * time.Millisecond)
	d.enqueue(Alert{Fingerprint: "b"})
	if d.enqueue(Alert{Fingerprint: "c"}) {
		t.Fatal("setup: third enqueue should have been refused")
	}

	close(release)
	for range 2 { // drain 'a' and 'b' so a slot frees
		select {
		case <-investigated:
		case <-time.After(time.Second):
			t.Fatal("queued alerts did not drain")
		}
	}

	if !d.enqueue(Alert{Fingerprint: "c"}) {
		t.Fatal("previously refused alert must be accepted once capacity frees")
	}
	select {
	case fp := <-investigated:
		if fp != "c" {
			t.Fatalf("investigated %s, want c", fp)
		}
	case <-time.After(time.Second):
		t.Fatal("retried alert was never investigated")
	}
	d.shutdown(context.Background())
}

func TestDispatcherShutdownDrains(t *testing.T) {
	var mu sync.Mutex
	done := 0
	h := handlerFunc(func(_ context.Context, _ Alert) error {
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		done++
		mu.Unlock()
		return nil
	})
	d := newDispatcher(h, 2, 8, time.Second, &counters{}, silentLogger())
	for i := range 4 {
		d.enqueue(Alert{Fingerprint: fmt.Sprintf("f%d", i)})
	}
	d.shutdown(context.Background()) // unbounded ctx: must wait for all 4

	mu.Lock()
	defer mu.Unlock()
	if done != 4 {
		t.Fatalf("drained %d, want 4", done)
	}
}

// A repeat notification for a fingerprint still queued or under investigation
// must be coalesced (acknowledged, not re-processed), while the same
// fingerprint must be investigable again once the first run has finished.
func TestDispatcherCoalescesInFlightFingerprint(t *testing.T) {
	release := make(chan struct{})
	processed := make(chan string, 4)
	h := handlerFunc(func(_ context.Context, a Alert) error {
		<-release
		processed <- a.Fingerprint
		return nil
	})
	d := newDispatcher(h, 1, 4, time.Second, &counters{}, silentLogger())

	if !d.enqueue(Alert{Fingerprint: "a"}) {
		t.Fatal("first enqueue must be accepted")
	}
	time.Sleep(20 * time.Millisecond) // let the worker pick up 'a'
	if !d.enqueue(Alert{Fingerprint: "a"}) {
		t.Fatal("repeat must be acknowledged (coalesced), not reported as dropped")
	}

	close(release)
	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("first investigation did not complete")
	}
	select {
	case fp := <-processed:
		t.Fatalf("repeat notification was investigated (%s); it must be coalesced", fp)
	case <-time.After(50 * time.Millisecond):
	}

	// After completion the fingerprint must be accepted and processed again.
	if !d.enqueue(Alert{Fingerprint: "a"}) {
		t.Fatal("fingerprint must be re-enqueueable after its investigation finished")
	}
	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("re-fired fingerprint was not investigated after the first run completed")
	}
	d.shutdown(context.Background())
}
