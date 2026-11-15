package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// a pixel outside the picture is refused, and so is a picture Sampled
// would refuse.
func TestAReplayOutsideThePictureIsRefused(t *testing.T) {
	eye := lookingAt(maths.V(0, 2, 6), centre)
	for _, at := range [][2]int{{-1, 0}, {0, -1}, {12, 0}, {0, 9}} {
		if _, _, err := Replay(smoking(), eye, film.Pinhole{}, 0, 12, 9,
			1, at[0], at[1]); err == nil {
			t.Errorf("pixel %v was replayed", at)
		}
	}
	if _, _, err := Replay(smoking(), eye, film.Pinhole{}, 0, 12, 9, 0,
		0, 0); err == nil {
		t.Error("no rays a pixel was replayed")
	}
}

// a mirror's look is told at its depth, after the mirror's own steps
// begin and before the share it brings: a mirror ball over a grey sky.
func TestAReplayTellsAMirrorAtItsDepth(t *testing.T) {
	eye := lookingAt(maths.V(0, 2, 6), centre)
	_, steps, err := Replay(mirrors(true), eye, film.Pinhole{}, 0, 12, 9,
		1, 6, 4)
	if err != nil {
		t.Fatal(err)
	}
	deepest, mirrored := 0, false
	for _, step := range steps {
		t.Logf("ray %d, bounce %d: %s: %s", step.Ray, step.Bounce,
			step.What, step.Said)
		deepest = max(deepest, step.Bounce)
		if step.What == "a mirror, by fresnel's share" {
			mirrored = step.Bounce == 0
		}
	}
	if deepest < 1 || !mirrored {
		t.Errorf("deepest bounce %d, a mirror told at its depth %v",
			deepest, mirrored)
	}
}
