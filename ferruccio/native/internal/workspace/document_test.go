package workspace

import (
	"github.com/janearc/slayer/ferruccio/internal/preview"
	"os"
	"path/filepath"
	"testing"
)

// TestRenderIdentity distinguishes identical text opened from different paths.
func TestRenderIdentity(t *testing.T) {
	folder := t.TempDir()
	source := "scene:\n  name: test\n"
	first, second := filepath.Join(folder, "first.yaml"), filepath.Join(folder, "second.yaml")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	state := &documentState{}
	if err := state.load(first); err != nil {
		t.Fatal(err)
	}
	if state.document == nil {
		t.Fatal("load did not install the document")
	}
	plan, err := preview.Prepare(preview.Request{Path: first, Text: source})
	if err != nil {
		t.Fatal(err)
	}
	defer plan.Close()
	if !state.accepts(plan) {
		t.Fatal("current render refused")
	}
	if err := state.load(second); err != nil {
		t.Fatal(err)
	}
	if state.accepts(plan) {
		t.Fatal("render for a different source accepted")
	}
	if err := state.load(first); err != nil {
		t.Fatal(err)
	}
	state.document.Replace(source + "# changed\n")
	if state.accepts(plan) {
		t.Fatal("render for older text accepted")
	}
}

// TestFailedLoadPreservesDraft protects edited work against a failed open.
func TestFailedLoadPreservesDraft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "draft.yaml")
	if err := os.WriteFile(path, []byte("scene: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	state := &documentState{}
	if err := state.load(path); err != nil {
		t.Fatal(err)
	}
	if state.document == nil {
		t.Fatal("no document")
	}
	state.document.Replace("unfinished: [")
	if err := state.load(path + ".missing"); err == nil {
		t.Fatal("missing file accepted")
	}
	if state.document.Text() != "unfinished: [" || state.path != path || !state.document.Dirty() {
		t.Fatal("failed open changed the draft")
	}
}
