package preview

import (
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

func queuedPlan(t *testing.T) (*Plan, Tools) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "scene.yaml")
	source := string(firstPicture(t))
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := Prepare(Request{Path: path, Text: source, OutputRoot: root, Profile: Profile{Width: 16, Height: 12}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = plan.Close() })
	return plan, Tools{Azazel: "/tools/azazel", Queue: "/queue/team", Agent: "preview-owner", Slayer: "/tools/slayer", Catalogue: "/tools/catalogue"}
}

func queueJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func queuePNG(t *testing.T, plan *Plan, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(plan.Output), 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(plan.Output)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewNRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func expectSubmit(t *testing.T, runner *MockRunner, tools Tools, plan *Plan, request *queueRequest, replyError error) *gomock.Call {
	t.Helper()
	return runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		want := []string{"submit", "--queue", tools.Queue, filepath.Join(plan.Directory, "azazel-request.json")}
		if call.Executable != tools.Azazel || !reflect.DeepEqual(call.Arguments, want) {
			t.Fatalf("submit invocation %+v", call)
		}
		raw, err := os.ReadFile(want[3])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, request); err != nil {
			t.Fatal(err)
		}
		if request.Agent != tools.Agent || request.Key == "" || request.Command != tools.Slayer || request.Lane != "gpu" || request.TimeoutSeconds <= 0 {
			t.Fatalf("bad queue request %+v", request)
		}
		if len(request.Args) == 0 || request.Args[0] != "draw" || !strings.Contains(strings.Join(request.Args, " "), "--width 16 --height 12 --rays 1 --quality preview") {
			t.Fatalf("producer argv %+v", request.Args)
		}
		if len(request.Inputs) == 0 || request.Inputs[0] != plan.Input {
			t.Fatalf("unbound frozen input %+v", request.Inputs)
		}
		for key := range request.Environment {
			if !strings.HasPrefix(key, "SLAYER_") {
				t.Fatalf("foreign environment override %q", key)
			}
		}
		if replyError != nil {
			return nil, replyError
		}
		return queueJSON(t, queueJob{Sequence: 41, Request: *request, State: "queued"}), nil
	})
}

func expectStatus(t *testing.T, runner *MockRunner, tools Tools, reply func() ([]byte, error)) *gomock.Call {
	t.Helper()
	return runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		if call.Executable != tools.Azazel || !reflect.DeepEqual(call.Arguments, []string{"status", "--queue", tools.Queue}) {
			t.Fatalf("status invocation %+v", call)
		}
		return reply()
	})
}

func jobSnapshot(t *testing.T, request queueRequest, state, held string, executing bool) []byte {
	t.Helper()
	result := queueResult{}
	if state == "succeeded" || state == "failed" || state == "cancelled" || state == "interrupted" {
		result.State = state
	}
	return queueJSON(t, map[string]any{"version": 1, "paused": held, "jobs": []queueJob{{Sequence: 41, Request: request, State: state, Executing: executing, Result: result}}})
}

