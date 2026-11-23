package queue

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/janearc/slayer/azazel/internal/budget"
	"strings"
	"sync"
	"time"
)

// Capacity names the sampled GPU counter and keeps host admission separate.
type Capacity struct {
	Cause    budget.StopCause
	Pressure string
	Ready    bool
	Stop     bool
	Reason   string
	GPU      int
	GPUKnown bool
}

// CapacityProbe supplies current host pressure and device utilization.
type CapacityProbe interface {
	Read(context.Context, string) (Capacity, error)
}

// Scheduling reports the capacity decision, including holds, to every client.
type Scheduling struct {
	Observed   time.Time `json:"observed"`
	GPUCounter string    `json:"gpu_counter"`
	GPUPercent int       `json:"gpu_percent"`
	GPUKnown   bool      `json:"gpu_known"`
	Target     int       `json:"gpu_target"`
	GPUActive  int       `json:"gpu_active"`
	GPULimit   int       `json:"gpu_limit"`
	CPUActive  int       `json:"cpu_active"`
	Held       string    `json:"held,omitempty"`
}

// These bounds limit in-flight staging while pressure measurements catch up.
const maximumGPUJobs = 64
const defaultGPUJobs = 4
const maximumCPUJobs = 2
const gpuAdmissionInterval = 5 * time.Second
const renderLeaseDuration = 30 * time.Second

// WithGPUJobs bounds concurrent GPU scene construction for this worker.
func (s Store) WithGPUJobs(limit int) (Store, error) {
	if limit < 1 || limit > maximumGPUJobs {
		return Store{}, fmt.Errorf("GPU jobs must be between 1 and %d", maximumGPUJobs)
	}
	s.gpuJobs = limit
	return s, nil
}

// gpuLimit preserves the default for callers without an explicit setting.
func (s Store) gpuLimit() int {
	if s.gpuJobs == 0 {
		return defaultGPUJobs
	}
	return s.gpuJobs
}

type dispatched struct {
	sequence uint64
	err      error
}

type dispatchMode int

const (
	allWork dispatchMode = iota
	cpuWork
	gpuWork
)

// accepts reports whether mode may start job. GPU work takes the GPU lane,
// CPU work takes every other lane, and all work takes both.
func (mode dispatchMode) accepts(job Job) bool {
	return mode == allWork || mode == gpuWork && job.Request.Lane == "gpu" || mode == cpuWork && job.Request.Lane != "gpu"
}

// Dispatch fills independently available slots and adopts surviving supervisors.
func (s Store) Dispatch(ctx context.Context, executor Executor, probe CapacityProbe, serve bool) error {
	return s.dispatch(ctx, executor, probe, serve, allWork, nil)
}

// dispatch holds one coordinator lock for its resource class.
func (s Store) dispatch(ctx context.Context, executor Executor, probe CapacityProbe, serve bool, mode dispatchMode, tick func(context.Context) error) error {
	if executor == nil || probe == nil {
		return ErrUnsafe
	}
	name := "worker.lock"
	if mode == gpuWork {
		name = "gpu-worker.lock"
	}
	lease, err := s.lockFile(name)
	if err != nil {
		return err
	}
	defer lease.Close()
	runCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() { cancel(); workers.Wait() }()
	finished := make(chan dispatched, maximumGPUJobs+maximumCPUJobs)
	owned := map[uint64]bool{}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.reconcileMode(ctx, owned, mode); err != nil {
			return err
		}
		if tick != nil {
			if err := tick(ctx); err != nil {
				return err
			}
		}
		capacity, err := probe.Read(ctx, s.root)
		if err != nil {
			capacity = Capacity{Reason: err.Error()}
		}
		if capacity.Stop {
			adjusted, err := s.runtimeVitals(ctx, Vitals{Ready: capacity.Ready, Stop: capacity.Stop, Reason: capacity.Reason, Pressure: capacity.Pressure, Cause: capacity.Cause})
			if err != nil {
				return err
			}
			capacity.Ready, capacity.Stop = adjusted.Ready, adjusted.Stop
		}
		if capacity.Stop {
			if err := s.Pause(ctx, capacity.Reason); err != nil {
				return err
			}
		}
		job, err := s.claimMode(ctx, rand.Text(), capacity, time.Now(), mode)
		switch {
		case err == nil:
			owned[job.Sequence] = true
			workers.Add(1)
			go func() {
				defer workers.Done()
				result := dispatched{job.Sequence, s.run(runCtx, executor, job)}
				select {
				case finished <- result:
				case <-runCtx.Done():
				}
			}()
		case errors.Is(err, ErrEmpty):
			if !serve && len(owned) == 0 {
				return nil
			}
		case errors.Is(err, ErrActive):
		case errors.Is(err, ErrPaused):
			if !serve {
				state, err := s.Status(ctx)
				if err != nil {
					return err
				}
				active := false
				for _, job := range state.Jobs {
					if job.State == Running && mode.accepts(job) {
						active = true
					}
				}
				if !active {
					return nil
				}
			}
		default:
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result := <-finished:
			delete(owned, result.sequence)
			if result.err != nil && !errors.Is(result.err, ErrActive) {
				return result.err
			}
		case <-ticker.C:
		}
	}
}

