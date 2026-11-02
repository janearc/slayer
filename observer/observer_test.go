package observer_test

import (
	"testing"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/radiation"
	theme "github.com/janearc/libtheme-css/primitives/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	theirs "github.com/janearc/libtheme-css/spaces/radiation"
	"github.com/janearc/slayer/observer"
	"pgregory.net/rapid"
)

const (
	nought = 0 // defined: no power
	unity  = 1 // defined: a power of one, and one band along
)

// lines is some lines of the eye's range in nanometres, each with a power from
// zero to one, no two in one band, so neither package sums two powers in an
// order of its own.
func lines(draw *rapid.T) map[float64]float64 {
	lo, hi := swatch.Observed()
	drawn := rapid.MapOf(rapid.Float64Range(float64(lo), float64(hi)),
		rapid.Float64Range(nought, unity)).Draw(draw, "lines")
	held, one := map[float64]float64{}, map[bands.Band]bool{}
	for nm, power := range drawn {
		if !one[bands.Of(nm)] {
			one[bands.Of(nm)], held[nm] = true, power
		}
	}
	return held
}

// a light is the colour libtheme gives the same light, bit for bit.
func TestALightIsTheColourLibthemeGivesIt(t *testing.T) {
	rapid.Check(t, sameColour)
}

func sameColour(draw *rapid.T) {
	spectrum := lines(draw)
	got := observer.Swatch(bands.Lines(spectrum))
	want := theme.Lines(spectrum).Swatch()
	if got != want {
		draw.Fatalf("%v is %v, not %v", spectrum, got, want)
	}
}

// the eye's range runs from the observer's first row to its last.
func TestTheVisibleIsTheObserversTable(t *testing.T) {
	lo, hi := swatch.Observed()
	short, long := observer.Visible().Edges()
	if short != float64(lo) || long != float64(hi) {
		t.Errorf("the visible is %v to %v, not %d to %d", short, long, lo,
			hi)
	}
}

// anti-tests: the visible holds its first row and not its long end, nor
// anything past either.
func TestTheVisibleHoldsNothingPastItsEnds(t *testing.T) {
	lo, hi := swatch.Observed()
	visible := observer.Visible()
	if !visible.Holds(float64(lo)) {
		t.Errorf("the visible does not hold %d", lo)
	}
	for _, nm := range []float64{float64(lo - unity), float64(hi)} {
		if visible.Holds(nm) {
			t.Errorf("the visible holds %v", nm)
		}
	}
}

// shown is celestial's named ranges, the ones slayer lays across the
// visible
var shown = []radiation.Range{radiation.Gamma(), radiation.XRay(),
	radiation.ExtremeUltraviolet(), radiation.Ultraviolet(),
	radiation.Infrared(), radiation.Radio(), radiation.Microwave()}

// theirsOf is libtheme's own profile of a range, on the same ends,
// across libtheme's visible.
func theirsOf(span radiation.Range) theirs.Profile {
	short, long := span.Edges()
	return theirs.Profile{Name: span.String(),
		Source: theirs.Range{Lo: short, Hi: long}, Render: theirs.Visible}
}

// a wavelength of a range stands for the colour libtheme's own profile of the
// same range gives it, bit for bit. the ends are celestial's: its extreme
// ultraviolet ends at 120 nm, where libtheme's ended at 121.
func TestAFalseColourIsLibthemesOwn(t *testing.T) {
	rapid.Check(t, falseAsTheirs)
}

func falseAsTheirs(draw *rapid.T) {
	span := rapid.SampledFrom(shown).Draw(draw, "range")
	short, long := span.Edges()
	nm := rapid.Float64Range(short, long).Filter(span.Holds).Draw(draw, "nm")
	if got, want := observer.FalseColour(span, nm),
		theirsOf(span).Swatch(nm); got != want {
		draw.Fatalf("%v at %v nm is %v, not %v", span, nm, got, want)
	}
}

// anti-tests: a wavelength outside a range, short of it or at its long
// end, stands for no colour, and nor does any wavelength of no range.
func TestAFalseColourOutsideItsRangeIsBlack(t *testing.T) {
	for _, span := range shown {
		short, long := span.Edges()
		for _, nm := range []float64{short / (unity + unity), long} {
			if got := observer.FalseColour(span, nm); got != swatch.Black {
				t.Errorf("%v at %v nm is %v", span, nm, got)
			}
		}
		if got := observer.FalseColour(radiation.Range{}, short); got != swatch.Black {
			t.Errorf("no range at %v nm is %v", short, got)
		}
	}
}

// line is a light of one, at the middle of a band.
func line(band bands.Band) bands.Light {
	short, long := band.Edges()
	return bands.Lines(map[float64]float64{(short + long) / (unity + unity): unity})
}

// anti-tests: no band's light is the colour of the band beside it, so a
// band is not carried across to another. the eye's range has bands at
// its ends the observer sees nothing in; every pair of neighbours that
// both have a colour differs, and there are such pairs.
func TestOneBandIsNotTheNext(t *testing.T) {
	lo, hi := swatch.Observed()
	black, pairs := swatch.Swatch{}, nought
	for band := bands.Of(float64(lo)); band < bands.Of(float64(hi)); band++ {
		one, next := observer.Swatch(line(band)), observer.Swatch(line(band+unity))
		if one == black || next == black {
			continue
		}
		pairs++
		if one == next {
			t.Errorf("bands %d and %d are both %v", band, band+unity, one)
		}
	}
	if pairs == nought {
		t.Error("no two neighbouring bands have a colour")
	}
}
