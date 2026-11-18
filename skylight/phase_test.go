package skylight

import (
	"math"
	"math/rand/v2"
	"testing"
)

// The draws the phases are checked by.
const (
	draws      = 1000000  // picked: five standard errors near 0.002
	phaseSeed  = 7        // picked: any seed
	twoFifths  = 0.4      // defined: Rayleigh's mean cosine squared
	fourthMean = 9.0 / 35 // defined: Rayleigh's mean of the fourth power,
	// three eighths of the integral of (1 + mu^2) mu^4, two fifths and
	// two sevenths
	third     = 1.0 / 3 // defined: the isotropic mean of the cosine squared
	wrongLean = 0.7     // picked: a twentieth off the lean asked for
)

// leans are asymmetries to draw by: forward, backward and none.
var leans = []float64{0.75, -0.4, 0} // picked: a cloud's, a backscatterer's

// drawn is the mean of what made makes of a walker's draws.
func drawn(made func(*walker) float64) float64 {
	w := &walker{random: rand.NewPCG(phaseSeed, phaseSeed)}
	sum := 0.0
	for range draws {
		sum += made(w)
	}
	return sum / draws
}

// squared is the square of a Rayleigh cosine drawn by the walker.
func squared(w *walker) float64 {
	cosine := rayleighCosine(w.uniform())
	return cosine * cosine
}

// Rayleigh's cosines, drawn by inverting its distribution, have a mean
// square of two fifths; its spread is the mean fourth power less the
// square of that. The ends of the draw are the ends of the turn.
func TestRayleighCosinesSquareToTwoFifths(t *testing.T) {
	spread := fourthMean - twoFifths*twoFifths
	if err := within(drawn(squared), twoFifths, spread, draws); err != nil {
		t.Error(err)
	}
	for u, want := range map[float64]float64{0: -1, 0.5: 0, 1: 1} { // defined:
		// the ends and the middle of the distribution
		if got := rayleighCosine(u); !(math.Abs(got-want) <= basisShare) {
			t.Errorf("draw %g turns by %g, want %g", u, got, want)
		}
	}
}

// the anti-test: cosines drawn alike every way square to a third, which
// the check refuses as Rayleigh's.
func TestIsotropicCosinesAreNotRayleighs(t *testing.T) {
	alike := func(w *walker) float64 {
		cosine := henyeyCosine(none, w.uniform())
		return cosine * cosine
	}
	spread := fourthMean - twoFifths*twoFifths
	if within(drawn(alike), twoFifths, spread, draws) == nil {
		t.Error("isotropic cosines pass as Rayleigh's")
	}
}

// turnedBy is the mean cosine between an axis and the ways turned from
// it by Henyey-Greenstein draws of a lean, through the walker's basis.
func turnedBy(lean float64) float64 {
	axis := axes[len(axes)-one]
	return drawn(func(w *walker) float64 {
		cosine := henyeyCosine(lean, w.uniform())
		return axis.dot(turned(axis, cosine, w.uniform()))
	})
}

// the Henyey-Greenstein draws, turned about an axis, have a mean cosine
// of their lean, forward, backward and at nought, where they are
// isotropic. Its second moment is a third of one and twice the lean
// squared, so the cosine's spread is one less the lean squared, over
// three.
func TestHenyeyMeanCosineIsItsLean(t *testing.T) {
	for _, lean := range leans {
		spread := (one - lean*lean) * third
		if err := within(turnedBy(lean), lean, spread, draws); err != nil {
			t.Errorf("lean %g: %v", lean, err)
		}
	}
}

// the anti-test: draws of a lean of 0.7 are refused as 0.75's, so the
// check would catch a lean taken wrongly by a twentieth.
func TestAWrongLeanIsCaught(t *testing.T) {
	asked := leans[none]
	spread := (one - asked*asked) * third
	if within(turnedBy(wrongLean), asked, spread, draws) == nil {
		t.Errorf("a lean of %g passes as %g", wrongLean, asked)
	}
}
