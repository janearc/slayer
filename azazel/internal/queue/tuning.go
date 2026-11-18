package queue

import (
	"context"
	"fmt"
	"github.com/janearc/slayer/azazel/internal/budget"
)

// Tuning is durable control reread by running coordinators and supervisors.
type Tuning struct {
	GPUJobs         int    `json:"gpu_jobs,omitempty"`
	GPUStartSeconds int    `json:"gpu_start_seconds,omitempty"`
	Pressure        string `json:"pressure,omitempty"`
}

// valid reports whether every control is within its allowed range.
func (t Tuning) valid() bool {
	return t.GPUJobs >= 0 && t.GPUJobs <= maximumGPUJobs && t.GPUStartSeconds >= 0 && t.GPUStartSeconds <= 3600 && (t.Pressure == "" || t.Pressure == "stop" || t.Pressure == "observe")
}

// ResetTuning restores worker defaults without changing active attempts.
func (s Store) ResetTuning(ctx context.Context) error {
	return s.transact(ctx, false, func(state *Snapshot) error { state.Tuning = Tuning{}; return nil })
}

// Tune merges supplied controls atomically without changing requests or attempts.
func (s Store) Tune(ctx context.Context, change Tuning) error {
	if !change.valid() || change == (Tuning{}) {
		return fmt.Errorf("invalid or empty tuning")
	}
	return s.transact(ctx, false, func(state *Snapshot) error {
		if change.GPUJobs != 0 {
			state.Tuning.GPUJobs = change.GPUJobs
		}
		if change.GPUStartSeconds != 0 {
			state.Tuning.GPUStartSeconds = change.GPUStartSeconds
		}
		if change.Pressure != "" {
			state.Tuning.Pressure = change.Pressure
		}
		return nil
	})
}

// runtimeVitals overrides only the explicitly selected memory-pressure decision.
func (s Store) runtimeVitals(ctx context.Context, vitals Vitals) (Vitals, error) {
	state, err := s.Status(ctx)
	if err != nil {
		return Vitals{}, err
	}
	if state.Tuning.Pressure == "observe" && vitals.Cause == budget.MemoryPressureStop && vitals.Stop {
		vitals.Ready, vitals.Stop = true, false
	}
	return vitals, nil
}
