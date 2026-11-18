package queue

import (
	"context"
	"errors"
	"go.uber.org/mock/gomock"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExecutionRefusalsCannotBecomeSuccess covers inputs and lost state.
func TestExecutionRefusalsCannotBecomeSuccess(t *testing.T) {
	for _, kind := range []string{"changed", "owner", "missing", "context", "interpreter", "removed-after-check"} {
		t.Run(kind, func(t *testing.T) {
			s, j := executableFixture(t, "exit 0")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			monitor := unreadHost(t)
			switch kind {
			case "changed":
				if err := os.WriteFile(j.Request.Command, []byte("#!/bin/sh\nexit 8\n"), 0700); err != nil {
					t.Fatal(err)
				}
			case "owner":
				j.Worker = "other"
			case "missing":
				j.Sequence = 2
			case "context":
				cancel()
			case "interpreter":
				monitor = observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil)
				if err := os.WriteFile(j.Request.Command, []byte("#!/azazel-no-such-interpreter\n"), 0700); err != nil {
					t.Fatal(err)
				}
				f, err := fingerprint(j.Request.Command, true)
				if err != nil {
					t.Fatal(err)
				}
				j.Files[0] = f
			case "removed-after-check":
				monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).DoAndReturn(func(context.Context, string, int) (Vitals, error) {
					if err := os.Remove(j.Request.Command); err != nil {
						t.Fatal(err)
					}
					return Vitals{Ready: true}, nil
				})
			}
			result := s.execute(ctx, j, io.Discard, monitor)
			if result.State == Succeeded || result.Uncertain {
				t.Fatal(kind, result)
			}
		})
	}
}

// TestRunningStopsOnCancellationPauseAndUnknownMetrics reaches the live monitor.
func TestRunningStopsOnCancellationPauseAndUnknownMetrics(t *testing.T) {
	for _, kind := range []string{"cancel", "pause", "unknown", "disk", "state"} {
		t.Run(kind, func(t *testing.T) {
			s, j := executableFixture(t, "exec /bin/sleep 5")
			monitor := unreadHost(t)
			first := monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
			monitor.EXPECT().Read(gomock.Any(), j.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).DoAndReturn(func(ctx context.Context, _ string, _ int) (Vitals, error) {
				switch kind {
				case "cancel":
					return Vitals{Ready: true}, s.Cancel(ctx, j.Sequence, j.Request.Agent)
				case "pause":
					return Vitals{Ready: true}, s.Pause(ctx, "paused during execution")
				case "unknown":
					return Vitals{}, errors.New("memory sample unavailable")
				case "disk":
					return Vitals{Stop: true, Reason: "disk floor"}, nil
				case "state":
					if err := os.Remove(filepath.Join(s.root, "queue.db")); err != nil {
						t.Fatal(err)
					}
					return Vitals{Ready: true}, nil
				}
				panic("unrecognized fixture")
			}).After(first)
			result := s.execute(t.Context(), j, io.Discard, monitor)
			if result.State == Succeeded || result.Uncertain {
				t.Fatal(kind, result)
			}
			if kind == "disk" && !result.Stop {
				t.Fatal(result)
			}
		})
	}
}

// TestGameBridgeRefusesChangedInputsAndAnUnstartableGame covers prestart failures.
func TestGameBridgeRefusesChangedInputsAndAnUnstartableGame(t *testing.T) {
	t.Setenv("GAME_LANES", "")
	s, j := executableFixture(t, "exit 0")
	bad := j
	bad.Files = nil
	executor := GameExecutor{Store: s, monitor: observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil), Executable: "/usr/bin/true"}
	if got := executor.Run(t.Context(), bad, io.Discard); got.State != Failed || got.Uncertain {
		t.Fatal(got)
	}
	game := filepath.Join(j.Request.Directory, "bad-game")
	if err := os.WriteFile(game, []byte("#!/azazel-missing-interpreter\n"), 0700); err != nil {
		t.Fatal(err)
	}
	file, err := fingerprint(game, true)
	if err != nil {
		t.Fatal(err)
	}
	j.Files[1] = file
	if got := executor.Run(t.Context(), j, io.Discard); got.State != Failed || got.Uncertain {
		t.Fatal(got)
	}
}

