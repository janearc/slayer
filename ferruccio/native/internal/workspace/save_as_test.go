package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const savedJobSource = `# keep the job comment
$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene:
  set: &original location.yaml # keep the source comment
outputs:
  picture: *original
`

func TestSaveAsKeepsTheJobSceneAndEditHistory(t *testing.T) {
	work, original, target := jobSaveFixture(t)
	edited := savedJobSource + "# authored edit\n"
	work.editor.ApplyText(edited)
	if !work.saveAt(target) {
		t.Fatal(work.diagnostic)
	}
	reopened, _, err := ReadDocument(target)
	if err != nil {
		t.Fatal(err)
	}
	scene := filepath.Join(filepath.Dir(original), "location.yaml")
	if got := filepath.Join(filepath.Dir(target), reopened.ReferencedSet()); got != scene {
		t.Fatalf("reopened job resolves the scene to %s, want %s", got, scene)
	}
	output, err := reopened.Value("outputs", "picture")
	if err != nil || output != "location.yaml" {
		t.Fatalf("unrelated output was changed to %q: %v", output, err)
	}
	for _, comment := range []string{"# keep the job comment", "# keep the source comment", "# authored edit"} {
		if !strings.Contains(reopened.Text(), comment) {
			t.Fatalf("saved job lost %q", comment)
		}
	}
	before, err := os.ReadFile(original)
	if err != nil || string(before) != savedJobSource {
		t.Fatalf("save changed the original job: %q, %v", before, err)
	}
	work.editor.Undo(work.ui.Window().Context())
	if work.state.document.ReferencedSet() != reopened.ReferencedSet() ||
		strings.Contains(work.editor.Text(), "# authored edit") {
		t.Fatal("undo reverted the relocation or left the authored edit")
	}
	work.editor.Redo(work.ui.Window().Context())
	if work.editor.Text() != reopened.Text() || work.state.document.Dirty() {
		t.Fatal("redo did not restore the saved edit")
	}
}

func TestSaveAsFailureKeepsOriginalJobAndHistory(t *testing.T) {
	work, original, target := jobSaveFixture(t)
	edited := savedJobSource + "# authored edit\n"
	work.editor.ApplyText(edited)
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if work.saveAt(target) {
		t.Fatal("saved a job over a directory")
	}
	if work.state.path != original || work.editor.Text() != edited || !work.state.document.Dirty() {
		t.Fatal("failed save changed the working path or draft")
	}
	work.editor.Undo(work.ui.Window().Context())
	if got := work.state.document.Text(); got != savedJobSource {
		t.Fatalf("failed save changed undo history: %q", got)
	}
}

func TestSaveAsKeepsUnnamedSceneInputsAndSameDirectoryJobsVerbatim(t *testing.T) {
	work, original, target := jobSaveFixture(t)
	sameDirectory := filepath.Join(filepath.Dir(original), "copy.yaml")
	if !work.saveAt(sameDirectory) || work.state.document.Text() != savedJobSource {
		t.Fatalf("same-directory save rewrote the job: %s", work.diagnostic)
	}
	named := "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\nscene: {smoke: 1}\n"
	work.editor.ApplyText(named)
	if !work.saveAt(target) || work.state.document.Text() != named {
		t.Fatalf("save rewrote a job with no input path: %s", work.diagnostic)
	}
}

func jobSaveFixture(t *testing.T) (*Workspace, string, string) {
	t.Helper()
	from, to := t.TempDir(), t.TempDir()
	path := filepath.Join(from, "job.yaml")
	if err := os.WriteFile(path, []byte(savedJobSource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(from, "location.yaml"), []byte(checkingSource), 0600); err != nil {
		t.Fatal(err)
	}
	work := libraryEventWorkspace(t)
	work.openPath(path)
	if work.state.path != path {
		t.Fatal(work.diagnostic)
	}
	return work, path, filepath.Join(to, "copy.yaml")
}
