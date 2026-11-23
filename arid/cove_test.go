package arid

import (
	"errors"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

func TestAdvanceSharesTheWaterAndKeepsTheShore(t *testing.T) {
	made, err := New([]thing.Thing{{Shape: shape.Plane{}}}, &Water{
		Name: "test cove", Sea: oceantex.Sea{Wind: 3, Fetch: 1100, Depth: 8},
		Extinction: maths.Grey(.1), Extent: 24, Collar: 6, Samples: 33, Seed: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	place := func(frame *thing.World, water *shape.HeightField) error {
		if water == nil || frame.OceanSurface == nil {
			t.Fatal("occupant received no water")
		}
		frame.Add(thing.Thing{Shape: shape.Sphere{Radius: 1}})
		return nil
	}
	if err := made.Advance(0, place); err != nil {
		t.Fatal(err)
	}
	world := made.World()
	start := world.OceanSurface.Boundary
	if len(world.Things) != 3 || world.Things[1] != start {
		t.Fatal("the frame did not assemble one occupant, one water surface and one shore")
	}
	shore := world.Things[2]
	refused := errors.New("occupant placement failed")
	err = made.Advance(time.Second, func(*thing.World, *shape.HeightField) error {
		return refused
	})
	if !errors.Is(err, refused) || world.OceanSurface.Boundary != start {
		t.Fatal("failed occupant placement changed the published frame")
	}
	if err := made.Advance(time.Second, nil); err != nil {
		t.Fatal(err)
	}
	if len(world.Things) != 2 || world.Things[1] != shore ||
		world.OceanSurface.Boundary.Material != start.Material {
		t.Fatal("advancing changed the shore or water medium")
	}
}
