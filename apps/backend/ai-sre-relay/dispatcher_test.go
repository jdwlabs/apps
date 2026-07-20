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
	d := newDispatcher(h, 1, 4, time.Second, silentLogger())

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
	d := newDispatcher(h, 1, 4, time.Second, silentLogger())
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
	h := handlerFunc(func(_ context.Context, _ Alert) error {
		<-release // hold the single worker busy
		return nil
	})
	d := newDispatcher(h, 1, 1, time.Second, silentLogger())

	d.enqueue(Alert{Fingerprint: "a"})       // taken by the worker (now blocked)
	time.Sleep(20 * time.Millisecond)        // ensure worker picked up 'a'
	if !d.enqueue(Alert{Fingerprint: "b"}) { // fills the size-1 queue
		t.Fatal("second enqueue should fit the queue")
	}
	if d.enqueue(Alert{Fingerprint: "c"}) { // queue full + worker busy -> drop
		t.Fatal("third enqueue should be dropped when the queue is full")
	}
	close(release)
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
	d := newDispatcher(h, 2, 8, time.Second, silentLogger())
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
	d := newDispatcher(h, 1, 4, time.Second, silentLogger())

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
