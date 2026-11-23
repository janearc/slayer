package medium

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// auroraTested is the aurora the test below holds, read from the chosen
// numbers.
func auroraTested(t *testing.T) Aurora {
	t.Helper()
	return Aurora{Radius: picked(t, "aurora-radius"),
		Low:   picked(t, "aurora-low"),
		High:  picked(t, "aurora-high"),
		Lat:   picked(t, "aurora-latitude-degrees") * maths.Degree,
		Width: picked(t, "aurora-width-degrees") * maths.Degree,
		Thick: unity,
		Seed:  uint64(pickedCount(t, "aurora-seed"))}
}

// auroraAt is the point at a latitude and longitude, up above the radius of
// the shell's base.
func auroraAt(radius, latitude, longitude, up float64) maths.Vec {
	return maths.V(math.Cos(latitude)*math.Cos(longitude), math.Sin(latitude),
		math.Cos(latitude)*math.Sin(longitude)).Scale(radius + up)
}

// An aurora is a glow on an oval round each magnetic pole, in its
// shell, brightest at the bottom, and nothing at the equator or beyond
// the shell; curtains vary it along the oval.
func TestAuroraIsAnOvalRoundEachPole(t *testing.T) {
	aurora := auroraTested(t)
	radius := aurora.Radius
	pole := picked(t, "aurora-latitude-degrees") * maths.Degree
	longitude := picked(t, "aurora-longitude")
	bottom := picked(t, "aurora-bottom-up")
	northPoint := auroraAt(radius, pole, longitude, bottom)
	southPoint := auroraAt(radius, -picked(t, "aurora-latitude-degrees")*
		maths.Degree, longitude, bottom)
	north, south := aurora.Density(northPoint), aurora.Density(southPoint)
	if north < picked(t, "aurora-north-least") ||
		!(math.Abs(north-south) <= picked(t, "aurora-tolerance")) {
		t.Errorf("the ovals are %v north and %v south", north, south)
	}
	equatorUp := picked(t, "aurora-equator-up")
	equator := auroraAt(radius, nought, longitude, equatorUp)
	if density := aurora.Density(equator); density != nought {
		t.Errorf("the equator glows %v", density)
	}
	under := auroraAt(radius, pole, longitude, picked(t, "aurora-under-up"))
	if density := aurora.Density(under); density != nought {
		t.Errorf("below the shell glows %v", density)
	}
	topPoint := auroraAt(radius, pole, longitude, picked(t, "aurora-top-up"))
	low, high := aurora.Density(northPoint), aurora.Density(topPoint)
	if !(high < picked(t, "aurora-top-share")*low) {
		t.Errorf("the bottom is %v and the top %v", low, high)
	}
	auroraCurtainsVary(t, aurora, pole)
}

// auroraCurtainsVary checks that with curtains the density along the oval
// varies by more than a factor of two.
func auroraCurtainsVary(t *testing.T, aurora Aurora, pole float64) {
	t.Helper()
	aurora.Curtains = picked(t, "aurora-curtains")
	var low, high float64 = picked(t, "aurora-low-start"), nought
	turn, step := picked(t, "aurora-turn"), picked(t, "aurora-step")
	up := picked(t, "aurora-bottom-up")
	for longitude := float64(nought); longitude < turn; longitude += step {
		density := aurora.Density(auroraAt(aurora.Radius, pole, longitude, up))
		low, high = math.Min(low, density), math.Max(high, density)
	}
	if !(high > double*low) {
		t.Errorf("curtains vary the oval only from %v to %v", low, high)
	}
}
