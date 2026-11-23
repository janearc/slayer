package queue

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/janearc/slayer/azazel/internal/footprint"
)

// Executor runs a selected job and reports whether its lifetime is known.
type Executor interface {
	Run(context.Context, Job, io.Writer) Result
}

// workerLock is permanent; participants never replace or remove it.
func (s Store) workerLock() (*os.File, error) {
	return s.lockFile("worker.lock")
}

// attemptLock stays held by the supervisor and is inherited by its command.
func (s Store) attemptLock(job Job) (*os.File, error) {
	return s.lockFile(fmt.Sprintf("%020d.%04d.lock", job.Sequence, job.Attempt))
}

// lockFile never replaces a lock another process may still hold.
func (s Store) lockFile(name string) (*os.File, error) {
	if err := s.location(); err != nil {
		return nil, err
	}
	path := filepath.Join(s.root, name)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = private(path, false); err == nil {
		lockDescriptor := int(file.Fd())
		err = syscall.Flock(lockDescriptor, syscall.LOCK_EX|syscall.LOCK_NB)
	}
	if err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, ErrActive
		}
		return nil, err
	}
	return file, nil
}

// Work drains FIFO jobs, or waits for later submissions when serve is true.
func (s Store) Work(ctx context.Context, executor Executor, serve bool) error {
	if executor == nil {
		return ErrUnsafe
	}
	lock, err := s.workerLock()
	if err != nil {
		return err
	}
	defer lock.Close()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		job, err := s.claim(ctx, rand.Text())
		if errors.Is(err, ErrEmpty) {
			if !serve {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(250 * time.Millisecond):
				continue
			}
		}
		if err != nil {
			return err
		}
		if err = s.run(ctx, executor, job); err != nil {
			return err
		}
	}
}

// run keeps uncertain executions active until an operator resolves them.
func (s Store) run(ctx context.Context, executor Executor, job Job) error {
	path := filepath.Join(s.root, fmt.Sprintf("%020d.%04d.log", job.Sequence, job.Attempt))
	log, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	result := executor.Run(ctx, job, log)
	closeErr := errors.Join(log.Sync(), log.Close())
	if result.Uncertain {
		return fmt.Errorf("%w: %s", ErrActive, result.Reason)
	}
	// A cancelled caller still needs to retain the terminal receipt.
	receiptCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if result.Stop {
		if err = s.Pause(receiptCtx, result.Reason); err != nil {
			return err
		}
	}
	if result.Deferred {
		return errors.Join(closeErr, s.deferAttempt(receiptCtx, job, result.Reason))
	}
	return errors.Join(closeErr, s.finish(receiptCtx, job.Sequence, job.Worker, result))
}

// Recover records interruption only after an explicit quiescence attestation.
func (s Store) Recover(ctx context.Context, seq uint64, owner string, stopped bool) error {
	if !stopped {
		return fmt.Errorf("recovery needs explicit confirmation that this attempt and its descendants have stopped")
	}
	lock, err := s.workerLock()
	if errors.Is(err, ErrActive) {
		return fmt.Errorf("%w: %w", ErrWorkerServing, err)
	}
	if err != nil {
		return err
	}
	defer lock.Close()
	return s.recoverAttempt(ctx, seq, owner, true)
}

// recoverAttempt preserves unstarted work and refuses live or unknown groups.
// A group recorded under another boot is not this boot's and is not probed.
func (s Store) recoverAttempt(ctx context.Context, seq uint64, owner string, attested bool) error {
	boot, err := footprint.BootID()
	return s.recoverAttemptOnBoot(ctx, seq, owner, attested, boot, err == nil)
}

// foreignBoot reports whether job's group was recorded under a boot other
// than boot. An unrecorded or unknown boot never counts as foreign.
func foreignBoot(job Job, boot string, known bool) bool {
	return known && footprint.ValidBootID(job.ProcessBootID) && job.ProcessBootID != boot
}

// recoverAttemptOnBoot is recoverAttempt for a given boot identity; known is
// false when this boot's identity could not be read.
func (s Store) recoverAttemptOnBoot(ctx context.Context, seq uint64, owner string, attested bool, boot string, known bool) error {
	snapshot, err := s.Status(ctx)
	if err != nil {
		return err
	}
	job, err := snapshot.job(seq)
	if err != nil {
		return err
	}
	if job.Worker != owner || job.State != Running {
		return ErrOwner
	}
	lease, err := s.attemptLock(*job)
	if err != nil {
		return err
	}
	defer lease.Close()
	foreign := foreignBoot(*job, boot, known)
	if !foreign && job.ProcessGroup > 1 && !errors.Is(syscall.Kill(-job.ProcessGroup, 0), syscall.ESRCH) {
		return fmt.Errorf("%w: process group %d still exists or is unknown", ErrActive, job.ProcessGroup)
	}
	if !job.Executing {
		return s.deferAttempt(ctx, *job, "worker stopped before execution")
	}
	if job.ProcessGroup == 0 && !attested {
		return fmt.Errorf("%w: launch outcome requires operator inspection", ErrActive)
	}
	reason := "supervisor ended; process group confirmed stopped"
	if foreign {
		reason = "supervisor ended; its process group was recorded under another boot, so it is gone"
	}
	return s.finish(ctx, seq, owner, Result{State: Interrupted, ExitCode: -1, Reason: reason})
}
