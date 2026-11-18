package workspace

import (
	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/widget"
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"os"
	"path/filepath"
	"testing"
)

func TestNewBuildsHeadlessWorkspaceAndPreservesInvalidDraftOnFailedLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unfinished.yaml")
	const invalid = "scene: [\n"
	if err := os.WriteFile(path, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}
	ui := app.New()
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), ui, Options{File: path})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if work.root == nil || ui.Window().Root() != work.root {
		t.Fatal("New did not install its real widget tree in the headless UI window")
	}
	if got := work.editor.Text(); got != invalid {
		t.Fatalf("editor text = %q, want invalid source %q", got, invalid)
	}
	if work.status != "Draft needs attention" || work.diagnostic == "" {
		t.Fatalf("invalid draft status/diagnostic = %q/%q", work.status, work.diagnostic)
	}

	document, originalPath := work.state.document, work.state.path
	if err := work.state.load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("missing replacement file was accepted")
	}
	if work.state.document != document || work.state.path != originalPath || work.editor.Text() != invalid {
		t.Fatal("failed load replaced the visible invalid draft or its source identity")
	}
}

func TestNewCopiesExampleToIndependentEditableFile(t *testing.T) {
	examples := t.TempDir()
	path := filepath.Join(examples, "first-picture.yaml")
	const example = "scene:\n  name: first picture\n"
	if err := os.WriteFile(path, []byte(example), 0o644); err != nil {
		t.Fatal(err)
	}
	ui := app.New()
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), ui, Options{Examples: examples})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if work.root == nil || ui.Window().Root() != work.root {
		t.Fatal("New did not install its real widget tree in the headless UI window")
	}
	if work.DocumentName() != "first-picture.yaml" || work.editor.Text() != example {
		t.Fatalf("active example name/text = %q/%q", work.DocumentName(), work.editor.Text())
	}
	if work.state.path == path || filepath.Dir(work.state.path) == examples {
		t.Fatalf("editable example still points into the bundled examples: %q", work.state.path)
	}
	copied, err := os.ReadFile(work.state.path)
	if err != nil {
		t.Fatalf("read editable example: %v", err)
	}
	if string(copied) != example {
		t.Fatalf("copied example = %q, want %q", copied, example)
	}
	info, err := os.Stat(work.state.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("editable example mode = %04o, want 0600", info.Mode().Perm())
	}
	if err := os.WriteFile(path, []byte("updated bundled example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unchanged, err := os.ReadFile(work.state.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != example || work.state.document.Text() != example {
		t.Fatal("changing the bundled example changed the user's editable copy")
	}
}

func TestWorkspaceSourceIdentitySurvivesConstruction(t *testing.T) {
	folder := t.TempDir()
	const source = "scene:\n  name: same text\n"
	first, second := filepath.Join(folder, "first.yaml"), filepath.Join(folder, "second.yaml")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{File: first})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	plan, err := preview.Prepare(preview.Request{Path: first, Text: source})
	if err != nil {
		t.Fatalf("prepare preview: %v", err)
	}
	defer plan.Close()
	if !work.state.accepts(plan) {
		t.Fatal("preview for the active workspace source was refused")
	}
	if err := work.state.load(second); err != nil {
		t.Fatalf("load second source: %v", err)
	}
	if work.state.accepts(plan) {
		t.Fatal("preview for another path with identical text was accepted")
	}
}

// TestSavingAnExampleRequiresDestination protects work from temporary-only saves.
func TestSavingAnExampleRequiresDestination(t *testing.T) {
	examples := t.TempDir()
	source := "scene: {}\n"
	if err := os.WriteFile(filepath.Join(examples, "first-picture.yaml"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{Examples: examples})
	if err != nil {
		t.Fatal(err)
	}
	temporaryPath := work.state.path
	work.state.document.Replace("unfinished: [")
	// The real native picker refuses to run before platform initialization.
	if work.save(false) {
		t.Fatal("example was marked saved without choosing a durable destination")
	}
	if !work.state.document.Dirty() {
		t.Fatal("refused save cleared unsaved work")
	}
	raw, err := os.ReadFile(temporaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != source {
		t.Fatal("refused destination silently wrote the temporary example")
	}
}

// TestStatusAndRenderLogRemainVisible protects the footer during render changes.
func TestStatusAndRenderLogRemainVisible(t *testing.T) {
	work, err := New(gogpu.NewApp(gogpu.DefaultConfig()), app.New(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{work.status, "Render log"} {
		var found bool
		var visit func(widget.Widget)
		visit = func(current widget.Widget) {
			if current == nil {
				return
			}
			if label, ok := current.(interface{ AccessibilityLabel() string }); ok && label.AccessibilityLabel() == want {
				found = true
			}
			if identified, ok := current.(interface{ ID() string }); ok && identified.ID() == want {
				found = true
			}
			for _, child := range current.Children() {
				visit(child)
			}
		}
		visit(work.root)
		if !found {
			t.Errorf("visible workspace tree has no %q", want)
		}
	}
}
