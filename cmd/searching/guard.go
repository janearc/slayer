package main

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/janearc/puzzlemouth/pzm"
)

// guard leaves headroom under the owner's doubled 3 GiB aggregate limit for
// Game and the launcher. It samples this native/Go process once a second;
// GOMEMLIMIT is only a soft Go-heap target and is not this guard.
func guard(parent context.Context, call pzm.Call, disk string) (context.Context, context.CancelFunc, error) {
	limit, err := integer(call, "limit-seconds", 600)
	if err != nil || limit < 1 || limit > 28800 {
		return nil, nil, fmt.Errorf("wall limit must be 1..28800 seconds")
	}
	if err := resources(disk); err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithCancelCause(parent)
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		deadline := time.NewTimer(time.Duration(limit) * time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-deadline.C:
				cancel(fmt.Errorf("shoot wall-time limit reached"))
				return
			case <-tick.C:
				if err := resources(disk); err != nil {
					cancel(err)
					return
				}
			}
		}
	}()
	return ctx, func() { cancel(nil) }, nil
}

// resources checks Darwin's byte-valued peak RSS and the output device's
// available space. Native frames must finish before cancellation is observed.
func resources(disk string) error {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return err
	}
	if usage.Maxrss > 2800*1024*1024 {
		return fmt.Errorf("shoot process exceeded 2800 MiB peak RSS")
	}
	var volume syscall.Statfs_t
	if err := syscall.Statfs(disk, &volume); err != nil {
		return err
	}
	if uint64(volume.Bavail)*uint64(volume.Bsize) < 30*1024*1024*1024 {
		return fmt.Errorf("output device has less than 30 GiB free")
	}
	return nil
}
