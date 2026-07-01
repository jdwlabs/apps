package main

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// handler is the pipeline seam; the dispatcher depends only on this interface.
type handler interface {
	Handle(ctx context.Context, a Alert) error
}

// dispatcher runs alert investigations on a bounded worker pool. The replica is
// memory-capped (32Mi) and each investigation makes several slow (tens of
// seconds) LLM/HTTP calls, so an unbounded goroutine-per-alert fan-out under a
// large Alertmanager batch would OOM-kill the process. A fixed pool with a
// bounded queue caps both concurrency and memory.
type dispatcher struct {
	handler    handler
	queue      chan Alert
	perAlertTO time.Duration
	log        *slog.Logger
	wg         sync.WaitGroup
}

func newDispatcher(h handler, workers, queueSize int, perAlertTO time.Duration, log *slog.Logger) *dispatcher {
	d := &dispatcher{
		handler:    h,
		queue:      make(chan Alert, queueSize),
		perAlertTO: perAlertTO,
		log:        log,
	}
	for range workers {
		d.wg.Add(1)
		go d.worker()
	}
	return d
}

func (d *dispatcher) worker() {
	defer d.wg.Done()
	for a := range d.queue {
		d.process(a)
	}
}

func (d *dispatcher) process(a Alert) {
	// A panicking client must not kill the worker — that would shrink the pool
	// and eventually stall the queue.
	defer func() {
		if r := recover(); r != nil {
			d.log.Error("pipeline panic", "fingerprint", a.Fingerprint, "panic", r)
		}
	}()
	// Bound the whole pipeline (several sequential calls) so one hung upstream
	// cannot occupy a worker slot indefinitely. The clients are ctx-aware, so
	// this also cancels in-flight requests on shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), d.perAlertTO)
	defer cancel()
	start := time.Now()
	_ = d.handler.Handle(ctx, a)
	d.log.Info("investigation complete",
		"fingerprint", a.Fingerprint, "alert", a.Name(),
		"duration_ms", time.Since(start).Milliseconds())
}

// enqueue submits an alert without blocking the HTTP response. It returns false
// (and logs) when the queue is saturated; Alertmanager re-sends on
// repeat_interval and Jira dedup absorbs the repeat, so a drop is recoverable,
// not silent.
func (d *dispatcher) enqueue(a Alert) bool {
	select {
	case d.queue <- a:
		return true
	default:
		d.log.Error("investigation queue full; dropping alert (alertmanager will retry)",
			"fingerprint", a.Fingerprint, "alert", a.Name())
		return false
	}
}

// shutdown stops accepting work and waits for in-flight and queued
// investigations to finish, bounded by ctx. Call only after the HTTP server has
// stopped, so no concurrent enqueue races the channel close.
func (d *dispatcher) shutdown(ctx context.Context) {
	close(d.queue)
	done := make(chan struct{})
	go func() { d.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		d.log.Warn("shutdown deadline reached; abandoning in-flight investigations")
	}
}
