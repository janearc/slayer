package queue

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

// TestGPUStartLimitLeavesCPUEligible bounds scene construction without serializing CPU.
func TestGPUStartLimitLeavesCPUEligible(t *testing.T) {
	s, request := fixture(t)
	for _, key := range []string{"one", "two"} {
		request.Key = key
		if _, err := s.Submit(t.Context(), request); err != nil {
			t.Fatal(err)
		}
	}
	request.Key, request.Lane = "cpu", "tests"
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	s, err := s.WithGPUJobs(1)
	if err != nil {
		t.Fatal(err)
	}
	sample := Capacity{Ready: true, GPUKnown: true, GPU: 25}
	now := time.Now()
	first, err := s.claimAvailable(t.Context(), "first", sample, now)
	if err != nil || first.Sequence != 1 {
		t.Fatal(first, err)
	}
	second, err := s.claimAvailable(t.Context(), "second", sample, now.Add(time.Minute))
	if err != nil || second.Sequence != 3 {
		t.Fatal(second, err)
	}
	if _, err := s.claimAvailable(t.Context(), "third", sample, now.Add(2*time.Minute)); !errors.Is(err, ErrActive) {
		t.Fatal(err)
	}
	for _, limit := range []int{-1, 0, 65} {
		if _, err := s.WithGPUJobs(limit); err == nil {
			t.Fatalf("accepted limit %d", limit)
		}
	}
}

// TestDispatchFillsGPUAndCPUIndependently catches global serialization.
func TestDispatchFillsGPUAndCPUIndependently(t *testing.T) {
	s, request := fixture(t)
	for _, item := range []struct{ key, lane string }{{"gpu-one", "gpu"}, {"gpu-two", "gpu"}, {"cpu", "tests"}} {
		request.Key, request.Lane = item.key, item.lane
		if _, err := s.Submit(t.Context(), request); err != nil {
			t.Fatal(err)
		}
	}
	controller := gomock.NewController(t)
	probe := NewMockCapacityProbe(controller)
	probe.EXPECT().Read(gomock.Any(), s.root).Return(Capacity{Ready: true, GPUKnown: true, GPU: 30}, nil).AnyTimes()
	executor := NewMockExecutor(controller)
	started := make(chan uint64, 3)
	release := make(chan struct{})
	defer close(release)
	executor.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, job Job, _ io.Writer) Result {
		started <- job.Sequence
		select {
		case <-release:
			return Result{State: Succeeded}
		case <-ctx.Done():
			return Result{Deferred: true, Reason: "test stopped"}
		}
	}).AnyTimes()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- s.Dispatch(ctx, executor, probe, false) }()
	defer func() { cancel(); <-finished }()
	seen := map[uint64]bool{}
	deadline := time.After(8 * time.Second)
	for len(seen) < 3 {
		select {
		case seq := <-started:
			seen[seq] = true
		case <-deadline:
			t.Fatalf("available resources left idle: started %v", seen)
		}
	}
}

// TestBusyGPUDoesNotBlockCPU proves a GPU hold cannot block unrelated work.
func TestUncertainExecutionDoesNotStopOtherResourceClasses(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	request.Key, request.Lane = "cpu-after-uncertain", "tests"
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	ctrl := gomock.NewController(t)
	probe := NewMockCapacityProbe(ctrl)
	probe.EXPECT().Read(gomock.Any(), s.root).Return(Capacity{Ready: true, GPUKnown: true, GPU: 35}, nil).AnyTimes()
	executor := NewMockExecutor(ctrl)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	executor.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, job Job, _ io.Writer) Result {
		if job.Sequence == 1 {
			if err := s.transact(t.Context(), false, func(state *Snapshot) error { state.Jobs[0].Executing = true; return nil }); err != nil {
				t.Fatal(err)
			}
			return Result{Uncertain: true, Reason: "supervisor receipt missing"}
		}
		cancel()
		return Result{State: Succeeded}
	}).Times(2)
	err := s.Dispatch(ctx, executor, probe, false)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("uncertain attempt stopped whole dispatcher", err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Running || state.Jobs[1].State != Succeeded {
		t.Fatal(state, err)
	}
}

// TestBusyGPUDoesNotBlockCPU proves a GPU hold cannot block unrelated work.
func TestBusyGPUDoesNotBlockCPU(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	request.Key, request.Lane = "cpu", "tests"
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	controller := gomock.NewController(t)
	probe := NewMockCapacityProbe(controller)
	probe.EXPECT().Read(gomock.Any(), s.root).Return(Capacity{Ready: true, GPUKnown: true, GPU: 100}, nil).AnyTimes()
	executor := NewMockExecutor(controller)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	executor.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, job Job, _ io.Writer) Result {
		if job.Sequence != 2 {
			t.Errorf("GPU admitted at full utilization: %d", job.Sequence)
		}
		cancel()
		return Result{State: Succeeded}
	}).Times(1)
	_ = s.Dispatch(ctx, executor, probe, false)
	snapshot, err := s.Status(t.Context())
	if err != nil || snapshot.Jobs[0].State != Queued || snapshot.Jobs[1].State != Succeeded {
		t.Fatal(snapshot, err)
	}
}
