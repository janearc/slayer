package preview

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
	"go.uber.org/mock/gomock"
)

func TestDirectoryPrefixMakesLabelledUniqueChildrenAndPreservesDefault(t *testing.T) {
	root := t.TempDir()
	prefix := "2026-11-15-preview-owner-ferruccio-preview-"
	var previous string
	for _, name := range []string{prefix, prefix, ""} {
		plan, err := Prepare(Request{Path: filepath.Join(root, "source.yaml"), Text: string(firstPicture(t)), OutputRoot: root, DirectoryPrefix: name})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = plan.Close() })
		want := name
		if want == "" {
			want = "ferruccio-render-"
		}
		if !strings.HasPrefix(filepath.Base(plan.Directory), want) || filepath.Dir(plan.Directory) != root || plan.Directory == previous {
			t.Fatal("directory identity/label lost", plan.Directory)
		}
		previous = plan.Directory
	}
}

func TestDirectoryPrefixRefusesPathsAndControlNamesBeforeArtifacts(t *testing.T) {
	for _, prefix := range []string{".", "..", "../escape", "nested/name", "nested\\name", "/absolute", " ", "bad\x00name", "bad\nname"} {
		t.Run(strings.ReplaceAll(prefix, "\x00", "nul"), func(t *testing.T) {
			root := t.TempDir()
			plan, err := Prepare(Request{Path: filepath.Join(root, "source.yaml"), Text: string(firstPicture(t)), OutputRoot: root, DirectoryPrefix: prefix})
			if plan != nil {
				t.Cleanup(func() { _ = plan.Close() })
			}
			if err == nil {
				t.Fatal("unsafe directory prefix accepted")
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatal("invalid prefix created artifacts", entries, err)
			}
		})
	}
}

func TestRecipeExistsBeforeSubmissionWithExactRequestedCommand(t *testing.T) {
	plan, tools := queuedPlan(t)
	runner := NewMockRunner(gomock.NewController(t))
	var request queueRequest
	submit := runner.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, call invocation) ([]byte, error) {
		if call.Executable != tools.Azazel || call.Arguments[0] != "submit" {
			t.Fatal("unexpected control call")
		}
		raw, err := os.ReadFile(call.Arguments[3])
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatal(err)
		}
		recipe, err := os.ReadFile(filepath.Join(plan.Directory, "recipe.txt"))
		if err != nil {
			t.Fatal("recipe missing before submit", err)
		}
		argv := append([]string{request.Command}, request.Args...)
		exact, err := json.Marshal(argv)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{string(exact), "Prepared preview submission", "16x12", "rays: 1", "azazel-request.json", "azazel-acknowledgement.json", "scene.yaml", "receipt.json", plan.Digest, "limitations"} {
			if !strings.Contains(string(recipe), want) {
				t.Fatal("recipe missing exact intent/provenance", want)
			}
		}
		if strings.Contains(string(recipe), "RENDERING since") || strings.Contains(string(recipe), "ETA") {
			t.Fatal("recipe invented active rendering/progress")
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

func TestReceiptUsesVerifiedProducerAndAcknowledgementNotConfiguredGame(t *testing.T) {
	for _, kind := range []string{"job", "catalogue"} {
		t.Run(kind, func(t *testing.T) {
			plan, tools := queuedPlan(t)
			if kind == "catalogue" {
				source := catalogueSettings(t)
				original := settingsFile(t, source, "")
				var err error
				plan, err = Prepare(Request{Path: original, Text: string(source), Specimen: "subaru", Profile: Profile{Width: 16, Height: 12}})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = plan.Close(); _ = os.RemoveAll(plan.Directory) })
				tools.Slayer = ""
			}
			tools.Game = "/configured-but-unused-game-sentinel"
			expected := tools.Slayer
			if kind == "catalogue" {
				expected = tools.Catalogue
			}
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
				return queueJSON(t, queueJob{Sequence: 41, Request: request, State: "queued"}), nil
			})
			expectStatus(t, runner, tools, func() ([]byte, error) {
				queuePNG(t, plan, 16, 12)
				return jobSnapshot(t, request, "succeeded", "", false), nil
			}).After(submit)
			if err := (queueClient{runner: runner}).run(context.Background(), tools, plan, io.Discard, nil); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(plan.Directory, "receipt.json"))
			if err != nil {
				t.Fatal(err)
			}
			var receipt map[string]any
			if err := json.Unmarshal(raw, &receipt); err != nil {
				t.Fatal(err)
			}
			if receipt["producer"] != expected || receipt["azazel"] != tools.Azazel || receipt["worker_provenance"] != "azazel-acknowledgement.json" {
				t.Fatal("verified producer/worker provenance not recorded", receipt)
			}
			if _, exists := receipt["game"]; exists {
				t.Fatal("unused client Game claimed as used")
			}
			if strings.Contains(string(raw), "configured-but-unused") {
				t.Fatal("configured Game leaked into used provenance")
			}
			if kind == "catalogue" {
				recipe, err := os.ReadFile(filepath.Join(plan.Directory, "recipe.txt"))
				if err != nil || !strings.Contains(string(recipe), "producer-authored quality") || strings.Contains(string(recipe), "quality: preview") {
					t.Fatal("catalogue recipe invented a quality flag", err)
				}
				if _, exists := receipt["slayer"]; exists {
					t.Fatal("catalogue claimed unused Slayer")
				}
			}
			if _, err := os.Stat(filepath.Join(plan.Directory, "azazel-acknowledgement.json")); err != nil {
				t.Fatal("worker provenance link missing", err)
			}
		})
	}
}

func TestCapturedOwnerJob3DecodesAsQueuedWithoutTerminalAuthority(t *testing.T) {
	requestRaw, err := os.ReadFile(filepath.Join("testdata", "azazel-job3-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	statusRaw, err := os.ReadFile(filepath.Join("testdata", "azazel-job3-status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request queueRequest
	if err := yamlfile.Read("captured owner request", requestRaw, &request); err != nil {
		t.Fatal(err)
	}
	observation, err := readObservation(statusRaw, request, 3)
	if err != nil {
		t.Fatal(err)
	}
	update := observation.update()
	if update.Sequence != 3 || update.State != "queued" || update.Terminal || update.Held == "" {
		t.Fatal("actual owner wire misread", update)
	}
	var filtered struct {
		Jobs []json.RawMessage `json:"jobs"`
	}
	if err := json.Unmarshal(statusRaw, &filtered); err != nil || len(filtered.Jobs) != 1 {
		t.Fatal("fixture retained another owner's job", err)
	}
	var metadata struct {
		Files []struct{ Path, SHA256 string } `json:"files"`
	}
	if err := json.Unmarshal(observation.raw, &metadata); err != nil || len(metadata.Files) < 2 {
		t.Fatal("actual worker fingerprint evidence not retained", err)
	}
}

func TestCapturedOwnerTerminalJob3AcceptsHistoricalExecutingMarker(t *testing.T) {
	requestRaw, err := os.ReadFile(filepath.Join("testdata", "azazel-job3-terminal-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	statusRaw, err := os.ReadFile(filepath.Join("testdata", "azazel-job3-terminal-status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request queueRequest
	if err := yamlfile.Read("captured owner terminal request", requestRaw, &request); err != nil {
		t.Fatal(err)
	}
	observation, err := readObservation(statusRaw, request, 3)
	if err != nil {
		t.Fatal(err)
	}
	update := observation.update()
	if observation.job.State != "succeeded" || !observation.job.Executing || update.Sequence != 3 || !update.Terminal {
		t.Fatal("actual owner terminal wire rejected", update)
	}
}
