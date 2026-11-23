package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

// fixedGlow is a glow of one mean radiance everywhere, for the one air.
type fixedGlow struct {
	light maths.Light
	air   medium.Medium
}

// Mean is the one radiance.
func (glow fixedGlow) Mean(maths.Vec) maths.Light { return glow.light }

// Lights says whether the medium is the glow's own air.
func (glow fixedGlow) Lights(made medium.Medium) bool {
	return made == glow.air
}

// glowTolerance is how close two lights drawn by one world are taken to
// be the same, as a share of the light: a pixel holds single precision.
const glowTolerance = 1e-5

// puffWorld is a world of one painted puff of air, albedo one, with no
// lamp in it, and a glow if one is given.
func puffWorld(air medium.Medium, glow thing.Glow) *thing.World {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: air,
		Material: &thing.Material{Colour: maths.Grey(1)}})
	world.Glow = glow
	return world
}

// glowRay goes up through the middle of the puff.
var glowRay = maths.Ray{At: maths.V(0, -3, 0), Dir: maths.Up}

// A glow lights the air it is of: a march through that air gathers the
// albedo times the mean radiance, so a puff with no lamp is not dark.
func TestAGlowLightsTheAirItIsOf(t *testing.T) {
	air := medium.Puff{Radius: 1, Thick: 2}
	bare := lone(puffWorld(air, nil), glowRay, 0)
	halfGlow := fixedGlow{maths.Grey(0.5), air}
	lit := lone(puffWorld(air, halfGlow), glowRay, 0)
	if bare.R != 0 || lit.R <= 0 || lit.G <= 0 || lit.B <= 0 {
		t.Fatal("a glow should light its air and a bare one stay dark",
			bare, lit)
	}
}

// The light a glow adds is the glow's, no more: twice the radiance adds
// twice the light, and none adds none.
func TestTheLightAGlowAddsIsLinearInTheGlow(t *testing.T) {
	air := medium.Puff{Radius: 1, Thick: 2}
	halfGlow := fixedGlow{maths.Grey(0.5), air}
	one := lone(puffWorld(air, halfGlow), glowRay, 0)
	fullGlow := fixedGlow{maths.Grey(1), air}
	two := lone(puffWorld(air, fullGlow), glowRay, 0)
	none := lone(puffWorld(air, fixedGlow{}), glowRay, 0)
	bare := lone(puffWorld(air, nil), glowRay, 0)
	if math.Abs(float64(two.R-2*one.R)) > glowTolerance*float64(two.R) ||
		none != bare {
		t.Fatal("the added light is not the glow's", one, two, none, bare)
	}
}

// Anti-test: a glow of one air lights no other. A volume that is not the
// air the glow was solved for takes nothing from it.
func TestAGlowOfOneAirLightsNoOtherAir(t *testing.T) {
	air := medium.Puff{Radius: 1, Thick: 2}
	other := medium.Puff{Radius: 2, Thick: 2}
	bare := lone(puffWorld(air, nil), glowRay, 0)
	halfGlow := fixedGlow{maths.Grey(0.5), other}
	got := lone(puffWorld(air, halfGlow), glowRay, 0)
	if got != bare {
		t.Fatal("a glow lit an air it is not of", got, bare)
	}
}

// Anti-test: a world with no glow draws as it always did, with a lamp
// scattering in the air as before.
func TestAWorldWithNoGlowIsDrawnAsBefore(t *testing.T) {
	air := medium.Puff{Radius: 1, Thick: 2}
	calls := 0
	world := puffWorld(air, nil)
	world.Lamp(watchedLamp{&calls})
	with := lone(world, glowRay, 0)
	world.Glow = fixedGlow{maths.Grey(1), medium.Puff{Radius: 9}}
	away := lone(world, glowRay, 0)
	if calls == 0 || with != away {
		t.Fatal("a glow of other air changed the lamps' scattering", calls,
			with, away)
	}
}
