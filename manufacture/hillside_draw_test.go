//go:build draw

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
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

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
func drawn(t *testing.T, world *thing.World, width, height, rays int) *frame.Frame {
	t.Helper()
	from := maths.V(6, 3.4, 15)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.V(-0.5, 1.8, 0))}
	f, err := trace.Sampled(world, eye, film.Pinhole{Wide: 55 * maths.Degree},
		0, width, height, rays)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestAHillsideIsDrawnWithItsThingsInIt(t *testing.T) {
	const width, height = 120, 75
	empty := drawn(t, bare(t), width, height, 1)
	full := drawn(t, hillside(t), width, height, 1)
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
	large := drawn(t, hillside(t), 960, 600, 2)
	picture := colour.Tone{Knee: 0.7, Top: 8}.Develop(large)
	if err := file.PNG(out, picture, 1); err != nil {
		t.Fatal(err)
	}
}
