package queue

import (
	"context"
	"go.uber.org/mock/gomock"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// observedHost requires an admission sample and checks every later sample's path.
func observedHost(t *testing.T, directory string, vitals Vitals, err error) *MockMonitor {
	t.Helper()
	monitor := NewMockMonitor(gomock.NewController(t))
	monitor.EXPECT().Read(gomock.Any(), directory, 0).Return(vitals, err)
	monitor.EXPECT().Read(gomock.Any(), directory, gomock.Cond(func(group int) bool {
		return group > 1
	})).Return(vitals, err).AnyTimes()
	return monitor
}

// unreadHost refuses any observation when validation should stop earlier.
func unreadHost(t *testing.T) *MockMonitor {
	t.Helper()
	return NewMockMonitor(gomock.NewController(t))
}

// executableFixture makes a bounded foreground process with exact arguments.
func executableFixture(t *testing.T, body string) (Store, Job) {
	t.Helper()
	s, r := fixture(t)
	path := filepath.Join(r.Directory, "tool")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	r.Command = path
	r.Args = []string{"a b", ";not-a-shell-command", "--vibes", "on"}
	_, err := s.Submit(t.Context(), r)
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.claim(t.Context(), "test-worker")
	if err != nil {
		t.Fatal(err)
	}
	return s, j
}

// TestForegroundArgumentsAndFailure are passed intact through process execution.
func TestForegroundArgumentsAndFailure(t *testing.T) {
	s, j := executableFixture(t, `[ "$#" = 4 ] && [ "$1" = "a b" ] && [ "$2" = ";not-a-shell-command" ] || exit 9
	[ "$GOMAXPROCS" = 8 ] || exit 10
exit 7`)
	r := s.execute(t.Context(), j, io.Discard, observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil))
	if r.State != Failed || r.ExitCode != 7 || r.Uncertain {
		t.Fatal(r)
	}
}

func TestRenderWorkerCountCanBeSetPerJob(t *testing.T) {
	s, j := executableFixture(t, `[ "$GOMAXPROCS" = 4 ] || exit 10`)
	if err := s.transact(t.Context(), false, func(state *Snapshot) error {
		state.Jobs[0].Request.Environment = map[string]string{"SLAYER_CPU_WORKERS": "4"}
		j.Request.Environment = state.Jobs[0].Request.Environment
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result := s.execute(t.Context(), j, io.Discard, observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil))
	if result.State != Succeeded {
		t.Fatal(result)
	}
}

func TestInvalidRenderWorkerCountsAreRefused(t *testing.T) {
	_, request := fixture(t)
	for _, value := range []string{"", "0", "-1", "1025", "not-a-number"} {
		request.Environment = map[string]string{"SLAYER_CPU_WORKERS": value}
		if err := request.validate(); err == nil {
			t.Fatalf("accepted worker count %q", value)
		}
	}
}

// TestCancellationStopsOwnedGroup cancels a controlled foreground sleep.
func TestCancellationStopsOwnedGroup(t *testing.T) {
	s, j := executableFixture(t, "exec /bin/sleep 30")
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(100*time.Millisecond, cancel)
	r := s.execute(ctx, j, io.Discard, observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil))
	if r.State != Cancelled || r.Uncertain {
		t.Fatal(r)
	}
}

// TestDiskStopsNeverBecomeSuccess preserves the filesystem stop policy.
func TestDiskStopsNeverBecomeSuccess(t *testing.T) {
	for _, v := range []Vitals{{Stop: true, Reason: "disk below floor"}} {
		s, j := executableFixture(t, "exec /bin/sleep 30")
		result := s.execute(t.Context(), j, io.Discard, observedHost(t, j.Request.Directory, v, nil))
		if !v.Stop && result.State != Failed || v.Stop && !result.Deferred || result.Uncertain {
			t.Fatal(result)
		}
	}
}

// TestLargeResidentUsageDoesNotImposeAHeapOrRSSLimit follows the pressure policy.
func TestLargeResidentUsageDoesNotImposeAHeapOrRSSLimit(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "64MiB")
	store, job := executableFixture(t, "[ -z \"$GOMEMLIMIT\" ] || exit 88\nsleep 0.35\nexit 0")
	monitor := unreadHost(t)
	monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, 0).Return(Vitals{Ready: true}, nil)
	monitor.EXPECT().Read(gomock.Any(), job.Request.Directory, gomock.Cond(func(group int) bool { return group > 1 })).
		Return(Vitals{Ready: true, Resident: 8 << 30}, nil).MinTimes(1)
	result := store.execute(t.Context(), job, io.Discard, monitor)
	if result.State != Succeeded || result.Uncertain {
		t.Fatal(result)
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || snapshot.Jobs[0].PeakResidentBytes != 8<<30 {
		t.Fatal(snapshot, err)
	}
}