// An inherited lane bypasses Game; a supervisor that writes no result must
// leave the attempt fenced rather than pretending it failed before launch.
func TestGameBridgeKeepsUnrecordedInheritedLaneAttemptUncertain(t *testing.T) {
	holdFixtureLane(t)
	s, job := executableFixture(t, "exit 0")
	executor := GameExecutor{Store: s, monitor: observedHost(t, job.Request.Directory, Vitals{Ready: true}, nil), Executable: "/usr/bin/true"}
	result := executor.Run(t.Context(), job, io.Discard)
	state, err := s.Status(t.Context())
	if err != nil || !result.Uncertain || state.Jobs[0].State != Running {
		t.Fatalf("unrecorded inherited-lane attempt lost its fence: result %v, state %v, error %v", result, state, err)
	}
}

// TestRequestObservationAndHoldRequireTheSameAttempt refuses changed records.
func TestRequestObservationAndHoldRequireTheSameAttempt(t *testing.T) {
	s, j := executableFixture(t, "exit 0")
	bad := j
	bad.Sequence = 99
	if err := s.hold(t.Context(), bad, "held"); !errors.Is(err, ErrMissing) {
		t.Fatal(err)
	}
	bad = j
	bad.Worker = "wrong"
	if err := s.hold(t.Context(), bad, "held"); !errors.Is(err, ErrOwner) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := s.requested(ctx, j); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// TestWorkerRetainsLowDiskPauseAndExistingLogs prevents destructive retry.
func TestWorkerRetainsLowDiskPauseAndExistingLogs(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	executor := expectedExecution(t, 1, Result{State: Failed, Stop: true, Reason: "disk floor"})
	if err := s.Work(t.Context(), executor, false); !errors.Is(err, ErrPaused) {
		t.Fatal(err)
	}
	s, r = fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.root, "00000000000000000001.0001.log"), []byte("old evidence"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Work(t.Context(), executor, false); !errors.Is(err, os.ErrExist) {
		t.Fatal(err)
	}
}

// TestEnvironmentCarriesSettingsAndEnforcesCPUs verifies forwarded render settings.
func TestEnvironmentCarriesSettingsAndEnforcesCPUs(t *testing.T) {
	r := Request{Agent: "agent", MemoryBytes: 512, Environment: map[string]string{"SLAYER_SHELF": "/shelf"}}
	env := environment(r)
	found := false
	for _, item := range env {
		if item == "SLAYER_SHELF=/shelf" {
			found = true
		}
	}
	if !found {
		t.Fatal(env)
	}
}

// TestLateReceiptStorageFailureLeavesTheAttemptVisible refuses lost stop state.
func TestLateReceiptStorageFailureLeavesTheAttemptVisible(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	executor := NewMockExecutor(gomock.NewController(t))
	executor.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ Job, _ io.Writer) Result {
		if err := os.Remove(filepath.Join(s.root, "queue.db")); err != nil {
			t.Fatal(err)
		}
		return Result{State: Failed, Stop: true, Reason: "disk floor"}
	})
	if err := s.Work(t.Context(), executor, false); err == nil {
		t.Fatal("ignored lost durable state")
	}
}

// TestWorkerRefusesAnUnsafeLockPermission never uses a public worker lock.
func TestWorkerRefusesAnUnsafeLockPermission(t *testing.T) {
	s, _ := fixture(t)
	if err := os.WriteFile(filepath.Join(s.root, "worker.lock"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.workerLock(); !errors.Is(err, ErrUnsafe) {
		t.Fatal(err)
	}
}

// TestStoreRefusesUnencodableReceiptAndMissingSchema preserves its predecessor.
func TestStoreRefusesUnencodableReceiptAndMissingSchema(t *testing.T) {
	s, r := fixture(t)
	if _, err := s.Submit(t.Context(), r); err != nil {
		t.Fatal(err)
	}
	err := s.transact(t.Context(), false, func(st *Snapshot) error {
		st.Jobs[0].Submitted = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		return nil
	})
	if err == nil {
		t.Fatal("accepted unencodable receipt")
	}
	if err = s.transact(t.Context(), true, func(*Snapshot) error { return nil }); err == nil {
		t.Fatal("reset existing schema")
	}
}
