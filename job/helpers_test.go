package job

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// private is defined as a file only its owner reads and writes.
const private = 0o600

// heading is every job's first line.
const heading = "$schema: " + schemaName + "\n"

// still is the least job: a still of a set.
const still = heading + "scene: {set: moonlake.yaml}\n"

// film is the least film: a set, a rate and a duration.
const film = still + "fps: 24\nduration: {value: 2, unit: s}\n"

//go:embed testdata/numbers.yaml
var numbersRaw []byte

// number is a value the tests chose, by name, from testdata/numbers.yaml.
func number(t *testing.T, name string) float64 {
	t.Helper()
	chosen, err := numbers.Read("testdata/numbers.yaml", numbersRaw)
	if err != nil {
		t.Fatal(err)
	}
	value, err := chosen.Get(name)
	if err != nil {
		t.Fatal(err)
	}
	return value.Value
}

// written is a job file holding text, in a directory of its own.
func written(t *testing.T, text string) string {
	t.Helper()
	return writtenIn(t, t.TempDir(), "job.yaml", text)
}

// writtenIn is a job file holding text, by name in directory, so jobs
// whose paths are the same are written beside each other.
func writtenIn(t *testing.T, directory, name, text string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(text), private); err != nil {
		t.Fatal(err)
	}
	return path
}

// reading is the job text says, read from a file.
func reading(t *testing.T, text string) (Job, error) {
	t.Helper()
	return Read(written(t, text))
}

// read is the job text says, which must be taken.
func read(t *testing.T, text string) Job {
	t.Helper()
	job, err := reading(t, text)
	if err != nil {
		t.Fatalf("%q was refused: %v", text, err)
	}
	return job
}

// refused is the refusal of the job text says, which must be refused as
// a job, and must say every one of words.
func refused(t *testing.T, text string, words ...string) error {
	t.Helper()
	_, err := reading(t, text)
	if err == nil {
		t.Fatalf("%q was taken", text)
	}
	saying(t, err, words...)
	return err
}

// saying fails unless err is a refusal saying every one of words.
func saying(t *testing.T, err error, words ...string) {
	t.Helper()
	if !errors.Is(err, ErrRefused) {
		t.Errorf("%v is not a refusal", err)
	}
	for _, word := range words {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("%q does not say %q", err, word)
		}
	}
}
