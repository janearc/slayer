package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/smoke"
)

func TestRunRefusesAShotWithNowhereToGo(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "bent"}, &out, &problems) != failed ||
		!strings.Contains(problems.String(), "-out") {
		t.Fatalf("a shot with no -out was taken: %q", problems.String())
	}
}

func TestRunRefusesAShotItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "sideways", "-out", t.TempDir() + "/x.png"},
		&out, &problems) != failed || !strings.Contains(
		problems.String(), "sideways") {
		t.Fatalf("an unknown shot was taken: %q", problems.String())
	}
}

func TestRunRefusesAFlagItDoesNotKnow(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-colour", "red"}, &out, &problems) != failed {
		t.Fatal("an unknown flag was taken")
	}
}

func TestMeasureSaysWhereTheBandsLie(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "measure", "-out", "unused"}, &out,
		&problems) != ok {
		t.Fatal(problems.String())
	}
	for _, said := range []string{"edge", "order 0", "order 2",
		"close-in"} {
		if !strings.Contains(out.String(), said) {
			t.Errorf("measure does not say %q: %s", said, out.String())
		}
	}
}

func TestDiagramIsWritten(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "diagram", "-width", "120", "-out",
		t.TempDir() + "/d.png"}, &out, &problems) != ok {
		t.Fatal(problems.String())
	}
}

// TestStraightDropsTheHolesField checks that the straight shot has no
// field and the bent one has, and that reach is carried into the field.
func TestStraightDropsTheHolesField(t *testing.T) {
	ring, found := smoke.Get(photonRing)
	if !found {
		t.Fatal("smoke 36 is gone")
	}
	given := asked{width: 16, height: 9, rays: 1, reach: 33}
	bentShot := bent(ring, given)
	if len(bentShot.world.Fields) == 0 {
		t.Fatal("the bent shot has no field")
	}
	for _, field := range bentShot.world.Fields {
		if field.Reach != 33 || field.LetGo != 33 {
			t.Errorf("reach %v and let-go %v, want 33", field.Reach, field.LetGo)
		}
	}
	if got := straight(ring, given).world.Fields; len(got) != 0 {
		t.Errorf("the straight shot has %d fields", len(got))
	}
}

// TestAimedTurnsTheCameraUp checks that a larger aim gives a different
// orientation from no aim.
func TestAimedTurnsTheCameraUp(t *testing.T) {
	ring, _ := smoke.Get(photonRing)
	level := aimed(ring, asked{width: 16, height: 9, rays: 1})
	up := aimed(ring, asked{width: 16, height: 9, rays: 1, aim: 0.2})
	if level.eye.Orientation == up.eye.Orientation {
		t.Error("an aim of 0.2 radians left the camera where it was")
	}
}

// TestMeasureAndDiagramRefuseASizeOfNothing checks the report and the
// diagram refuse a width of nothing, and the diagram an unwritable path.
func TestMeasureAndDiagramRefuseASizeOfNothing(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "measure", "-out", "x", "-width", "0"}, &out, &problems) != failed {
		t.Error("measure accepted a width of nothing")
	}
	problems.Reset()
	if run([]string{"-shot", "diagram", "-out", "x", "-width", "0"}, &out, &problems) != failed {
		t.Error("diagram accepted a width of nothing")
	}
	problems.Reset()
	missing := filepath.Join(t.TempDir(), "no", "such", "d.png")
	if run([]string{"-shot", "diagram", "-out", missing, "-width", "64"}, &out, &problems) != failed ||
		problems.Len() == 0 {
		t.Error("diagram wrote into a directory that is not there")
	}
}

// TestMeasureHonoursAnAskedWidthOfLens checks the degrees given to -wide
// change what measure says.
func TestMeasureHonoursAnAskedWidthOfLens(t *testing.T) {
	say := func(wide string) string {
		var out, problems bytes.Buffer
		if run([]string{"-shot", "measure", "-out", "x", "-wide", wide}, &out, &problems) != ok {
			t.Fatal(problems.String())
		}
		return out.String()
	}
	if say("2") == say("4") {
		t.Error("measure said the same at two widths of lens")
	}
}

// TestMeasureRefusesALensNarrowerThanTheCloseIn checks that a lens
// narrower than the third image's close-in is refused with a message.
func TestMeasureRefusesALensNarrowerThanTheCloseIn(t *testing.T) {
	var out, problems bytes.Buffer
	if run([]string{"-shot", "measure", "-out", "x", "-wide", "0.0001"}, &out, &problems) != failed ||
		!strings.Contains(problems.String(), "no narrower") {
		t.Errorf("measure said %q", problems.String())
	}
}
