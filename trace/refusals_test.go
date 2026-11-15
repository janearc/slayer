package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/thing"
)

// A lamp that sends no light at a point adds nothing to a gas step there,
// whatever way it says that light comes from: the pixel through a gas lit
// by a lamp with no strength and no direction is the pixel of the same
// gas with no lamp, as a surface treats a lamp that sends nothing.
func TestALampOfNoLightAddsNothingToAGas(t *testing.T) {
	air := gas.Air()
	slab := thing.Volume{Medium: gas.Of{Shape: plate(1e4, -1, 1),
		In: air, Unit: 1}}
	nan := math.NaN()
	bare := thing.NewWorld()
	bare.AddVolume(slab)
	want := lone(bare, alongZ, 0)
	for name, toward := range map[string]maths.Vec{
		"a direction that is no number": maths.V(nan, nan, nan),
		"a direction of nothing":        {},
		"a direction":                   maths.V(0, 1, 0)} {
		world := thing.NewWorld()
		world.AddVolume(slab)
		world.Lamp(film.Sun{Toward: toward})
		got := lone(world, alongZ, 0)
		if got != want {
			t.Errorf("a lamp of no light from %s gives %v, want %v",
				name, got, want)
		}
	}
}
