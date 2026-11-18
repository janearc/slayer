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
	stop := s.keepRenderLease(t.Context(), job, io.Discard)
	defer stop()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		state, err := s.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if state.Jobs[0].LeaseUntil.After(job.LeaseUntil) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("preparing worker did not renew the lease")
}