func TestQueuePausedAndClaimedAreNotAcceptedImages(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := expectSubmit(t, runner, tools, plan, &request, nil)
	paused := expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		return jobSnapshot(t, request, "queued", "intake paused", false), nil
	}).After(submit)
	claimed := expectStatus(t, runner, tools, func() ([]byte, error) { return jobSnapshot(t, request, "running", "waiting for admission", false), nil }).After(paused)
	executing := expectStatus(t, runner, tools, func() ([]byte, error) { return jobSnapshot(t, request, "running", "", true), nil }).After(claimed)
	expectStatus(t, runner, tools, func() ([]byte, error) { return jobSnapshot(t, request, "succeeded", "", false), nil }).After(executing)
	var updates []Update
	err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, func(update Update) {
		updates = append(updates, update)
		if update.State != "succeeded" {
			if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); !os.IsNotExist(err) {
				t.Fatal("nonterminal job accepted image")
			}
			if err := plan.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(plan.Input); err != nil {
				t.Fatal("queued/running input deleted", err)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []Update{{State: "queued", Sequence: 41}, {State: "queued", Held: "intake paused", Sequence: 41}, {State: "waiting", Held: "waiting for admission", Sequence: 41}, {State: "running", Sequence: 41}, {State: "succeeded", Sequence: 41, Terminal: true}}
	if !reflect.DeepEqual(updates, want) {
		t.Fatalf("updates %+v; want %+v", updates, want)
	}
	raw, err := os.ReadFile(filepath.Join(plan.Directory, "receipt.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt["width"] != float64(16) || receipt["height"] != float64(12) {
		t.Fatal("dimensions not measured", receipt)
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); !os.IsNotExist(err) {
		t.Fatal("terminal input not cleaned")
	}
	if _, err := os.Stat(filepath.Join(plan.Directory, "scene.yaml")); err != nil {
		t.Fatal("frozen audit YAML lost", err)
	}
}

func TestQueueFailedCancelledAndWrongDimensionsRejectImages(t *testing.T) {
	for _, state := range []string{"failed", "cancelled", "interrupted", "succeeded"} {
		t.Run(state, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			runner := NewMockRunner(gomock.NewController(t))
			var request queueRequest
			submit := expectSubmit(t, runner, tools, plan, &request, nil)
			expectStatus(t, runner, tools, func() ([]byte, error) {
				queuePNG(t, plan, 16, 11)
				return jobSnapshot(t, request, state, "", false), nil
			}).After(submit)
			if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err == nil {
				t.Fatal("invalid result accepted")
			}
			if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); !os.IsNotExist(err) {
				t.Fatal("failed result got receipt")
			}
			if err := plan.Close(); err != nil {
				t.Fatal(err)
			}
			_, inputError := os.Stat(plan.Input)
			if state == "interrupted" {
				if inputError != nil {
					t.Fatal("operator-interrupted input removed without confirmed descendant quiescence", inputError)
				}
			} else if !os.IsNotExist(inputError) {
				t.Fatal("confirmed terminal input retained")
			}
		})
	}
}

func TestQueueUncertainAcknowledgementRetainsAndRetriesSameRequest(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var first, second queueRequest
	lost := expectSubmit(t, runner, tools, plan, &first, errors.New("ack lost"))
	expectStatus(t, runner, tools, func() ([]byte, error) { return nil, errors.New("status unavailable") }).After(lost)
	client := queueClient{runner: runner}
	if err := client.run(context.Background(), tools, plan, io.Discard, nil); err == nil {
		t.Fatal("lost ack reported success")
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("uncertain input deleted", err)
	}
	initial, err := os.ReadFile(filepath.Join(plan.Directory, "azazel-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	submit := expectSubmit(t, runner, tools, plan, &second, nil)
	expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		return jobSnapshot(t, second, "succeeded", "", false), nil
	}).After(submit)
	if err := client.run(context.Background(), tools, plan, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(plan.Directory, "azazel-request.json"))
	if err != nil || string(again) != string(initial) || !reflect.DeepEqual(first, second) {
		t.Fatal("uncertain retry changed identity or content", err)
	}
}

func TestQueueCancellationNamesOnlyVerifiedOwnedSequence(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	submit := expectSubmit(t, runner, tools, plan, &request, nil)
	queued := expectStatus(t, runner, tools, func() ([]byte, error) {
		other := queueJob{Sequence: 7, Request: queueRequest{Agent: "other", Key: "other"}, State: "succeeded", Result: queueResult{State: "succeeded"}}
		own := queueJob{Sequence: 41, Request: request, State: "queued"}
		return queueJSON(t, map[string]any{"version": 1, "paused": "paused", "jobs": []queueJob{other, own}}), nil
	}).After(submit)
	cancelled := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(fresh context.Context, call invocation) ([]byte, error) {
		if fresh.Err() != nil {
			t.Fatal("cancel reused cancelled caller context")
		}
		if _, ok := fresh.Deadline(); !ok {
			t.Fatal("cancel context unbounded")
		}
		want := []string{"cancel", "--queue", tools.Queue, "--agent", tools.Agent, "41"}
		if call.Executable != tools.Azazel || !reflect.DeepEqual(call.Arguments, want) {
			t.Fatalf("wrong cancellation %+v", call)
		}
		return nil, nil
	}).After(queued)
	expectStatus(t, runner, tools, func() ([]byte, error) { return jobSnapshot(t, request, "cancelled", "", false), nil }).After(cancelled)
	err := (queueClient{runner: runner}).run(ctx, tools, plan, io.Discard, func(update Update) {
		if update.Held == "paused" {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error %v", err)
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); !os.IsNotExist(err) {
		t.Fatal("confirmed cancellation retained input")
	}
}

func TestQueueUnconfirmedCancelAndForeignStatusRetainInput(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(map[bool]string{false: "unconfirmed cancellation", true: "foreign sequence"}[foreign], func(t *testing.T) {
			plan, tools := queuedPlan(t)
			runner := NewMockRunner(gomock.NewController(t))
			var request queueRequest
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			submit := expectSubmit(t, runner, tools, plan, &request, nil)
			observed := expectStatus(t, runner, tools, func() ([]byte, error) {
				own := request
				if foreign {
					own.Agent = "other"
					return jobSnapshot(t, own, "succeeded", "", false), nil
				}
				return jobSnapshot(t, own, "running", "hold", false), nil
			}).After(submit)
			if !foreign {
				cancelling := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
					if len(call.Arguments) == 0 || call.Arguments[0] != "cancel" {
						t.Fatal("expected owned cancellation")
					}
					return nil, nil
				}).After(observed)
				expectStatus(t, runner, tools, func() ([]byte, error) { return nil, errors.New("status unavailable") }).After(cancelling)
			}
			err := (queueClient{runner: runner}).run(ctx, tools, plan, io.Discard, func(update Update) {
				if update.Held == "hold" {
					cancel()
				}
			})
			if err == nil {
				t.Fatal("uncertain/foreign status accepted")
			}
			if err := plan.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(plan.Input); err != nil {
				t.Fatal("uncertain live input deleted", err)
			}
		})
	}
}

