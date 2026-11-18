package queue

import (
	"context"
	"fmt"
	"io"
	"time"
)

// keepRenderLease covers fingerprinting and host holds before a supervisor
// starts. The supervisor independently renews after the command begins.
func (s Store) keepRenderLease(ctx context.Context, job Job, out io.Writer) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := s.renewRenderLease(ctx, job, now); err != nil && ctx.Err() == nil {
					fmt.Fprintf(out, "render lease renewal: %v\n", err)
				}
			}
		}
	}()
	return func() { cancel(); <-done }
}

// renewRenderLease extends only the current, unexpired attempt's grant.
func (s Store) renewRenderLease(ctx context.Context, attempt Job, now time.Time) error {
	return s.transact(ctx, false, func(state *Snapshot) error {
		job, err := state.job(attempt.Sequence)
		if err != nil {
			return err
		}
		if job.State != Running || job.Worker != attempt.Worker || job.Attempt != attempt.Attempt {
			return ErrOwner
		}
		if job.LeaseUntil.IsZero() || !now.Before(job.LeaseUntil) {
			return ErrExpiredLease
		}
		until := now.UTC().Add(renderLeaseDuration)
		if until.After(job.LeaseUntil) {
			job.LeaseUntil = until
		}
		return nil
	})
}
