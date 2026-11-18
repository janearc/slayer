package queue

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

// TestReconcileKeepsUncertainJobsFencedWithoutBlockingCPU guards restart progress.
func TestReconcileKeepsUncertainJobsFencedWithoutBlockingCPU(t *testing.T) {
	for _, kind := range []string{"unstarted", "lease", "unknown-launch", "live-group"} {
		t.Run(kind, func(t *testing.T) {
			s, request := fixture(t)
			if _, err := s.Submit(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			job, err := s.claim(t.Context(), "prior-worker")
			if err != nil {
				t.Fatal(err)
			}
			group := 0
			if kind == "live-group" {
				command := exec.Command("/bin/sleep", "30")
				command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				if err := command.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { command.Process.Kill(); command.Wait() })
				group = command.Process.Pid
			}
			if kind != "unstarted" {
				if err := s.transact(t.Context(), false, func(state *Snapshot) error {
					state.Jobs[0].Executing, state.Jobs[0].ProcessGroup = true, group
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "lease" {
				lease, err := s.attemptLock(job)
				if err != nil {
					t.Fatal(err)
				}
				defer lease.Close()
			}
			request.Key, request.Lane = "cpu", "tests"
			if _, err := s.Submit(t.Context(), request); err != nil {
				t.Fatal(err)
			}
			if err := s.reconcile(t.Context(), nil); err != nil {
				t.Fatal("restart blocked unrelated jobs", err)
			}
			next, err := s.claimMode(t.Context(), "replacement", Capacity{Ready: true}, time.Now(), cpuWork)
			if err != nil || next.Sequence != 2 {
				t.Fatal(next, err)
			}
			state, err := s.Status(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			want := Running
			if kind == "unstarted" {
				want = Queued
			}
			if state.Jobs[0].State != want {
				t.Fatal(state.Jobs[0])
			}
			if kind == "unknown-launch" || kind == "live-group" {
				if !strings.Contains(state.Jobs[0].Held, "recovery:") || !strings.Contains(state.Scheduling.Held, "job 1") {
					t.Fatal(state)
				}
			}
		})
	}
}

// TestRecoveryRequeuesUnstartedWork retains its key and fences the old child.
func TestRecoveryRequeuesUnstartedWork(t *testing.T) {
	holdFixtureLane(t)
	s, job := executableFixture(t, "exit 99")
	if err := s.Recover(t.Context(), job.Sequence, job.Worker, true); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.Status(t.Context())
	if err != nil || snapshot.Jobs[0].State != Queued {
		t.Fatal("unstarted request was not retained for dispatch", snapshot, err)
	}
	next, err := s.claim(t.Context(), "replacement-worker")
	if err != nil || next.Sequence != job.Sequence || next.Attempt != 2 {
		t.Fatal(next, err)
	}
	if err = s.executeSelected(t.Context(), job.Sequence, job.Worker, io.Discard, unreadHost(t)); !errors.Is(err, ErrOwner) {
		t.Fatal("stale child was not refused", err)
	}
}

// TestKilledSupervisorCannotReleaseItsLiveRender exercises the inherited lease.
func TestKilledSupervisorCannotReleaseItsLiveRender(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AZAZEL_QUEUE_HELPER", "game")
	s, job := executableFixture(t, "exec /bin/sleep 20")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "lane", "gpu", "--", executable, "execute", "--queue", s.root, "--worker", job.Worker, strconv.FormatUint(job.Sequence, 10))
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	group := 0
	t.Cleanup(func() {
		command.Process.Kill()
		command.Wait()
		if group > 1 {
			syscall.Kill(-group, syscall.SIGKILL)
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for group == 0 {
		state, err := s.Status(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		group = state.Jobs[0].ProcessGroup
		if time.Now().After(deadline) {
			t.Fatal("command never started", state)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err = command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	command.Wait()
	if err = s.Recover(t.Context(), job.Sequence, job.Worker, true); !errors.Is(err, ErrActive) {
		t.Fatalf("recovered while orphan group %d remained alive: %v", group, err)
	}
	if err = syscall.Kill(-group, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for !errors.Is(syscall.Kill(-group, 0), syscall.ESRCH) {
		if time.Now().After(deadline) {
			t.Fatal("orphan group did not stop")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err = s.Recover(t.Context(), job.Sequence, job.Worker, true); err != nil {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Interrupted {
		t.Fatal(state, err)
	}
}

// TestRecoveryRefusesALiveSupervisor covers both sides of command start.
func TestRecoveryRefusesALiveSupervisor(t *testing.T) {
	for _, beforeStart := range []bool{true, false} {
		t.Run(map[bool]string{true: "before-start", false: "running"}[beforeStart], func(t *testing.T) {
			holdFixtureLane(t)
			s, job := executableFixture(t, "exec /bin/sleep 30")
			monitor := unreadHost(t)
			probe := func(context.Context, string, int) (Vitals, error) {
				if err := s.Recover(t.Context(), job.Sequence, job.Worker, true); !errors.Is(err, ErrActive) {
					t.Errorf("recovery accepted a live supervisor: %v", err)
				}
				if beforeStart {
					return Vitals{Stop: true, Reason: "bounded probe complete"}, nil
				}
				return Vitals{Ready: true}, s.Cancel(t.Context(), job.Sequence, job.Request.Agent)
			}
			if beforeStart {
				monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, 0).DoAndReturn(probe)
			} else {
				monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
				monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).DoAndReturn(probe)
			}
			if err := s.executeSelected(t.Context(), job.Sequence, job.Worker, io.Discard, monitor); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestStoppingWorkerBeforeAdmissionKeepsItsRequest distinguishes shutdown from cancel.
func TestStoppingWorkerBeforeAdmissionKeepsItsRequest(t *testing.T) {
	s, request := fixture(t)
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	monitor := unreadHost(t)
	monitor.EXPECT().Read(gomock.Any(), request.Directory, 0).DoAndReturn(func(context.Context, string, int) (Vitals, error) {
		stop()
		return Vitals{Ready: true}, nil
	})
	_ = s.Work(ctx, GameExecutor{Store: s, Executable: "/usr/bin/true", monitor: monitor}, false)
	snapshot, err := s.Status(t.Context())
	if err != nil || snapshot.Jobs[0].State != Queued || snapshot.Jobs[0].CancelRequested {
		t.Fatal("worker shutdown cancelled unstarted work", snapshot, err)
	}
}
