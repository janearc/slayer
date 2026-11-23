package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/janearc/slayer/azazel/internal/footprint"
)

// supervisorGonePrefix starts the held text that says a render runs with no
// supervisor. It begins as recovery notes do, so the scheduling line shows it.
const supervisorGonePrefix = "recovery: supervisor gone: "

// supervisorGone reports whether job's render lives with no supervisor to
// time it, cancel it or stop it. The attempt lock is held by the render,
// and nothing has renewed the lease for a whole lease past its end: the
// supervisor renews every few seconds, and the dispatcher renews only for
// the supervisors it started itself.
func supervisorGone(job Job, now time.Time) bool {
	return job.State == Running && job.Executing && job.ProcessGroup > 1 &&
		!job.LeaseUntil.IsZero() && now.After(job.LeaseUntil.Add(renderLeaseDuration))
}

// supervisorGoneNote says which group runs unsupervised and since when.
func supervisorGoneNote(job Job) string {
	return fmt.Sprintf("%sprocess group %d runs with no supervisor; its lease ended at %s",
		supervisorGonePrefix, job.ProcessGroup, job.LeaseUntil.UTC().Format(time.RFC3339))
}

// watchHeldAttempt keeps the ledger's note for a job whose attempt lock is
// held: written when its supervisor is gone, and withdrawn if the lease is
// renewed after all, as it is when a clock step only looked like silence.
func (s Store) watchHeldAttempt(ctx context.Context, job Job, now time.Time) error {
	note := ""
	if supervisorGone(job, now) {
		note = supervisorGoneNote(job)
	}
	if note == job.Held || note == "" && !strings.HasPrefix(job.Held, supervisorGonePrefix) {
		return nil
	}
	return s.hold(ctx, job, note)
}

// groupSignaller is what a cancel uses to find a render's group and signal it.
type groupSignaller struct {
	boot     func() (string, error)
	identity func(int) (footprint.Identity, error)
	kill     func(int, syscall.Signal) error
}

// nativeSignaller reads this host's processes and signals them.
func nativeSignaller() groupSignaller {
	return groupSignaller{boot: footprint.BootID, identity: footprint.Native().Identity, kill: syscall.Kill}
}

// signalOrphan kills the process group of a render whose supervisor is gone.
// It does so only when the ledger recorded the leader's identity under this
// boot and the process now holding that group number is that leader: a
// group number alone may belong to another process by now.
func signalOrphan(job Job, via groupSignaller) error {
	group := job.ProcessGroup
	if job.ProcessStartSeconds == 0 || !footprint.ValidBootID(job.ProcessBootID) {
		return fmt.Errorf("process group %d runs with no supervisor and its identity was not recorded, so it is not signaled", group)
	}
	boot, err := via.boot()
	if err != nil {
		return fmt.Errorf("process group %d runs with no supervisor and this boot is unreadable, so it is not signaled: %w", group, err)
	}
	if boot != job.ProcessBootID {
		return nil
	}
	found, err := via.identity(group)
	if err != nil {
		if errors.Is(via.kill(-group, 0), syscall.ESRCH) {
			return nil
		}
		return fmt.Errorf("process group %d runs with no supervisor and its leader cannot be read, so it is not signaled: %w", group, err)
	}
	if found.PID != group || found.Group != group || found.StartSeconds != job.ProcessStartSeconds || found.StartMicroseconds != job.ProcessStartMicroseconds {
		return nil
	}
	if err := via.kill(-group, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
