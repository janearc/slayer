package main

import (
	"errors"
	"io"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// -from puts the eye straight over the hillside's middle at that
// height, looking straight down at it.
func TestFromAHeightTheEyeLooksStraightDown(t *testing.T) {
	given := mustParse(t, "-from", "100000")
	eye := eyeFor(given, 82.5)
	if eye.Origin.Sub(maths.V(0, 100000, 0)).Len() > 1e-9 {
		t.Errorf("the eye is at %v, not 100 km over the middle",
			eye.Origin)
	}
	if !eye.Forward().Near(maths.V(0, -1, 0), 1e-9) {
		t.Errorf("from 100 km the eye looks %v, not down", eye.Forward())
	}
}

// -wide is the lens's width in degrees; 55 when not given.
func TestTheLensIsAsWideAsAsked(t *testing.T) {
	if got := mustParse(t).wide; math.Abs(got-lensWide) > 1e-12 {
		t.Errorf("the lens is %g wide when not asked", got)
	}
	if got := mustParse(t, "-wide", "0.06").wide; got != 0.06 {
		t.Errorf("-wide 0.06 gives %g", got)
	}
}

// a height of zero or less, and a lens of zero or of half a turn or more, are
// refused at the flags.
func TestNoHeightAndNoLensAreRefused(t *testing.T) {
	for _, wrong := range [][]string{{"-from", "-1"}, {"-wide", "0"},
		{"-wide", "180"}} {
		if _, err := parse(wrong, io.Discard); !errors.Is(err, errAsked) {
			t.Errorf("%v is taken: %v", wrong, err)
		}
	}
}
