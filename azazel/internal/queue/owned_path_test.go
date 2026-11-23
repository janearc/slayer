package queue

import (
	"context"
	"fmt"
	"go.uber.org/mock/gomock"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// patience bounds every wait on a real process in these tests; it is never
// asserted against, only a limit so a broken run ends.
const patience = 40 * time.Second

// ownedHelper stands in for the azazel binary's execute-owned verb: the
// dispatcher's RenderExecutor starts this test binary with the verb's
// arguments, and it runs the same ExecuteOwned the verb runs.
func ownedHelper() int {
	if len(os.Args) != 7 || os.Args[1] != "execute-owned" || os.Args[2] != "--queue" || os.Args[4] != "--worker" {
		return 90
	}
	sequence, err := strconv.ParseUint(os.Args[6], 10, 64)
	if err != nil {
		return 91
	}
	if err := At(os.Args[3]).ExecuteOwned(context.Background(), sequence, os.Args[5], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// ownedQueue submits one job whose command is the shell script body.
// Memory pressure is observed rather than stopped on, so the host's state
// does not decide the test.
func ownedQueue(t *testing.T, body string, timeoutSeconds int64) (Store, Job) {
	t.Helper()
	s, request := fixture(t)
	script := filepath.Join(request.Directory, "mock-render")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	request.Command, request.Args, request.TimeoutSeconds = script, nil, timeoutSeconds
	job, err := s.Submit(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Tune(t.Context(), Tuning{Pressure: "observe"}); err != nil {
		t.Fatal(err)
	}
	return s, job
}

// dispatching runs the dispatcher with RenderExecutor in the background and
// returns its result and the means to stop it.
func dispatching(t *testing.T, s Store) (<-chan error, context.CancelFunc) {
	t.Helper()
	t.Setenv("AZAZEL_QUEUE_HELPER", "owned")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	probe := NewMockCapacityProbe(gomock.NewController(t))
	probe.EXPECT().Read(gomock.Any(), gomock.Any()).Return(Capacity{Ready: true, GPUKnown: true}, nil).AnyTimes()
	ctx, cancel := context.WithTimeout(t.Context(), 2*patience)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- s.Dispatch(ctx, RenderExecutor{Store: s, Executable: executable}, probe, false)
	}()
	return done, cancel
}

// awaitJob polls the ledger until ready holds of the job, or fails.
func awaitJob(t *testing.T, s Store, what string, ready func(Job) bool) Job {
	t.Helper()
	deadline := time.Now().Add(patience)
	for time.Now().Before(deadline) {
		state, err := s.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if ready(state.Jobs[0]) {
			return state.Jobs[0]
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("never saw", what)
	return Job{}
}

// running is true once the render's group is recorded.
func running(job Job) bool { return job.Executing && job.ProcessGroup > 1 }

// finishes waits for the dispatcher to return and fails if it errs.
func finishes(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal("the dispatcher failed", err)
		}
	case <-time.After(patience):
		t.Fatal("the dispatcher did not finish")
	}
}

// gone says whether no process holds the group.
func gone(group int) bool { return syscall.Kill(-group, 0) == syscall.ESRCH }

// A job runs through the dispatcher, RenderExecutor and execute-owned to
// success, and the supervisor's record is the job's.
func TestDispatcherRunsAJobThroughExecuteOwned(t *testing.T) {
	s, _ := ownedQueue(t, "exit 0", 60)
	done, _ := dispatching(t, s)
	finishes(t, done)
	state, err := s.Status(t.Context())
	job := state.Jobs[0]
	if err != nil || job.State != Succeeded || job.Result.ExitCode != 0 || job.Attempt != 1 || !job.Executing {
		t.Fatal(job, err)
	}
}

// Anti-test: a command that exits non-zero is Failed with its code, not
// Succeeded, through the same path.
func TestDispatcherRecordsAFailingCommandAsFailed(t *testing.T) {
	s, _ := ownedQueue(t, "exit 7", 60)
	done, _ := dispatching(t, s)
	finishes(t, done)
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Failed || state.Jobs[0].Result.ExitCode != 7 {
		t.Fatal(state.Jobs[0], err)
	}
}

// Cancel of a running job stops the render's group and records Cancelled.
func TestCancelStopsARenderRunByTheDispatcher(t *testing.T) {
	s, submitted := ownedQueue(t, "exec /bin/sleep 120", 600)
	done, _ := dispatching(t, s)
	job := awaitJob(t, s, "the render running", running)
	if err := s.Cancel(t.Context(), job.Sequence, submitted.Request.Agent); err != nil {
		t.Fatal(err)
	}
	finishes(t, done)
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Cancelled || !gone(job.ProcessGroup) {
		t.Fatal(state.Jobs[0], err)
	}
}

// A render past its timeout is stopped and recorded Failed, its group gone.
func TestTimeoutStopsARenderRunByTheDispatcher(t *testing.T) {
	s, _ := ownedQueue(t, "exec /bin/sleep 120", 1)
	done, _ := dispatching(t, s)
	job := awaitJob(t, s, "the render running", running)
	finishes(t, done)
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Failed || !strings.Contains(state.Jobs[0].Result.Reason, "deadline") || !gone(job.ProcessGroup) {
		t.Fatal(state.Jobs[0], err)
	}
}

// Anti-test: a render within its timeout is not stopped by it.
func TestARenderWithinItsTimeoutIsNotStopped(t *testing.T) {
	s, _ := ownedQueue(t, "exec /bin/sleep 1", 300)
	done, _ := dispatching(t, s)
	finishes(t, done)
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Succeeded {
		t.Fatal(state.Jobs[0], err)
	}
}

// The supervisor holds the render's lease while it runs: the deadline moves
// out past the one the claim granted.
func TestRenderLeaseIsRenewedWhileTheRenderRuns(t *testing.T) {
	s, submitted := ownedQueue(t, "exec /bin/sleep 120", 600)
	done, _ := dispatching(t, s)
	first := awaitJob(t, s, "the render running", running)
	renewed := awaitJob(t, s, "the lease renewed", func(job Job) bool { return job.LeaseUntil.After(first.LeaseUntil) })
	if !renewed.LeaseUntil.After(first.LeaseUntil) || renewed.State != Running {
		t.Fatal(renewed)
	}
	if err := s.Cancel(t.Context(), first.Sequence, submitted.Request.Agent); err != nil {
		t.Fatal(err)
	}
	finishes(t, done)
}

// A supervisor killed while its render lives leaves the render running and
// the job Running, run once; the ledger then says so, and cancel stops the
// render, after which the attempt resolves as Interrupted.
func TestKilledSupervisorLeavesTheRenderAndCancelStopsIt(t *testing.T) {
	s, submitted := ownedQueue(t, `echo $PPID > supervisor.pid
echo started >> runs
exec /bin/sleep 120`, 600)
	done, stop := dispatching(t, s)
	job := awaitJob(t, s, "the render running", running)
	directory := submitted.Request.Directory
	var supervisor int
	awaitJob(t, s, "the supervisor's number", func(Job) bool {
		data, err := os.ReadFile(filepath.Join(directory, "supervisor.pid"))
		trimmed := strings.TrimSpace(string(data))
		value, parsed := strconv.Atoi(trimmed)
		supervisor = value
		return err == nil && parsed == nil
	})
	if err := syscall.Kill(supervisor, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	// The lease no one renews has run out for a whole lease by now.
	awaitJob(t, s, "the ledger saying the supervisor is gone", func(current Job) bool {
		if strings.HasPrefix(current.Held, supervisorGonePrefix) {
			return true
		}
		if err := s.transact(t.Context(), false, func(state *Snapshot) error {
			state.Jobs[0].LeaseUntil = time.Now().Add(-2 * renderLeaseDuration)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return false
	})
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Running || gone(job.ProcessGroup) {
		t.Fatal("the render was lost with its supervisor", state.Jobs[0], err)
	}
	if err := s.Cancel(t.Context(), job.Sequence, submitted.Request.Agent); err != nil {
		t.Fatal(err)
	}
	final := awaitJob(t, s, "the attempt resolved", func(current Job) bool { return current.State != Running })
	runs, err := os.ReadFile(filepath.Join(directory, "runs"))
	if final.State != Interrupted || !gone(job.ProcessGroup) || err != nil || strings.Count(string(runs), "started") != 1 {
		t.Fatal("the render was not stopped once, or ran twice", final, string(runs), err)
	}
	stop()
	<-done
}
