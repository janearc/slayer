package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// TestAgentHelpExplainsSubmissionAndRecovery checks the shipped entry point.
func TestAgentHelpExplainsSubmissionAndRecovery(t *testing.T) {
	told, code := run(t, azazel, "--agent-help")
	if code != 0 {
		t.Fatal(code, told)
	}
	for _, required := range []string{
		"submit --queue", "status --queue", "cancel --queue", "agent", "key",
		"same request", "failed", "running", "process success", "system memory pressure",
		"schema", "do not run execute", "under development", "--gpu-jobs",
	} {
		if !strings.Contains(told, required) {
			t.Errorf("agent help omits %q", required)
		}
	}
}

// TestMixedHelpAndSubmitIsRefusedWithoutChangingQueue checks Pzm's refusal.
func TestMixedHelpAndSubmitIsRefusedWithoutChangingQueue(t *testing.T) {
	directory := t.TempDir()
	queuePath := filepath.Join(directory, "queue")
	store := queue.At(queuePath)
	if err := store.Init(t.Context(), queue.Config{Game: "/usr/bin/true"}); err != nil {
		t.Fatal(err)
	}
	request := queue.Request{Agent: "test", Key: "help", Command: "/usr/bin/true", Args: []string{}, Directory: directory, Lane: "gpu", TimeoutSeconds: 10}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "request.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	told, code := run(t, azazel, "--agent-help", "submit", "--queue", queuePath, path)
	if code != exitUsage {
		t.Fatal(code, told)
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || len(snapshot.Jobs) != 0 {
		t.Fatal("help changed the queue", snapshot, err)
	}
}

// TestQueueAliasesAdvertisedByPzmReachTheSameHandler checks the public contract.
func TestQueueAliasesAdvertisedByPzmReachTheSameHandler(t *testing.T) {
	for _, args := range [][]string{{"--status", "--queue", "/missing-azazel-queue"}, {"--queue", "/missing-azazel-queue", "status"}} {
		told, code := run(t, azazel, args...)
		if code != 1 || !strings.Contains(told, "azazel status:") {
			t.Fatal(args, code, told)
		}
	}
}
