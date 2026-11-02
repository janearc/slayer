//go:build darwin

package gpu

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// stage is a floor with a ball on it under a spot: a lamp kept to a
// cone fourteen degrees wide with eight more of soft edge, aimed at
// the floor beside the ball, so the picture has the beam's pool, its
// soft rim, the dark past it, and the ball's shadow thrown across it.
func stage(t *testing.T, width float64) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	grey := &thing.Material{Name: "floor", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: floor(t, 30), Material: grey})
	red := &thing.Material{Name: "ball", Colour: maths.RGB(0.7, 0.2, 0.2)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.7}, Material: red,
		Motion: at(0.8, 0.7, 0.2)})
	lamp := maths.V(-3, 6, 2)
	world.Lamp(emit.Lit{At: lamp, Strength: maths.Grey(60),
		Toward: maths.V(0.5, 0, 0).Sub(lamp).Unit(), Width: width,
		Soft: 8 * maths.Degree})
	from := maths.V(0, 7, 13)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// the truth for a lamp kept to a cone is slayer's tracer, which draws
// emit's own Lit and so emit's own fall across the cone.
func TestTheGPUDrawsALampInAConeAsSlayerDoes(t *testing.T) {
	world, eye := stage(t, 14*maths.Degree)
	againstAt(t, world, eye, 97, 65, 1)
}

// and the cone is in the picture: comparing dark with dark would pass
// too. kept to its cone the lamp lights a pool and leaves most of the
// floor dark; with no width the same lamp lights the whole floor.
func TestAConeLeavesMostOfTheFloorDark(t *testing.T) {
	lit := func(width float64) int {
		world, eye := stage(t, width)
		drawn, err := Sampled(world, eye, film.Pinhole{}, 0, 96, 64, 1)
		if errors.Is(err, metal.ErrNoDevice) {
			t.Skip("no gpu here")
		}
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, pixel := range drawn.Pix {
			if pixel.Cover > 0 && pixel.G > 1e-4 {
				count++
			}
		}
		return count
	}
	pool, whole := lit(14*maths.Degree), lit(0)
	if pool < 96*64/100 || pool > whole/2 {
		t.Errorf("%d pixels lit in the cone, %d with no cone", pool,
			whole)
	}
}

// the hall, with its star kept to a cone down onto the floor: the
// same fall has to reach the haze and the baked shadows as well as
// the surfaces.
func TestTheHallUnderAConeIsSlayers(t *testing.T) {
	world, eye := hall(t)
	star, ok := world.Lamps[0].(emit.Lit)
	if !ok {
		t.Fatalf("the hall's lamp is a %T", world.Lamps[0])
	}
	star.Toward, star.Width = maths.V(0.3, -1, 0.4).Unit(), 0.5
	star.Soft = 0.3
	world.Lamps[0] = star
	against(t, world, eye)
	within(t, world, eye, 96, 64, 2,
		tolerance{mean: 3e-4, largest: 1e-3})
}