// reconcile requeues unstarted attempts and observes children of a prior worker.
func (s Store) reconcile(ctx context.Context, owned map[uint64]bool) error {
	return s.reconcileMode(ctx, owned, allWork)
}

// reconcileMode leaves other coordinators' unstarted claims alone.
func (s Store) reconcileMode(ctx context.Context, owned map[uint64]bool, mode dispatchMode) error {
	state, err := s.Status(ctx)
	if err != nil {
		return err
	}
	for _, job := range state.Jobs {
		if job.State != Running || owned[job.Sequence] || !mode.accepts(job) {
			continue
		}
		lease, err := s.attemptLock(job)
		if errors.Is(err, ErrActive) {
			if watched := s.watchHeldAttempt(ctx, job, time.Now()); watched != nil && !errors.Is(watched, ErrOwner) {
				return watched
			}
			continue
		}
		if err != nil {
			return err
		}
		lease.Close()
		if err := s.recoverAttempt(ctx, job.Sequence, job.Worker, false); err != nil {
			if errors.Is(err, ErrOwner) {
				continue
			}
			if errors.Is(err, ErrActive) {
				if held := s.hold(ctx, job, "recovery: "+err.Error()); held != nil && !errors.Is(held, ErrOwner) {
					return held
				}
				continue
			}
			return err
		}
	}
	return nil
}

// claimAvailable selects the oldest eligible request without blocking CPU on GPU.
func (s Store) claimAvailable(ctx context.Context, owner string, sample Capacity, now time.Time) (Job, error) {
	return s.claimMode(ctx, owner, sample, now, allWork)
}

// claimMode shares one ledger while each coordinator owns only its resource class.
func (s Store) claimMode(ctx context.Context, owner string, sample Capacity, now time.Time, mode dispatchMode) (Job, error) {
	var selected Job
	decision := error(ErrActive)
	err := s.transact(ctx, false, func(state *Snapshot) error {
		defer func() {
			if state.Scheduling == nil {
				return
			}
			for _, job := range state.Jobs {
				if job.State == Running && strings.HasPrefix(job.Held, "recovery:") {
					if state.Scheduling.Held != "" {
						state.Scheduling.Held += "; "
					}
					state.Scheduling.Held += fmt.Sprintf("job %d %s", job.Sequence, job.Held)
				}
			}
		}()
		if state.Paused != "" {
			return ErrPaused
		}
		gpu, cpu, waiting, relevant := 0, 0, 0, 0
		var lastGPU time.Time
		for _, job := range state.Jobs {
			if (job.State == Queued || job.State == Running) && (mode != gpuWork || mode.accepts(job)) {
				relevant++
			}
			// A start time after now came from a clock that has since
			// stepped back; it is long enough ago, not an hour ahead.
			if job.Request.Lane == "gpu" && job.Started.After(lastGPU) && !job.Started.After(now) {
				lastGPU = job.Started
			}
			if job.State == Queued {
				waiting++
			}
			if job.State != Running {
				continue
			}
			if job.Request.Lane == "gpu" {
				gpu++
			} else {
				cpu++
			}
		}
		state.Scheduling = &Scheduling{Observed: now, GPUCounter: "Device Utilization %", GPUPercent: sample.GPU, GPUKnown: sample.GPUKnown, Target: 100, GPUActive: gpu, GPULimit: s.gpuLimit(), CPUActive: cpu, Held: sample.Reason}
		limit, interval := s.gpuLimit(), gpuAdmissionInterval
		if state.Tuning.GPUJobs != 0 {
			limit = state.Tuning.GPUJobs
		}
		if state.Tuning.GPUStartSeconds != 0 {
			interval = time.Duration(state.Tuning.GPUStartSeconds) * time.Second
		}
		state.Scheduling.GPULimit = limit
		if relevant == 0 {
			decision = ErrEmpty
			return nil
		}
		if !sample.Ready {
			return nil
		}
		gpuRoom := sample.GPUKnown && sample.GPU >= 0 && sample.GPU < 100 && gpu < limit && now.Sub(lastGPU) >= interval
		for i := range state.Jobs {
			job := &state.Jobs[i]
			if job.State != Queued || !mode.accepts(*job) {
				continue
			}
			if job.Request.Lane == "gpu" && !gpuRoom || job.Request.Lane != "gpu" && cpu >= maximumCPUJobs {
				continue
			}
			job.State, job.Worker, job.Started = Running, owner, now.UTC()
			job.Attempt++
			job.LeaseUntil = now.UTC().Add(renderLeaseDuration)
			job.Executing, job.ProcessGroup, job.Held = false, 0, ""
			job.ProcessStartSeconds, job.ProcessStartMicroseconds, job.ProcessBootID = 0, 0, ""
			selected, decision = *job, nil
			if job.Request.Lane == "gpu" {
				state.Scheduling.GPUActive++
			} else {
				state.Scheduling.CPUActive++
			}
			return nil
		}
		if waiting > 0 {
			state.Scheduling.Held = "waiting for GPU utilization, staging interval, or free CPU/GPU slots"
		}
		return nil
	})
	if err != nil {
		return Job{}, err
	}
	return selected, decision
}
