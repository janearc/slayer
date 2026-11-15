package hillside

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a plan's file lifts to the job of the same frames: its smoke, its
// rate and its frames, its output as the job's frames.
func TestLiftSequenceIsThePlansJob(t *testing.T) {
	path := shortSmokePlan(t, sessionPlainSmoke)
	lifted, err := LiftSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := readSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	if lifted.Key() != plan.lifted.Key() {
		t.Error("the lifted job is not the plan's own")
	}
	smoke, isSmoke := lifted.Scene().Smoke()
	frames, hasFrames := lifted.Outputs().Frames()
	if !isSmoke || smoke != sessionPlainSmoke || !hasFrames ||
		frames != plan.output {
		t.Errorf("lifted smoke %d %v, frames %q %v", smoke, isSmoke,
			frames, hasFrames)
	}
}

// the anti-test: a plan the sequence verb refuses is refused here, and
// a job file is not taken for a plan.
func TestLiftSequenceRefusesWhatIsNotAPlan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.yaml")
	text := "$schema: https://github.com/janearc/slayer/schemas/" +
		"job-v1.json\nscene: {smoke: 1}\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LiftSequence(path); err == nil {
		t.Error("a job file was lifted as a plan")
	}
	_, err := LiftSequence(filepath.Join(t.TempDir(), "gone.yaml"))
	if err == nil || !strings.Contains(err.Error(), "gone.yaml") {
		t.Errorf("a plan that is not there gave %v", err)
	}
}
