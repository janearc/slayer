//go:build draw

package trace

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

const (
	// unitMass is the mass of the test fields, defined as one, the unit
	// the spins below are shares of.
	unitMass = 1
	// fieldReach is how far the test fields reach, defined as forty
	// masses, past the horizon so that the field is in the frame.
	fieldReach = 40
	// backwardSpin is a spin below zero, defined as the half a mass turned
	// the wrong way: celestial refuses a spin below zero.
	backwardSpin = -0.5
	// edgeSpin is a spin at the mass, defined as the share where a hole has
	// no horizon: celestial refuses it.
	edgeSpin = 1
	// pastSpin is a spin past the mass, defined as twice the mass, which
	// celestial refuses as it refuses a spin at the mass.
	pastSpin = 2
	// noReach is the reach of a field that reaches nothing, defined: zero.
	noReach = 0
	// quarrelSpin is the spin of the field of no reach, defined as a spin
	// celestial would refuse if it were asked: it is not.
	quarrelSpin = -5
)

// refusedSpins are the spins celestial refuses: below zero, at the mass
// and past it.
var refusedSpins = []float64{backwardSpin, edgeSpin, pastSpin}

// A field whose mass and spin celestial says make no hole is refused by
// every entry point, with an error that names the field, its mass, its
// spin and what celestial said, and no frame: it is not left out of the
// picture. The spins are below zero, which celestial refuses, and at the
// mass, where it has no horizon. The sound field beside them is drawn.
func TestAFieldCelestialRefusesIsRefusedByName(t *testing.T) {
	for _, spin := range refusedSpins {
		world := thing.NewWorld()
		world.AddField(thing.Field{Mass: unitMass, Reach: fieldReach})
		world.AddField(thing.Field{Mass: unitMass, Spin: spin, Reach: fieldReach})
		for name, draw := range entries(world) {
			drawn, err := draw()
			if err == nil || drawn {
				t.Errorf("%s drew a field of spin %v: %v", name, spin, err)
				continue
			}
			for _, want := range []string{"field 1", "mass of 1",
				"spin of", "hole: a spin of"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%s, spin %v: %q lacks %q", name, spin,
						err, want)
				}
			}
		}
	}
}

// A field of no reach or no mass is no field, and is left out as it
// always was; it is not a field celestial was asked about.
func TestAFieldOfNoReachIsStillNoField(t *testing.T) {
	world := thing.NewWorld()
	world.AddField(thing.Field{Mass: unitMass, Spin: quarrelSpin, Reach: noReach})
	for name, draw := range entries(world) {
		if _, err := draw(); err != nil {
			t.Errorf("%s refused a field of no reach: %v", name, err)
		}
	}
}
