package queue

import (
	"bytes"
	"context"
	"errors"
	"go.uber.org/mock/gomock"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestStatusNeverWritesLedger protects all new fields from read-side rewriting.
func TestStatusNeverWritesLedger(t *testing.T) {
	s, _ := fixture(t)
	if err := s.Tune(t.Context(), Tuning{GPUJobs: 4, Pressure: "observe"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.root, "queue.db")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("status wrote ledger", err)
	}
	if state.Version <= 1 {
		t.Fatal("new records still accepted by legacy writers", state.Version)
	}
}

// TestServingDispatcherWaitsForResume requires no worker restart after pause.
func TestServingDispatcherWaitsForResume(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if err := s.Pause(t.Context(), "operator inspection"); err != nil {
		t.Fatal(err)
	}
	ctrl := gomock.NewController(t)
	probe := NewMockCapacityProbe(ctrl)
	observed := make(chan struct{}, 1)
	probe.EXPECT().Read(gomock.Any(), s.root).DoAndReturn(func(context.Context, string) (Capacity, error) {
		select {
		case observed <- struct{}{}:
		default:
		}
		return Capacity{Ready: true, GPUKnown: true, GPU: 35}, nil
	}).AnyTimes()
	executor := NewMockExecutor(ctrl)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	executor.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, Job, interface{ Write([]byte) (int, error) }) Result {
		cancel()
		return Result{State: Succeeded}
	}).Times(1)
	done := make(chan error, 1)
	go func() { done <- s.Dispatch(ctx, executor, probe, true) }()
	<-observed
	select {
	case err := <-done:
		t.Fatal("serving worker exited on pause", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := s.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Succeeded {
		t.Fatal(state, err)
	}
}
