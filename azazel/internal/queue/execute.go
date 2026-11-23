package queue

import (
	"context"
	"errors"
	"fmt"
	"github.com/janearc/slayer/azazel/internal/budget"
	"github.com/janearc/slayer/azazel/internal/footprint"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Vitals separates a host hold from a stop and measured group residency.
type Vitals struct {
	Cause    budget.StopCause
	Ready    bool
	Stop     bool
	Reason   string
	Resident uint64
	Pressure string
}

// Monitor reads the selected output filesystem and the owned process group.
type Monitor interface {
	Read(context.Context, string, int) (Vitals, error)
}

// environment carries rendering settings and the job's CPU worker count.
func environment(r Request) []string {
	// The worker's listeners describe the worker, not each renderer. A job
	// may explicitly request separate addresses in its own environment.
	env := slices.DeleteFunc(os.Environ(), func(item string) bool {
		return strings.HasPrefix(item, "SLAYER_METRICS_ADDR=") || strings.HasPrefix(item, "SLAYER_PPROF_ADDR=") ||
			strings.HasPrefix(item, "AZAZEL_FRAME_FD=") || strings.HasPrefix(item, "AZAZEL_FRAME_NONCE=") ||
			strings.HasPrefix(item, "AZAZEL_FRAME_SEQUENCE=") || strings.HasPrefix(item, "AZAZEL_FRAME_ATTEMPT=")
	})
	for key, value := range r.Environment {
		env = setEnvironment(env, key, value)
	}
	workers := r.Environment["SLAYER_CPU_WORKERS"]
	if workers == "" {
		workers = "8"
	}
	env = setEnvironment(env, "GOMAXPROCS", workers)
	env = slices.DeleteFunc(env, func(item string) bool { return strings.HasPrefix(item, "GOMEMLIMIT=") })
	env = setEnvironment(env, "SLAYER_TEAM", r.Agent)
	return env
}

// setEnvironment replaces rather than duplicates a named environment variable.
func setEnvironment(env []string, key, value string) []string {
	env = slices.DeleteFunc(env, func(s string) bool { return strings.HasPrefix(s, key+"=") })
	return append(env, key+"="+value)
}

// prepare observes a hold without occupying or retaining a Game lane.
func (s Store) prepare(ctx context.Context, j Job, monitor Monitor) (Result, bool) {
	fail := func(err error) (Result, bool) {
		if ctx.Err() != nil {
			return Result{Deferred: true, Reason: "worker stopped before execution"}, false
		}
		return Result{State: Failed, ExitCode: -1, Reason: err.Error()}, false
	}
	if err := ctx.Err(); err != nil {
		return Result{Deferred: true, Reason: "worker stopped before execution"}, false
	}
	cancelled, paused, err := s.requested(ctx, j)
	if err != nil {
		return fail(err)
	}
	if cancelled {
		return Result{State: Cancelled, ExitCode: -1, Reason: "cancelled before execution"}, false
	}
	if paused != "" {
		return Result{Deferred: true, Stop: true, Reason: paused}, false
	}
	vitals, err := monitor.Read(ctx, j.Request.Directory, 0)
	if err == nil {
		vitals, err = s.runtimeVitals(ctx, vitals)
	}
	if err != nil {
		vitals.Reason = err.Error()
		vitals.Ready = false
	}
	if err = s.observe(ctx, j, vitals); err != nil {
		return fail(err)
	}
	if err = s.hold(ctx, j, vitals.Reason); err != nil {
		return fail(err)
	}
	if vitals.Stop || !vitals.Ready {
		return Result{Deferred: true, Stop: vitals.Stop, Reason: vitals.Reason}, false
	}
	return Result{}, true
}

// execute monitors one foreground group inside its parent's held Game lane.
func (s Store) execute(ctx context.Context, j Job, out io.Writer, monitor Monitor) Result {
	lease, err := s.attemptLock(j)
	if err != nil {
		return Result{Uncertain: true, Reason: err.Error()}
	}
	defer lease.Close()
	return s.executeUnderLock(ctx, j, out, monitor, lease)
}

// executeUnderLock retains attempt ownership across the process-start window.
func (s Store) executeUnderLock(ctx context.Context, j Job, out io.Writer, monitor Monitor, lease *os.File) Result {
	return s.executeUnderLockWithLeader(ctx, j, out, monitor, lease, func(pid int) (footprint.Leader, bool) {
		return footprint.CaptureLeader(pid, footprint.Native(), footprint.BootID)
	})
}

// executeUnderLockWithLeader runs job under its lease, capturing the process
// group leader with capture and using the standard frame receipt setup.
func (s Store) executeUnderLockWithLeader(ctx context.Context, j Job, out io.Writer, monitor Monitor, lease *os.File, capture func(int) (footprint.Leader, bool)) Result {
	return s.executeUnderLockWithObservers(ctx, j, out, monitor, lease, capture, newFrameReceipt)
}

// executeUnderLockWithObservers runs job under its lease with the given leader
// capture and frame receipt setup.
func (s Store) executeUnderLockWithObservers(ctx context.Context, j Job, out io.Writer, monitor Monitor, lease *os.File, capture func(int) (footprint.Leader, bool), setup func(string, Job) (*frameReceipt, error)) Result {
	failure := func(err error) Result { return Result{State: Failed, ExitCode: -1, Reason: err.Error()} }
	if err := j.verify(); err != nil {
		return failure(err)
	}
	result, ready := s.prepare(ctx, j, monitor)
	if !ready {
		return result
	}
	if err := j.verify(); err != nil {
		return failure(err)
	}
	if !j.LeaseUntil.IsZero() {
		if err := s.renewRenderLease(ctx, j, time.Now()); err != nil {
			return Result{Deferred: true, Reason: "render lease before launch: " + err.Error()}
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(j.Request.TimeoutSeconds)*time.Second)
	defer cancel()
	command := exec.Command(j.Request.Command, j.Request.Args...)
	command.Dir = j.Request.Directory
	command.Env = environment(j.Request)
	command.Stdout, command.Stderr = out, out
	command.ExtraFiles = []*os.File{lease}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var frames *frameReceipt
	if observeFrames(j.Request) {
		frames = frameReceiptWithin(runCtx, s.root, j, setup)
		if frames != nil {
			command.ExtraFiles = append(command.ExtraFiles, frames.child)
			command.Env = frames.environment(command.Env)
			defer frames.close()
		}
	}
	if runCtx.Err() != nil {
		state := Cancelled
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			state = Failed
		}
		return Result{State: state, ExitCode: -1, Reason: runCtx.Err().Error()}
	}
	if err := command.Start(); err != nil {
		return failure(err)
	}
	if frames != nil {
		frames.start()
	}
	// Bound this optional observation before Wait can reap and recycle the PID.
	// A late result is discarded and cannot change the attempt record.
	leader, leaderKnown := captureWithin(runCtx, command.Process.Pid, capture)
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	if runCtx.Err() != nil {
		state := Cancelled
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			state = Failed
		}
		return stopGroup(command.Process.Pid, done, Result{State: state, ExitCode: -1, Reason: runCtx.Err().Error()})
	}
	if err := s.transact(ctx, false, func(st *Snapshot) error {
		current, err := st.job(j.Sequence)
		if err != nil {
			return err
		}
		if current.Worker != j.Worker || current.State != Running {
			return ErrOwner
		}
		recordLeader(current, command.Process.Pid, leader, leaderKnown)
		current.Executing = true
		return nil
	}); err != nil {
		return stopGroup(command.Process.Pid, done, failure(err))
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	nextRenewal := time.Now().Add(5 * time.Second)
	var failures readFailures
	for {
		select {
		case err := <-done:
			return completion(command.Process.Pid, err)
		case <-runCtx.Done():
			state := Cancelled
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				state = Failed
			}
			return stopGroup(command.Process.Pid, done, Result{State: state, ExitCode: -1, Reason: runCtx.Err().Error()})
		case <-ticker.C:
			if !j.LeaseUntil.IsZero() && !time.Now().Before(nextRenewal) {
				now := time.Now()
				if err := s.renewHeldRenderLease(runCtx, j, now); err != nil {
					fmt.Fprintf(out, "render lease renewal: %v\n", err)
				}
				nextRenewal = now.Add(5 * time.Second)
			}
			verdict, stop, err := s.supervisionTick(runCtx, j, monitor, command.Process.Pid)
			if err != nil {
				if !readMayRetry(runCtx, err) {
					return stopGroup(command.Process.Pid, done, failure(err))
				}
				now := time.Now()
				if failures.spent(now, s.readAllowance()) {
					explained := failures.explain(err, now)
					return stopGroup(command.Process.Pid, done, failure(explained))
				}
				continue
			}
			failures.clear()
			if stop {
				return stopGroup(command.Process.Pid, done, verdict)
			}
		}
	}
}

