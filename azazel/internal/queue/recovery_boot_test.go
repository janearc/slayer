package queue

import (
	"errors"
	"strings"
	"syscall"
	"testing"
	"time"
)

// bootTailForTest is the part of a boot identity after its first eight
// digits.
const bootTailForTest = "-89ab-cdef-0123-456789abcdef"

// The two boot identities the recovery tests use; both are canonical, and
// they differ in their first eight digits.
const (
	thisBootForTest  = "11111111" + bootTailForTest
	otherBootForTest = "22222222" + bootTailForTest
)

// executingUnderBoot claims the fixture job and records this process's own
// group, which is alive, as the group its supervisor started, under boot.
func executingUnderBoot(t *testing.T, boot string) (Store, Job) {
	t.Helper()
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.transact(t.Context(), false, func(state *Snapshot) error {
		state.Jobs[0].Executing = true
		state.Jobs[0].ProcessGroup = syscall.Getpgrp()
		state.Jobs[0].ProcessBootID = boot
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return s, job
}

// A group recorded under another boot is not this boot's: the attempt
// resolves and its slot is free, though the same number is alive here.
func TestRecoveryIgnoresAGroupRecordedUnderAnotherBoot(t *testing.T) {
	s, job := executingUnderBoot(t, otherBootForTest)
	if err := s.recoverAttemptOnBoot(t.Context(), job.Sequence, job.Worker, false, thisBootForTest, true); err != nil {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Interrupted || !strings.Contains(state.Jobs[0].Result.Reason, "another boot") {
		t.Fatal("foreign boot group still held the slot", state, err)
	}
}

// Anti-test: under the same boot the present check stands, and a boot that
// is unknown or unrecorded is never treated as foreign.
func TestRecoveryStillRefusesALiveGroupOfThisBoot(t *testing.T) {
	for name, c := range map[string]struct {
		recorded string
		known    bool
	}{
		"same boot":         {thisBootForTest, true},
		"this boot unknown": {otherBootForTest, false},
		"none recorded":     {"", true},
	} {
		t.Run(name, func(t *testing.T) {
			s, job := executingUnderBoot(t, c.recorded)
			err := s.recoverAttemptOnBoot(t.Context(), job.Sequence, job.Worker, false, thisBootForTest, c.known)
			if !errors.Is(err, ErrActive) {
				t.Fatal("a live group was resolved", err)
			}
			state, err := s.Status(t.Context())
			if err != nil || state.Jobs[0].State != Running {
				t.Fatal(state, err)
			}
		})
	}
}

// Recovery against a queue whose worker is serving says why it cannot run.
func TestRecoverNamesTheServingWorker(t *testing.T) {
	s, job := executingUnderBoot(t, "")
	lock, err := s.workerLock()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	err = s.Recover(t.Context(), job.Sequence, job.Worker, true)
	if !errors.Is(err, ErrWorkerServing) || !errors.Is(err, ErrActive) || !strings.Contains(err.Error(), "stop the worker") {
		t.Fatal("recover did not say the worker holds the lock", err)
	}
}

// Anti-test: with no worker serving, recovery does not claim one is.
func TestRecoverDoesNotBlameAWorkerThatIsNotServing(t *testing.T) {
	s, _ := executingUnderBoot(t, "")
	err := s.Recover(t.Context(), 99, "owner", true)
	if errors.Is(err, ErrWorkerServing) || !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
}
