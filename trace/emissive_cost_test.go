package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
	"github.com/janearc/slayer/scene/thing"
)

type watchedLamp struct{ calls *int }

func (l watchedLamp) Light(_, _ maths.Vec) (maths.Vec, float64, maths.Light) {
	*l.calls++
	return maths.V(1, 1, 1).Unit(), 10, maths.Grey(4)
}
func TestEmissiveVolumeDoesNotTraceZeroScatteringShadows(t *testing.T) {
	for _, painted := range []bool{false, true} {
		calls := 0
		material := &thing.Material{Glow: maths.RGB(0, 0, 2)}
		if painted {
			material.Colour = maths.Grey(1)
			material.Texture = flat(maths.Light{})
		}
		world := thing.NewWorld()
		world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 2}, Material: material})
		world.Lamp(watchedLamp{&calls})
		ray := maths.Ray{At: maths.V(0, -3, 0), Dir: maths.Up}
		lit := lone(world, ray, 0)
		if calls != 0 {
			t.Fatal("zero-scattering volume asked lamps", calls)
		}
		world.Lamps = nil
		bare := lone(world, ray, 0)
		if lit != bare || lit.B <= 0 || !(lit.Cover > 0 && lit.Cover < 1) {
			t.Fatal("emission or absorption changed", lit, bare)
		}
	}
}
func TestPaintedScatteringStillAsksLamps(t *testing.T) {
	calls := 0
	world := thing.NewWorld()
	texture := flat(maths.Grey(1))
	world.AddVolume(thing.Volume{Medium: medium.Puff{Radius: 1, Thick: 2},
		Material: &thing.Material{Texture: texture}})
	world.Lamp(watchedLamp{&calls})
	got := lone(world, maths.Ray{At: maths.V(0, -3, 0), Dir: maths.Up}, 0)
	if calls == 0 || got.R <= 0 {
		t.Fatal("nonzero painted scattering skipped", calls, got)
	}
}