// observeFrames is request-local: inherited worker environment cannot opt in.
func observeFrames(request Request) bool { return request.Environment["SLAYER_OBSERVE_FRAMES"] == "on" }

// frameReceiptWithin runs setup in its own goroutine and returns the receipt.
// A receipt that arrives after the caller stops waiting is closed.
func frameReceiptWithin(ctx context.Context, root string, job Job, setup func(string, Job) (*frameReceipt, error)) *frameReceipt {
	type answer struct {
		receipt *frameReceipt
		err     error
	}
	result := make(chan answer)
	abandoned := make(chan struct{})
	go func() {
		var value answer
		func() { defer func() { _ = recover() }(); value.receipt, value.err = setup(root, job) }()
		select {
		case result <- value:
		case <-abandoned:
			if value.receipt != nil {
				value.receipt.close()
			}
		}
	}()
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	select {
	case value := <-result:
		if value.err != nil || ctx.Err() != nil {
			if value.receipt != nil {
				value.receipt.close()
			}
			return nil
		}
		return value.receipt
	case <-ctx.Done():
	case <-timer.C:
	}
	close(abandoned)
	return nil
}

// captureWithin runs capture for pid in a goroutine and waits at most 50
// milliseconds. A panic reports nothing known.
func captureWithin(ctx context.Context, pid int, capture func(int) (footprint.Leader, bool)) (footprint.Leader, bool) {
	type answer struct {
		leader footprint.Leader
		known  bool
	}
	result := make(chan answer, 1)
	go func() {
		defer func() {
			if recover() != nil {
				result <- answer{}
			}
		}()
		leader, known := capture(pid)
		result <- answer{leader, known}
	}()
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	select {
	case value := <-result:
		if ctx.Err() != nil {
			return footprint.Leader{}, false
		}
		return value.leader, value.known
	case <-ctx.Done():
	case <-timer.C:
	}
	return footprint.Leader{}, false
}

