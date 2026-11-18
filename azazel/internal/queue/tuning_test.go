package queue

import (
	"context"
	"errors"
	"github.com/janearc/slayer/azazel/internal/budget"
	"go.uber.org/mock/gomock"
	"io"
	"testing"
	"time"
)

// TestObservePolicyKeepsActualCommandAlive checks the supervisor, not just policy data.
func TestObservePolicyKeepsActualCommandAlive(t *testing.T) {
	for _, restoreStop := range []bool{false, true} {
		t.Run(map[bool]string{false: "observe", true: "restore-stop"}[restoreStop], func(t *testing.T) {
			s, job := executableFixture(t, "exec /bin/sleep 0.5")
			if err := s.Tune(t.Context(), Tuning{Pressure: "observe"}); err != nil {
				t.Fatal(err)
			}
			monitor := unreadHost(t)
			monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
			reads := 0
			monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).DoAndReturn(func(ctx context.Context, _ string, _ int) (Vitals, error) {
				reads++
				if restoreStop && reads == 2 {
					if err := s.Tune(ctx, Tuning{Pressure: "stop"}); err != nil {
						t.Fatal(err)
					}
				}
				return Vitals{Stop: true, Cause: budget.MemoryPressureStop, Pressure: "elevated", Reason: "memory pressure wording changed"}, nil
			}).MinTimes(2)
			result := s.execute(t.Context(), job, io.Discard, monitor)
			if restoreStop {
				if result.State != Failed || !result.Stop {
					t.Fatal(result)
				}
			} else if result.State != Succeeded {
				t.Fatal(result)
			}
			state, err := s.Status(t.Context())
			if err != nil || state.Jobs[0].MemoryPressure != "elevated" {
				t.Fatal(state, err)
			}
		})
	}
}

// TestLiveTuningChangesAdmissionWithoutRestart preserves existing attempts on reductions.
func TestLiveTuningChangesAdmissionWithoutRestart(t *testing.T) {
	s, request := fixture(t)
	for _, key := range []string{"one", "two", "three"} {
		request.Key = key
		if _, err := s.Submit(t.Context(), request); err != nil {
			t.Fatal(err)
		}
	}
	s, _ = s.WithGPUJobs(1)
	now := time.Now()
	sample := Capacity{Ready: true, GPUKnown: true, GPU: 35}
	first, err := s.claimAvailable(t.Context(), "one", sample, now)
	if err != nil {
		t.Fatal(err)
	}
	other := At(s.root)
	if err := other.Tune(t.Context(), Tuning{GPUJobs: 4, GPUStartSeconds: 1, Pressure: "observe"}); err != nil {
		t.Fatal(err)
	}
	second, err := s.claimAvailable(t.Context(), "two", sample, now.Add(time.Second))
	if err != nil || second.Sequence != 2 {
		t.Fatal(second, err)
	}
	if err := other.Tune(t.Context(), Tuning{GPUJobs: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.claimAvailable(t.Context(), "three", sample, now.Add(time.Minute)); !errors.Is(err, ErrActive) {
		t.Fatal(err)
	}
	state, err := other.Status(t.Context())
	if err != nil || state.Tuning.Pressure != "observe" || state.Tuning.GPUStartSeconds != 1 || state.Jobs[0].State != Running || state.Jobs[1].State != Running || state.Jobs[2].State != Queued || state.Scheduling.GPULimit != 1 {
		t.Fatal(state, err)
	}
	if err := s.finish(t.Context(), first.Sequence, first.Worker, Result{State: Succeeded}); err != nil {
		t.Fatal(err)
	}
}

// TestLivePressurePolicyIsSpecific preserves disk stops and reacts without restart.
func TestResetTuningRestoresWorkerDefaultsWithoutTouchingAttempts(t *testing.T) {
	s, r := fixture(t)
	for _, key := range []string{"one", "two"} {
		r.Key = key
		if _, err := s.Submit(t.Context(), r); err != nil {
			t.Fatal(err)
		}
	}
	s, _ = s.WithGPUJobs(2)
	if err := s.Tune(t.Context(), Tuning{GPUJobs: 1, Pressure: "observe", GPUStartSeconds: 1}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	sample := Capacity{Ready: true, GPUKnown: true, GPU: 35}
	first, err := s.claimAvailable(t.Context(), "first", sample, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.claimAvailable(t.Context(), "second", sample, now.Add(time.Minute)); !errors.Is(err, ErrActive) {
		t.Fatal(err)
	}
	if err := s.ResetTuning(t.Context()); err != nil {
		t.Fatal(err)
	}
	second, err := s.claimAvailable(t.Context(), "second", sample, now.Add(time.Minute))
	if err != nil || second.Sequence != 2 {
		t.Fatal(second, err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Tuning != (Tuning{}) || state.Jobs[0].Worker != first.Worker || state.Jobs[0].Attempt != first.Attempt || state.Jobs[0].State != Running {
		t.Fatal(state, err)
	}
}

func TestLivePressurePolicyIsSpecific(t *testing.T) {
	s, _ := fixture(t)
	elevated := Vitals{Stop: true, Cause: budget.MemoryPressureStop, Pressure: "elevated", Reason: "memory pressure wording changed"}
	for _, policy := range []string{"observe", "stop", "observe"} {
		if err := s.Tune(t.Context(), Tuning{Pressure: policy}); err != nil {
			t.Fatal(err)
		}
		got, err := s.runtimeVitals(t.Context(), elevated)
		if err != nil || got.Stop != (policy == "stop") || got.Ready != (policy == "observe") || got.Pressure != "elevated" {
			t.Fatal(got, err)
		}
		disk := Vitals{Stop: true, Cause: budget.DiskSpaceStop, Pressure: "elevated", Reason: "free disk below 30 GiB"}
		got, err = s.runtimeVitals(t.Context(), disk)
		if err != nil || got != disk {
			t.Fatal(got, err)
		}
	}
	for _, invalid := range []Tuning{{}, {GPUJobs: -1}, {GPUJobs: 65}, {Pressure: "ignore"}, {GPUStartSeconds: -1}} {
		if err := s.Tune(t.Context(), invalid); err == nil {
			t.Fatalf("accepted %+v", invalid)
		}
	}
}
