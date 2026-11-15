// Package cloudsea_test holds the corrected cloud sea to what its README
// and its comments say of it.
package cloudsea_test

import (
	"math"
	"os"
	"testing"

	"github.com/janearc/slayer/set"
)

// The scene's own numbers, as its comments say them.
const (
	// beamRadius is half ESO's 30 cm beam diameter (eso-4lgsf), metres.
	beamRadius = 0.15
	// sodiumLine is the guide stars' wavelength, nanometres.
	sodiumLine = 589
	// slitAltitude is the elt recipe's authored slit altitude, degrees.
	slitAltitude = 55
	// mostApart is GALACSI's 64 arc seconds, in degrees (eso-galacsi-layout).
	mostApart = 64.0 / 3600
	// apartSlack is how far past it rounding may take a beam. Picked.
	apartSlack = 1e-6
)

// corrected reads the scene.
func corrected(t *testing.T) set.Set {
	t.Helper()
	source, err := os.ReadFile("cloudsea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	read, err := set.Read("cloudsea.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	return read
}

// TestTheGuideStarsShareOneAim: four beams, each from a launch tube, each
// within GALACSI's 64 arc seconds of the slit's 55 degrees, 30 cm across
// at sodium's line; none is the study's splayed aim.
func TestTheGuideStarsShareOneAim(t *testing.T) {
	lasers := corrected(t).Lasers()
	if len(lasers) != 4 {
		t.Fatalf("%d beams, want four", len(lasers))
	}
	for _, laser := range lasers {
		_, altitude := laser.Pointing()
		radius, _, _ := laser.Envelope()
		wavelength, _ := laser.Spectrum()
		if math.Abs(altitude-slitAltitude) > mostApart+apartSlack {
			t.Errorf("%s: aimed %v degrees up, want within %v of %v",
				laser.Name(), altitude, mostApart, slitAltitude)
		}
		if !laser.Launcher() || radius != beamRadius ||
			wavelength != sodiumLine {
			t.Errorf("%s: launcher %t, radius %v, line %v", laser.Name(),
				laser.Launcher(), radius, wavelength)
		}
	}
}

// TestTheStudysCamerasAreKept: the study's four framings are in the scene, so
// the corrected scene draws them as the study drew them.
func TestTheStudysCamerasAreKept(t *testing.T) {
	have := map[string]bool{}
	for _, name := range corrected(t).Cameras() {
		have[name] = true
	}
	for _, name := range []string{"southwest", "close",
		"overhead", "south east overlook"} {
		if !have[name] {
			t.Errorf("no camera %q", name)
		}
	}
}
