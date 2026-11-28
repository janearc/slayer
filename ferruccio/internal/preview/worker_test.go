package preview

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkerDispatchRequiresPendingWorkAndStaleScheduling(t *testing.T) {
	now := time.Now()
	for _, example := range []struct {
		name     string
		state    string
		paused   string
		observed time.Time
		snapshot bool
		want     bool
	}{
		{name: "unverified acknowledgement", state: "queued"},
		{name: "new job", state: "queued", snapshot: true, want: true},
		{name: "abandoned active job", state: "running", snapshot: true, observed: now.Add(-time.Minute), want: true},
		{name: "active scheduler", state: "queued", snapshot: true, observed: now.Add(-time.Second)},
		{name: "paused", state: "queued", paused: "inspection", snapshot: true},
		{name: "finished", state: "succeeded", snapshot: true},
	} {
		t.Run(example.name, func(t *testing.T) {
			status := observedJob{
				job: queueJob{State: example.state}, paused: example.paused,
				hasSnapshot: example.snapshot, lastSchedule: example.observed,
			}
			if got := workerNeeded(status, now); got != example.want {
				t.Fatalf("worker needed = %t, want %t", got, example.want)
			}
		})
	}
}

func TestWorkerStartupReportsFailureAndRetries(t *testing.T) {
	directory := t.TempDir()
	tools := Tools{Azazel: filepath.Join(directory, "missing-azazel"), Queue: directory, ManageWorker: true}
	plan := &Plan{Directory: directory}
	status := observedJob{job: queueJob{State: "queued"}, hasSnapshot: true}
	now := time.Now()
	var worker workerDispatch
	if message := worker.update(tools, plan, status, now); !strings.Contains(message, "missing-azazel") {
		t.Fatalf("missing startup error in %q", message)
	}
	tools.Azazel = "/usr/bin/true"
	if message := worker.update(tools, plan, status, now); !strings.Contains(message, "missing-azazel") || worker.done != nil {
		t.Fatalf("retried without backoff: %q", message)
	}
	if message := worker.update(tools, plan, status, now.Add(workerRetry)); message != "Starting Azazel worker" || worker.done == nil {
		t.Fatalf("did not retry after backoff: %q", message)
	}
	if err := <-worker.done; err != nil {
		t.Fatal(err)
	}
}
