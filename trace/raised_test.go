package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// The point the lamps and the sky are asked from on a smooth mesh is the
// lifted point, the one the corners' tangent planes give, except where the
// smooth normal is opposed to the face the eye sees, a hundred and twenty
// degrees or more from it: there it is the point the ray met.

// The numbers of these tests' floor, each a choice of the test.
const (
	// raiseRun is how far the floor's corner normals lean along x, against
	// one of rise: forty-five degrees, on the floor's upper side, so the
	// floor's tangent planes leave its far side well under them and the
	// lift has a length to be seen.
	raiseRun = 1.0
	// raiseSeen is the least a lift is to carry the point, for the test to
	// say it is a lift and not rounding.
	raiseSeen = 1e-3
)

// raiseWorld is a world of one square floor looking up in the plane y = 0,
// shaded smooth, whose corners' normals are all lean, or shaded flat when
// lean is the zero vector.
func raiseWorld(test *testing.T, lean maths.Vec) *thing.World {
	test.Helper()
	corners := []maths.Vec{maths.V(-ridgeSide, 0, -ridgeSide),
		maths.V(-ridgeSide, 0, ridgeSide), maths.V(ridgeSide, 0, ridgeSide),
		maths.V(ridgeSide, 0, -ridgeSide)}
	var normals []maths.Vec
	if lean != (maths.Vec{}) {
		normals = []maths.Vec{lean, lean, lean, lean}
	}
	skin, err := mesh.New(corners, ridgeHalves, normals, lean != (maths.Vec{}))
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky = even(maths.Grey(1))
	world.Add(thing.Thing{Shape: skin, Material: &thing.Material{
		Name: "ground", Colour: maths.Grey(ridgeColour)}})
	return world
}

// raisedAt is the point the lamps are asked from for the ridge ray that meets
// the floor in a world, the point it met, and the point lifted raises it to.
func raisedAt(test *testing.T, world *thing.World) (raised, met,
	lifted maths.Vec) {
	test.Helper()
	scene := stage(world, 0)
	ray := ridgeRay()
	seen, found := scene.crowd.first(ray, 0)
	if !found {
		test.Fatal("the ray missed the floor")
	}
	point := ray.Point(seen.dist)
	return seen.raised(point, ray.Dir), point, seen.lifted(point)
}

// TestAPointWhoseSmoothNormalIsOpposedIsNotLifted is the witness: a floor
// seen from above whose corners' normals point down under it has no lift, and
// the point the lamps are asked from is the point met, although the corners'
// tangent planes would carry it a long way.
func TestAPointWhoseSmoothNormalIsOpposedIsNotLifted(test *testing.T) {
	under := maths.V(0, -ridgeUnder, -1)
	raised, met, lifted := raisedAt(test, raiseWorld(test, under))
	if !(lifted.Sub(met).Len() > raiseSeen) {
		test.Fatalf("the floor lifts the point by %v, want over %v",
			lifted.Sub(met).Len(), raiseSeen)
	}
	if raised != met {
		test.Errorf("the point was %v, want the point met %v", raised, met)
	}
}

// TestAPointWhoseSmoothNormalIsNotOpposedIsLifted is the anti-test: a floor
// whose corners' normals lean up, or lean down under it by less than a hundred
// and twenty degrees from it, is lifted, and the point the lamps are asked
// from is the lifted one, to the last bit.
func TestAPointWhoseSmoothNormalIsNotOpposedIsLifted(test *testing.T) {
	for _, lean := range []maths.Vec{maths.V(raiseRun, 1, 0).Unit(),
		maths.V(0, -ridgeNear, -1)} {
		raised, met, lifted := raisedAt(test, raiseWorld(test, lean))
		if !(lifted.Sub(met).Len() > raiseSeen) {
			test.Fatalf("the floor lifts the point by %v, want over %v",
				lifted.Sub(met).Len(), raiseSeen)
		}
		if raised != lifted {
			test.Errorf("the point for %v was %v, want the lifted %v",
				lean, raised, lifted)
		}
	}
}

// TestAPointOnAFloorShadedFlatIsTheOneMet is the anti-test: a mesh shaded flat
// has no lift, and the point is the point met.
func TestAPointOnAFloorShadedFlatIsTheOneMet(test *testing.T) {
	raised, met, _ := raisedAt(test, raiseWorld(test, maths.Vec{}))
	if raised != met {
		test.Errorf("the point was %v, want the point met %v", raised, met)
	}
}