func TestQueueRefusesMissingConfigurationWithoutStartingSubprocess(t *testing.T) {
	for _, field := range []string{"azazel", "queue", "agent"} {
		plan, tools := queuedPlan(t)
		switch field {
		case "azazel":
			tools.Azazel = ""
		case "queue":
			tools.Queue = ""
		case "agent":
			tools.Agent = ""
		}
		runner := NewMockRunner(gomock.NewController(t))
		if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err == nil {
			t.Fatal("missing configuration accepted", field)
		}
		if err := plan.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(plan.Input); !os.IsNotExist(err) {
			t.Fatal("unsubmitted input retained")
		}
	}
}

func TestQueueLostAcknowledgementRecoversOnlyOwnExactRequest(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	lost := expectSubmit(t, runner, tools, plan, &request, errors.New("ack response lost"))
	recovered := expectStatus(t, runner, tools, func() ([]byte, error) {
		other := queueJob{Sequence: 7, Request: queueRequest{Agent: "other", Key: "other"}, State: "succeeded", Result: queueResult{State: "succeeded"}}
		own := queueJob{Sequence: 41, Request: request, State: "queued"}
		return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{other, own}}), nil
	}).After(lost)
	expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		return jobSnapshot(t, request, "succeeded", "", false), nil
	}).After(recovered)
	var sequence uint64
	err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, func(update Update) { sequence = update.Sequence })
	if err != nil || sequence != 41 {
		t.Fatal("failed own-key recovery", sequence, err)
	}
}

func TestQueueUncertainRetryRefusesDifferentQueue(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	lost := expectSubmit(t, runner, tools, plan, &request, errors.New("ack lost"))
	expectStatus(t, runner, tools, func() ([]byte, error) { return nil, errors.New("unavailable") }).After(lost)
	client := queueClient{runner: runner}
	if err := client.run(context.Background(), tools, plan, io.Discard, nil); err == nil {
		t.Fatal("uncertain submission accepted")
	}
	tools.Queue = "/another/queue"
	if err := client.run(context.Background(), tools, plan, io.Discard, nil); err == nil {
		t.Fatal("same intended job submitted to another queue")
	}
}

func TestQueueMalformedTerminalDoesNotProveCleanupOrSuccess(t *testing.T) {
	for _, missing := range []string{"executing", "exit_code", "request_memory"} {
		t.Run(missing, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			runner := NewMockRunner(gomock.NewController(t))
			var request queueRequest
			submit := expectSubmit(t, runner, tools, plan, &request, nil)
			expectStatus(t, runner, tools, func() ([]byte, error) {
				queuePNG(t, plan, 16, 12)
				raw := queueJSON(t, queueJob{Sequence: 41, Request: request, State: "succeeded", Result: queueResult{State: "succeeded"}})
				var job map[string]any
				if err := json.Unmarshal(raw, &job); err != nil {
					t.Fatal(err)
				}
				switch missing {
				case "executing":
					delete(job, "executing")
				case "exit_code":
					delete(job["result"].(map[string]any), "exit_code")
				case "request_memory":
					job["request"].(map[string]any)["memory_bytes"] = 1
				}
				return queueJSON(t, map[string]any{"version": 1, "jobs": []any{job}}), nil
			}).After(submit)
			if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err == nil {
				t.Fatal("malformed terminal accepted")
			}
			if err := plan.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(plan.Input); err != nil {
				t.Fatal("unconfirmed terminal input deleted", err)
			}
		})
	}
}

