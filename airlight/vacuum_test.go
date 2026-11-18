package airlight

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// scatteredPhotons is how many photons the scattering air is walked
// with: a tenth of them scatter, and the bins about a sun 30 degrees up
// need some in each.
const scatteredPhotons = 200_000 // picked: see above

// solvedThin is the fields of the thin air's shell scattering so much a
// metre, walked from its floor with so many photons.
func solvedThin(t *testing.T, scattering float64, photons int) *Fields {
	t.Helper()
	plan, err := NewPlan(thinKnown(t, scattering, photons), none)
	if err != nil {
		t.Fatal(err)
	}
	fields, err := plan.Solve()
	if err != nil {
		t.Fatal(err)
	}
	return fields
}

// lookedAt are ways a sky is looked at: up, toward the sun, away from
// it low, level, and down.
var lookedAt = []maths.Vec{straightUp, wayAt(thirty*degree, none),
	wayAt(sixty*degree, math.Pi), wayAt(math.Pi/two, math.Pi/two),
	straightDown}

// the full sun is the sun's light at the top, all of it, by the
// spectrum's route.
func fullSun(fields *Fields) maths.Light {
	return fields.seen(func(int) float64 { return one }).Picture(handExposure)
}

// a column with no scattering and no absorption over a black ground
// gives a black sky every way, and the whole sun.
func TestAVacuumGivesABlackSkyAndTheFullSun(t *testing.T) {
	fields := solvedThin(t, none, fewPhotons)
	sun := wayAt(thirty*degree, none)
	made := NewSky(fields, sun, handExposure)
	for _, look := range lookedAt {
		if got := made.Light(look); got != (maths.Light{}) {
			t.Errorf("a vacuum's sky along %v is %v", look, got)
		}
	}
	if got, want := Sun(fields, sun, handExposure),
		fullSun(fields); !near(got, want) {
		t.Errorf("a vacuum's sun %v, want %v", got, want)
	}
}

// the anti-test: a thin scattering air in the same column lights the
// sky and dims the sun, so the checks above would see either.
func TestAScatteringAirLightsTheSkyAndDimsTheSun(t *testing.T) {
	fields := solvedThin(t, thinDepth/thinTop, scatteredPhotons)
	sun := wayAt(thirty*degree, none)
	overhead := NewSky(fields, sun, handExposure).Light(straightUp)
	if !(overhead.Luminance() > none) {
		t.Errorf("a scattering air's sky overhead is %v", overhead)
	}
	direct, full := Sun(fields, sun, handExposure), fullSun(fields)
	if !(direct.Luminance() < full.Luminance()) {
		t.Errorf("a scattering air's sun %v is not dimmed from %v", direct,
			full)
	}
}
