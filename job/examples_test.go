package job

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// examples are the example jobs in jobs/, each with its scene's form and
// whether it is a still.
var examples = map[string]struct {
	form  string
	still bool
}{
	"first-picture.yaml":    {formSmoke, true},
	"first-film.yaml":       {formSmoke, false},
	"moonlake-still.yaml":   {formSet, true},
	"smoke-still.yaml":      {formSmoke, true},
	"moonlake-film.yaml":    {formSet, false},
	"portrait-short.yaml":   {formSet, false},
	"frames-short.yaml":     {formFrames, false},
	"short-fire-night.yaml": {formSet, false},
	"blur-fire-night.yaml":  {formSet, false},
	"lantern-portrait.yaml": {formNamed, true},
	"orbital-portrait.yaml": {formNamed, true},
}

// TestTheExampleJobsRead holds every example in jobs/ to a job that is
// read, of its form, with every file it reads there to read.
func TestTheExampleJobsRead(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join(root, "jobs", "*.yaml"))
	if err != nil || len(paths) != len(examples) {
		t.Fatalf("jobs/ holds %d examples, want %d: %v", len(paths),
			len(examples), err)
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
