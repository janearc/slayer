package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRecordRejectsTruncatedPNG checks that a file whose header is good but
// whose pixels are cut short is a failure before any receipt is written.
func TestRecordRejectsTruncatedPNG(t *testing.T) {
	plan, tools := queuedPlan(t)
	queuePNG(t, plan, 16, 12)
	info, err := os.Stat(plan.Output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(plan.Output, info.Size()-12); err != nil {
		t.Fatal(err)
	}
	err = plan.record(tools)
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("record = %v; want a truncated PNG failure", err)
	}
	if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); !os.IsNotExist(err) {
		t.Fatal("a truncated PNG received a receipt")
	}
}

// TestRecordAcceptsWholePNG is the anti-test: a complete file of the frozen
// size is accepted and receives its receipt.
func TestRecordAcceptsWholePNG(t *testing.T) {
	plan, tools := queuedPlan(t)
	queuePNG(t, plan, 16, 12)
	if err := plan.record(tools); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(plan.Directory, "receipt.json")); err != nil {
		t.Fatal("a whole PNG received no receipt", err)
	}
}

// TestPrepareRemovesItsFolderOnLaterError checks that a failure after the
// manifest copy leaves nothing under the output root.
func TestPrepareRemovesItsFolderOnLaterError(t *testing.T) {
	root, set := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(set, "sources.yaml"), []byte("version: 1\nsources: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(set, "scene.yaml")
	text := string(firstPicture(t))
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(set, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(set, 0700) })
	if _, err := Prepare(Request{Path: path, Text: text, OutputRoot: root, Profile: Profile{Width: 16, Height: 12}}); err == nil {
		t.Skip("the set folder stayed writable, so the later error cannot be made")
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("output root holds %v after a failed prepare (%v)", entries, err)
	}
}

// TestPrepareKeepsItsFolderOnSuccess is the anti-test: a prepare that works
// leaves exactly its folder.
func TestPrepareKeepsItsFolderOnSuccess(t *testing.T) {
	plan, _ := queuedPlan(t)
	if _, err := os.Stat(plan.Directory); err != nil {
		t.Fatal("a successful prepare lost its folder", err)
	}
	if _, err := os.Stat(plan.Input); err != nil {
		t.Fatal("a successful prepare lost its input", err)
	}
}

// TestManifestSearchStopsAtTheSetsTree checks that a manifest above a set
// outside any repository is not picked up, and that one inside the set's own
// repository still is.
func TestManifestSearchStopsAtTheSetsTree(t *testing.T) {
	outer := t.TempDir()
	if err := os.WriteFile(filepath.Join(outer, "sources.yaml"), []byte("outer\n"), 0600); err != nil {
		t.Fatal(err)
	}
	set := filepath.Join(outer, "sets")
	if err := os.Mkdir(set, 0700); err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if err := CopySourceManifest(filepath.Join(set, "scene.yaml"), destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "sources.yaml")); !os.IsNotExist(err) {
		t.Fatal("a manifest outside the set's tree was copied")
	}
	if err := os.Mkdir(filepath.Join(outer, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := CopySourceManifest(filepath.Join(set, "scene.yaml"), destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, "sources.yaml")); err != nil {
		t.Fatal("the repository's own manifest was not copied", err)
	}
}
