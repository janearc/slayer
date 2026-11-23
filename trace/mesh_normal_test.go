package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A smooth mesh is shaded by its smooth normal, turned against the ray. Where
// the eye sees the front of the face and that normal lies under the face, the
// normal is the other way over the face, leaned back to face the eye. The
// ground is such a mesh: at a ridge the normal a corner takes from the slope
// beside it is that of the ridge's other flank. A point the lamps are asked
// from is the lifted one, except where the smooth normal is opposed to the face
// the eye sees.

// The numbers of these tests' face and rays, each a choice of the test.
const (
	// ridgeSide is half the width of the square face.
	ridgeSide = 1.0
	// ridgeWide is half the width of the face that is wide enough for no ray
	// to leave it by its edge.
	ridgeWide = 50.0
	// ridgeFar is how far along a ray the tests look for the face.
	ridgeFar = 100.0
	// ridgeAside is how far off the face's middle the rays land, in x.
	ridgeAside = 0.3
	// ridgeHeight is how high the ray starts over the face's plane.
	ridgeHeight = 0.3
	// ridgeBack is how far before the face's middle the ray starts, in z: it
	// meets the plane a tenth of the face's half width past the middle.
	ridgeBack = -0.9
	// ridgeFall is how far the ray falls for each unit it runs: a ray
	// of about seventeen degrees below level.
	ridgeFall = 0.3
	// ridgeLean is how far a normal rises for each unit it leans along the
	// ray: it is over eighty degrees from straight up.
	ridgeLean = 0.1
	// ridgeNear is how far a normal that is on the far side of the face but
	// not opposed to it falls for each unit it leans toward the eye: about
	// a hundred and six degrees from straight up.
	ridgeNear = 0.3
	// ridgeUnder is how far a normal that is opposed to the face falls for
	// each unit it leans toward the eye: a hundred and thirty-five degrees
	// from straight up.
	ridgeUnder = 1.0
	// ridgeRounding is how far a normal's part may be from its exact value.
	ridgeRounding = 1e-12
	// ridgeFloor is how far under the face the black floor lies.
	ridgeFloor = 0.5
	// ridgeSkyShare is the least share of the sky's light a grey face lit
	// by the sky above it brings back; the sky is one and the face's colour
	// is a half, and a normal held on the eye's side lights it with over a
	// twentieth of that.
	ridgeSkyShare = 0.05
	// ridgeColour is the grey of the face.
	ridgeColour = 0.5
)

// ridgeHalves is the square's two halves, each wound to look up.
var ridgeHalves = [][3]int{{0, 1, 2}, {0, 2, 3}} // defined: by corner

// ridgeFaceOf is a square face of the half width given in the plane y = 0,
// looking up, whose smooth normal at every corner is lean.
func ridgeFaceOf(test *testing.T, side float64, lean maths.Vec) *mesh.Mesh {
	test.Helper()
	corners := []maths.Vec{maths.V(-side, 0, -side), maths.V(-side, 0, side),
		maths.V(side, 0, side), maths.V(side, 0, -side)}
	skin, err := mesh.New(corners, ridgeHalves,
		[]maths.Vec{lean, lean, lean, lean}, true)
	if err != nil {
		test.Fatal(err)
	}
	return skin
}

// ridgeFace is ridgeFaceOf at ridgeSide.
func ridgeFace(test *testing.T, lean maths.Vec) *mesh.Mesh {
	test.Helper()
	return ridgeFaceOf(test, ridgeSide, lean)
}

// ridgeMet is the normal the tracer shades a ray's hit on the mesh by.
func ridgeMet(test *testing.T, skin *mesh.Mesh, ray maths.Ray) maths.Vec {
	test.Helper()
	placed := placedThing{thing: &thing.Thing{Shape: skin,
		Material: &thing.Material{Name: "ridge",
			Colour: maths.Grey(ridgeColour)}},
		placement: placement{at: maths.NewSubspace(maths.Vec{},
			maths.Identity, 1), scale: 1}}
	along, landed, found := placed.reach(ray, 0, ridgeFar)
	if !found {
		test.Fatal("the ray missed the face")
	}
	met := hit{thing: placed, dist: along, on: landed}
	return met.normal(ray.Point(along), ray.Dir)
}

