package preview

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

func firstPicture(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "first-picture.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func readMapping(t *testing.T, name string, source []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := yamlfile.Read(name, source, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPrepareMakesAdjacentJobSnapshotWithoutOutputs(t *testing.T) {
	source := strings.Replace(string(firstPicture(t)), "scene: {smoke: 1}", "scene: {set: ../scenes/moonlake.yaml}", 1)
	root := t.TempDir()
	jobDir := filepath.Join(root, "jobs")
	if err := os.MkdirAll(jobDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(jobDir, "first-picture.yaml")
	originalBytes := []byte(source)
	if err := os.WriteFile(original, originalBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := Prepare(Request{Path: original, Text: source})
	if err != nil || plan == nil {
		t.Fatalf("Prepare: plan=%v, err=%v", plan, err)
	}
	if plan.Input == original || filepath.Dir(plan.Input) != jobDir {
		t.Errorf("snapshot input %q is not adjacent to original %q", plan.Input, original)
	}
	if !filepath.IsAbs(plan.Output) || filepath.Dir(plan.Output) != plan.Directory || !filepath.IsAbs(plan.Directory) {
		t.Errorf("output is not an absolute path in its new directory: plan=%+v", plan)
	}
	if plan.Digest == "" {
		t.Error("prepared plan has no digest")
	}
	closed := false
	t.Cleanup(func() {
		if !closed {
			if err := plan.Close(); err != nil {
				t.Errorf("Close: %v", err)
			}
		}
	})

	snapshotBytes, err := os.ReadFile(plan.Input)
	if err != nil {
		t.Fatal(err)
	}
	originalDocument := readMapping(t, original, originalBytes)
	snapshotDocument := readMapping(t, plan.Input, snapshotBytes)
	if _, exists := originalDocument["outputs"]; !exists {
		t.Fatal("fixture setup lost original outputs")
	}
	delete(originalDocument, "outputs")
	if !reflect.DeepEqual(snapshotDocument, originalDocument) {
		t.Errorf("snapshot changed fields beyond outputs removal:\n got %#v\nwant %#v", snapshotDocument, originalDocument)
	}
	scene := snapshotDocument["scene"].(map[string]any)
	if scene["set"] != "../scenes/moonlake.yaml" {
		t.Errorf("relative scene.set = %v; want path retained beside original", scene["set"])
	}
	if got, err := os.ReadFile(original); err != nil || string(got) != source {
		t.Fatalf("Prepare changed original source: %v", err)
	}
	for _, name := range []string{"first-picture.png", "first-picture.pfm"} {
		path := filepath.Join(root, "out", name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Prepare created original job output %q: %v", path, err)
		}
	}
	if err := os.WriteFile(plan.Output, []byte("preview output sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plan.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	if _, err := os.Stat(plan.Input); !os.IsNotExist(err) {
		t.Errorf("Close left temporary input: %v", err)
	}
	if info, err := os.Stat(plan.Directory); err != nil || !info.IsDir() {
		t.Errorf("Close removed preview directory: %v", err)
	}
	if got, err := os.ReadFile(plan.Output); err != nil || string(got) != "preview output sentinel" {
		t.Errorf("Close changed preview output: %q, %v", got, err)
	}
	if got, err := os.ReadFile(original); err != nil || string(got) != source {
		t.Errorf("Close changed original source: %v", err)
	}
}

func TestPrepareRefusesFilmJobsAndMalformedYAMLBeforeSnapshot(t *testing.T) {
	film := strings.Replace(string(firstPicture(t)), "at: {value: 0, unit: s}", "fps: 24\nduration: {value: 10, unit: s}", 1)
	for _, testCase := range []struct {
		name string
		text string
	}{
		{"film", film},
		{"malformed", "$schema: [unfinished"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			original := filepath.Join(root, "draft.yaml")
			if err := os.WriteFile(original, []byte(testCase.text), 0o600); err != nil {
				t.Fatal(err)
			}
			if plan, err := Prepare(Request{Path: original, Text: testCase.text}); err == nil {
				if plan != nil {
					plan.Close()
				}
				t.Fatalf("Prepare accepted %s job", testCase.name)
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != filepath.Base(original) {
				t.Errorf("%s refusal created snapshot artifacts: %v", testCase.name, entries)
			}
			got, err := os.ReadFile(original)
			if err != nil || string(got) != testCase.text {
				t.Errorf("%s refusal changed original source: %v", testCase.name, err)
			}
		})
	}
}

func catalogueSettings(t *testing.T) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "subaru-settings.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func settingsFile(t *testing.T, source []byte, sidecar string) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "settings.yaml")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	if sidecar != "" {
		if err := os.WriteFile(filepath.Join(directory, "render.json"), []byte(sidecar), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestPrepareCatalogueSelectsOneSpecimenDetailWithoutChangingSource(t *testing.T) {
	source := []byte(strings.Replace(string(catalogueSettings(t)), "details: [demo]", "details: [demo, quick]", 1))
	original := settingsFile(t, source, "")
	plan, err := Prepare(Request{Path: original, Text: string(source), Specimen: "subaru"})
	if err != nil || plan == nil {
		t.Fatalf("Prepare catalogue: plan=%+v, err=%v", plan, err)
	}
	defer func() {
		if err := plan.Close(); err != nil {
			t.Errorf("Close input: %v", err)
		}
		if err := os.RemoveAll(plan.Directory); err != nil {
			t.Errorf("remove output folder: %v", err)
		}
	}()
	if plan.Kind != "catalogue" || plan.Specimen != "subaru" || plan.Detail != "demo" {
		t.Errorf("catalogue plan identity = %q/%q/%q, want catalogue/subaru/demo", plan.Kind, plan.Specimen, plan.Detail)
	}
	if want := filepath.Join(plan.Directory, "book", "subaru-demo", "specimen.png"); plan.Output != want {
		t.Errorf("output = %q, want fresh specimen path %q", plan.Output, want)
	}
	if !filepath.IsAbs(plan.Directory) || !filepath.IsAbs(plan.Output) {
		t.Errorf("catalogue output paths are not absolute: %+v", plan)
	}
	snapshot, err := os.ReadFile(plan.Input)
	if err != nil {
		t.Fatal(err)
	}
	prepared := readMapping(t, plan.Input, snapshot)
	if prepared["width"] != 640 || prepared["height"] != 360 || prepared["rays"] != 1 {
		t.Errorf("snapshot rendering bounds = %v x %v, rays=%v; want 640x360, rays=1", prepared["width"], prepared["height"], prepared["rays"])
	}
	if got := prepared["details"]; !reflect.DeepEqual(got, []any{"demo"}) {
		t.Errorf("snapshot details = %#v, want only first detail [demo]", got)
	}
	originalSettings := readMapping(t, original, source)
	wantSnapshot := readMapping(t, original, source)
	wantSnapshot["width"] = 640
	wantSnapshot["height"] = 360
	wantSnapshot["rays"] = 1
	wantSnapshot["details"] = []any{"demo"}
	if !reflect.DeepEqual(prepared, wantSnapshot) {
		t.Errorf("catalogue snapshot changed fields beyond render bounds and selected detail:\n got %#v\nwant %#v", prepared, wantSnapshot)
	}
	if !reflect.DeepEqual(originalSettings, readMapping(t, original, source)) {
		t.Error("reading snapshot comparison changed original settings")
	}
	if got, err := os.ReadFile(original); err != nil || !reflect.DeepEqual(got, source) {
		t.Errorf("Prepare changed original catalogue settings: %v", err)
	}
}

func TestPrepareCatalogueRecoversKindOnlyFromAdjacentRenderReceipt(t *testing.T) {
	source := catalogueSettings(t)
	receipt := `{"specimen":{"entry":{"kind":"subaru"}}}`
	original := settingsFile(t, source, receipt)
	plan, err := Prepare(Request{Path: original, Text: string(source)})
	if err != nil || plan == nil {
		t.Fatalf("Prepare from adjacent render receipt: plan=%+v, err=%v", plan, err)
	}
	defer func() {
		_ = plan.Close()
		_ = os.RemoveAll(plan.Directory)
	}()
	if plan.Kind != "catalogue" || plan.Specimen != "subaru" || plan.Detail != "demo" {
		t.Errorf("recovered plan identity = %q/%q/%q, want catalogue/subaru/demo", plan.Kind, plan.Specimen, plan.Detail)
	}
}

func TestPrepareCatalogueExplicitSpecimenOverridesInvalidReceipt(t *testing.T) {
	source := catalogueSettings(t)
	original := settingsFile(t, source, `{"specimen":{"entry":{"kind":[]}}}`)
	plan, err := Prepare(Request{Path: original, Text: string(source), Specimen: "subaru"})
	if err != nil || plan == nil {
		t.Fatalf("explicit specimen should override invalid adjacent receipt: plan=%+v, err=%v", plan, err)
	}
	defer func() {
		_ = plan.Close()
		_ = os.RemoveAll(plan.Directory)
	}()
	if plan.Specimen != "subaru" || plan.Detail != "demo" {
		t.Errorf("explicit plan identity = %q/%q; want subaru/demo", plan.Specimen, plan.Detail)
	}
}

func TestPrepareCatalogueRefusesMissingOrInvalidReceiptBeforeArtifacts(t *testing.T) {
	source := catalogueSettings(t)
	for _, testCase := range []struct {
		name    string
		receipt string
	}{
		{"missing", ""},
		{"malformed", "{"},
		{"wrong type", `{"specimen":{"entry":{"kind":[]}}}`},
		{"duplicate kind", `{"specimen":{"entry":{"kind":"subaru","kind":"subaru"}}}`},
		{"unknown specimen", `{"specimen":{"entry":{"kind":"not-in-menu"}}}`},
		{"oversized", strings.Repeat(" ", 4*1024*1024+1)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			original := settingsFile(t, source, testCase.receipt)
			plan, err := Prepare(Request{Path: original, Text: string(source)})
			if plan != nil {
				_ = plan.Close()
				_ = os.RemoveAll(plan.Directory)
			}
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "select a specimen") {
				t.Fatalf("catalogue without a valid specimen receipt error = %v, want a request to select a specimen", err)
			}
			entries, readErr := os.ReadDir(filepath.Dir(original))
			if readErr != nil {
				t.Fatal(readErr)
			}
			wantEntries := 1
			if testCase.receipt != "" {
				wantEntries = 2
			}
			if len(entries) != wantEntries {
				t.Errorf("refusal created snapshot artifacts: %v", entries)
			}
		})
	}
}

func TestPrepareCatalogueRejectsUnsafeAndUnknownSpecimenSelectorsBeforeArtifacts(t *testing.T) {
	source := catalogueSettings(t)
	for _, selector := range []string{"../subaru", "-all", "subaru night", " subaru", "not-in-menu"} {
		t.Run(selector, func(t *testing.T) {
			original := settingsFile(t, source, "")
			plan, err := Prepare(Request{Path: original, Text: string(source), Specimen: selector})
			if plan != nil {
				_ = plan.Close()
				_ = os.RemoveAll(plan.Directory)
			}
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), "specimen") {
				t.Fatalf("selector %q error = %v; want specimen selector refusal", selector, err)
			}
			entries, readErr := os.ReadDir(filepath.Dir(original))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 1 || entries[0].Name() != filepath.Base(original) {
				t.Errorf("selector refusal created snapshot artifacts: %v", entries)
			}
		})
	}
}
