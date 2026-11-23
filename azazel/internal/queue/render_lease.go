package queue

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// leaseRenewEvery is how often a holder extends its render lease.
const leaseRenewEvery = 5 * time.Second

// keepRenderLease covers fingerprinting and host holds before a supervisor
// starts. The supervisor independently renews after the command begins.
func (s Store) keepRenderLease(ctx context.Context, job Job, out io.Writer) func() {
	ticker := time.NewTicker(leaseRenewEvery)
	stop := s.keepLeaseOnTicks(ctx, job, out, ticker.C)
	return func() { ticker.Stop(); stop() }
}

// keepLeaseOnTicks renews the lease of job at each time received on ticks,
// until the returned stop is called. The time received is the clock the
// renewal reads, so a caller may step it.
func (s Store) keepLeaseOnTicks(ctx context.Context, job Job, out io.Writer, ticks <-chan time.Time) func() {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticks:
				if err := s.renewHeldRenderLease(ctx, job, now); err != nil && ctx.Err() == nil {
					fmt.Fprintf(out, "render lease renewal: %v\n", err)
				}
			}
		}
	}()
	return func() { cancel(); <-done }
}

// renewRenderLease extends only the current, unexpired attempt's grant.
func (s Store) renewRenderLease(ctx context.Context, attempt Job, now time.Time) error {
	return s.renewLease(ctx, attempt, now, false)
}

// renewHeldRenderLease extends the current attempt's grant. A grant that
// reads as expired is renewed anyway while the attempt lock is held: the
// lock is inherited by the render, so a held lock means the render lives,
// and a clock that stepped forward is not a reason to give it up.
func (s Store) renewHeldRenderLease(ctx context.Context, attempt Job, now time.Time) error {
	return s.renewLease(ctx, attempt, now, s.attemptHeld(attempt))
}

// attemptHeld reports whether another open file holds the attempt lock.
func (s Store) attemptHeld(attempt Job) bool {
	lock, err := s.attemptLock(attempt)
	if err == nil {
		lock.Close()
		return false
	}
	return errors.Is(err, ErrActive)
}

// renewLease extends the lease of the current attempt; held lets an
// expired grant be renewed.
func (s Store) renewLease(ctx context.Context, attempt Job, now time.Time, held bool) error {
	return s.transact(ctx, false, func(state *Snapshot) error {
		job, err := state.job(attempt.Sequence)
		if err != nil {
			return err
		}
		if job.State != Running || job.Worker != attempt.Worker || job.Attempt != attempt.Attempt {
			return ErrOwner
		}
		if job.LeaseUntil.IsZero() || !now.Before(job.LeaseUntil) && !held {
			return ErrExpiredLease
		}
		until := now.UTC().Add(renderLeaseDuration)
		if until.After(job.LeaseUntil) {
			job.LeaseUntil = until
		}
		return nil
	})
}
