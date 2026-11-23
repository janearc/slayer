package queue

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// defaultReadAllowance is how long a supervisor lets its reads of the host
// and the ledger keep failing before it stops the render. Each read waits
// up to five seconds on the ledger, so this spans several of them.
const defaultReadAllowance = 30 * time.Second

// readAllowance is the time supervision reads may fail in a row.
func (s Store) readAllowance() time.Duration {
	if s.readLimit == 0 {
		return defaultReadAllowance
	}
	return s.readLimit
}

// readFailures counts the supervision ticks in a row whose reads failed.
type readFailures struct {
	first time.Time
	count int
}

// spent records one more failed tick at now and reports whether the
// failures have run for allowance or longer.
func (f *readFailures) spent(now time.Time, allowance time.Duration) bool {
	if f.count == 0 {
		f.first = now
	}
	f.count++
	return now.Sub(f.first) >= allowance
}

// clear forgets earlier failures after a tick whose reads all succeeded.
func (f *readFailures) clear() { f.count = 0 }

// explain is the reason stored when the failures stop a render: how many
// reads failed, over how long, and the last error. An error that is not
// transient is reported as it is.
func (f readFailures) explain(err error, now time.Time) error {
	if f.count == 0 {
		return err
	}
	return fmt.Errorf("%d host and ledger reads failed over %s: %w", f.count, now.Sub(f.first).Round(time.Millisecond), err)
}

// readMayRetry says whether a failed read may be tried again: not when the
// run is over, and not when the record says the job is no longer ours.
func readMayRetry(ctx context.Context, err error) bool {
	return ctx.Err() == nil && !errors.Is(err, ErrOwner) && !errors.Is(err, ErrMissing) && !errors.Is(err, ErrUnsafe)
}

// supervisionTick reads whether the job is cancelled or paused and the
// host's vitals, and records the usage. It returns the result to stop the
// group with when stop is set; an error means a read failed.
func (s Store) supervisionTick(ctx context.Context, j Job, monitor Monitor, group int) (result Result, stop bool, err error) {
	cancelled, paused, err := s.requested(ctx, j)
	if err != nil {
		return Result{}, false, err
	}
	if cancelled || paused != "" {
		return Result{State: Cancelled, ExitCode: -1, Reason: "cancellation or queue pause requested"}, true, nil
	}
	vitals, err := monitor.Read(ctx, j.Request.Directory, group)
	if err == nil {
		vitals, err = s.runtimeVitals(ctx, vitals)
	}
	if err != nil {
		return Result{}, false, err
	}
	if err := s.observe(ctx, j, vitals); err != nil {
		return Result{}, false, err
	}
	if vitals.Stop {
		return Result{State: Failed, ExitCode: -1, Stop: true, Reason: vitals.Reason}, true, nil
	}
	return Result{}, false, nil
}
