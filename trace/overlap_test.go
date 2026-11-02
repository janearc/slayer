package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// glows is a material that gives off light and scatters none.
func glows(light maths.Light) *thing.Material {
	return &thing.Material{Glow: light}
}

// unlit is a material that neither gives off nor scatters light.
var unlit = &thing.Material{Name: "black"}

// nearly is whether two lights are within a share of each other, by
// luminance.
func nearly(one, other maths.Light, share float64) bool {
	x, y := one.Luminance(), other.Luminance()
	return math.Abs(x-y) <= share*math.Max(x, y)
}

// shell is a glowing shell round a ball of radius one, six units deep,
// the same all round, rising from nothing at its base as an aurora's
// does: a hard base would test the march's step against the edge, not
// the order the volumes are added in.
func shell() thing.Volume {
	return thing.Volume{Medium: medium.Aurora{Radius: 1, Low: 0, High: 6,
		Lat: 0, Width: 100, Thick: 0.2}, Material: glows(maths.Grey(1))}
}

// A glowing shell round a gas giant shows only its near side over the
// ball: at the ball's centre, what it is over a solid ball the same
// size, within a few percent. Added one whole crossing at a time, the
// shell's far side came through the giant, nearly twice as bright.
func TestAShellRoundAGiantShowsItsNearSide(t *testing.T) {
	ray := maths.Ray{At: maths.V(0, 0, 20), Dir: maths.V(0, 0, -1)}
	giant := thing.NewWorld()
	giant.AddVolume(thing.Volume{Medium: medium.Giant{Radius: 1,
		Depth: 0.08,
		Thick: 500}, Material: unlit})
	giant.AddVolume(shell())
	solid := thing.NewWorld()
	solid.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: unlit})
	solid.AddVolume(shell())
	overGiant := lone(giant, ray, 0).Light()
	overSolid := lone(solid, ray, 0).Light()
	if overSolid.Luminance() <= 0 {
		t.Fatalf("the shell over the solid ball gives %v", overSolid)
	}
	// they differ by 0.0015 of each other as measured, the march's
	// own step error; twice that is let
	if !nearly(overGiant, overSolid, 0.003) {
		t.Errorf("over the giant the shell gives %v, over a solid "+
			"ball %v",
			overGiant, overSolid)
	}
}

// Glowing air under an opaque deck of cloud does not show through it:
// straight down, the picture is what it is with the ground raised to the
// cloud's base, where there is no air below the deck to see.
func TestCloudsHideTheAirBelowThem(t *testing.T) {
	ray := maths.Ray{At: maths.V(0, 5, 0), Dir: maths.V(0, -1, 0)}
	world := func(ground float64) *thing.World {
		made := thing.NewWorld()
		made.Add(thing.Thing{Shape: shape.Sphere{Radius: ground},
			Material: unlit})
		air := medium.Air{Radius: 1, Height: 0.3, Thick: 0.5}
		made.AddVolume(thing.Volume{Medium: air,
			Material: glows(maths.Grey(1))})
		deck := medium.Clouds{Radius: 1, Low: 0.2, High: 0.4, Cover: 1,
			Thick: 400, Seed: 1}
		made.AddVolume(thing.Volume{Medium: deck, Material: unlit})
		return made
	}
	ground := lone(world(1), ray, 0).Light()
	raised := lone(world(1.2), ray, 0).Light()
	if raised.Luminance() <= 0 {
		t.Fatalf("the air above the deck gives %v", raised)
	}
	// measured 0.0002 apart; twice that is let
	if !nearly(ground, raised, 0.0004) {
		t.Errorf("down to the ground the air gives %v; with "+
			"nothing below the deck, %v", ground, raised)
	}
}

// A jet's half behind an opaque disc does not show through it: down the
// jet, through the disc, the picture is what it is with a black floor
// just under the disc, which hides the counter-jet.
func TestADiscHidesTheCounterJet(t *testing.T) {
	ray := maths.Ray{At: maths.V(0.3, 5, 0), Dir: maths.V(0, -1, 0)}
	world := func(floor bool) *thing.World {
		made := thing.NewWorld()
		jet := medium.Jet{Length: 5, Base: 0.5, Spread: 0.05,
			Thick: 5, From: 0.5}
		made.AddVolume(thing.Volume{Medium: jet,
			Material: glows(maths.Grey(1))})
		made.AddVolume(thing.Volume{Medium: medium.Disc{Inner: 0.05,
			Outer: 3,
			Thick: 0.1, Dense: 400}, Material: unlit})
		if floor {
			made.Add(thing.Thing{Shape: shape.Sphere{Radius: 100},
				Material: unlit,
				Motion:   resting(maths.V(0, -100.06, 0))})
		}
		return made
	}
	open := lone(world(false), ray, 0).Light()
	floored := lone(world(true), ray, 0).Light()
	if floored.Luminance() <= 0 {
		t.Fatalf("the jet above the disc gives %v", floored)
	}
	// measured 0.0022 apart; twice that is let
	if !nearly(open, floored, 0.0045) {
		t.Errorf("through the disc the jet gives %v; with the "+
			"counter-jet floored, %v", open, floored)
	}
}
