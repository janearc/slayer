package job

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// examples are the example jobs in doc/examples/jobs/, each with its
// scene's form and whether it is a still.
var examples = map[string]struct {
	form  string
	still bool
}{
	"first-picture.yaml":              {formSmoke, true},
	"first-film.yaml":                 {formSmoke, false},
	"smoke-still.yaml":                {formSmoke, true},
	"lantern-portrait.yaml":           {formNamed, true},
	"orbital-portrait.yaml":           {formNamed, true},
	"city-clustered-10000.yaml":       {formSet, true},
	"city-clustered-green-10000.yaml": {formSet, true},
	"city-clustered-night-10000.yaml": {formSet, true},
	"city-grid-10000.yaml":            {formSet, true},
	"macrostructures.yaml":            {formSet, true},
}

// TestTheExampleJobsRead holds every example in doc/examples/jobs/ to a
// job that is read, of its form, with every file it reads there to read.
func TestTheExampleJobsRead(t *testing.T) {
	paths, err := filepath.Glob(
		filepath.Join("..", "doc", "examples", "jobs", "*.yaml"))
	if err != nil || len(paths) != len(examples) {
		t.Fatalf("doc/examples/jobs/ holds %d examples, want %d: %v",
			len(paths), len(examples), err)
	}
	for _, path := range paths {
		want, listed := examples[filepath.Base(path)]
		job, err := Read(path)
		if !listed || err != nil {
			t.Errorf("%s: listed %v, read %v", path, listed, err)
			continue
		}
		if job.Scene().Form() != want.form ||
			job.Clock().Still() != want.still {
			t.Errorf("%s is a %s still %v", path, job.Scene().Form(),
				job.Clock().Still())
		}
		for _, used := range readsFrom(job) {
			if _, err := os.Stat(used); err != nil {
				t.Errorf("%s reads %s: %v", path, used, err)
			}
		}
	}
}

// readsFrom is every file the job reads that must already be there: its
// set, and its tune and voicing where they are not built-ins.
func readsFrom(job Job) []string {
	var used []string
	if set, has := job.Scene().Set(); has {
		used = append(used, set)
	}
	if track, has := job.Sound(); has {
		score, voicing, _ := track.Tune()
		used = append(used, score, voicing)
	}
	return slices.DeleteFunc(used, func(path string) bool {
		return path == "" || strings.HasPrefix(path, builtIn)
	})
}

// TestAnExampleNamingAMissingSetIsFound is the anti-test: a job whose
// set is not there is caught by the same walk.
// anti-test of TestTheExampleJobsRead
func TestAnExampleNamingAMissingSetIsFound(t *testing.T) {
	job := read(t, still)
	for _, used := range readsFrom(job) {
		if _, err := os.Stat(used); err == nil {
			t.Errorf("%s was found, and should not be", used)
		}
	}
	if len(readsFrom(job)) != one {
		t.Error("the walk did not take the set")
	}
}
