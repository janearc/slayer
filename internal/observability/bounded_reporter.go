package observability

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// SummaryShutdownWait bounds a normal final-summary opportunity.
const SummaryShutdownWait = 500 * time.Millisecond

// StartBoundedReporter runs one serial reporter. On a normal stop it gives a
// final callback one bounded opportunity, then cancels without waiting on a
// stuck sampler or sink. A cancelled parent never starts a final callback.
// Callbacks must check ctx after blocking operations and before side effects.
func StartBoundedReporter(parent context.Context, interval, wait time.Duration, initial, tick, final func(context.Context)) func() {
	return startBoundedReporter(parent, interval, wait, initial, tick, final, nil)
}

// StartExclusiveReporter keeps one in-flight reporter per gate, including
// one stuck after shutdown, preventing repeated starts from leaking workers.
func StartExclusiveReporter(gate *atomic.Bool, parent context.Context, interval, wait time.Duration, initial, tick, final func(context.Context)) (func(), bool) {
	if gate == nil || !gate.CompareAndSwap(false, true) {
		return func() {}, false
	}
	return startBoundedReporter(parent, interval, wait, initial, tick, final, func() { gate.Store(false) }), true
}

// startBoundedReporter runs the initial, tick and final callbacks on an
// interval, and returns the function that stops the reporter.
func startBoundedReporter(parent context.Context, interval, wait time.Duration, initial, tick, final func(context.Context), finished func()) func() {
	if parent == nil {
		parent = context.Background()
	}
	if interval <= 0 {
		interval = time.Minute
	}
	if wait <= 0 {
		wait = SummaryShutdownWait
	}
	ctx, cancel := context.WithCancel(parent)
	quit, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		if finished != nil {
			defer finished()
		}
		if ctx.Err() != nil {
			return
		}
		if initial != nil {
			reporterCallback(ctx, initial)
		}
		if ctx.Err() != nil {
			return
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-quit:
				if ctx.Err() == nil && final != nil {
					reporterCallback(ctx, final)
				}
				return
			case <-ticker.C:
				if ctx.Err() == nil && tick != nil {
					reporterCallback(ctx, tick)
				}
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(quit)
			timer := time.NewTimer(wait)
			defer timer.Stop()
			select {
			case <-done:
			case <-timer.C:
			}
			cancel()
		})
	}
}

// reporterCallback runs one callback, and records a failure if it panics.
func reporterCallback(ctx context.Context, call func(context.Context)) {
	defer func() {
		if recover() != nil {
			NoteSummaryFailure(nil, "")
		}
	}()
	call(ctx)
}
