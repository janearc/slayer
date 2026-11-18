package job_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/slayer/job"
)

// TestWithKeepsImportedSnapshotAfterItsFilesGoAway holds an override over
// the admitted import snapshot, not a fresh read of files that may be gone.
func TestWithKeepsImportedSnapshotAfterItsFilesGoAway(t *testing.T) {
	path, buffer := importJobFiles(t)
	first, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	originalSources, has := first.ImportedSet()
	if !has || len(originalSources.ImportSources()) != 2 {
		t.Fatal("job did not freeze its imported source closure")
	}
	originalInputs := inputDigests(t, first.Canonical())
	originalKey, originalPicture := first.Key(), first.PictureKey()

	for _, source := range []string{buffer,
		filepath.Join(filepath.Dir(buffer), "two-metre-human.gltf"),
		filepath.Join(filepath.Dir(path), "set.yaml")} {
		if err := os.Remove(source); err != nil {
			t.Fatal(err)
		}
	}

	output, err := first.With(job.Output("new-picture.png"))
	if err != nil {
		t.Fatal(err)
	}
	retained, retainedSet := output.ImportedSet()
	if !retainedSet || len(retained.ImportSources()) != 2 {
		t.Fatal("With dropped the admitted imported source closure")
	}
	if output.Key() != originalKey || output.PictureKey() != originalPicture {
		t.Fatal("changing only output changed picture identity")
	}
	if !reflect.DeepEqual(inputDigests(t, output.Canonical()), originalInputs) {
		t.Fatal("changing output changed the fixed input digests")
	}

	width, err := first.With(job.Width(640))
	if err != nil {
		t.Fatal(err)
	}
	if width.PictureKey() == originalPicture {
		t.Fatal("changing width retained the picture identity")
	}
	if !reflect.DeepEqual(inputDigests(t, width.Canonical()), originalInputs) {
		t.Fatal("changing width changed the fixed input digests")
	}
}

// inputDigests reads the exact admitted hashes from the canonical job form.
func inputDigests(t *testing.T, canonical []byte) map[string]any {
	t.Helper()
	var form struct {
		Inputs map[string]any `json:"inputs"`
	}
	if err := json.Unmarshal(canonical, &form); err != nil {
		t.Fatal(err)
	}
	return form.Inputs
}
