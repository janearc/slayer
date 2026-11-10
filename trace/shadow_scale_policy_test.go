package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestSecondaryRayScaleUsesReflectingSurfacesRegardlessOfDistance(t *testing.T) {
	for _, test := range []struct {
		name     string
		material *thing.Material
		reflects bool
	}{
		{"black", &thing.Material{}, false},
		{"pure glow", &thing.Material{Glow: maths.Grey(1)}, false},
		{"emission texture", &thing.Material{Shine: flat(maths.Grey(1))}, false},
		{"default material", nil, true},
		{"diffuse", &thing.Material{Colour: maths.Grey(0.1)}, true},
		{"reflectance texture", &thing.Material{Texture: flat(maths.Grey(0.1))}, true},
		{"gloss", &thing.Material{Gloss: 0.5}, true},
		{"spectral", &thing.Material{Radiant: thing.Spectrum{}}, true},
	} {
		for _, far := range []bool{false, true} {
			world := thing.NewWorld()
			world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Colour: maths.Grey(1)}})
			local := Spread(world, 0)
			space := maths.Subspace{}
			if far {
				space.Origin = maths.V(0, 0, -3.8e8)
			}
			world.Add(thing.Thing{Shape: shape.Sphere{Radius: 100}, Material: test.material, Motion: thing.Still(space)})
			got := Spread(world, 0)
			if test.reflects && got <= local || !test.reflects && got != local {
				t.Fatalf("%s far=%v: local %g, scale %g", test.name, far, local, got)
			}
			if scene := stage(world, 0); scene.spread != got {
				t.Fatal("CPU stage changed the shared policy")
			}
		}
	}
}

func TestBlackGeometryStillOccludesAfterLeavingTheOffsetScale(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Colour: maths.Grey(1)}})
	world.Lamp(thing.Sun(maths.V(0, 1, 1).Unit(), maths.Grey(1)))
	eye := maths.Subspace{Origin: maths.V(0, 0, 4)}
	first, err := Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Pix[0].R <= 0 {
		t.Fatal("control surface is not illuminated")
	}
	before := Spread(world, 0)
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 7, 7)})})
	if Spread(world, 0) != before {
		t.Fatal("black occluder changed local offset scale")
	}
	second, err := Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Pix[0].R != 0 {
		t.Fatal("black occluder stopped casting a shadow", second.Pix[0])
	}
}
