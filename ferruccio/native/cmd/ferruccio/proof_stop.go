package main

import "sync/atomic"

// proofPreview exposes only the owned cancellation and completion boundary.
type proofPreview interface {
	StopPreview()
	Running() bool
}

// proofStop receives timer expiry and drives cleanup on the native UI thread.
type proofStop struct {
	expired             atomic.Bool
	requested, complete bool
}

// expire is safe on the timer goroutine; it never touches native UI state.
func (stop *proofStop) expire() { stop.expired.Store(true) }

// poll reports expiry and whether the owned preview has finished stopping.
func (stop *proofStop) poll(work proofPreview) (expired, complete bool) {
	if !stop.expired.Load() {
		return false, false
	}
	if stop.complete {
		return true, true
	}
	if !stop.requested {
		stop.requested = true
		work.StopPreview()
	}
	stop.complete = !work.Running()
	return true, stop.complete
}
