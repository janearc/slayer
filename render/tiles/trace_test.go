package tiles_test

import (
	"image"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/tiles"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

func TestPlannedSectionsMatchWholeTrace(t *testing.T) {
	world := thing.NewWorld()
	motion := thing.Still(maths.Subspace{Origin: maths.V(0, 0, -4)})
	world.Add(thing.Thing{
		Shape:    shape.Sphere{Radius: 1},
		Material: &thing.Material{Name: "sphere", Colour: maths.Grey(0.6)},
		Motion:   motion,
	})
	white := maths.Grey(1)
	world.Lamp(thing.Sun(maths.XAxis, white))
	eye := maths.NewSubspace(maths.V(0, 0, 0), maths.Identity, 0)
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	const width, height, side = 13, 9, 4
	full, err := trace.Sampled(world, eye, lens, 0, width, height, side)
	if err != nil {
		t.Fatal(err)
	}
	var boxes []image.Rectangle
	if err := tiles.Each(width, height, side, 20, 0, 2<<30,
		func(box image.Rectangle, _ uint64) error {
			boxes = append(boxes, box)
			return nil
		}); err != nil {
		t.Fatal(err)
	}
	parts, err := trace.SampledEach(world, eye, lens, 0, width, height, side, boxes)
	if err != nil {
		t.Fatal(err)
	}
	lit := 0
	for i, box := range boxes {
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				if full.At(x, y).R > 0 {
					lit++
				}
				if full.At(x, y) != parts[i].At(x-box.Min.X, y-box.Min.Y) {
					t.Fatalf("pixel (%d, %d) differs across tile %v", x, y, box)
				}
			}
		}
	}
	if lit == 0 {
		t.Fatal("the test scene did not illuminate any pixels")
	}

}
