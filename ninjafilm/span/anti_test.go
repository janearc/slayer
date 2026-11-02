package span_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/span"
	"pgregory.net/rapid"
)

// anti-tests: no span reaches one nanometre past what is seen, at either
// end. The ends themselves are inside it.
func TestNoSpanReachesOneNanometrePastWhatIsSeen(t *testing.T) {
	var zero span.Span
	for _, ends := range [][]int{{shortest - next, shortest},
		{longest, longest + next}, {shortest - next, longest + next}} {
		got, err := span.New(ends[dark], ends[lit])
		if err == nil || got != zero {
			t.Errorf("a span of %v was made", ends)
		}
	}
}

// anti-tests: spans between the lamps hold no lamp. The light held
// between blue and green is dark. So is the light between green and red.
func TestSpansBetweenTheLampsHoldNoLamp(t *testing.T) {
	var none maths.Light
	for _, ends := range [][]int{{blue + next, green - next},
		{green + next, red - next}, {shortest, blue - next},
		{red + next, longest}} {
		if got := spanned(t, ends[dark], ends[lit]).Holds(); got != none {
			t.Errorf("a span of %v holds %v", ends, got)
		}
	}
}

// anti-tests: lamps do not shine a nanometre either side of their hue.
// Spans of the nanometre before or after hold none of them.
func TestALampDoesNotShineBesideItsHue(t *testing.T) {
	var none maths.Light
	for _, hue := range []int{red, green, blue} {
		before := spanned(t, hue-next, hue-next).Holds()
		after := spanned(t, hue+next, hue+next).Holds()
		if before != none || after != none {
			t.Errorf("the nanometres either side of %d hold %v and %v", hue,
				before, after)
		}
	}
}

// anti-tests: a span that stops short of a lamp never holds it, and one
// that starts past a lamp never holds it either.
func TestASpanNeverHoldsALampItDoesNotReach(t *testing.T) {
	rapid.Check(t, spanMissesLampsOutsideIt)
}

func spanMissesLampsOutsideIt(t *rapid.T) {
	from, to := ends(t)
	made, err := span.New(from, to)
	if err != nil {
		t.Skip("a span past what is seen")
	}
	held := made.Holds()
	for _, hue := range []struct {
		wavelength int
		lamp       float64
	}{{red, held.R}, {green, held.G}, {blue, held.B}} {
		outside := hue.wavelength < from || hue.wavelength > to
		if outside && hue.lamp != dark {
			t.Fatalf("%d to %d holds a lamp at %d", from, to, hue.wavelength)
		}
	}
}

// anti-tests: a lamp is not held twice when a span is split. The halves
// hold each lamp at most once, and never a part of one.
func TestSplittingASpanNeverHoldsALampTwice(t *testing.T) {
	rapid.Check(t, splitHoldsEachLampOnce)
}

func splitHoldsEachLampOnce(t *rapid.T) {
	from := rapid.IntRange(shortest, longest-next).Draw(t, "from")
	to := rapid.IntRange(from+next, longest).Draw(t, "to")
	cut := rapid.IntRange(from, to-next).Draw(t, "cut")
	halves := spanned(t, from, cut).Holds().Add(spanned(t, cut+next,
		to).Holds())
	for _, lamp := range []float64{halves.R, halves.G, halves.B} {
		if lamp != dark && lamp != lit {
			t.Fatalf("the halves of %d to %d hold %v of a lamp", from, to,
				lamp)
		}
	}
}

// anti-tests: the outer span can hold more than the inner. All that is
// seen holds all three lamps and one nanometre holds fewer.
func TestTheOuterSpanCanHoldMoreThanTheInnerOne(t *testing.T) {
	all := spanned(t, shortest, longest).Holds()
	one := spanned(t, red, red).Holds()
	if all != maths.RGB(lit, lit, lit) || one == all {
		t.Errorf("all that is seen holds %v and a nanometre holds %v", all,
			one)
	}
}

// anti-tests: no lamp is held in part. Whatever span is drawn, each lamp
// is held whole or not at all.
func TestNoLampIsHeldInPart(t *testing.T) {
	rapid.Check(t, lampsAreHeldWholeOrNot)
}

func lampsAreHeldWholeOrNot(t *rapid.T) {
	from, to := ends(t)
	made, err := span.New(from, to)
	if err != nil {
		t.Skip("a span past what is seen")
	}
	held := made.Holds()
	for _, lamp := range []float64{held.R, held.G, held.B} {
		if lamp != dark && lamp != lit {
			t.Fatalf("%d to %d holds %v of a lamp", from, to, lamp)
		}
	}
}
