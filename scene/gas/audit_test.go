package gas

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
)

// through zero metres all comes, an opaque band too; a gas that does not take a
// band, or of no density, passes all of it, a sun's endless path too, not zero
// times infinity.
func TestNoDistanceAndNoExtinctionPassAll(t *testing.T) {
	air := Air()
	if got := air.Transmit(maths.Band{Nanometres: 1e20}, 0); got != 1 {
		t.Errorf("an undeclared band through nought metres is %v", got)
	}
	haze := Uniform(Haze(), 1)
	if got := haze(maths.Vec{}, maths.Up, math.Inf(1), maths.Band{Nanometres: 1e9}); got != 1 {
		t.Errorf("haze's radio to the sun is %v", got)
	}
	none := Uniform(Air(), 0)
	if got := none(maths.Vec{}, maths.Up, math.Inf(1), maths.Band{Nanometres: 550}); got != 1 {
		t.Errorf("air of no density to the sun is %v", got)
	}
	got := Single(Haze(), 1, maths.Band{Nanometres: 1e9},
		[]film.Emitter{film.Sun{Toward: maths.Up, Strength: maths.Grey(1)}},
		maths.Vec{}, maths.V(0, 0, 1), haze)
	if math.IsNaN(got.R) {
		t.Errorf("haze lit by the sun in radio is %v", got)
	}
}

// an asymmetry of one or more either way, or of no number, is no lobe and
// is refused by name; one inside is henyey and greenstein's, worked at
// g a half face on: (1 - 1/4) / (4 pi (1/4)^1.5), 3/(2 pi).
func TestALobeIsInsideOne(t *testing.T) {
	if got, want := (Gas{Phase: 0.5}).PhaseAt(1), 3/(2*math.Pi); math.Abs(got-want) > 1e-12 {
		t.Errorf("a lobe of a half, forward, is %v, want %v", got, want)
	}
	for _, g := range []float64{1, -1, 1.5, math.NaN()} {
		func() {
			defer func() {
				got, _ := recover().(string)
				if !strings.Contains(got, "no lobe") {
					t.Errorf("an asymmetry of %v: %q", g, got)
				}
			}()
			(Gas{Name: "test", Phase: g}).PhaseAt(0.3)
		}()
	}
}
