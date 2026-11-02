package trace

import (
	"math"
	"testing"

	kerr "github.com/janearc/celestial/hole"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/gas"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// reddish is a gas that takes red and leaves blue, the other way
// round from air: it absorbs in step with the wavelength and scatters
// nothing.
func reddish() gas.Gas {
	return gas.Gas{Name: "takes red", Visible: func(
		nanometres float64) (absorb, scatter float64) {
		return nanometres / 1000, 0
	}}
}

// underRedGas is a slab of the gas that takes red, two units deep
// across the ray alongZ, in an empty world, with a weak field round
// it or not.
func underRedGas(field bool) *thing.World {
	world := thing.NewWorld()
	world.AddVolume(thing.Volume{Medium: gas.Of{
		Shape: plate(1, -1, 1), In: reddish(), Unit: 1}})
	if field {
		world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	}
	return world
}

// A pixel is covered by as much as the least-covered channel is: one
// less what the volumes let through of the channel they let through
// most. Through a gas that takes red and leaves blue, against no sky,
// that is one less what is kept of blue, the same on the way out of a
// field as with none. Read from the red channel it would be more.
func TestCoverIsByTheChannelThatGetsThroughMost(t *testing.T) {
	keep := byEye(func(band maths.Band) float64 {
		return reddish().Transmit(band, 2)
	})
	want, byRed := 1-keep.B, 1-keep.R
	if !(keep.B > keep.R+0.05) {
		t.Fatalf("the test's gas keeps %v: blue should be well over "+
			"red", keep)
	}
	// three units off the field's middle, where a ray is black
	from := maths.Ray{At: maths.V(3, 0, -20), Dir: maths.V(0, 0, 1)}
	for name, world := range map[string]*thing.World{
		"with no field":  underRedGas(false),
		"out of a field": underRedGas(true)} {
		got := float64(lone(world, from, 0).Cover)
		if !within(got, want, 1e-3) {
			t.Errorf("%s the gas covers %v of the pixel, want %v; "+
				"by red it would be %v", name, got, want, byRed)
		}
	}
}

// A ball in front of a field, seen from outside, is the ball as it is
// with no field: the ray meets it before it reaches the field.
func TestABallBeforeAFieldIsDrawnAsItIs(t *testing.T) {
	build := func(field bool) *thing.World {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Material: &thing.Material{Colour: maths.Grey(0.8)},
			Motion:   resting(maths.V(0, 0, 5))})
		world.Lamp(thing.Sun(maths.V(0, 0, 1), maths.Grey(1)))
		if field {
			world.AddField(thing.Field{Mass: 0.1, Reach: 2})
		}
		return world
	}
	at := maths.Ray{At: maths.V(0.2, 0.1, 20), Dir: maths.V(0, 0, -1)}
	plain, before := lone(build(false), at, 0), lone(build(true), at, 0)
	// lit: past half of a white matte surface's own, which is one over pi
	if plain.Cover != 1 || !(bright(plain) > 0.5/math.Pi) || before != plain {
		t.Errorf("before a field the ball is %+v, and %+v with none",
			before, plain)
	}
}

// An eye inside a field has the four-velocity the field was told, its
// time and its way in the hole's own axes; told none, it is an eye
// held still there.
func TestAnEyeInAFieldMovesAsItWasTold(t *testing.T) {
	world := thing.NewWorld()
	world.AddField(thing.Field{Mass: 1, Reach: 40, EyeTime: 1.5,
		Eye: maths.V(0.2, 0, -0.1)})
	world.AddField(thing.Field{Mass: 1, Reach: 40})
	fields := placeFields(world, 0)
	here := kerr.NewVec3(0, 10, 0)
	way := fields[0].source.InHole(maths.V(0.2, 0, -0.1))
	want := kerr.NewVec4(1.5, way[0], way[1], way[2])
	if got := fields[0].eye(here); got != want {
		t.Errorf("the eye told to move has %v, want %v", got, want)
	}
	still := fields[1].hole.Static(here)
	if got := fields[1].eye(here); got != still {
		t.Errorf("the eye told nothing has %v, want it still: %v", got,
			still)
	}
}

// A volume in a field is still unless it has a beam whose speed is between zero
// and one; a round beam with such a speed goes round at the hole's own rate,
// whatever its speed.
func TestAVolumesFlowInAField(t *testing.T) {
	world := thing.NewWorld()
	world.AddField(thing.Field{Mass: 1, Reach: 40})
	field := placeFields(world, 0)[0]
	point := maths.V(10, 0, 0)
	here := field.in(point)
	flow := func(beam *thing.Beam) kerr.Vec4 {
		volume := staging(thing.Volume{Medium: plate(1, -1, 1),
			Beam: beam}).volumes[0]
		return field.emitter(&volume, point)
	}
	still, round := field.hole.Static(here), field.hole.Circular(here)
	if round == still {
		t.Fatal("the hole's circular flow is the still one here")
	}
	if got := flow(nil); got != still {
		t.Errorf("with no beam the volume flows %v, not still", got)
	}
	stopped := &thing.Beam{Round: true}
	if got := flow(stopped); got != still {
		t.Errorf("a round beam of no speed flows %v, not still", got)
	}
	turning := &thing.Beam{Round: true, Speed: 0.3}
	if got := flow(turning); got != round {
		t.Errorf("a round beam flows %v, want the hole's rate %v", got,
			round)
	}
	along := flow(&thing.Beam{Speed: 0.3})
	if along == still || along == round || math.IsNaN(along.Time()) {
		t.Errorf("a beam along its axis flows %v", along)
	}
}
