package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// TestGameHandoffVerbsAlwaysRefuse checks execute and work-gpu run nothing.
func TestGameHandoffVerbsAlwaysRefuse(t *testing.T) {
	root := filepath.Join(t.TempDir(), "queue")
	store := queue.At(root)
	if err := store.Init(t.Context(), queue.Config{}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"execute", "--queue", root, "--worker", "owner", "1"},
		{"work-gpu", "--queue", root},
	} {
		var out, wrong bytes.Buffer
		if code := queueTool(t.Context()).Run(args, &out, &wrong); code == 0 || !strings.Contains(wrong.String(), "always refuses") {
			t.Fatal(args, code, wrong.String())
		}
	}
}

// TestWorkerConcurrencyIsValidatedBeforeDispatch checks the public bound.
func TestTuneUpdatesTheExistingQueue(t *testing.T) {
	root := filepath.Join(t.TempDir(), "queue")
	s := queue.At(root)
	if err := s.Init(t.Context(), queue.Config{Game: "/usr/bin/true"}); err != nil {
		t.Fatal(err)
	}
	var out, wrong bytes.Buffer
	if code := queueTool(t.Context()).Run([]string{"tune", "--queue", root, "--gpu-jobs", "4", "--gpu-start-seconds", "1", "--pressure-policy", "observe"}, &out, &wrong); code != 0 {
		t.Fatal(code, wrong.String())
	}
	state, err := s.Status(t.Context())
	if err != nil || state.Tuning.GPUJobs != 4 || state.Tuning.Pressure != "observe" {
		t.Fatal(state, err)
	}
	out.Reset()
	wrong.Reset()
	if code := queueTool(t.Context()).Run([]string{"tune", "--queue", root, "--reset", "--gpu-jobs", "2"}, &out, &wrong); code == 0 {
		t.Fatal("accepted ambiguous reset")
	}
	out.Reset()
	wrong.Reset()
	if code := queueTool(t.Context()).Run([]string{"tune", "--queue", root, "--reset"}, &out, &wrong); code != 0 {
		t.Fatal(code, wrong.String())
	}
	state, err = s.Status(t.Context())
	if err != nil || state.Tuning != (queue.Tuning{}) {
		t.Fatal(state, err)
	}
}

// TestWorkerConcurrencyIsValidatedBeforeDispatch checks the public bound.
func TestWorkerConcurrencyIsValidatedBeforeDispatch(t *testing.T) {
	for _, action := range []string{"work", "work-gpu"} {
		for _, value := range []string{"0", "-1", "65", "many"} {
			var out, wrong bytes.Buffer
			code := queueTool(t.Context()).Run([]string{action, "--queue", "/missing", "--gpu-jobs", value}, &out, &wrong)
			if code == 0 || !strings.Contains(wrong.String(), "GPU jobs") && !strings.Contains(wrong.String(), "gpu-jobs") {
				t.Fatal(action, value, code, wrong.String())
			}
		}
	}
}

// TestEmptyOptionalFieldsKeepSubmissionIdentity exercises the stored JSON round trip.
func TestEmptyOptionalFieldsKeepSubmissionIdentity(t *testing.T) {
	for _, optional := range []string{`,"inputs":[]`, `,"environment":{}`, `,"inputs":[],"environment":{}`} {
		t.Run(optional, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "queue")
			store := queue.At(root)
			if err := store.Init(t.Context(), queue.Config{Game: "/usr/bin/true"}); err != nil {
				t.Fatal(err)
			}
			raw := fmt.Sprintf(`{"agent":"one","key":"retry","command":"/usr/bin/true","args":[],"directory":%q,"lane":"tests","timeout_seconds":5%s}`, t.TempDir(), optional)
			file := filepath.Join(t.TempDir(), "request.json")
			if err := os.WriteFile(file, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				var out, wrong bytes.Buffer
				if code := queueTool(t.Context()).Run([]string{"submit", "--queue", root, file}, &out, &wrong); code != 0 {
					t.Fatalf("identical submission refused: %s", wrong.String())
				}
				var job queue.Job
				if err := json.Unmarshal(out.Bytes(), &job); err != nil || job.Sequence != 1 {
					t.Fatal(job, err)
				}
			}
			status, err := store.Status(t.Context())
			if err != nil || len(status.Jobs) != 1 {
				t.Fatal(status, err)
			}
		})
	}
}

