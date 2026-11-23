package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestBlackGeometryStillOccludesAfterLeavingTheOffsetScale(t *testing.T) {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}, Material: &thing.Material{Colour: maths.Grey(1)}})
	toward := maths.V(0, 1, 1).Unit()
	light := maths.Grey(1)
	world.Lamp(thing.Sun(toward, light))
	eye := maths.Subspace{Origin: maths.V(0, 0, 4)}
	first, err := Render(world, eye, film.Pinhole{}, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Pix[0].R <= 0 {
		t.Fatal("control surface is not illuminated")
	}
	before := Spread(world, 0)
	sphereAt := thing.Still(maths.Subspace{Origin: maths.V(0, 7, 7)})
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 2}, Material: &thing.Material{},
		Motion: sphereAt})
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
