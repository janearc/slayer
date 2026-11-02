package manufacture

import (
	"os"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/forest"
	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/trace"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// the smoke test: everything the manufactory makes so far, stood on a
// ground and drawn by slayer's tracer (render/trace) on the cpu. it proves
// a picture comes out and that the things are in it. it cannot prove a fern
// looks like a fern: set HILLSIDE_OUT to a path and it writes a larger
// picture there, to be looked at.
//
// the scene, its ground and its light are this test's own, set by
// eye. the things and their looks are the kinds' rows.
var soil = &thing.Material{Name: "soil",
	Colour: maths.RGB(0.2, 0.17, 0.1)}

const (
	// the ground is so many cells of so many metres, and things stand
	// on a grid of places this far apart
	groundCells, groundSize = 8, 160.0
	placesASide, apart      = 6, 2.4
)

// bare is the hillside with nothing on it: ground, sky and sun.
func bare(t *testing.T) *thing.World {
	t.Helper()
	world := thing.NewWorld()
	world.Sky = sky.Gradient{Horizon: maths.RGB(0.7, 0.8, 0.95),
		Zenith: maths.RGB(0.25, 0.45, 0.9),
		Ground: maths.RGB(0.34, 0.29, 0.18)}
	// a sun, and a faint second one from the other side standing in
	// for the light of the sky, which the tracer does not gather
	world.Lamp(thing.Sun(maths.V(-0.5, 0.8, 0.45), maths.Grey(1.5)))
	world.Lamp(thing.Sun(maths.V(0.6, 0.5, -0.3), maths.Grey(0.4)))
	ground, err := forest.Ground(groundCells, groundSize)
	if err != nil {
		t.Fatal(err)
	}
	corner := maths.V(-groundSize/2, 0, -groundSize/2)
	world.Add(thing.Thing{Shape: ground, Material: soil,
		Motion: thing.Still(maths.Subspace{Origin: corner})})
	return world
}

// kindAt is which kind stands at a place: mostly stones, a fern at
// every third place and a spruce at every ninth.
func kindAt(index int) string {
	if index%9 == 0 {
		return "spruce"
	}
	if index%3 == 1 {
		return "fern"
	}
	return []string{"boulder", "cobble", "boulder", "log"}[index%4]
}

// hillside is the bare hillside with a thing stood at every place, and
// somebody's can at the front.
func hillside(t *testing.T) *thing.World {
	t.Helper()
	world := bare(t)
	places, err := forest.Placements(11, placesASide, apart, 0.8)
	if err != nil {
		t.Fatal(err)
	}
	middle := apart * (placesASide - 1) / 2
	for index, placed := range places {
		made := mustMake(t, kindAt(index), uint64(1000+index))
		at := placed.At().Sub(maths.V(middle, 0, middle))
		put(t, world, made, maths.Subspace{Origin: at,
			Orientation: maths.About(maths.YAxis, placed.Turn())})
	}
	put(t, world, mustMake(t, "can", 2),
		maths.Subspace{Origin: maths.V(3.2, 0, middle+4)})
	return world
}

// put stands a made thing in a world.
func put(t *testing.T, world *thing.World, made good,
	at maths.Subspace) {
	t.Helper()
	things, err := made.things(at)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range things {
		world.Add(one)
	}
}

// drawn is a world seen from above head height at its front corner.
func drawn(world *thing.World, width, height, rays int) *frame.Frame {
	from := maths.V(6, 3.4, 15)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(-0.5, 1.8, 0))}
	return trace.Sampled(world, eye, film.Pinhole{Wide: 55 * maths.Degree},
		0, width, height, rays)
}

func TestAHillsideIsDrawnWithItsThingsInIt(t *testing.T) {
	const width, height = 120, 75
	empty := drawn(bare(t), width, height, 1)
	full := drawn(hillside(t), width, height, 1)
	lit, changed := 0, 0
	for index, pixel := range full.Pix {
		light, was := pixel.Light(), empty.Pix[index].Light()
		if light.R+light.G+light.B > 0 {
			lit++
		}
		if light != was {
			changed++
		}
	}
	if lit < len(full.Pix)/2 {
		t.Errorf("%d of %d pixels have any light: the picture is "+
			"dark", lit, len(full.Pix))
	}
	// the things stand across the middle of the picture; if under a
	// tenth of it changed when they were added, they are not in it
	if changed < len(full.Pix)/10 {
		t.Errorf("%d of %d pixels changed when the things were stood "+
			"on the ground", changed, len(full.Pix))
	}
	out := os.Getenv("HILLSIDE_OUT")
	if out == "" {
		return
	}
	large := drawn(hillside(t), 960, 600, 2)
	picture := colour.Tone{Knee: 0.7, Top: 8}.Develop(large)
	if err := file.PNG(out, picture, 1); err != nil {
		t.Fatal(err)
	}
}
