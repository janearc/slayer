package queue

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// TestExecutionRequiresALiveGameLane refuses missing and forged lane marks.
func TestExecutionRequiresALiveGameLane(t *testing.T) {
	t.Setenv("GAME_LANES", "")
	if heldLane("gpu") == nil {
		t.Fatal("accepted missing lane")
	}
	t.Setenv("GAME_LANES", "gpu=/not/the/game/cache/lane-1")
	if heldLane("gpu") == nil {
		t.Fatal("accepted foreign lane")
	}
	// Isolate the known Game cache location, never touch its live locks.
	t.Setenv("HOME", t.TempDir())
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(cache, "game", "lanes", "gpu")
	if err = os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "lane-1")
	t.Setenv("GAME_LANES", "tests=ignored:gpu="+path)
	if heldLane("gpu") == nil {
		t.Fatal("accepted absent lock")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if heldLane("gpu") == nil {
		t.Fatal("accepted unheld lock")
	}
	descriptor := int(file.Fd())
	if err = syscall.Flock(descriptor, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	if err = heldLane("gpu"); err != nil {
		t.Fatal(err)
	}
}

// holdFixtureLane owns an isolated Game-compatible lock for bounded fixtures.
func holdFixtureLane(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(cache, "game", "lanes", "gpu")
	if err = os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "lane-1")
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		t.Fatal(err)
	}
	descriptor := int(file.Fd())
	if err = syscall.Flock(descriptor, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })
	t.Setenv("GAME_LANES", "gpu="+path)
}

// TestSelectedAttemptRequiresItsWorkerAndCannotExecuteTwice closes replay races.
func TestSelectedAttemptRequiresItsWorkerAndCannotExecuteTwice(t *testing.T) {
	holdFixtureLane(t)
	s, j := executableFixture(t, "exit 0")
	if err := s.executeSelected(t.Context(), 99, j.Worker, os.Stdout, unreadHost(t)); err == nil {
		t.Fatal("accepted missing job")
	}
	if err := s.executeSelected(t.Context(), j.Sequence, "wrong", os.Stdout, unreadHost(t)); err == nil {
		t.Fatal("accepted wrong worker")
	}
	if err := s.executeSelected(t.Context(), j.Sequence, j.Worker, os.Stdout, observedHost(t, j.Request.Directory, Vitals{Ready: true}, nil)); err != nil {
		t.Fatal(err)
	}
	if err := s.executeSelected(t.Context(), j.Sequence, j.Worker, os.Stdout, unreadHost(t)); err == nil {
		t.Fatal("executed twice")
	}
}

// TestSelectedHostHoldReturnsWithoutRetainingExecution makes lane handback possible.
func TestSelectedHostHoldReturnsWithoutRetainingExecution(t *testing.T) {
	for _, stop := range []bool{false, true} {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			holdFixtureLane(t)
			s, j := executableFixture(t, "exit 99")
			monitor := observedHost(t, j.Request.Directory, Vitals{Stop: stop, Reason: "host held"}, nil)
			if err := s.executeSelected(t.Context(), j.Sequence, j.Worker, os.Stdout, monitor); err != nil {
				t.Fatal(err)
			}
			status, err := s.Status(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			got := status.Jobs[0]
			if got.State != Running || got.Executing || got.Held != "host held" || (status.Paused != "") != stop {
				t.Fatal(status)
			}
		})
	}
}

// TestNativeSelectedEntryUsesTheSameProtocol observes the real host policy.
func TestNativeSelectedEntryUsesTheSameProtocol(t *testing.T) {
	holdFixtureLane(t)
	s, j := executableFixture(t, "exit 0")
	if err := s.Execute(t.Context(), j.Sequence, j.Worker, os.Stdout); err != nil {
		t.Fatal(err)
	}
	status, err := s.Status(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := status.Jobs[0]
	if got.State != Succeeded && (got.State != Running || got.Executing || got.Held == "") {
		t.Fatal(status)
	}
}