// recordLeader stores pid as the job's process group. The start time and boot
// ID are kept only when the leader identity is valid.
func recordLeader(job *Job, pid int, leader footprint.Leader, known bool) {
	job.ProcessGroup = pid
	job.ProcessStartSeconds, job.ProcessStartMicroseconds, job.ProcessBootID = 0, 0, ""
	id := leader.Identity
	if known && footprint.ValidBootID(leader.BootID) && id.PID == pid && id.Group == pid && id.StartSeconds > 0 && id.StartMicroseconds < 1_000_000 {
		job.ProcessStartSeconds, job.ProcessStartMicroseconds, job.ProcessBootID = id.StartSeconds, id.StartMicroseconds, leader.BootID
	}
}

// requested observes cancellation without allowing an identity change.
func (s Store) requested(ctx context.Context, j Job) (bool, string, error) {
	st, err := s.Status(ctx)
	if err != nil {
		return false, "", err
	}
	current, err := st.job(j.Sequence)
	if err != nil {
		return false, "", err
	}
	if current.Worker != j.Worker || current.State != Running {
		return false, "", ErrOwner
	}
	return current.CancelRequested, st.Paused, nil
}

// completion requires an empty group before releasing the queue slot.
func completion(group int, err error) Result {
	result := Result{State: Succeeded}
	if err != nil {
		result = Result{State: Failed, ExitCode: -1, Reason: err.Error()}
		var exited *exec.ExitError
		if errors.As(err, &exited) {
			result.ExitCode = exited.ExitCode()
		}
	}
	if alive := syscall.Kill(-group, 0); !errors.Is(alive, syscall.ESRCH) {
		result.Uncertain = true
		result.Reason = "process group still present or unknown after parent exit"
	}
	return result
}