func TestQueueDifferentPlansKeepDifferentStableKeys(t *testing.T) {
	first, tools := queuedPlan(t)
	second, _ := queuedPlan(t)
	one, err := first.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	same, err := first.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	two, err := second.freezeRequest(tools)
	if err != nil || one.Key == two.Key || one.Key != same.Key {
		t.Fatal("unstable or colliding keys", err)
	}
}

func TestQueueFailureMetadataIsOwnedActionableAndBounded(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := expectSubmit(t, runner, tools, plan, &request, nil)
	expectStatus(t, runner, tools, func() ([]byte, error) {
		raw := jobSnapshot(t, request, "failed", "", false)
		var snapshot map[string]any
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			t.Fatal(err)
		}
		job := snapshot["jobs"].([]any)[0].(map[string]any)
		job["attempt"] = 2
		result := job["result"].(map[string]any)
		result["reason"] = "producer refused scene: " + strings.Repeat("x", 8192)
		result["exit_code"] = 1
		return queueJSON(t, snapshot), nil
	}).After(submit)
	err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil)
	if err == nil || len(err.Error()) > 2048 || !strings.Contains(err.Error(), "producer refused scene") || !strings.Contains(err.Error(), "00000000000000000041.0002.log") {
		t.Fatal("unbounded or unactionable own failure metadata", err)
	}
}

func TestQueueRequestDeclaresDirectReferencedJobScene(t *testing.T) {
	plan, tools := queuedPlan(t)
	source := strings.Replace(string(firstPicture(t)), "scene: {smoke: 1}", "scene: {set: other.yaml}", 1)
	original := filepath.Join(filepath.Dir(plan.Source), "job-with-set.yaml")
	other := filepath.Join(filepath.Dir(plan.Source), "other.yaml")
	if err := os.WriteFile(other, []byte("scene: {name: direct reference}"), 0600); err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(Request{Path: original, Text: source, OutputRoot: filepath.Dir(plan.Directory), Profile: plan.Profile})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = prepared.Close() })
	request, err := prepared.freezeRequest(tools)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, path := range request.Inputs {
		if path == other {
			found = true
		}
	}
	if !found {
		t.Fatal("direct scene.set omitted from declared fingerprints", request.Inputs)
	}
}

func TestQueueCPUPreviewKeepsActualProducerAndTestsLane(t *testing.T) {
	plan, tools := queuedPlan(t)
	plan.Profile.CPURenderer = true
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		if call.Executable != tools.Azazel || call.Arguments[0] != "submit" {
			t.Fatal("direct renderer or Game fallback")
		}
		raw, err := os.ReadFile(call.Arguments[3])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatal(err)
		}
		if request.Command != tools.Slayer || request.Lane != "tests" || !strings.Contains(strings.Join(request.Args, " "), "--renderer cpu") {
			t.Fatalf("CPU request %+v", request)
		}
		return queueJSON(t, queueJob{Sequence: 41, Request: request, State: "queued"}), nil
	})
	expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		return jobSnapshot(t, request, "succeeded", "", false), nil
	}).After(submit)
	if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
}

func TestQueueCatalogueUsesBoundedNativeCatalogueCLI(t *testing.T) {
	_, tools := queuedPlan(t)
	source := catalogueSettings(t)
	original := settingsFile(t, source, "")
	plan, err := Prepare(Request{Path: original, Text: string(source), Specimen: "subaru", Profile: Profile{Width: 16, Height: 12, CPURenderer: true}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = plan.Close(); _ = os.RemoveAll(plan.Directory) })
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		if call.Executable != tools.Azazel || call.Arguments[0] != "submit" {
			t.Fatal("nonqueue catalogue execution")
		}
		raw, err := os.ReadFile(call.Arguments[3])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatal(err)
		}
		want := []string{"render", "subaru", "--plan", plan.Input, "--out", filepath.Join(plan.Directory, "book")}
		if request.Command != tools.Catalogue || request.Lane != "gpu" || !reflect.DeepEqual(request.Args, want) {
			t.Fatalf("catalogue request %+v", request)
		}
		return queueJSON(t, queueJob{Sequence: 41, Request: request, State: "queued"}), nil
	})
	expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		return jobSnapshot(t, request, "succeeded", "", false), nil
	}).After(submit)
	if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err != nil {
		t.Fatal(err)
	}
}

