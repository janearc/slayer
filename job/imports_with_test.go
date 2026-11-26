package job_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/slayer/job"
)

func TestStillFromRetainsBufferedImportClosure(t *testing.T) {
	path, buffer := importJobFiles(t)
	setPath := filepath.Join(filepath.Dir(path), "set.yaml")
	raw, err := os.ReadFile(setPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte("value: 2, unit: m"), []byte("value: 3, unit: m"), 1)
	if bytes.Equal(raw, changed) {
		t.Fatal("fixture did not change")
	}
	still, err := job.StillFrom(setPath, changed)
	if err != nil {
		t.Fatal(err)
	}
	read, has := still.ImportedSet()
	if !has || len(read.ImportSources()) != 2 {
		t.Fatal("buffered still did not retain its relative import closure")
	}
	before := still.Canonical()
	if err := os.Remove(buffer); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(still.Canonical(), before) {
		t.Fatal("buffered still changed after its imported asset was removed")
	}
	if _, err := job.StillFrom(setPath, changed); err == nil {
		t.Fatal("a fresh buffered still accepted a missing asset")
	}
}

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
	outputDigests := inputDigests(t, output.Canonical())
	if !reflect.DeepEqual(outputDigests, originalInputs) {
		t.Fatal("changing output changed the fixed input digests")
	}

	width, err := first.With(job.Width(640))
	if err != nil {
		t.Fatal(err)
	}
	if width.PictureKey() == originalPicture {
		t.Fatal("changing width retained the picture identity")
	}
	widthDigests := inputDigests(t, width.Canonical())
	if !reflect.DeepEqual(widthDigests, originalInputs) {
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
