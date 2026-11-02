package trace

import (
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// busy is a small world with every kind of march in it: a floor, a ball
// that has only a distance to be found by, a painted volume and a gas
// volume over them, and a sun to shadow through both.
func busy() *thing.World {
	w := thing.NewWorld()
	w.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: maths.Grey(0.5)}})
	w.Add(thing.Thing{Shape: struct{ shape.Shape }{shape.Sphere{Radius: 0.5}},
		Material: &thing.Material{Colour: maths.Grey(0.5)},
		Motion:   at(maths.V(1, 0.5, 0), 1)})
	w.AddVolume(thing.Volume{Medium: ball{k: 0.2, radius: 1},
		Material: &thing.Material{Colour: maths.Grey(1)},
		Motion:   at(maths.V(-1, 1.5, 0), 1)})
	w.AddVolume(thing.Volume{Medium: gas.Of{Shape: block{k: 0.2, half: 1},
		In: tinted(gas.Optics{Absorb: 0.1}, gas.Optics{Absorb: 0.1},
			gas.Optics{Absorb: 0.1, Scatter: 0.1}), Unit: 1},
		Motion: at(maths.V(0, 1.5, 0), 1)})
	w.Lamp(thing.Sun(maths.V(0, 1, 0.3), maths.Grey(1)))
	return w
}

// a picture of a few pixels is drawn in no time, whatever way the rows,
// the steps of a march and the shadow samples count. a loop that counts
// the wrong way never ends, and these fail the run instead of holding it
// up. the file name puts this before every test that draws.
func TestEveryPictureIsDrawnInNoTime(t *testing.T) {
	w := busy()
	from := maths.V(0, 3, 6)
	eye := maths.Subspace{Origin: from, Orientation: maths.LookAt(from, maths.V(0, 0.7, 0))}
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	ends(t, "a picture", func() { Render(w, eye, lens, 0, 6, 5) })
	ends(t, "a picture sampled twice a pixel", func() { Sampled(w, eye, lens, 0, 3, 3, 2) })
	ends(t, "a picture in a band", func() {
		RenderIn(w, eye, lens, 0, 6, 5, maths.Band{Nanometres: 1})
	})
	ends(t, "a picture in a band with a sun to light it", func() {
		RenderIn(w, eye, lens, 0, 6, 5, maths.Band{Nanometres: 656})
	})
	// a ray that gets out of a field comes back lit, which is asked first:
	// a march that goes on after it is done does not come back at all
	h := hole(nil)
	var escaped maths.Light
	ends(t, "a ray bent by a field", func() {
		escaped = Trace(h, maths.Ray{At: maths.V(0, 8, -100), Dir: maths.V(0, 0, 1)}, 0).Light()
	})
	if escaped == (maths.Light{}) {
		t.Fatal("a ray that gets out of a field came back dark")
	}
	ends(t, "rays in a field", func() {
		Trace(h, maths.Ray{At: maths.V(0, 0, -20), Dir: maths.V(0, 0, -1)}, 0)
		Trace(h, maths.Ray{At: maths.V(0, 0, -100), Dir: maths.V(0, 0, 1)}, 0)
	})
	ends(t, "a ray through the volumes", func() {
		Trace(w, maths.Ray{At: maths.V(0, 6, 0.1), Dir: maths.V(0, -1, 0)}, 0)
		Trace(w, maths.Ray{At: maths.V(-4, 1.5, 0), Dir: maths.V(1, 0, 0)}, 0)
	})
}
