package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/ferruccio/internal/draft"
)

func TestBundledStudyCopyRetainsIndependentRelativeAssets(t *testing.T) {
	examples := makeBundledStudyFixture(t)
	path, study, err := copyBundledStudy(examples, "oakland-larson/set.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if study != "oakland-larson" || filepath.Dir(path) == filepath.Join(examples, "oakland-larson") {
		t.Fatalf("study path/id = %q/%q", path, study)
	}
	if err := os.RemoveAll(filepath.Join(examples, "oakland-larson")); err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	document, err := draft.Open("set.yaml", []byte("scene: {name: edited}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := saveBundledStudyDocument(document, filepath.Join(target, "set.yaml"), filepath.Dir(path), study); err != nil {
		t.Fatal(err)
	}
	for _, name := range bundledStudyFiles(study) {
		working, err := os.ReadFile(filepath.Join(filepath.Dir(path), name))
		if err != nil {
			t.Fatal(err)
		}
		saved, err := os.ReadFile(filepath.Join(target, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(working) != string(saved) {
			t.Fatalf("saved %s differs from working resource", name)
		}
	}
	if err := os.WriteFile(filepath.Join(target, "two-metre-human.bin"), []byte("destination edit"), 0600); err != nil {
		t.Fatal(err)
	}
	working, err := os.ReadFile(filepath.Join(filepath.Dir(path), "two-metre-human.bin"))
	if err != nil || string(working) != "binary fixture bytes" {
		t.Fatalf("saving changed the working source asset: %q, %v", working, err)
	}
}

func TestWorkspaceExampleLoadsTheBundledStudyWithItsAssets(t *testing.T) {
	examples := makeBundledStudyFixture(t)
	work := &Workspace{options: Options{Examples: examples}}
	if err := work.example("oakland-larson/set.yaml"); err != nil {
		t.Fatal(err)
	}
	if work.study != "oakland-larson" || !work.exampleDraft || filepath.Dir(work.state.path) == filepath.Join(examples, "oakland-larson") {
		t.Fatalf("selected study state id=%q exampleDraft=%t path=%q", work.study, work.exampleDraft, work.state.path)
	}
	if err := os.RemoveAll(filepath.Join(examples, "oakland-larson")); err != nil {
		t.Fatal(err)
	}
	for _, name := range bundledStudyFiles(work.study) {
		if _, err := os.Stat(filepath.Join(filepath.Dir(work.state.path), name)); err != nil {
			t.Fatalf("materialized study lost %s: %v", name, err)
		}
	}
}

func TestBundledStudySaveRefusesConflictingAssetBeforeWritingDraft(t *testing.T) {
	examples := makeBundledStudyFixture(t)
	path, study, err := copyBundledStudy(examples, "oakland-larson/set.yaml")
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	conflict := filepath.Join(target, "two-metre-human.bin")
	if err := os.WriteFile(conflict, []byte("user asset must win"), 0600); err != nil {
		t.Fatal(err)
	}
	document, err := draft.Open("set.yaml", []byte("scene: {name: edited}\n"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(target, "set.yaml")
	err = saveBundledStudyDocument(document, output, filepath.Dir(path), study)
	if err == nil || !errors.Is(err, ErrStudyAssetConflict) {
		t.Fatalf("conflicting resource error = %v, want asset conflict", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("conflicting asset still saved draft: stat error %v", err)
	}
	got, err := os.ReadFile(conflict)
	if err != nil || string(got) != "user asset must win" {
		t.Fatalf("conflicting destination asset changed: %q, %v", got, err)
	}
	if document.Text() != "scene: {name: edited}\n" {
		t.Fatal("refused Save As changed the working document")
	}
}

func TestBundledStudySaveAcceptsIdenticalAssets(t *testing.T) {
	examples := makeBundledStudyFixture(t)
	path, study, err := copyBundledStudy(examples, "oakland-larson/set.yaml")
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	for _, name := range bundledStudyFiles(study) {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(path), name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	document, err := draft.Open("set.yaml", []byte("scene: {name: saved}\n"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(target, "set.yaml")
	if err := saveBundledStudyDocument(document, output, filepath.Dir(path), study); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil || string(got) != document.Text() {
		t.Fatalf("draft was not saved beside identical assets: %q, %v", got, err)
	}
}

func makeBundledStudyFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	folder := filepath.Join(root, "oakland-larson")
	if err := os.MkdirAll(folder, 0700); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{
		"set.yaml":             "scene:\n  imports: [two-metre-human.gltf]\n",
		"two-metre-human.gltf": "gltf fixture bytes",
		"two-metre-human.bin":  "binary fixture bytes",
		"LICENSE.txt":          "fixture license bytes",
	} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