// stopGroup stops the owned group while Game retains its lane.
func stopGroup(group int, done <-chan error, result Result) Result {
	if group < 2 {
		result.Uncertain = true
		result.Reason = "invalid owned process group"
		return result
	}
	if err := syscall.Kill(-group, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		result.Uncertain = true
		result.Reason = err.Error()
		return result
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		result.Uncertain = true
		result.Reason = "process did not reap after stop"
		return result
	}
	deadline := time.Now().Add(2 * time.Second)
	for !errors.Is(syscall.Kill(-group, 0), syscall.ESRCH) {
		if time.Now().After(deadline) {
			result.Uncertain = true
			result.Reason = "process group quiescence remains unknown"
			return result
		}
		time.Sleep(10 * time.Millisecond)
	}
	return result
}

// Execute is the checked child entry invoked inside the selected Game lane.
func (s Store) Execute(ctx context.Context, seq uint64, owner string, out io.Writer) error {
	return s.executeSelected(ctx, seq, owner, out, nativeMonitor{})
}

// ExecuteOwned supervises a render claimed by Azazel's own scheduler.
func (s Store) ExecuteOwned(ctx context.Context, seq uint64, owner string, out io.Writer) error {
	return s.executeSelectedMode(ctx, seq, owner, out, nativeMonitor{}, false)
}

// executeSelected retains the same ownership checks with a supplied monitor.
func (s Store) executeSelected(ctx context.Context, seq uint64, owner string, out io.Writer, monitor Monitor) error {
	return s.executeSelectedMode(ctx, seq, owner, out, monitor, true)
}

// executeSelectedMode runs the job at seq under its attempt lock. When
// requireGame is set, the job's lane must pass the Game lane check first.
func (s Store) executeSelectedMode(ctx context.Context, seq uint64, owner string, out io.Writer, monitor Monitor, requireGame bool) error {
	snapshot, err := s.Status(ctx)
	if err != nil {
		return err
	}
	selected, err := snapshot.job(seq)
	if err != nil {
		return err
	}
	lease, err := s.attemptLock(*selected)
	if err != nil {
		return err
	}
	defer lease.Close()
	var job Job
	err = s.transact(ctx, false, func(st *Snapshot) error {
		j, err := st.job(seq)
		if err != nil {
			return err
		}
		if requireGame {
			if err := heldLane(j.Request.Lane); err != nil {
				return err
			}
		} else if j.LeaseUntil.IsZero() || !time.Now().Before(j.LeaseUntil) {
			return ErrExpiredLease
		}
		if j.State != Running || j.Worker != owner || j.Executing {
			return ErrOwner
		}
		j.Executing = true
		job = *j
		return nil
	})
	if err != nil {
		return err
	}
	result := s.executeUnderLock(ctx, job, out, monitor, lease)
	if result.Deferred {
		if result.Stop {
			if err = s.Pause(ctx, result.Reason); err != nil {
				return err
			}
		}
		receipt, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		return s.transact(receipt, false, func(st *Snapshot) error {
			current, err := st.job(seq)
			if err != nil {
				return err
			}
			if current.Worker != owner || current.State != Running {
				return ErrOwner
			}
			current.Executing = false
			return nil
		})
	}
	if result.Uncertain {
		return fmt.Errorf("%w: %s", ErrActive, result.Reason)
	}
	receipt, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if result.Stop {
		if err = s.Pause(receipt, result.Reason); err != nil {
			return err
		}
	}
	return s.finish(receipt, seq, owner, result)
}

// GameExecutor puts the supervising child inside Game's shared lane.
type GameExecutor struct {
	Store      Store
	Executable string
	monitor    Monitor
}

type jobExecutor struct {
	Store      Store
	Executable string
	monitor    Monitor
	direct     bool
}

// RenderExecutor starts independent supervisors without a Game lane.
type RenderExecutor struct {
	Store      Store
	Executable string
}

// Run runs job through the direct executor, without the Game handoff.
func (r RenderExecutor) Run(ctx context.Context, job Job, out io.Writer) Result {
	return (jobExecutor{Store: r.Store, Executable: r.Executable, direct: true}).Run(ctx, job, out)
}

// Run starts the supervisor inside a Game lane, for library callers.
func (g GameExecutor) Run(ctx context.Context, j Job, out io.Writer) Result {
	return (jobExecutor{Store: g.Store, Executable: g.Executable, monitor: g.monitor}).Run(ctx, j, out)
}

