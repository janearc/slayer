package skylight

import (
	"errors"
	"math"
	"testing"
)

// The numbers of the band's tests.
const (
	bandPhotons = 40000 // picked: a few hundred crossings a coarse bin
	litBin      = 1     // picked: the second of the coarse zenith bins
)

// litSuns is the span of the coarse grid's litBin: 45 to 90 degrees.
func litSuns() Suns {
	width := math.Pi / coarseBins
	return Suns{From: litBin * width, To: (litBin + one) * width}
}

// zenithSum is a field's radiance and counts summed over a zenith bin
// at its first kept height.
func zenithSum(field *Field, zenith int) (float64, int64) {
	light, count := 0.0, int64(none)
	first := field.index(none, zenith, none, none)
	for bin := first; bin < first+field.views*field.azimuths; bin++ {
		light += field.radiance[bin]
		count += field.counts[bin]
	}
	return light, count
}

// walkedFor is the mixed column walked on the ground grid for a span.
func walkedFor(t *testing.T, suns Suns) *Field {
	t.Helper()
	field, _, err := SolveFor(mixed(), groundGrid(), suns, bandPhotons,
		seedOf, workersOf)
	if err != nil {
		t.Fatal(err)
	}
	return field
}

// every sun's band is the whole disc the top sphere shows the sun.
func TestEverySunsBandIsTheDisc(t *testing.T) {
	made := layered(mixed(), groundGrid().Heights)
	on := bandOf(&made, everySun)
	if on.inner != none || on.outer != made.top {
		t.Errorf("every sun's band runs %g to %g, not 0 to %g", on.inner,
			on.outer, made.top)
	}
}

// a walk for a span finds, in the span's bins, the light a walk of the
// whole planet finds, within their noise, from more crossings.
func TestABandWalkFindsTheWholeWalksLight(t *testing.T) {
	band := walkedFor(t, litSuns())
	whole := walkedFor(t, everySun)
	for bin := range band.views * band.azimuths {
		at := band.index(none, litBin, none, none) + bin
		ours, theirs := band.radiance[at], whole.radiance[at]
		if band.counts[at] == none || whole.counts[at] == none {
			continue
		}
		noise := math.Sqrt(ours*ours/float64(band.counts[at]) +
			theirs*theirs/float64(whole.counts[at]))
		if math.Abs(ours-theirs) > sigmas*noise {
			t.Errorf("bin %d: the band's %g, the whole's %g, noise %g",
				bin, ours, theirs, noise)
		}
	}
	_, ours := zenithSum(band, litBin)
	_, theirs := zenithSum(whole, litBin)
	if !(ours > theirs) {
		t.Errorf("the band's walk crossed %d times, the whole's %d", ours,
			theirs)
	}
}

// the anti-test: a bin outside the span holds less light from the band's
// walk than from the whole's, so the check would see a walk that never
// narrowed.
func TestABandWalkLeavesOtherSunsShort(t *testing.T) {
	lit := litSuns()
	ours, _ := zenithSum(walkedFor(t, lit), none)
	theirs, _ := zenithSum(walkedFor(t, everySun), none)
	if !(ours < theirs) {
		t.Errorf("outside the span the band's walk found %g, the whole's %g",
			ours, theirs)
	}
}

// a span that does not run up from nought to pi is refused by name.
func TestASpanOfSunsOutOfOrderIsRefused(t *testing.T) {
	for _, suns := range []Suns{{From: one, To: none},
		{From: -one, To: one}, {From: none, To: two * math.Pi},
		{From: math.NaN(), To: one}} {
		if err := checkSuns(suns); !errors.Is(err, errSuns) {
			t.Errorf("%+v: %v", suns, err)
		}
		_, _, err := SolveFor(mixed(), groundGrid(), suns, bandPhotons,
			seedOf, workersOf)
		if !errors.Is(err, errSuns) {
			t.Errorf("SolveFor walked a span of %+v: %v", suns, err)
		}
	}
}

// the anti-test: every sun, and one sun alone, are spans a walk can be
// for.
func TestEverySunAndOneAreSpans(t *testing.T) {
	for _, suns := range []Suns{everySun, {From: one, To: one}} {
		if err := checkSuns(suns); err != nil {
			t.Errorf("%+v: %v", suns, err)
		}
	}
}
