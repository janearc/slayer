package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a line is a gaussian, strength * exp(-d^2) with d = (band - line) /
// width, the same either side. worked by hand for a line of strength 2
// at 500 nm, width 2, from exp(-0.25) = 0.77880078307, exp(-1) =
// 0.36787944117, exp(-2.25) = 0.10539922456 and exp(-4) = 0.01831563889:
//
//	501 nm  d 0.5   2 exp(-0.25) = 1.55760156614
//	502 nm  d 1     2 exp(-1)    = 0.73575888234
//	503 nm  d 1.5   2 exp(-2.25) = 0.21079844912
//	504 nm  d 2     2 exp(-4)    = 0.03663127778
func TestALineIsAGaussian(t *testing.T) {
	e := Excited{Lines: map[float64]float64{500: 2}, Width: 2}
	for _, c := range []struct{ nm, want float64 }{
		{500, 2}, {501, 1.55760156614}, {499, 1.55760156614},
		{502, 0.73575888234}, {498, 0.73575888234},
		{503, 0.21079844912}, {504, 0.03663127778}, {496, 0.03663127778},
	} {
		if got := e.Emits(maths.Band{Nanometres: c.nm}); math.Abs(got-c.want) > 1e-10 {
			t.Errorf("at %v nm the line gives %v, want %v", c.nm, got, c.want)
		}
	}
}

// the lines sum: lines of 2 at 500 and 1 at 504, each 2 wide, at 502
// are both a width away, d = 1 and d = -1, so (2 + 1) exp(-1) =
// 1.10363832351.
func TestLinesSum(t *testing.T) {
	e := Excited{Lines: map[float64]float64{500: 2, 504: 1}, Width: 2}
	if got := e.Emits(maths.Band{Nanometres: 502}); math.Abs(got-1.10363832351) > 1e-10 {
		t.Errorf("between two lines the gas gives %v, want 1.10363832351", got)
	}
	// a line at zero counts once, and no line is added that is not there
	zero := Excited{Lines: map[float64]float64{0: 1}, Width: 2}
	if got := zero.Emits(maths.Band{}); got != 1 {
		t.Errorf("a line at nought gives %v at nought, want 1", got)
	}
}

// a width of zero or below is one nanometre. at 0.5 nm off the line, d = 0.5
// and exp(-0.25) = 0.77880078307; at 1 nm, exp(-1). a width of half a nanometre
// is just that, and at 0.5 nm off, d = 1.
func TestAWidthOfNothingIsOne(t *testing.T) {
	for _, w := range []float64{0, -2} {
		e := Excited{Lines: map[float64]float64{500: 1}, Width: w}
		for _, c := range []struct{ nm, want float64 }{
			{500.5, 0.77880078307}, {501, 0.36787944117}, {500, 1},
		} {
			if got := e.Emits(maths.Band{Nanometres: c.nm}); math.Abs(got-c.want) > 1e-10 {
				t.Errorf("a width of %v at %v nm gives %v, want %v", w, c.nm, got, c.want)
			}
		}
	}
	half := Excited{Lines: map[float64]float64{500: 1}, Width: 0.5}
	if got := half.Emits(maths.Band{Nanometres: 500.5}); math.Abs(got-0.36787944117) > 1e-10 {
		t.Errorf("a width of a half at 0.5 nm gives %v, want exp(-1)", got)
	}
}

// excited gas returns no radar at any band, whether on a line or not.
func TestExcitedGasReturnsNoRadar(t *testing.T) {
	e := Excited{Lines: map[float64]float64{500: 1, 1e8: 1}, Width: 2, Passes: 0.5}
	for _, nm := range []float64{0, 500, 1e8, 1e9, 1} {
		if got := e.Scatters(maths.Band{Nanometres: nm}); got != 0 {
			t.Errorf("at %v nm excited gas scatters %v", nm, got)
		}
	}
}

// the excitation is the nearest line's, or one where that line has none or one
// that is not greater than zero. the nearest of lines at 500 and 502 to a band
// at 1 nm is 500, not some line at zero.
func TestExcitationIsTheNearestLinesOrOne(t *testing.T) {
	e := Excited{Lines: map[float64]float64{500: 1, 502: 1},
		Excitations: map[float64]float64{500: 2}}
	if got := e.Excitation(maths.Band{Nanometres: 1}); got != 2 {
		t.Errorf("at 1 nm the excitation is %v, want the nearest line's 2", got)
	}
	for _, c := range []struct{ set, want float64 }{
		{2, 2}, {0.5, 0.5}, {0, 1}, {-2, 1},
	} {
		e.Excitations = map[float64]float64{500: c.set}
		if got := e.Excitation(maths.Band{Nanometres: 500}); got != c.want {
			t.Errorf("a line with an excitation of %v gives %v, want %v", c.set, got, c.want)
		}
	}
	e.Excitations = map[float64]float64{502: 3}
	if got := e.Excitation(maths.Band{Nanometres: 500}); got != 1 {
		t.Errorf("a line with none gives %v, want 1", got)
	}
}

// a band as far from two lines as from each other takes the shorter's
// excitation, even when only the longer has one: 501 is a nanometre from
// 500 and from 502, and 503 from 502 and from 504.
func TestATieTakesTheShorterLine(t *testing.T) {
	e := Excited{Lines: map[float64]float64{500: 1, 502: 1, 504: 1},
		Excitations: map[float64]float64{502: 3, 504: 4}}
	if got := e.Excitation(maths.Band{Nanometres: 501}); got != 1 {
		t.Errorf("at 501 the excitation is %v, want 500's one, not 502's 3", got)
	}
	if got := e.Excitation(maths.Band{Nanometres: 503}); got != 3 {
		t.Errorf("at 503 the excitation is %v, want 502's 3, not 504's 4", got)
	}
}