// Run leaves uncertain supervisor exits fenced; no acknowledgement means no retry.
func (g jobExecutor) Run(ctx context.Context, j Job, out io.Writer) Result {
	if g.direct {
		stop := g.Store.keepRenderLease(ctx, j, out)
		defer stop()
	}
	if err := j.verify(); err != nil {
		return Result{State: Failed, ExitCode: -1, Reason: err.Error()}
	}
	monitor := g.monitor
	if monitor == nil {
		monitor = nativeMonitor{}
	}
	for {
		result, ready := g.Store.prepare(ctx, j, monitor)
		if ready {
			result = g.runAdmitted(ctx, j, out)
		}
		if !result.Deferred || result.Stop || ctx.Err() != nil {
			return result
		}
		select {
		case <-ctx.Done():
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// runAdmitted starts a supervisor after the host currently permits work.
func (g jobExecutor) runAdmitted(ctx context.Context, j Job, out io.Writer) Result {
	uncertain := func(err error) Result { return Result{Uncertain: true, Reason: err.Error()} }

	var command *exec.Cmd
	if g.direct {
		command = exec.Command(g.Executable, "execute-owned", "--queue", g.Store.root, "--worker", j.Worker, strconv.FormatUint(j.Sequence, 10))
	} else {
		command = exec.Command(j.Files[1].Path, "lane", j.Request.Lane, "--", g.Executable, "execute",
			"--queue", g.Store.root, "--worker", j.Worker, strconv.FormatUint(j.Sequence, 10))
	}
	command.Dir = j.Request.Directory
	command.Env = slices.DeleteFunc(os.Environ(), func(s string) bool { return strings.HasPrefix(s, "GAME_LANES=") })
	if !g.direct && j.Request.Lane == "gpu" && heldLane("gpu") == nil {
		command = exec.Command(g.Executable, "execute", "--queue", g.Store.root, "--worker", j.Worker, strconv.FormatUint(j.Sequence, 10))
		command.Dir, command.Env = j.Request.Directory, os.Environ()
	}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Stdout, command.Stderr = out, out
	if err := command.Start(); err != nil {
		return Result{State: Failed, ExitCode: -1, Reason: err.Error()}
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case <-ctx.Done():
		receipt, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		st, err := g.Store.Status(receipt)
		if err != nil {
			return uncertain(err)
		}
		current, err := st.job(j.Sequence)
		if err != nil {
			return uncertain(err)
		}
		if current.State != Running {
			return current.Result
		}
		if !current.Executing {
			return Result{Deferred: true, Reason: "worker stopped before execution"}
		}
		return uncertain(fmt.Errorf("worker stopped; supervisor retains attempt"))
	case <-done:
	}
	st, err := g.Store.Status(context.Background())
	if err != nil {
		return uncertain(err)
	}
	finished, err := st.job(j.Sequence)
	if err != nil {
		return uncertain(err)
	}
	if finished.State == Running {
		if !finished.Executing && finished.Held != "" {
			return Result{Deferred: true, Stop: st.Paused != "", Reason: finished.Held}
		}
		return uncertain(ErrActive)
	}
	return finished.Result
}

// hold makes infrastructure waits observable without failing the request.
func (s Store) hold(ctx context.Context, j Job, reason string) error {
	return s.transact(ctx, false, func(st *Snapshot) error {
		current, err := st.job(j.Sequence)
		if err != nil {
			return err
		}
		if current.Worker != j.Worker || current.State != Running {
			return ErrOwner
		}
		current.Held = reason
		return nil
	})
}

// observe retains measured usage without treating an RSS estimate as a limit.
func (s Store) observe(ctx context.Context, job Job, vitals Vitals) error {
	return s.transact(ctx, false, func(snapshot *Snapshot) error {
		current, err := snapshot.job(job.Sequence)
		if err != nil {
			return err
		}
		if current.Worker != job.Worker || current.State != Running {
			return ErrOwner
		}
		current.PeakResidentBytes = max(current.PeakResidentBytes, vitals.Resident)
		if vitals.Pressure != "" {
			current.MemoryPressure = vitals.Pressure
		}
		return nil
	})
}
