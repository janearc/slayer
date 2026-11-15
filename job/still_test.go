package job

import (
	"errors"
	"path/filepath"
	"testing"
)

// a set file given alone is a still of that set, its path absolute, with
// no job file and nothing else said.
func TestStillOfIsTheLeastStill(t *testing.T) {
	directory := t.TempDir()
	set := filepath.Join(directory, "moonlake.yaml")
	made, err := StillOf(set)
	if err != nil {
		t.Fatal(err)
	}
	said, isSet := made.Scene().Set()
	if !isSet || said != set {
		t.Errorf("the still's set is %q, %v", said, isSet)
	}
	if !made.Clock().Still() || made.File() != "" {
		t.Errorf("a set alone is not a still with no job file: %q",
			made.File())
	}
	if _, has := made.Outputs().Picture(); has {
		t.Error("a set alone said a picture")
	}
	read, err := Read(written(t, still))
	if err != nil {
		t.Fatal(err)
	}
	if read.Key() == made.Key() {
		t.Error("two different sets made one key")
	}
}

// the anti-test: a set given by a relative path is made absolute, never
// kept relative to a directory nobody can see, and an empty one is
// refused as a job with no scene is.
// anti-test of TestStillOfIsTheLeastStill
func TestStillOfRefusesNoSet(t *testing.T) {
	made, err := StillOf("moonlake.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if said, _ := made.Scene().Set(); !filepath.IsAbs(said) {
		t.Errorf("a relative set stayed relative: %q", said)
	}
	if _, err := StillOf(""); err == nil {
		t.Error("an empty set path was taken")
	} else if !errors.Is(err, ErrRefused) {
		t.Errorf("an empty set path was not refused as a job: %v", err)
	}
}
