package session

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// plannedAt is a box of a given side, in metres, with its centre on the
// axis the camera of lookingNorth looks along, so far from it in metres
// (negative is behind it).
func plannedAt(side, ahead float64) plannedThing {
	half := side / 2
	return plannedThing{line: planLine{name: "box", count: 1},
		box: maths.Box{Min: maths.V(-half, -half, -ahead-half),
			Max: maths.V(half, half, -ahead+half)}}
}

// TestAThingIsBuiltAveragedOrDroppedByItsPixels checks the plan's three
// verdicts and that each thing is counted under exactly one of them: one
// that is out of view is dropped, one under a pixel is averaged, and one
// of more is built.
func TestAThingIsBuiltAveragedOrDroppedByItsPixels(t *testing.T) {
	const (
		near    = 10.0
		far     = 100000.0
		metre   = 1.0
		centims = 0.01
	)
	eye := lookingNorth()
	for _, each := range []struct {
		about                    string
		thing                    plannedThing
		verdict                  string
		built, averaged, dropped int
	}{
		{"a metre at ten metres", plannedAt(metre, near), verdictBuild,
			1, 0, 0},
		{"a hair of a thing far off", plannedAt(centims, far),
			verdictAverage, 0, 1, 0},
		{"a metre behind the camera", plannedAt(metre, -near), verdictDrop,
			0, 0, 1},
	} {
		got := each.thing.judged(eye)
		if got.verdict != each.verdict || got.built != each.built ||
			got.averaged != each.averaged || got.dropped != each.dropped {
			t.Errorf("%s: %q built %d, averaged %d, dropped %d; want %q "+
				"%d, %d, %d", each.about, got.verdict, got.built,
				got.averaged, got.dropped, each.verdict, each.built,
				each.averaged, each.dropped)
		}
	}
}

// TestAThingItsKindCannotSizeIsBuilt is the anti-test of the verdict: a
// thing whose kind said it cannot say how big it is keeps that verdict,
// built, even where it stands behind the camera.
func TestAThingItsKindCannotSizeIsBuilt(t *testing.T) {
	unsized := plannedAt(1, -10)
	unsized.line.verdict = verdictBuild
	got := unsized.judged(lookingNorth())
	if got.verdict != verdictBuild || got.built != 1 || got.dropped != 0 {
		t.Errorf("an unsized thing: %q built %d, dropped %d", got.verdict,
			got.built, got.dropped)
	}
}