func TestQueueDryRunAcknowledgementNeverAcceptsAnotherJobImage(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	submit := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		if call.Arguments[0] != "submit" {
			t.Fatal("expected submission")
		}
		return []byte(`{"dry_run":true}`), nil
	})
	expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		other := queueJob{Sequence: 99, Request: queueRequest{Agent: "other", Key: "other"}, State: "succeeded", Result: queueResult{State: "succeeded"}}
		return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{other}}), nil
	}).After(submit)
	if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err == nil {
		t.Fatal("dry-run/foreign image accepted")
	}
	if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); !os.IsNotExist(err) {
		t.Fatal("unacknowledged job got receipt")
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("ambiguous submission lost input")
	}
}

func TestQueueUncertainProcessResultRetainsFrozenInput(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := expectSubmit(t, runner, tools, plan, &request, nil)
	expectStatus(t, runner, tools, func() ([]byte, error) {
		job := queueJob{Sequence: 41, Request: request, State: "failed", Result: queueResult{State: "failed", ExitCode: 1, Uncertain: true}}
		return queueJSON(t, map[string]any{"version": 1, "jobs": []queueJob{job}}), nil
	}).After(submit)
	if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err == nil {
		t.Fatal("uncertain process result accepted")
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("uncertain descendants lost frozen input")
	}
}

func TestQueueConcurrentCloseAndDuplicateRunDoNotRemoveLiveInput(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := expectSubmit(t, runner, tools, plan, &request, nil)
	expectStatus(t, runner, tools, func() ([]byte, error) {
		queuePNG(t, plan, 16, 12)
		return jobSnapshot(t, request, "succeeded", "", false), nil
	}).After(submit)
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	client := queueClient{runner: runner}
	go func() {
		done <- client.run(context.Background(), tools, plan, io.Discard, func(update Update) {
			if update.State == "queued" {
				close(entered)
				<-release
			}
		})
	}()
	<-entered
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("concurrent close removed queued input", err)
	}
	if err := client.run(context.Background(), tools, plan, io.Discard, nil); err == nil {
		t.Fatal("concurrent duplicate run accepted")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestPublicRunRequiresQueueEvenWhenLegacyGameIsConfigured(t *testing.T) {
	plan, _ := queuedPlan(t)
	err := Run(context.Background(), Tools{Game: "/unavailable/game", Slayer: "/unavailable/slayer"}, plan, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "Azazel") {
		t.Fatal("legacy Game path bypassed queue", err)
	}
}

func TestQueueRetryAcknowledgementReportsActualOwnerState(t *testing.T) {
	for _, row := range []struct {
		state     string
		executing bool
		want      string
	}{
		{"running", false, "waiting"}, {"running", true, "running"}, {"succeeded", false, "succeeded"}, {"cancelled", false, "cancelled"},
	} {
		t.Run(row.want, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			runner := NewMockRunner(gomock.NewController(t))
			var request queueRequest
			submit := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
				raw, err := os.ReadFile(call.Arguments[3])
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(raw, &request); err != nil {
					t.Fatal(err)
				}
				result := queueResult{}
				if row.state == "succeeded" || row.state == "cancelled" {
					result.State = row.state
				}
				return queueJSON(t, queueJob{Sequence: 41, Request: request, State: row.state, Executing: row.executing, Result: result}), nil
			})
			expectStatus(t, runner, tools, func() ([]byte, error) {
				if row.state == "cancelled" {
					return jobSnapshot(t, request, "cancelled", "", false), nil
				}
				queuePNG(t, plan, 16, 12)
				return jobSnapshot(t, request, "succeeded", "", false), nil
			}).After(submit)
			var first string
			err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, func(update Update) {
				if first == "" {
					first = update.State
					if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); !os.IsNotExist(err) {
						t.Fatal("acknowledgement accepted image")
					}
				}
			})
			wrongError := row.state == "cancelled" && err == nil || row.state != "cancelled" && err != nil
			if wrongError || first != row.want {
				t.Fatal("retry acknowledgement invented queued transition", first, row.want, err)
			}
		})
	}
}
