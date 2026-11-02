package gas

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// rayleigh's lobe is three over sixteen pi times one plus the cosine
// squared, worked at three angles: square on, 3/(16 pi); along the line
// either way, twice that; and at a cosine of a half, 1.25 times the first.
// the sum to one over the sphere cannot tell this lobe from an even one.
func TestRayleighsLobeIsWorkedByHand(t *testing.T) {
	side := 3 / (16 * math.Pi)
	for _, g := range []Gas{Air(), {}, {Name: "bare"}} {
		wantClose(t, "square on", g.PhaseAt(0), side)
		wantClose(t, "along, forward", g.PhaseAt(1), 2*side)
		wantClose(t, "along, back", g.PhaseAt(-1), 2*side)
		wantClose(t, "at a half", g.PhaseAt(0.5), 1.25*side)
	}
}

// henyey and greenstein's lobe at an asymmetry of a half, worked: head on
// forward it is (1+g)/(4 pi (1-g)^2), 1.5/pi; straight back it is
// (1-g)/(4 pi (1+g)^2), 1/(18 pi); square on, 0.75/(4 pi 1.25^1.5). a
// negative asymmetry is the same lobe turned about.
func TestHenyeyAndGreensteinsLobeIsWorkedByHand(t *testing.T) {
	fwd := Gas{Name: "half", Phase: 0.5}
	wantClose(t, "forward", fwd.PhaseAt(1), 1.5/math.Pi)
	wantClose(t, "back", fwd.PhaseAt(-1), 1/(18*math.Pi))
	wantClose(t, "square on", fwd.PhaseAt(0), 0.75/(4*math.Pi*math.Pow(1.25, 1.5)))
	back := Gas{Name: "minus half", Phase: -0.5}
	wantClose(t, "turned about, forward", back.PhaseAt(1), 1/(18*math.Pi))
	wantClose(t, "turned about, back", back.PhaseAt(-1), 1.5/math.Pi)
}

// inscatter is the lamp's light times the gas's scattering per metre times
// the phase at the angle. air at 450 nm scatters 1.16e-5 times (550/450)^4
// a metre, that being (11/9)^4, 14641/6561; its phase at a cosine of a half
// is 1.25 times 3/(16 pi). the absorption has no part in it.
func TestInscatterIsTheProductOfThreeThings(t *testing.T) {
	scatter := 1.16e-5 * 14641.0 / 6561
	phase := 1.25 * 3 / (16 * math.Pi)
	got := Air().Inscatter(maths.Band{Nanometres: 450}, maths.RGB(2, 3, 5), 0.5)
	wantLight(t, "air at 450", got, maths.RGB(2*scatter*phase, 3*scatter*phase,
		5*scatter*phase))
}

// a gas that scatters and absorbs different amounts, here haze's table
// figures (1e-4 and 1e-5 a metre, held so nothing moves, not measured
// values) and its asymmetry of 0.7, forward at a cosine of one: the
// scattering only, times 1.7/(4 pi 0.09).
func TestInscatterTakesTheScatteringAndNotTheAbsorption(t *testing.T) {
	forward := 1.7 / (4 * math.Pi * 0.09)
	got := Haze().Inscatter(maths.Band{Nanometres: 550}, maths.Grey(1), 1)
	wantLight(t, "haze forward", got, maths.Grey(1e-4*forward))
}
