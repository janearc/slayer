//go:build draw

package trace

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// the same answers, from a whole picture: the sky above the horizon is empty
// and the ball is in the middle.
func TestAPictureOfTheScene(t *testing.T) {
	from := maths.V(0, 2, 6)
	eye := lookingAt(from, centre)
	picture := draw(smoking(), eye, film.Pinhole{Pixel: 0.75}, 0, 64, 36)
	if picture.At(0, 0).Cover != 0 {
		t.Fatal("the top corner should be sky")
	}
	mid := picture.At(32, 18)
	near := from.Sub(centre).Len() - 1
	if mid.Cover != 1 || !within(float64(mid.Depth), near, 0.05) {
		t.Fatalf("the middle should be the near side of the ball, got "+
			"%+v", mid)
	}
}

// A clear thing is drawn and casts no shadow: a ball of light over a
// floor leaves the floor lit beneath it.
func TestAClearThingCastsNoShadow(t *testing.T) {
	floor := func(shadowless bool) float32 {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Name: "floor",
				Colour: maths.Grey(0.8)}})
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Name: "ball",
				Glow: maths.Grey(2)},
			Motion: resting(maths.V(0, 2, 0)),
			Attrs:  thing.Attrs{Clear: shadowless}})
		world.Lamp(thing.Sun(maths.Up, maths.Grey(1)))
		from := maths.V(0, 1.2, 8)
		eye := lookingAt(from, maths.Vec{})
		lens := film.Pinhole{Wide: 30 * maths.Degree}
		picture := draw(world, eye, lens, 0, 41, 41)
		// the floor straight under the ball: the middle of the
		// picture, where the eye looks, is the floor's origin
		return picture.At(20, 20).R
	}
	// the sun is overhead, so the lit floor is its own 0.8 over pi,
	// matte, and the shadowed floor has no light at all
	lit, shadowed := floor(true), floor(false)
	if !within(float64(lit), 0.8/math.Pi, 1e-6) || shadowed != 0 {
		t.Errorf("under a clear ball the floor is %v, under a solid "+
			"one %v", lit, shadowed)
	}
}

// The tracer ticks what moves: a thin disc with an orbital rate, drawn at two
// times, differs, and at time zero it is a disc standing still, byte for byte.
func TestTheTracerTicksAMovingDisc(t *testing.T) {
	disc := func(omega float64) *thing.World {
		world := thing.NewWorld()
		shine := texture.Thin{Inner: 1, Outer: 10, Hot: 6000, Bright: 1,
			Streaks: 1, Seed: 3, Omega: omega}
		world.Add(thing.Thing{Shape: shape.Plane{},
			Material: &thing.Material{Shine: shine}})
		return world
	}
	from := maths.V(0, 12, 6)
	eye := lookingAt(from, maths.Vec{})
	lens := film.Pinhole{Wide: 60 * maths.Degree}
	still := draw(disc(0), eye, lens, 0, 32, 32)
	now := draw(disc(1), eye, lens, 0, 32, 32)
	later := draw(disc(1), eye, lens, 20*time.Second, 32, 32)
	same, moved := true, false
	for i := range now.Pix {
		same = same && now.Pix[i] == still.Pix[i]
		moved = moved || later.Pix[i] != now.Pix[i]
	}
	if !same {
		t.Error("at nought the moving disc is not the still one")
	}
	if !moved {
		t.Error("twenty seconds on the disc has not moved")
	}
}
