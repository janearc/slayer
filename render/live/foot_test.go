package live

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// the foot keeps the last few lines, newest at the bottom, and a trouble
// with the file below them until it is cleared.
func TestFootScrollsAndHoldsTheTrouble(t *testing.T) {
	var shown foot
	for _, line := range []string{"one", "two", "three", "four"} {
		shown.note(line)
	}
	if got := shown.lines(); strings.Join(got, ",") != "two,three,four" {
		t.Errorf("the foot shows %q", got)
	}
	shown.troubled("the job\ndoes not read")
	got := shown.lines()
	if got[len(got)-one] != "the job does not read" ||
		len(got) != FootRows {
		t.Errorf("the trouble shows as %q", got)
	}
	if !shown.cleared() || shown.cleared() {
		t.Error("a trouble did not clear once and only once")
	}
}

// anti-test of TestFootScrollsAndHoldsTheTrouble
// a note does not clear the trouble, and a foot with fewer
// lines than rows is padded above, not below.
func TestFootNotesLeaveTheTrouble(t *testing.T) {
	var shown foot
	shown.troubled("bad")
	shown.note("a note")
	got := shown.lines()
	if got[len(got)-one] != "bad" || got[none] != "" {
		t.Errorf("the foot shows %q", got)
	}
}

// the edits are written into the file: a still's moment, a camera by
// name with the zoom kept, the film's exposure; the comments stay.
func TestWithEditsWritesTheKeys(t *testing.T) {
	raw := []byte("# the job\n" + stillJob +
		"camera: {name: far, zoom: {from: {value: 1, unit: x}, " +
		"to: {value: 2, unit: x}, duration: {value: 1, unit: s}}}\n")
	written, err := withEdits(raw, edits{moment: 2500 * time.Millisecond,
		said: true, camera: "near", exposure: 4})
	if err != nil {
		t.Fatal(err)
	}
	var read struct {
		At     struct{ Value float64 }
		Camera struct {
			Name string
			Zoom any
		}
		Film struct{ Exposure, Width float64 }
	}
	if err := yaml.Unmarshal(written, &read); err != nil {
		t.Fatal(err)
	}
	if read.At.Value != 2.5 || read.Camera.Name != "near" ||
		read.Camera.Zoom == nil || read.Film.Exposure != 4 ||
		read.Film.Width != 40 {
		t.Errorf("wrote %s", written)
	}
	if !bytes.Contains(written, []byte("# the job")) {
		t.Errorf("the comment was lost: %s", written)
	}
}

// anti-test of TestWithEditsWritesTheKeys
// no edits leave the keys as they were, a film is made where
// the job has none, and a file that is not one mapping is refused.
func TestWithEditsLeavesTheRest(t *testing.T) {
	written, err := withEdits([]byte(jobLine), edits{})
	if err != nil || bytes.Contains(written, []byte("camera")) ||
		bytes.Contains(written, []byte("at:")) {
		t.Errorf("no edits wrote %s, %v", written, err)
	}
	made, err := withEdits([]byte(jobLine), edits{exposure: 3})
	if err != nil || !bytes.Contains(made, []byte("exposure: 3")) {
		t.Errorf("an exposure with no film wrote %s, %v", made, err)
	}
	if _, err := withEdits([]byte("- a list\n"), edits{camera: "x"}); err ==
		nil {
		t.Error("a list was edited as a job")
	}
	if _, err := withEdits([]byte("a: [b"), edits{}); err == nil {
		t.Error("broken YAML was edited")
	}
}

// writeEdits keeps the file's mode and leaves no file beside it.
func TestWriteEditsKeepsTheMode(t *testing.T) {
	path := jobFile(t, stillJob)
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeEdits(path, edits{camera: "near"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("the mode is now %v, %v", info.Mode(), err)
	}
	entries, _ := os.ReadDir(strings.TrimSuffix(path, "test.yaml"))
	if len(entries) != one {
		t.Errorf("the folder holds %d files", len(entries))
	}
}

// anti-test of TestWriteEditsKeepsTheMode
// a file that is not there, or not a job's shape, is not
// written.
func TestWriteEditsRefusesWhatItCannotEdit(t *testing.T) {
	if err := writeEdits("/nowhere/at/all.yaml", edits{}); err == nil {
		t.Error("a file that is not there was edited")
	}
	path := jobFile(t, "- a list\n")
	if err := writeEdits(path, edits{camera: "x"}); err == nil ||
		!strings.Contains(err.Error(), path) {
		t.Errorf("a list was edited, or refused without its name: %v",
			err)
	}
}
