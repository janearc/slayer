package planet

import (
	"math"
	"testing"
)

// near is whether two finite numbers agree to a share of the larger.
func near(got, want, share float64) bool {
	if math.IsInf(got, 0) || math.IsNaN(got) {
		return false
	}
	return math.Abs(got-want) <= share*math.Max(math.Abs(got),
		math.Abs(want))
}

// with no air, a white deck sends back its light as a cosine-law
// surface does: a cosine of the star's height over pi, whichever way
// it is seen from.
func TestABareDeckIsACosineReflector(t *testing.T) {
	for _, look := range []struct{ star, eye, turn float64 }{
		{1, 1, -1}, {0.5, 0.9, -0.2}, {0.2, 0.3, 0.7},
	} {
		got := reflected(nil, 1, look.star, look.eye, look.turn)
		want := look.star / math.Pi
		if !near(got, want, 1e-12) {
			t.Errorf("star %g eye %g: %g, want %g", look.star, look.eye,
				got, want)
		}
	}
}

// a layer that only absorbs adds no light of its own and dims the deck
// below by its depth along the way down and the way back.
func TestAnAbsorbingLayerOnlyDims(t *testing.T) {
	layer := optics{absorb: 0.3}
	star, eye := 0.6, 0.8
	got := reflected([]optics{layer}, 0.5, star, eye, -0.5)
	want := 0.5 * star / math.Pi * math.Exp(-0.3*(1/star+1/eye))
	if !near(got, want, 1e-12) {
		t.Errorf("%g, want %g", got, want)
	}
}

// a layer of nothing changes nothing.
func TestAnEmptyLayerChangesNothing(t *testing.T) {
	alone := reflected(nil, 0.4, 0.7, 0.5, 0.1)
	under := reflected([]optics{{}}, 0.4, 0.7, 0.5, 0.1)
	if alone != under {
		t.Errorf("%g alone, %g under an empty layer", alone, under)
	}
}

// a thin layer of gas over black sends back its single scattering,
// which for a depth tau seen from straight above and lit from straight
// above is tau times the phase over four pi, over the cosine.
func TestAThinGasLayerScattersItsDepth(t *testing.T) {
	tau := 1e-6
	layer := optics{scatter: tau}
	got := reflected([]optics{layer}, 0, 1, 1, -1)
	want := tau * rayleigh(-1) / (4 * math.Pi)
	if !near(got, want, 1e-5) {
		t.Errorf("%g, want %g", got, want)
	}
}

// what the column sends back for each watt falling on a square metre of
// it is the same with the star and the eye swapped: single scattering
// is reciprocal. that is the radiance over the star's cosine, since the
// light falling on the ground is the star's times that cosine.
func TestLightIsReciprocal(t *testing.T) {
	stack := []optics{{scatter: 0.2, absorb: 0.05},
		{scatter: 0.1, haze: 0.8, hazeAlbedo: 0.9, hazeLean: 0.6}}
	one := reflected(stack, 0.7, 0.3, 0.8, -0.4) / 0.3
	other := reflected(stack, 0.7, 0.8, 0.3, -0.4) / 0.8
	if !near(one, other, 1e-12) {
		t.Errorf("%g one way, %g the other", one, other)
	}
}

// both phases are whole: over every way out they sum to four pi, so a
// scatterer that absorbs nothing loses nothing.
func TestThePhasesAreWhole(t *testing.T) {
	for name, phase := range map[string]func(float64) float64{
		"rayleigh": rayleigh,
		"haze":     func(turn float64) float64 { return henyey(0.7, turn) },
	} {
		const steps = 200000
		sum := 0.0
		for i := 0; i < steps; i++ {
			turn := -1 + (float64(i)+0.5)*2/steps
			sum += phase(turn) * 2 / steps
		}
		if got := 2 * math.Pi * sum; !near(got, 4*math.Pi, 1e-6) {
			t.Errorf("%s sums to %g, want 4 pi", name, got)
		}
	}
}