// TestQueueCommandsAcceptAndCancel tests the public multi-agent submission path.
func TestQueueCommandsAcceptAndCancel(t *testing.T) {
	root := filepath.Join(t.TempDir(), "queue")
	invoke := func(args ...string) string {
		t.Helper()
		var out, wrong bytes.Buffer
		if code := queueTool(context.Background()).Run(args, &out, &wrong); code != 0 {
			t.Fatalf("%v: %d %s", args, code, wrong.String())
		}
		return out.String()
	}
	invoke("init", "--queue", root, "--game", "/usr/bin/true")
	request := queue.Request{Agent: "studio", Key: "film-one", Command: slayer,
		Args: []string{"draw", "doc/examples/jobs/first-film.yaml", "--dry-run"}, Directory: t.TempDir(), Lane: "gpu", MemoryBytes: 256 << 20, TimeoutSeconds: 60}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "request.json")
	if err = os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	var job queue.Job
	submitted := invoke("submit", "--queue", root, file)
	if err = json.Unmarshal([]byte(submitted), &job); err != nil || job.Sequence != 1 {
		t.Fatal(job, err)
	}
	invoke("cancel", "--queue", root, "--agent", "studio", "1")
	var status queue.Snapshot
	reported := invoke("status", "--queue", root)
	if err = json.Unmarshal([]byte(reported), &status); err != nil || status.Jobs[0].State != queue.Cancelled {
		t.Fatal(status, err)
	}
	invoke("pause", "--queue", root, "--reason", "review")
	invoke("resume", "--queue", root)
}

// TestQueueCommandRefusals requires complete and unambiguous requests.
func TestQueueCommandRefusals(t *testing.T) {
	for _, args := range [][]string{
		{"init"}, {"status"}, {"submit"}, {"cancel", "--queue", "/missing", "no"},
		{"work", "--queue", "/missing"}, {"recover", "--queue", "/missing", "1"},
	} {
		var out, wrong bytes.Buffer
		if queueTool(t.Context()).Run(args, &out, &wrong) == 0 {
			t.Fatal("accepted", args)
		}
	}
}

// TestRequestSchemaRejectsAmbiguity checks complete JSON and YAML boundaries.
func TestRequestSchemaRejectsAmbiguity(t *testing.T) {
	for _, raw := range []string{
		"{}", "null", "not a request", `{"agent":"one","agent":"two"}`,
		"agent: one\n---\nagent: two\n", "agent: one\n---\n[invalid",
	} {
		if _, err := decodeRequest([]byte(raw)); err == nil {
			t.Fatal("accepted ambiguous request", raw)
		}
	}
	if _, err := decodeRequestSchema([]byte("{}"), []byte("broken")); err == nil {
		t.Fatal("accepted invalid schema")
	}
	raw := `agent: studio
key: one
command: /usr/bin/true
args: ["draw", "file with spaces.yaml", "--renderer", "gpu-native"]
directory: /private/tmp
lane: gpu
timeout_seconds: 60
memory_bytes: 268435456
`
	request, err := decodeRequest([]byte(raw))
	if err != nil || request.Args[1] != "file with spaces.yaml" || request.TimeoutSeconds != 60 {
		t.Fatal(request, err)
	}
}

// TestQueueMutationsRefuseStrayArguments prevents a typo from changing state.
func TestQueueMutationsRefuseStrayArguments(t *testing.T) {
	root := filepath.Join(t.TempDir(), "queue")
	store := queue.At(root)
	if err := store.Init(t.Context(), queue.Config{Game: "/usr/bin/true"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"pause", "--queue", root, "--reason", "test", "stray"},
		{"resume", "--queue", root, "stray"}, {"work", "--queue", root, "stray"},
	} {
		var output, wrong bytes.Buffer
		if code := queueTool(t.Context()).Run(args, &output, &wrong); code == 0 {
			t.Fatal(args, "accepted stray argument")
		}
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || snapshot.Paused != "" {
		t.Fatal(snapshot, err)
	}
}
