//go:build darwin

package metal

/*
#include "bridge.h"
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// A drawing waits for the device in the bridge, polling the command, and
// Metal cannot take a running command back. So a context that stops a
// drawing does it by ending the wait: StopOn has every wait end when the
// context is done, with the context's error, and the device is left to
// finish or lose the command it was given. The kernel the wait was in is
// not used again, as after any wait that gave up.

// stopping is the context the waits are stopped on, kept to say its error.
var stopping struct {
	sync.Mutex
	ctx context.Context
}

// StopOn stops every wait for the device when ctx is done: the wait
// returns an error that wraps the context's. It gives back the function
// that lets go of the context and lets later waits wait again, when the
// work ctx covers is over; calling it again does nothing. A context that
// is never done stops nothing, and a nil one is a context never done.
func StopOn(ctx context.Context) (release func()) {
	if ctx == nil {
		return func() {}
	}
	stopping.Lock()
	stopping.ctx = ctx
	stopping.Unlock()
	ended := make(chan struct{})
	cancel := context.AfterFunc(ctx, func() {
		defer close(ended)
		C.slayer_stop_waits(1)
	})
	var once sync.Once
	return func() {
		once.Do(func() {
			// a stop that has begun is let finish, so it cannot land
			// after the waits are let go and stop the next drawing's
			if !cancel() {
				<-ended
			}
			C.slayer_stop_waits(0)
			stopping.Lock()
			stopping.ctx = nil
			stopping.Unlock()
		})
	}
}

// errStoppedWait is why a wait ended when no context says; it is a
// cancellation.
var errStoppedWait = fmt.Errorf("metal: a wait was stopped: %w",
	context.Canceled)

// stoppedError is the error of a wait that was stopped, wrapping the
// error of the context it stopped on.
func stoppedError(why string) error {
	stopping.Lock()
	ctx := stopping.ctx
	stopping.Unlock()
	if ctx == nil || ctx.Err() == nil {
		return errors.Join(errStoppedWait, errors.New("metal: "+why))
	}
	return fmt.Errorf("metal: %s: %w", why, ctx.Err())
}
