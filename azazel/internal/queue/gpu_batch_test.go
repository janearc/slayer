package queue

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBatchLaunchRetainsExitAndLog verifies detached command completion and failures.
func TestBatchLaunchRetainsExitAndLog(t *testing.T) {
	for _, executable := range []string{"/usr/bin/true", "/usr/bin/false", "/missing/game"} {
		s, _ := fixture(t)
		var out bytes.Buffer
		done, err := s.startGPUBatch(executable, "/test/azazel", &out)
		if executable == "/missing/game" {
			if err == nil {
				t.Fatal("missing command accepted")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		select {
		case err := <-done:
			if (err == nil) != (executable == "/usr/bin/true") {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("batch exit not observed")
		}
		if !strings.Contains(out.String(), "GPU batch coordinator pid") {
			t.Fatal(out.String())
		}
		if _, err := os.Stat(filepath.Join(s.root, "gpu-batches.log")); err != nil {
			t.Fatal(err)
		}
	}
}

// TestGameDispatcherDrainsAndCancels checks the actual main coordinator entry.
func TestGameDispatcherDrainsAndCancels(t *testing.T) {
	s, request := fixture(t)
	if err := s.DispatchGame(t.Context(), "/usr/bin/true", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	request.Lane = "gpu"
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	var out bytes.Buffer
	// /usr/bin/true stands in for a completed Game process: the queued request
	// must stay queued because no supervisor wrote an acknowledgement.
	finished := make(chan error, 1)
	go func() { finished <- s.DispatchGame(ctx, "/usr/bin/true", true, &out) }()
	time.AfterFunc(1500*time.Millisecond, cancel)
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Jobs[0].State != Queued {
		t.Fatal(state, err)
	}
	if out.Len() == 0 {
		t.Fatal("batch not launched")
	}
}

// TestGPUCoordinatorRequiresAndUsesItsLease checks the internal handoff entry.
func TestGameDispatcherRestartsFailedBatch(t *testing.T) {
	s, request := fixture(t)
	if err := s.transact(t.Context(), false, func(state *Snapshot) error { state.Config.Game = "/usr/bin/false"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2500*time.Millisecond)
	defer cancel()
	var out bytes.Buffer
	if err := s.DispatchGame(ctx, "/usr/bin/true", true, &out); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("batch failure stopped main dispatcher", err)
	}
	if strings.Count(out.String(), "GPU batch coordinator pid") < 2 {
		t.Fatal("failed batch was not replaced", out.String())
	}
}

// TestGPUCoordinatorRequiresAndUsesItsLease checks the internal handoff entry.
func TestGPUCoordinatorRequiresAndUsesItsLease(t *testing.T) {
	s, _ := fixture(t)
	t.Setenv("GAME_LANES", "")
	if err := s.DispatchGPU(t.Context(), "/usr/bin/true"); err == nil {
		t.Fatal("accepted missing Game lease")
	}
	holdFixtureLane(t)
	if err := s.DispatchGPU(t.Context(), "/usr/bin/true"); err != nil {
		t.Fatal(err)
	}
}
