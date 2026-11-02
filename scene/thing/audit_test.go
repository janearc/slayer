package thing

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// excited gas gives the same light every time it is asked: three lines
// that overlap at a band, asked two hundred times, give one answer, and
// a band as near two lines takes the shorter's excitation.
func TestExcitedGasIsTheSameEveryTime(t *testing.T) {
	e := Excited{Lines: map[float64]float64{500: 0.3, 501: 0.7, 502: 0.11,
		503: 0.9, 504: 0.13, 505: 0.5}, Width: 2,
		Excitations: map[float64]float64{500: 2, 502: 3}}
	band := maths.Band{Nanometres: 502.5}
	first := e.Emits(band)
	for range 200 {
		if got := e.Emits(band); math.Float64bits(got) != math.Float64bits(first) {
			t.Fatalf("the gas gave %v and then %v", first, got)
		}
	}
	tie := Excited{Lines: map[float64]float64{500: 1, 502: 1},
		Excitations: map[float64]float64{500: 2, 502: 3}}
	for range 200 {
		if got := tie.Excitation(maths.Band{Nanometres: 501}); got != 2 {
			t.Fatalf("midway between 500 and 502 the excitation is %v, not 500's 2", got)
		}
	}
}

// an orbit longer than a Duration holds, or of no number, is refused by
// name: worked by hand, Sedna at 506 au round the sun is some 11,400
// years, and would have been 292.
func TestAnOrbitPastADurationIsRefused(t *testing.T) {
	const sun, au = 1.98847e30, 1.495978707e11
	for _, r := range []float64{506 * au, math.Inf(1)} {
		func() {
			defer func() {
				got, _ := recover().(string)
				if !strings.Contains(got, "292") {
					t.Errorf("an orbit of %v m: %q", r, got)
				}
			}()
			Period(sun, r)
		}()
	}
}

// a surface mirrors no more than lands on it: a Gloss of two is a Gloss
// of one, face on and grazing.
func TestNoSurfaceMirrorsMoreThanLands(t *testing.T) {
	two, one := &Material{Gloss: 2, Index: 1.5}, &Material{Gloss: 1, Index: 1.5}
	for _, cos := range []float64{1, 0.5, 0} {
		if got, want := two.Mirrors(cos), one.Mirrors(cos); got != want || got > 1 {
			t.Errorf("a gloss of two at %v mirrors %v, want %v", cos, got, want)
		}
	}
}
