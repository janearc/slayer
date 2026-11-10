package trace

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"testing"
)

func emitterScaleFixture() (*thing.World, *thing.Material) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{Colour: maths.Grey(.3)}})
	far := &thing.Material{Glow: maths.Grey(1)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1.7e6}, Material: far, Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -3.8e8)})})
	return world, far
}
func TestDistantEmissionDoesNotLiftLocalShadowRays(t *testing.T) {
	world, _ := emitterScaleFixture()
	only := thing.NewWorld()
	only.Add(*world.Things[0])
	want := Spread(only, 0)
	got := Spread(world, 0)
	if got != want {
		t.Fatal("emitter changes local offset scale", got, want)
	}
	if scene := stage(world, 0); scene.spread != want {
		t.Fatal("CPU stage disagrees with shared GPU scale", scene.spread, want)
	}
}
func TestReflectingAndPaintedBodiesStillSetOffsetScale(t *testing.T) {
	for _, change := range []func(*thing.Material){
		func(m *thing.Material) { m.Colour = maths.Grey(.1) },
		func(m *thing.Material) { m.Texture = flat(maths.Grey(.1)) },
		func(m *thing.Material) { m.Gloss = .5 },
	} {
		world, m := emitterScaleFixture()
		change(m)
		if got := Spread(world, 0); got < 3e8 {
			t.Fatal("reflecting body omitted", got)
		}
	}
}
func TestAllEmittingSceneStillOffsetsAtDistantPoints(t *testing.T) {
	world, _ := emitterScaleFixture()
	world.Things = world.Things[1:]
	scene := stage(world, 0)
	if scene.spread != 0 {
		t.Fatal("emitter-only scene has diffuse scale", scene.spread)
	}
	point := maths.V(0, 0, -3.8e8)
	if got := scene.lifting(point); got != lift*point.Len() || got <= 0 {
		t.Fatal("lost far-coordinate offset", got)
	}
}
