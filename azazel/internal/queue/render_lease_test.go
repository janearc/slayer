package queue

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
	"time"
)

func TestNewRenderQueueDoesNotNeedGame(t *testing.T) {
	root := filepath.Join(t.TempDir(), "queue")
	s := At(root)
	if err := s.Init(t.Context(), Config{}); err != nil {
		t.Fatal(err)
	}
	job, err := s.Submit(t.Context(), Request{Agent: "test", Key: "one", Command: "/usr/bin/true", Directory: t.TempDir(), Lane: "gpu", TimeoutSeconds: 60})
	if err != nil || len(job.Files) != 1 {
		t.Fatal(job, err)
	}
	if err := job.verify(); err != nil {
		t.Fatal(err)
	}
}

// TestRenderLeaseIsRenewedByItsOwner fences stale attempts and persists renewal.
func TestRenderLeaseIsRenewedByItsOwner(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, now)
	if err != nil || !job.LeaseUntil.After(now) {
		t.Fatal(job, err)
	}
	if err := s.renewRenderLease(t.Context(), job, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	state, err := At(s.root).Status(t.Context())
	if err != nil || !state.Jobs[0].LeaseUntil.After(job.LeaseUntil) {
		t.Fatal(state, err)
	}
	if err := s.renewRenderLease(t.Context(), job, now.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	afterClockStep, err := s.Status(t.Context())
	if err != nil || !afterClockStep.Jobs[0].LeaseUntil.Equal(state.Jobs[0].LeaseUntil) {
		t.Fatal("clock moved the lease deadline backward", afterClockStep, err)
	}
	stale := job
	stale.Worker = "another-owner"
	if err := s.renewRenderLease(t.Context(), stale, now.Add(2*time.Second)); !errors.Is(err, ErrOwner) {
		t.Fatal("another owner renewed the lease", err)
	}
	stale = job
	stale.Attempt++
	if err := s.renewRenderLease(t.Context(), stale, now.Add(2*time.Second)); !errors.Is(err, ErrOwner) {
		t.Fatal("another attempt renewed the lease", err)
	}
	if err := s.finish(t.Context(), job.Sequence, job.Worker, Result{State: Succeeded}); err != nil {
		t.Fatal(err)
	}
	if err := s.renewRenderLease(t.Context(), job, now.Add(3*time.Second)); !errors.Is(err, ErrOwner) {
		t.Fatal("finished work renewed the lease", err)
	}
}

// TestExpiredRenderLeaseIsNotRevived keeps an expired grant from being reused.
func TestExpiredRenderLeaseIsNotRevived(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.renewRenderLease(t.Context(), job, job.LeaseUntil.Add(time.Second)); !errors.Is(err, ErrExpiredLease) {
		t.Fatal("expired lease was revived", err)
	}
}

func TestExpiredClaimCannotStartRender(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, time.Now().Add(-2*renderLeaseDuration))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.executeSelectedMode(t.Context(), job.Sequence, job.Worker, io.Discard, nil, false); !errors.Is(err, ErrExpiredLease) {
		t.Fatal("expired claim could start a render", err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].Executing {
		t.Fatal(state, err)
	}
}

// An expired deadline alone never proves the old render stopped.
func TestExpiredLeaseDoesNotReplayLiveRender(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := s.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if err := s.transact(t.Context(), false, func(state *Snapshot) error {
		state.Jobs[0].LeaseUntil = now.Add(-time.Second)
		state.Jobs[0].Executing = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcile(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Running || state.Jobs[0].Attempt != 1 {
		t.Fatal("live render was reclaimed", state, err)
	}
}

func TestPreparingWorkerRenewsRenderLease(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time)
	stop := s.keepLeaseOnTicks(t.Context(), job, io.Discard, ticks)
	defer stop()
	// The holder takes a second tick only after it finished the first.
	ticks <- job.LeaseUntil.Add(-renderLeaseDuration / 2)
	ticks <- job.LeaseUntil.Add(-renderLeaseDuration / 2)
	state, err := s.Status(t.Context())
	if err != nil || !state.Jobs[0].LeaseUntil.After(job.LeaseUntil) {
		t.Fatal("preparing worker did not renew the lease", state, err)
	}
	renewed := state.Jobs[0].LeaseUntil
	ticks <- job.LeaseUntil.Add(-time.Hour)
	ticks <- job.LeaseUntil.Add(-time.Hour)
	state, err = s.Status(t.Context())
	if err != nil || !state.Jobs[0].LeaseUntil.Equal(renewed) {
		t.Fatal("a clock stepped back moved the deadline", state, err)
	}
}

// A lease that a forward clock step expired is renewed while the attempt
// lock is held, because the render holding it lives.
func TestHeldAttemptRenewsALeaseExpiredByAClockStep(t *testing.T) {
	s, job := claimedFixture(t)
	lock, err := s.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	stepped := job.LeaseUntil.Add(time.Hour)
	if err := s.renewHeldRenderLease(t.Context(), job, stepped); err != nil {
		t.Fatal("held attempt lost its lease to a clock step", err)
	}
	state, err := s.Status(t.Context())
	if err != nil || !state.Jobs[0].LeaseUntil.After(stepped) {
		t.Fatal(state, err)
	}
}

// Anti-test: with no lock held nothing lives to keep, so the expired lease
// stays expired, and the strict renewal before launch never revives one.
func TestExpiredLeaseIsRefusedWithoutAHeldLockOrBeforeLaunch(t *testing.T) {
	s, job := claimedFixture(t)
	stepped := job.LeaseUntil.Add(time.Hour)
	if err := s.renewHeldRenderLease(t.Context(), job, stepped); !errors.Is(err, ErrExpiredLease) {
		t.Fatal("a lease with no held lock was revived", err)
	}
	lock, err := s.attemptLock(job)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := s.renewRenderLease(t.Context(), job, stepped); !errors.Is(err, ErrExpiredLease) {
		t.Fatal("the strict renewal revived an expired lease", err)
	}
}

// claimedFixture claims the fixture job with a fresh lease.
func claimedFixture(t *testing.T) (Store, Job) {
	t.Helper()
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	job, err := s.claimAvailable(t.Context(), "owner", Capacity{Ready: true, GPUKnown: true}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return s, job
}

// A start time in the future, left by a clock that stepped back, does not
// hold GPU admission; a start inside the interval still does.
func TestGPUSpacingIgnoresAStartTimeInTheFuture(t *testing.T) {
	for name, c := range map[string]struct {
		offset time.Duration
		admits bool
	}{
		"an hour ahead":       {time.Hour, true},
		"inside the interval": {-time.Second, false},
		"past the interval":   {-2 * gpuAdmissionInterval, true},
	} {
		t.Run(name, func(t *testing.T) {
			s, request := fixture(t)
			now := time.Now().UTC()
			capacity := Capacity{Ready: true, GPUKnown: true}
			if _, err := s.Submit(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			first, err := s.claimAvailable(t.Context(), "owner", capacity, now.Add(c.offset))
			if err != nil {
				t.Fatal(err)
			}
			if err := s.finish(t.Context(), first.Sequence, first.Worker, Result{State: Succeeded}); err != nil {
				t.Fatal(err)
			}
			request.Key = "two"
			if _, err := s.Submit(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			_, err = s.claimAvailable(t.Context(), "owner", capacity, now)
			if c.admits != (err == nil) {
				t.Fatal("admission differs from the rule", err)
			}
		})
	}
}