// ridgeRay is a ray from before the face, running along z and falling onto
// it from above.
func ridgeRay() maths.Ray {
	return maths.Ray{At: maths.V(ridgeAside, ridgeHeight, ridgeBack),
		Dir: maths.V(0, -ridgeFall, 1).Unit()}
}

// ridgeSame is whether two vectors are the same to a rounding.
func ridgeSame(got, want maths.Vec) bool {
	return within(got.X, want.X, ridgeRounding) &&
		within(got.Y, want.Y, ridgeRounding) &&
		within(got.Z, want.Z, ridgeRounding)
}

// TestANormalUnderTheFaceIsLeanedBackToFaceTheEye is the witness: a face
// looking up, seen from above, whose smooth normal points down under it and
// toward the eye is shaded by a normal over the face, on the eye's side and
// facing the eye; its y is over nought where the normal it was given has
// -0.71.
func TestANormalUnderTheFaceIsLeanedBackToFaceTheEye(test *testing.T) {
	skin := ridgeFace(test, maths.V(0, -ridgeUnder, -1))
	ray := ridgeRay()
	normal := ridgeMet(test, skin, ray)
	if !(normal.Y > 0) {
		test.Errorf("the normal %v points under the face", normal)
	}
	facing := -normal.Dot(ray.Dir)
	if !(facing > 0) || facing > leastFacing {
		test.Errorf("the normal faces the eye by %v, want over nought "+
			"and not over %v", facing, leastFacing)
	}
	if !within(normal.Len(), 1, ridgeRounding) {
		test.Errorf("the normal is %v long, want one", normal.Len())
	}
}

// TestANormalUnderTheFaceLightsTheSky is the same witness as light: a wide
// grey face under an even sky of one, over a black floor, whose smooth normal
// points down under it, brings back over a twentieth of the sky by its
// colour. A normal under the face lights nothing, since the face shuts out
// every ray it asks the sky along; the normal held over the face lights it
// with the sky above the face.
func TestANormalUnderTheFaceLightsTheSky(test *testing.T) {
	world := thing.NewWorld()
	world.Sky = even(maths.Grey(1))
	face := ridgeFaceOf(test, ridgeWide, maths.V(0, -ridgeUnder, -1))
	world.Add(thing.Thing{Shape: face,
		Material: &thing.Material{Name: "ridge",
			Colour: maths.Grey(ridgeColour)}})
	under := maths.NewSubspace(maths.V(0, -ridgeFloor, 0), maths.Identity, 1)
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "black floor"},
		Motion:   thing.Still(under)})
	got := lone(world, ridgeRay(), 0).Light()
	if !(got.R > ridgeColour*ridgeSkyShare) {
		test.Errorf("the face brought back %v of an even sky of one, "+
			"want over %v", got.R, ridgeColour*ridgeSkyShare)
	}
}

// TestAnOpposedNormalTurnedOverIsTheNormalOverTheFace is the anti-test for a
// normal opposed to the face that, turned against the ray, is over it: it is
// that normal, as for any shape.
func TestAnOpposedNormalTurnedOverIsTheNormalOverTheFace(test *testing.T) {
	lean := maths.V(0, -ridgeUnder, 1)
	normal := ridgeMet(test, ridgeFace(test, lean), ridgeRay())
	if want := lean.Neg().Unit(); !ridgeSame(normal, want) {
		test.Errorf("the normal is %v, want %v", normal, want)
	}
}

