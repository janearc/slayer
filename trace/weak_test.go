package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// weakHaze is the solid scene in haze of its own colour.
func weakHaze() *thing.World {
	world := smoking()
	world.Haze = thing.Haze{Density: 0.1, Colour: maths.Grey(0.25)}
	return world
}

// weakMirror is the solid scene on a glossy floor under a white sky.
func weakMirror() *thing.World {
	world := smoking()
	world.Things[0].Material = &thing.Material{Name: "glass",
		Colour: maths.Grey(0.1), Gloss: 1}
	world.Sky = even(maths.Grey(1))
	return world
}

// A ray that passes a ball and goes on to the floor behind it meets
// that floor with a weak field round the ball as it does with none.
// The ray's nearest point to the field's middle is short of the floor,
// so it has turned outward and left the field's march before it lands.
func TestARayLeavingAFieldStillMeetsTheFloor(t *testing.T) {
	past := maths.Ray{At: maths.V(3, 4, 6),
		Dir: maths.V(-0.2, -0.5, -1).Unit()}
	plain := lone(smoking(), past, 0)
	// lit: past half of a white matte surface's own, which is one over pi
	if plain.Cover != 1 || !(bright(plain) > 0.5/math.Pi) {
		t.Fatalf("with no field the ray should land on lit floor: %+v",
			plain)
	}
	world := smoking()
	world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	curved := lone(world, past, 0)
	if curved.Cover != 1 || !within(bright(curved), bright(plain), 1e-6) {
		t.Errorf("in a weak field the ray brings back %+v, with "+
			"none %+v", curved, plain)
	}
}

// Through a weak field a ball and the floor behind it are as far away
// as they are with no field: the march counts every piece of its way,
// the last piece onto a surface and the straight run out to the reach.
func TestDepthThroughAWeakFieldIsTheDistance(t *testing.T) {
	at := maths.Ray{At: maths.V(0, 1, 9), Dir: maths.V(0, 0, -1)}
	past := maths.Ray{At: maths.V(3, 4, 6),
		Dir: maths.V(-0.2, -0.5, -1).Unit()}
	beyond := maths.Ray{At: maths.V(0, 60, 0),
		Dir: maths.V(0, -1, 0.3).Unit()}
	world := smoking()
	world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
	for name, aim := range map[string]maths.Ray{"at the ball": at,
		"past the ball": past, "from beyond the reach": beyond} {
		smoke := smoking()
		want := float64(lone(smoke, aim, 0).Depth)
		got := float64(lone(world, aim, 0).Depth)
		if !(want > 1) || !within(got, want, 1e-3*want) {
			t.Errorf("%s the depth is %v in the field and %v with "+
				"none", name, got, want)
		}
	}
}

// A volume in front of a surface is gathered on the march's last step
// too. Through a thin glowing slab lying on a glowing floor, thin
// enough to sit inside that one step, a weak field gives what no field
// gives: the floor through the slab, and the slab's own glow.
func TestAFieldGathersTheVolumeBeforeASurface(t *testing.T) {
	const density, deep, glow = 70, 0.01, 0.3
	over := slab{density: density, box: maths.Box{
		Min: maths.V(-10, 0, -10), Max: maths.V(10, deep, 10)}}
	build := func(field bool) *thing.World {
		world := glowing()
		world.AddVolume(thing.Volume{Medium: over,
			Material: &thing.Material{Glow: maths.Grey(glow)}})
		if field {
			world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
		}
		return world
	}
	aim := maths.Ray{At: maths.V(0.5, 3, 0), Dir: maths.V(0, -1, 0)}
	kept := math.Exp(-density * deep)
	want := kept + glow*(1-kept)
	// a pixel is kept to seven figures, and the two marches add the
	// slab up in different pieces, so five places is asked
	plainWorld := build(false)
	plain := float64(lone(plainWorld, aim, 0).R)
	if !within(plain, want, 1e-5) {
		t.Fatalf("with no field the floor is %v, want %v", plain, want)
	}
	curvedWorld := build(true)
	curved := float64(lone(curvedWorld, aim, 0).R)
	if !within(curved, want, 1e-5) {
		t.Errorf("in a weak field the floor is %v, want %v", curved,
			want)
	}
}

// A look keeps its count of bounces through a field's march, so a
// mirror seen in a mirror is no more followed inside a weak field than
// outside one. The ray lands a unit from the field's middle, where the
// march is an ordinary one, and mirrors up toward the ball.
//
// With no ball the floor mirrors the sky by fresnel's share, the same
// with the field as without: so the check below cannot pass by the
// ray simply coming back black.
func TestAMirrorInAFieldIsStillNotFollowed(t *testing.T) {
	steep := maths.Ray{At: maths.V(1.3, 1, 0),
		Dir: maths.V(-0.3, -1, 0).Unit()}
	mirrored := func(ball, field bool) float64 {
		world := mirrors(ball)
		if field {
			world.AddField(thing.Field{Mass: 1e-6, Reach: 40})
		}
		return bright(lone(world, steep, 0))
	}
	open, openInField := mirrored(false, false), mirrored(false, true)
	if !(open > 0.03) || !within(openInField, open, 1e-6) {
		t.Fatalf("with no ball the floor mirrors %v in a weak field "+
			"and %v with none", openInField, open)
	}
	under, underInField := mirrored(true, false), mirrored(true, true)
	if under != 0 || underInField != 0 {
		t.Errorf("the ball in the floor is %v in a weak field and %v "+
			"with none: a second bounce was taken", underInField,
			under)
	}
}
