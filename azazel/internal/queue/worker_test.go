package queue

import (
	"context"
	"errors"
	"io"
	"testing"

	"go.uber.org/mock/gomock"
)

// expectedExecution requires the selected job identity at the process boundary.
func expectedExecution(t *testing.T, sequence uint64, result Result) *MockExecutor {
	t.Helper()
	executor := NewMockExecutor(gomock.NewController(t))
	executor.EXPECT().Run(gomock.Any(), gomock.Cond(func(job Job) bool {
		return job.Sequence == sequence && job.State == Running && job.Worker != ""
	}), gomock.Any()).Return(result)
	return executor
}

// TestWorkerDrainsInOrderAndKeepsFailures exercises the durable worker loop.
func TestWorkerDrainsInOrderAndKeepsFailures(t *testing.T) {
	s, r := fixture(t)
	for _, key := range []string{"one", "two", "three"} {
		r.Key = key
		if _, err := s.Submit(t.Context(), r); err != nil {
			t.Fatal(err)
		}
	}
	executor := NewMockExecutor(gomock.NewController(t))
	var previous *gomock.Call
	for sequence := uint64(1); sequence <= 3; sequence++ {
		call := executor.EXPECT().Run(gomock.Any(), gomock.Cond(func(job Job) bool {
			return job.Sequence == sequence && job.State == Running && job.Worker != ""
		}), gomock.Any()).Return(Result{State: Failed, ExitCode: 3, Reason: "unsupported feature"})
		if previous != nil {
			call.After(previous)
		}
		previous = call
	}
	if err := s.Work(t.Context(), executor, false); err != nil {
		t.Fatal(err)
	}
	status, err := s.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range status.Jobs {
		if j.State != Failed || j.Result.ExitCode != 3 {
			t.Fatal(j)
		}
	}
	if err = s.Work(t.Context(), executor, false); err != nil {
		t.Fatal(err)
	}
}

// TestUncertainCompletionFencesTheQueue refuses to spend abandoned capacity.
func TestUncertainCompletionFencesTheQueue(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	r.Key = "next"
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	executor := expectedExecution(t, 1, Result{Uncertain: true, Reason: "worker vanished"})
	if err := s.Work(t.Context(), executor, false); !errors.Is(err, ErrActive) {
		t.Fatal(err)
	}
	if err := s.Work(t.Context(), executor, false); !errors.Is(err, ErrActive) {
		t.Fatal(err)
	}
	status, err := s.Status(t.Context())
	if err != nil || status.Jobs[0].State != Running || status.Jobs[1].State != Queued {
		t.Fatal(status, err)
	}
	if err = s.Recover(t.Context(), 1, status.Jobs[0].Worker, false); err == nil {
		t.Fatal("recovered without confirmation")
	}
	if err = s.Recover(t.Context(), 1, status.Jobs[0].Worker, true); err != nil {
		t.Fatal(err)
	}
}

// TestAnotherWorkerCannotStartWhileTheFirstOwnsExecution tests the real lock.
func TestAnotherWorkerCannotStartWhileTheFirstOwnsExecution(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	executor := NewMockExecutor(gomock.NewController(t))
	executor.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, j Job, _ io.Writer) Result {
		if err := s.Work(t.Context(), NewMockExecutor(gomock.NewController(t)), false); !errors.Is(err, ErrActive) {
			t.Fatal(err)
		}
		if err := s.Recover(t.Context(), j.Sequence, j.Worker, true); !errors.Is(err, ErrActive) {
			t.Fatal(err)
		}
		return Result{State: Succeeded}
	})
	if err := s.Work(t.Context(), executor, false); err != nil {
		t.Fatal(err)
	}
}