// TestANormalOnTheOtherSideOfTheFaceIsTurnedOver is the witness for a ridge
// whose corners lean the other way: a smooth normal opposite the face's is
// turned over, to the face's own way, the side the eye is on.
func TestANormalOnTheOtherSideOfTheFaceIsTurnedOver(test *testing.T) {
	face := ridgeFace(test, maths.V(0, -1, 0))
	normal := ridgeMet(test, face, ridgeRay())
	if !ridgeSame(normal, maths.V(0, 1, 0)) {
		test.Errorf("the normal is %v, want straight up", normal)
	}
}

// TestALeaningNormalIsLeanedBackToFaceTheEye is the witness for a ridge top:
// a face looking up, seen from above, whose smooth normal leans along the ray
// past the horizontal, is shaded by a normal over the face that faces the
// eye; its y is over nought where the normal turned against the ray has -0.1.
func TestALeaningNormalIsLeanedBackToFaceTheEye(test *testing.T) {
	ray := ridgeRay()
	face := ridgeFace(test, maths.V(0, ridgeLean, 1))
	normal := ridgeMet(test, face, ray)
	if !(normal.Y > 0) {
		test.Errorf("the normal %v points under the face", normal)
	}
	if facing := -normal.Dot(ray.Dir); !(facing > 0) || facing > leastFacing {
		test.Errorf("the normal faces the eye by %v, want over nought "+
			"and not over %v", facing, leastFacing)
	}
}

// TestANormalOverTheFaceIsTurnedAgainstTheRay is the anti-test: a smooth
// normal that, turned against the ray, is over the face, whether it faces the
// eye or leans past the face's edge-on away from it, is that normal and no
// other.
func TestANormalOverTheFaceIsTurnedAgainstTheRay(test *testing.T) {
	for _, lean := range []maths.Vec{maths.V(0, 1, -ridgeLean*2),
		maths.V(0, ridgeNear, -1), maths.V(0, -ridgeNear, 1)} {
		ray := ridgeRay()
		normal := ridgeMet(test, ridgeFace(test, lean), ray)
		want := lean.Unit()
		if want.Dot(ray.Dir) > 0 {
			want = want.Neg()
		}
		if !ridgeSame(normal, want) {
			test.Errorf("the normal for %v is %v, want %v", lean, normal,
				want)
		}
	}
}

// TestAFaceSeenFromBehindIsTurnedOver is the anti-test: a ray that comes up
// through the face meets its back, and the normal, whichever way its smooth
// normal points, is the one turned against the ray, even where the smooth
// normal is opposed to the face.
func TestAFaceSeenFromBehindIsTurnedOver(test *testing.T) {
	ray := maths.Ray{At: maths.V(ridgeAside, -ridgeHeight, ridgeBack),
		Dir: maths.V(0, ridgeFall, 1).Unit()}
	for _, lean := range []maths.Vec{maths.V(0, 1, 0), maths.V(0, -1, 0)} {
		normal := ridgeMet(test, ridgeFace(test, lean), ray)
		if want := maths.V(0, -1, 0); !ridgeSame(normal, want) {
			test.Errorf("the normal for %v is %v, want %v", lean, normal,
				want)
		}
	}
}

// TestAFlatMeshIsShadedByItsFace is the anti-test: a mesh shaded flat has no
// smooth normal, and its face's normal is the one the eye sees.
func TestAFlatMeshIsShadedByItsFace(test *testing.T) {
	corners := []maths.Vec{maths.V(-ridgeSide, 0, -ridgeSide),
		maths.V(-ridgeSide, 0, ridgeSide), maths.V(ridgeSide, 0, ridgeSide),
		maths.V(ridgeSide, 0, -ridgeSide)}
	skin, err := mesh.New(corners, ridgeHalves, nil, false)
	if err != nil {
		test.Fatal(err)
	}
	normal := ridgeMet(test, skin, ridgeRay())
	if want := maths.V(0, 1, 0); !ridgeSame(normal, want) {
		test.Errorf("the flat face's normal is %v, want straight up", normal)
	}
}
