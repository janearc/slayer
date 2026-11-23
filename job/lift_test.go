package job

import (
	"path/filepath"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// planned is a sequence plan's values as the session would fill them: a
// smoke, its film, a stretched span and a print.
const planned = "smoke: 3\nwidth: 64\nheight: 48\nrays: 2\nfps: 6\n" +
	"start: 45\nduration: 30\nsceneduration: 3600\n" +
	"knee: 0.7\ntop: 8\nbloom: 0.5\n"

// plan is the planned values with a camera, in a directory of its own.
func plan(t *testing.T) Plan {
	t.Helper()
	directory := t.TempDir()
	var values Plan
	if err := yaml.Unmarshal([]byte(planned), &values); err != nil {
		t.Fatal(err)
	}
	values.File = filepath.Join(directory, "plan.yaml")
	values.Camera = []byte(pathCamera[len("camera:\n"):])
	values.Output = filepath.Join(directory, "frames")
	return values
}

// TestALiftedPlanHasTheSameFrames holds a lifted plan to its scene,
// film, clock, print, camera and output.
func TestALiftedPlanHasTheSameFrames(t *testing.T) {
	job, err := Lift(plan(t))
	if err != nil {
		t.Fatal(err)
	}
	smoke, _ := job.Scene().Smoke()
	width, _ := job.Film().Width()
	rate, _ := job.Clock().Rate()
	curve, _ := job.Print()
	bloom, _ := curve.Bloom()
	lens, _ := job.Camera()
	path, _ := lens.Path()
	frames, _ := job.Outputs().Frames()
	times := job.Times()
	got := spoken(smoke, width, rate, len(times),
		times[len(times)/halves], curve.Knee(), curve.Top(), bloom,
		path.Duration(), filepath.Base(frames), job.File() == "")
	if want := "3 64 6 180 30m45s 0.7 8 0.5 2s frames true"; got != want {
		t.Errorf("the lifted plan is %s, want %s", got, want)
	}
}

// TestALiftedSetPlanHasNoFilmOrStretch is the anti-test: a plan that
// says no film and no scene duration lifts to a job that says neither.
// anti-test of TestALiftedPlanHasTheSameFrames
func TestALiftedSetPlanHasNoFilmOrStretch(t *testing.T) {
	values := plan(t)
	values.Smoke, values.Width, values.Height, values.Rays = nothing,
		nothing, nothing, nothing
	values.SceneDuration, values.Camera = nothing, nil
	values.Scene = "relative.yaml"
	job, err := Lift(values)
	if err != nil {
		t.Fatal(err)
	}
	_, stretched := job.Clock().SceneDuration()
	_, hasCamera := job.Camera()
	set, _ := job.Scene().Set()
	if filmSaid(job.Film()) || stretched || hasCamera ||
		!filepath.IsAbs(set) {
		t.Error("a set plan lifted to a film, a stretch, a camera or a " +
			"relative scene")
	}
}

// TestALiftedPlanIsCheckedAsAJob is the anti-test: a plan's values a job
// cannot take are refused, naming the plan.
// anti-test of TestALiftedPlanHasTheSameFrames
func TestALiftedPlanIsCheckedAsAJob(t *testing.T) {
	values := plan(t)
	values.Camera = []byte("unit: m\nlook-at: {x: 0, y: 0, z: 0}\n" +
		"moves: []\nspeed: 2\n")
	_, err := Lift(values)
	if err == nil {
		t.Fatal("a camera with a strange key was lifted")
	}
	saying(t, err, "plan.yaml, lifted to a job", "speed")
	values = plan(t)
	values.Scene = "both.yaml"
	_, err = Lift(values)
	if err == nil {
		t.Fatal("a plan with a scene and a smoke was lifted")
	}
	saying(t, err, "lifted to a job", "set and smoke")
}
