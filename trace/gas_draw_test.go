//go:build draw

package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// looking is one pixel of the world seen from a place along a direction.
func looking(world *thing.World, from, along maths.Vec) maths.Light {
	eye := lookingAt(from, from.Add(along))
	lens := film.Pinhole{Wide: 1 * maths.Degree}
	return draw(world, eye, lens, 0, 3, 3).At(1, 1).Light()
}

// A gas air lights itself from its own numbers: straight up from the
// ground at noon the sky is blue, more blue than green than red, by
// rayleigh's fourth power across the eye's bands, and not the magenta
// the air's material is painted, which a gas volume never reads.
//
// Toward the sun along the horizon the light that gets through the
// long path is redder than the light from straight up.
func TestAGasAirMakesABlueSky(t *testing.T) {
	world := earthlike()
	up := maths.V(1.0001, 0, 0) // on the ground under the sun
	sky := looking(world, up, maths.V(1, 0, 0))
	if !(sky.B > sky.G && sky.G > sky.R && sky.R > 0) {
		t.Fatalf("the noon sky is %v", sky)
	}
	// how blue: the gas's own scattering at the blue band over the red,
	// which for air is rayleigh's fourth power, about three. What gets
	// through the air on the way moves it, so fifteen percent is let.
	air := gas.Air()
	for _, channel := range []struct {
		name string
		got  float64
		band maths.Band
	}{{"blue", sky.B, eyeBands[2]}, {"green", sky.G, eyeBands[1]}} {
		want := air.Scatter(channel.band) / air.Scatter(eyeBands[0])
		if !within(channel.got/sky.R, want, 0.15*want) {
			t.Errorf("the sky's %s over its red is %.3f, want %.3f",
				channel.name, channel.got/sky.R, want)
		}
	}
	// from a point on the terminator the sun lies along the horizon
	edge := maths.V(0, 1.0001, 0)
	low := looking(world, edge, maths.V(1, 0, 0))
	if !(low.R > 0 && low.B > 0) {
		t.Fatalf("the horizon toward the sun is %v", low)
	}
	if low.R/low.B <= sky.R/sky.B {
		t.Errorf("the horizon (%v) is not redder than the zenith (%v)",
			low, sky)
	}
	// at night, away from the sun, the air gives nothing
	night := looking(world, maths.V(-1.0001, 0, 0), maths.V(-1, 0, 0))
	if night != (maths.Light{}) {
		t.Errorf("the night sky is %v", night)
	}
}