// light sent back over the whole sky never exceeds the light that came
// in, even for deep air that absorbs nothing over a white deck: single
// scattering only ever keeps less.
func TestNoMoreLightLeavesThanArrives(t *testing.T) {
	stack := []optics{{scatter: 5}, {haze: 5, hazeAlbedo: 1,
		hazeLean: 0.3}}
	for _, star := range []float64{1, 0.5, 0.1} {
		left := 0.0
		const rings, rounds = 400, 400
		for i := 0; i < rings; i++ {
			eye := (float64(i) + 0.5) / rings
			for k := 0; k < rounds; k++ {
				round := 2 * math.Pi * (float64(k) + 0.5) / rounds
				turn := scatterTurn(star, eye, round)
				left += reflected(stack, 1, star, eye, turn) * eye *
					(1.0 / rings) * (2 * math.Pi / rounds)
			}
		}
		if left > star*(1+1e-3) {
			t.Errorf("star %g: %g leaves of %g", star, left, star)
		}
	}
}

// a thin layer of gas and haze together sends back each one's
// scattering by its own phase: gas by rayleigh's, the haze by its
// lean, the haze only for the share it scatters.
func TestGasAndHazeScatterEachByItsOwn(t *testing.T) {
	gas, haze, albedo, lean := 1e-6, 3e-6, 0.5, 0.6
	layer := optics{scatter: gas, haze: haze, hazeAlbedo: albedo,
		hazeLean: lean}
	for _, turn := range []float64{-1, 0.5} {
		got := reflected([]optics{layer}, 0, 1, 1, turn)
		want := (gas*rayleigh(turn) + haze*albedo*henyey(lean, turn)) /
			(4 * math.Pi)
		if !near(got, want, 1e-5) {
			t.Errorf("turned %g: %g, want %g", turn, got, want)
		}
	}
}

// a haze leaning forward throws light on: straight on it is one and
// the lean over one less the lean, squared.
func TestHazeLeansForward(t *testing.T) {
	lean := 0.6
	got, want := henyey(lean, 1), (1+lean)/((1-lean)*(1-lean))
	if !near(got, want, 1e-12) {
		t.Errorf("straight on %g, want %g", got, want)
	}
	if !(henyey(lean, 1) > henyey(lean, -1)) {
		t.Error("the haze throws more light back than on")
	}
}

// a layer with nothing that scatters has no phase.
func TestNothingScatteringHasNoPhase(t *testing.T) {
	if got := (optics{absorb: 1}).phase(0.3); got != 0 {
		t.Errorf("phase %g", got)
	}
}

// gas under an absorbing layer is dimmed by that layer's depth on the
// way in and on the way out.
func TestGasUnderALayerIsDimmedBothWays(t *testing.T) {
	star, eye, turn := 0.5, 0.8, -0.3
	thin := optics{scatter: 1e-6}
	alone := reflected([]optics{thin}, 0, star, eye, turn)
	under := reflected([]optics{{absorb: 0.4}, thin}, 0, star, eye, turn)
	if want := alone * math.Exp(-0.4*(1/star+1/eye)); !near(under, want,
		1e-9) {
		t.Errorf("%g under, want %g", under, want)
	}
}

// the light is turned by the angle between the way from the star and
// the way to the eye: straight back when the eye is where the star is,
// and by the eye's height alone under a star overhead.
func TestTheTurnIsBetweenStarAndEye(t *testing.T) {
	for _, look := range []struct{ star, eye, round, want float64 }{
		{0.6, 0.6, 0, -1},
		{1, 0.3, 2, -0.3},
		{0.6, 0.8, math.Pi / 3, -0.72},
		{0.6, 0.8, math.Pi, 0}, // 0.48 less 0.8 times 0.6: square on
	} {
		got := scatterTurn(look.star, look.eye, look.round)
		if math.Abs(got-look.want) > 1e-12 {
			t.Errorf("%+v: %g", look, got)
		}
	}
}

// gas and haze scattering equally throw light by the mean of their two
// phases.
func TestEqualScatterersShareThePhase(t *testing.T) {
	layer := optics{scatter: 1, haze: 2, hazeAlbedo: 0.5, hazeLean: 0.4}
	got := layer.phase(0.3)
	want := (rayleigh(0.3) + henyey(0.4, 0.3)) / 2
	if !near(got, want, 1e-12) {
		t.Errorf("%g, want %g", got, want)
	}
}
