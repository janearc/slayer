package preview

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestManagedWorkerDrainsTemporaryQueue uses harmless commands and a queue it
// creates for this test. It never accesses the configured operator queue.
func TestManagedWorkerDrainsTemporaryQueue(t *testing.T) {
	azazel := os.Getenv("FERRUCCIO_TEST_AZAZEL")
	if azazel == "" {
		t.Skip("set FERRUCCIO_TEST_AZAZEL to a compatible Azazel executable")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
	defer cancel()
	plan, tools := queuedPlan(t)
	plan.Profile.CPURenderer = true
	root := t.TempDir()
	tools.Azazel, tools.Queue, tools.ManageWorker = azazel, filepath.Join(root, "queue"), true
	tools.Slayer = workerFixture(t, root, "own-command", "exit 7")
	control := func(verb string, args ...string) []byte {
		t.Helper()
		command := exec.CommandContext(ctx, azazel, append([]string{verb, "--queue", tools.Queue}, args...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v: %s", verb, err, output)
		}
		return output
	}
	control("init")
	control("tune", "--pressure-policy", "observe")

	barrier := filepath.Join(root, "let-finish")
	other := queueRequest{Agent: "another-agent", Key: "kept-running", Directory: root, Lane: "tests", TimeoutSeconds: 30}
	other.Command = workerFixture(t, root, "other-command", "while [ ! -f \"$1\" ]; do sleep 0.1; done")
	other.Args = []string{barrier}
	otherFile := filepath.Join(root, "other.json")
	if err := os.WriteFile(otherFile, queueJSON(t, other), 0600); err != nil {
		t.Fatal(err)
	}
	control("submit", otherFile)
	release := func() {
		if err := os.WriteFile(barrier, nil, 0600); err != nil {
			t.Error(err)
		}
	}
	defer release()

	var started bool
	err := RunObserved(ctx, tools, plan, io.Discard, func(update Update) {
		started = started || update.Held == "Starting Azazel worker"
	})
	if err == nil || !strings.Contains(err.Error(), "exit 7") || !started {
		t.Fatalf("own command did not run under the newly started worker: started=%t, err=%v", started, err)
	}
	otherState, err := readObservation(control("status"), other, 0)
	if err != nil || otherState.job.State != "running" || otherState.job.Attempt != 1 {
		t.Fatalf("other agent job should still be running: %+v, %v", otherState.job, err)
	}
	competing, err := startQueueWorker(tools, root)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-competing:
		if err == nil || !strings.Contains(err.Error(), "active attempt") {
			t.Fatalf("competing coordinator did not report the existing lock: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("competing coordinator did not honor the existing lock")
	}
	release()
	for {
		observed, err := readObservation(control("status"), other, 0)
		if err != nil {
			t.Fatal(err)
		}
		if observed.job.State == "succeeded" && observed.job.Attempt == 1 {
			return
		}
		if terminalJob(observed.job) || sleep(ctx, 100*time.Millisecond) != nil {
			t.Fatalf("other agent job did not finish unchanged: %+v", observed.job)
		}
	}
}

func workerFixture(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}
