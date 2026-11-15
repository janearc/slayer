//go:build draw

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	slayer "github.com/janearc/slayer/trace"
)

// spotLit is a clay ball on a floor under one spot from above and to
// one side, its cone narrow enough that its edge falls across the floor
// in the picture: inside the pool is lit, outside is dark, and the ball
// casts a shadow in it.
func spotLit(lamp film.Spot) (*thing.World, maths.Subspace) {
	world := thing.NewWorld()
	clay := &thing.Material{Name: "clay", Colour: maths.Grey(0.7)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 0.5}, Material: clay})
	floor := &thing.Material{Name: "floor", Colour: maths.Grey(0.6)}
	world.Add(thing.Thing{Shape: shape.Sphere{Radius: 200}, Material: floor,
		Motion: at(0, -200.5, 0)})
	world.Lamp(lamp)
	from := maths.V(0, 2.5, 6)
	return world, maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, maths.Vec{})}
}

// key is the spot these tests light by: four units up and two aside,
// aimed at the ball, twenty degrees to its edge.
func key(soft float64) film.Spot {
	at := maths.V(2, 4, 1)
	return film.Spot{At: at, Aim: maths.Vec{}.Sub(at),
		Strength: maths.Grey(30), Cone: 20 * maths.Degree,
		Soft: soft * maths.Degree}
}

// a spot is the point emitter kept to its cone, and no other lamp: the
// kernel is handed one as the other, so they must be one. On the gpu
// the two are one picture to the bit, since they are one row of
// numbers; in slayer's tracer, where each is its own code, they are
// one to rounding. What is left between the tracer's spot and the
// kernel's is then the kernel's cone, and not this handing over.
func TestASpotIsThePointEmitterKeptToItsCone(t *testing.T) {
	spot := key(6)
	lit, eye := spotLit(spot)
	lit.Lamps = []thing.Lamp{spotAsLit(spot)}
	spotted, _ := spotLit(spot)
	onGPU, asLit := drawn(t, spotted, eye), drawn(t, lit, eye)
	for at := range onGPU.Pix {
		if onGPU.Pix[at] != asLit.Pix[at] {
			t.Fatalf("pixel %d is %+v under the spot and %+v under the "+
				"emitter, on the gpu", at, onGPU.Pix[at], asLit.Pix[at])
		}
	}
	lens := film.Pinhole{Wide: 50 * maths.Degree}
	traced, err := slayer.Sampled(spotted, eye, lens, 0, 96, 64, 2)
	if err != nil {
		t.Fatal(err)
	}
	tracedLit, err := slayer.Sampled(lit, eye, lens, 0, 96, 64, 2)
	if err != nil {
		t.Fatal(err)
	}
	if found := compare(traced, tracedLit, 96, 64); found.mean > 1e-7 ||
		found.largest > 1e-6 {
		t.Errorf("the spot and the emitter are %+v apart in slayer's "+
			"tracer", found)
	}
	t.Logf("the emitter, tracer against gpu: %+v",
		compare(tracedLit, asLit, 96, 64))
}

// softEdged is the tolerance for a spot whose edge fades across six
// degrees: the kernel finds how far off its aim a way lies in floats,
// and across so narrow an edge that shows. Measured on 2026-11-12 the
// gpu is 1.4e-5 off slayer's tracer on average and 2.0e-5 at most, with
// the spot or with the emitter it is handed over as; a hard-edged spot
// is 3.5e-6 and 3.7e-6. This is about twice the first.
var softEdged = tolerance{mean: 3e-5, largest: 1e-3}

// the gpu draws a spot as slayer's tracer does: a pool of light with a
// soft edge, falling by the square of the distance, and a hard shadow
func TestTheGPUDrawsASpotAsSlayerDoes(t *testing.T) {
	world, eye := spotLit(key(6))
	baking := bakeSide
	bakeSide = 0
	defer func() { bakeSide = baking }()
	within(t, world, eye, 96, 64, 2, softEdged)
}

// a spot with no soft edge is drawn too: all of its light inside its
// cone and none outside
func TestTheGPUDrawsAHardEdgedSpotAsSlayerDoes(t *testing.T) {
	world, eye := spotLit(key(0))
	against(t, world, eye)
}

// the anti-test of the two above: the spot's cone is drawn. the same
// lamp with a cone twice as wide lights more of the floor, and is
// another picture on the gpu
func TestASpotsConeShowsOnTheGPU(t *testing.T) {
	narrow, eye := spotLit(key(6))
	wide := key(6)
	wide.Cone = 40 * maths.Degree
	wider, _ := spotLit(wide)
	found := compare(drawn(t, narrow, eye), drawn(t, wider, eye), 96, 64)
	if found.mean < 1e-2 {
		t.Errorf("a spot of twenty degrees and one of forty are %.5f "+
			"apart: the cone is not drawn", found.mean)
	}
}

// a spot the kernel's cone cannot say is refused by name: one soft all
// the way to its middle, whose cone would have no width and so light
// every way, and one aimed no way. The anti-test is the key, which is
// neither and is not refused
func TestASpotTheConeCannotSayIsRefused(t *testing.T) {
	soft := key(20)
	softer := key(25)
	aimless := key(6)
	aimless.Aim = maths.Vec{}
	lost := key(6)
	lost.Aim = maths.V(math.NaN(), 0, 0)
	for name, each := range map[string]struct {
		lamp film.Spot
		says string
	}{
		"soft to its middle":   {soft, "a spot soft to its middle"},
		"soft past its middle": {softer, "a spot soft to its middle"},
		"aimed no way":         {aimless, "a spot aimed no way"},
		"aimed at no number":   {lost, "a spot aimed no way"},
	} {
		var cause unsupported
		if err := refuseLamp(each.lamp, 1); !errors.As(err, &cause) ||
			cause.what != each.says {
			t.Errorf("%s: %v, want it refused as %q", name, err, each.says)
		}
	}
	if err := refuseLamp(key(6), 1); err != nil {
		t.Errorf("the key was refused: %v", err)
	}
}
