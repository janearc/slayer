package queue

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/mock/gomock"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// TestMain provides a bounded stand-in for Game's process boundary.
func TestMain(m *testing.M) {
	if os.Getenv("AZAZEL_QUEUE_HELPER") == "ledger" {
		os.Exit(ledgerHelper())
	}
	if os.Getenv("AZAZEL_QUEUE_HELPER") == "game" {
		os.Exit(gameHelper())
	}
	os.Exit(m.Run())
}

// gameHelper holds a real isolated flock and runs the exact child protocol.
func gameHelper() int {
	if len(os.Args) != 11 || os.Args[1] != "lane" || os.Args[3] != "--" || os.Args[5] != "execute" {
		return 90
	}
	s := At(os.Args[7])
	seq, err := strconv.ParseUint(os.Args[10], 10, 64)
	if err != nil {
		return 91
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return 92
	}
	dir := filepath.Join(cache, "game", "lanes", os.Args[2])
	if err = os.MkdirAll(dir, 0700); err != nil {
		return 93
	}
	path := filepath.Join(dir, "lane-1")
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return 94
	}
	defer file.Close()
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return 95
	}
	os.Setenv("GAME_LANES", os.Args[2]+"="+path)
	vitals := Vitals{Ready: true}
	if os.Getenv("AZAZEL_QUEUE_HELPER_HOLD") == "yes" {
		vitals = Vitals{Reason: "changed host"}
	}
	controller := gomock.NewController(helperReporter{})
	defer controller.Finish()
	snapshot, err := s.Status(context.Background())
	if err != nil {
		return 97
	}
	job, err := snapshot.job(seq)
	if err != nil {
		return 98
	}
	monitor := NewMockMonitor(controller)
	// A durable cancellation can precede the child's admission sample.
	monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, 0).Return(vitals, nil).MaxTimes(1)
	monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).Return(vitals, nil).AnyTimes()
	err = s.executeSelected(context.Background(), seq, os.Args[9], os.Stdout, monitor)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 96
	}
	return 0
}

// TestGameBridgeCompletesAndRecordsFailure exercises the actual child handoff.
func TestGameBridgeCompletesAndRecordsFailure(t *testing.T) {
	t.Setenv("AZAZEL_QUEUE_HELPER", "game")
	t.Setenv("HOME", t.TempDir())
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{"exit 0", "exit 7"} {
		s, r := fixture(t)
		if err = s.transact(t.Context(), false, func(st *Snapshot) error { st.Config.Game = executable; return nil }); err != nil {
			t.Fatal(err)
		}
		command := filepath.Join(r.Directory, "fixture")
		if err = os.WriteFile(command, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
			t.Fatal(err)
		}
		r.Command = command
		if _, err = s.Submit(t.Context(), r); err != nil {
			t.Fatal(err)
		}
		executor := GameExecutor{Store: s, monitor: observedHost(t, r.Directory, Vitals{Ready: true}, nil), Executable: executable}
		if err = s.Work(t.Context(), executor, false); err != nil {
			t.Fatal(err)
		}
		snapshot, err := s.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		want := Succeeded
		if body == "exit 7" {
			want = Failed
		}
		if snapshot.Jobs[0].State != want || !snapshot.Jobs[0].Executing {
			t.Fatal(snapshot)
		}
	}
}

// TestGameExitWithoutAReceiptKeepsTheAttemptActive rejects a false success.
func TestGameExitWithoutAReceiptKeepsTheAttemptActive(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	j, err := s.claim(t.Context(), "w")
	if err != nil {
		t.Fatal(err)
	}
	result := (GameExecutor{Store: s, monitor: observedHost(t, r.Directory, Vitals{Ready: true}, nil), Executable: "/usr/bin/true"}).Run(t.Context(), j, io.Discard)
	if !result.Uncertain {
		t.Fatal(result)
	}
}

// TestGameBridgeWorkerStopPreservesSupervisor separates worker shutdown from cancel.
func TestGameBridgeWorkerStopPreservesSupervisor(t *testing.T) {
	t.Setenv("AZAZEL_QUEUE_HELPER", "game")
	t.Setenv("HOME", t.TempDir())
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	s, r := fixture(t)
	if err = s.transact(t.Context(), false, func(st *Snapshot) error { st.Config.Game = executable; return nil }); err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(r.Directory, "fixture")
	if err = os.WriteFile(command, []byte("#!/bin/sh\nexec /bin/sleep 5\n"), 0700); err != nil {
		t.Fatal(err)
	}
	r.Command = command
	if _, err = s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	j, err := s.claim(t.Context(), "w")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan Result, 1)
	monitor := observedHost(t, r.Directory, Vitals{Ready: true}, nil)
	go func() {
		finished <- (GameExecutor{Store: s, monitor: monitor, Executable: executable}).Run(ctx, j, io.Discard)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		snapshot, err := s.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Jobs[0].ProcessGroup > 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("supervisor did not start", snapshot)
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	result := <-finished
	if !result.Uncertain {
		t.Fatal(result)
	}
	snapshot, err := s.Status(t.Context())
	if err != nil || snapshot.Jobs[0].State != Running || snapshot.Jobs[0].CancelRequested {
		t.Fatal("worker stop cancelled the render", snapshot, err)
	}
	if err = s.Cancel(t.Context(), j.Sequence, j.Request.Agent); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for {
		snapshot, err := s.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Jobs[0].State == Cancelled {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("supervisor did not finish cancelled attempt", snapshot)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestExecutionHoldsUnknownHostAndStopsOnOperatorCancellation preserves a wait.
func TestExecutionHoldsUnknownHostAndStopsOnOperatorCancellation(t *testing.T) {
	for _, sample := range []struct {
		vitals Vitals
		err    error
	}{{vitals: Vitals{Reason: "load held"}}, {err: errors.New("sample unknown")}} {
		s, j := executableFixture(t, "exit 0")
		ctx, cancel := context.WithTimeout(t.Context(), 550*time.Millisecond)
		result := s.execute(ctx, j, io.Discard, observedHost(t, j.Request.Directory, sample.vitals, sample.err))
		cancel()
		if !result.Deferred || result.Uncertain {
			t.Fatal(result)
		}
		snapshot, err := s.Status(t.Context())
		if err != nil || snapshot.Jobs[0].Held == "" {
			t.Fatal(snapshot, err)
		}
	}
}

// TestTimeoutIsAFailureNeverAnAutomaticRetry runs a bounded slow command.
func TestTimeoutIsAFailureNeverAnAutomaticRetry(t *testing.T) {
	s, j := executableFixture(t, "exec /bin/sleep 5")
	j.Request.TimeoutSeconds = 1
	result := s.execute(t.Context(), j, io.Discard, observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil))
	if result.State != Failed || result.Uncertain {
		t.Fatal(result)
	}
}

// TestPrestartCancellationAndPauseSkipTheExecutable tests both durable holds.
func TestPrestartCancellationAndPauseSkipTheExecutable(t *testing.T) {
	for _, pause := range []bool{false, true} {
		s, j := executableFixture(t, "exit 99")
		var err error
		if pause {
			err = s.Pause(t.Context(), "operator pause")
		} else {
			err = s.Cancel(t.Context(), j.Sequence, j.Request.Agent)
		}
		if err != nil {
			t.Fatal(err)
		}
		result := s.execute(t.Context(), j, io.Discard, unreadHost(t))
		if !pause && result.State != Cancelled || pause && !result.Deferred || result.ExitCode == 99 {
			t.Fatal(result)
		}
	}
}

// TestChangedHostHandsBackTheGameLaneBeforeWaiting proves the observed repair.
func TestChangedHostHandsBackTheGameLaneBeforeWaiting(t *testing.T) {
	t.Setenv("AZAZEL_QUEUE_HELPER", "game")
	t.Setenv("AZAZEL_QUEUE_HELPER_HOLD", "yes")
	t.Setenv("HOME", t.TempDir())
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	s, r := fixture(t)
	if err = s.transact(t.Context(), false, func(st *Snapshot) error { st.Config.Game = executable; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	j, err := s.claim(t.Context(), "worker")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	outside := make(chan struct{})
	monitor := unreadHost(t)
	first := monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
	monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).DoAndReturn(func(context.Context, string, int) (Vitals, error) {
		close(outside)
		return Vitals{Reason: "wait outside Game"}, nil
	}).After(first)
	finished := make(chan Result, 1)
	go func() {
		finished <- (GameExecutor{Store: s, Executable: executable, monitor: monitor}).Run(ctx, j, io.Discard)
	}()
	select {
	case <-outside:
	case <-time.After(5 * time.Second):
		t.Fatal("Game did not return its lane")
	}
	cancel()
	result := <-finished
	if result.Uncertain || !result.Deferred {
		t.Fatal(result)
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(cache, "game", "lanes", "gpu", "lane-1"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal("idle supervisor retained Game lane", err)
	}
}

// TestUnstartedDiskStopRetainsTheQueuedRequest requires explicit resume.
func TestUnstartedDiskStopRetainsTheQueuedRequest(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	executor := expectedExecution(t, 1, Result{Deferred: true, Stop: true, Reason: "low disk"})
	if err := s.Work(t.Context(), executor, false); !errors.Is(err, ErrPaused) {
		t.Fatal(err)
	}
	status, err := s.Status(t.Context())
	if err != nil || status.Jobs[0].State != Queued || status.Paused == "" {
		t.Fatal(status, err)
	}
	if err = s.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	executor = expectedExecution(t, 1, Result{State: Succeeded})
	if err = s.Work(t.Context(), executor, false); err != nil {
		t.Fatal(err)
	}
	status, err = s.Status(t.Context())
	if err != nil || status.Jobs[0].Attempt != 2 || status.Jobs[0].State != Succeeded {
		t.Fatal(status, err)
	}
}

// helperReporter makes generated mock failures fail the protocol subprocess.
type helperReporter struct{}

func (helperReporter) Errorf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}

func (helperReporter) Fatalf(format string, args ...any) {
	panic(fmt.Sprintf(format, args...))
}
