package trace

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// A smooth face's normal is not its own. Where the eye grazes the face,
// the way mirrored about the smooth normal can point into the face, and
// on 733bdc72 the mirror then saw the face itself, or what lay under it.
// The way is now turned back above the face, and the mirror sees what is
// there: here a sky of one, over a black floor below the face.

// faceTilt is how far the smooth face's normal leans along the ray, as a
// share of its rise: about 1.7 degrees, as a round tube's leans by up to
// 1.9 between its 96 strips.
const faceTilt = 0.03 // picked: under a strip's lean, over the graze's

// faceGraze is how steeply the grazing ray falls onto the face, as a
// share of its run: about 2.9 degrees.
const faceGraze = 0.05 // picked: so the mirrored way dips 0.6 degrees

// faceSide is half the width of the square face, and faceAside how far
// off its diagonal the rays land.
const (
	faceSide  = 1   // picked: a square two across
	faceAside = 0.3 // picked: clear of the diagonal both faces share
)

// floorDrop is how far under the face the black floor lies.
const floorDrop = 0.5 // picked: anywhere under the face

// faceSky is the sky's light, the same every way.
const faceSky = 1 // picked: so the mirror brings back its own share

// faceShare is the least share of what it mirrors at that look that the
// grazing ray must bring back from the sky.
const faceShare = 0.5 // picked: on 733bdc72 it brought back nought

// faceRounding is how far a pixel's 32 bits may be from the exact light.
const faceRounding = 1e-6 // picked: seven figures

// faceSteep is the fall of the anti-test's ray, one in one: 45 degrees,
// and the rise of the face's leaning normal.
const faceSteep = 1 // defined: a run as long as the rise

// faceGloss and faceIndex are the face's finish: a whole mirror by
// fresnel's share for glass, as the glint tests' lacquer is.
const (
	faceGloss = 1   // picked: all of fresnel's share mirrored
	faceIndex = 1.5 // picked: glass
)

// facePlane is the height of the face's plane, and nought in any way
// that keeps to a plane of constant z.
const facePlane = 0 // defined: the plane y = 0

// faceMoment is when the scene is staged.
const faceMoment = 0 // defined: its start

// faceHalves is the square's two halves, each wound to look up.
var faceHalves = [][3]int{{0, 1, 2}, {0, 2, 3}} // defined: by corner

// glossBlack is the face's finish: no colour of its own, so all it
// shows is what it mirrors.
func glossBlack() *thing.Material {
	return &thing.Material{Name: "black gloss", Gloss: faceGloss,
		Index: faceIndex}
}

// tiltedFace is a square face in the plane y = 0, looking up, shaded
// smooth by a normal leaning toward +x at every corner, gloss black over
// a black floor and under an even sky.
func tiltedFace(test *testing.T) *thing.World {
	test.Helper()
	corners := []maths.Vec{maths.V(-faceSide, facePlane, -faceSide),
		maths.V(-faceSide, facePlane, faceSide),
		maths.V(faceSide, facePlane, faceSide),
		maths.V(faceSide, facePlane, -faceSide)}
	lean := maths.V(faceTilt, faceSteep, facePlane)
	normals := []maths.Vec{lean, lean, lean, lean}
	skin, err := mesh.New(corners, faceHalves, normals, true)
	if err != nil {
		test.Fatal(err)
	}
	world := thing.NewWorld()
	world.Sky = even(maths.Grey(faceSky))
	world.Add(thing.Thing{Shape: skin, Material: glossBlack()})
	under := maths.NewSubspace(maths.V(facePlane, -floorDrop, facePlane),
		maths.Identity, faceSteep)
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Name: "black floor"},
		Motion:   thing.Still(under)})
	return world
}

// mirrorsBack is what the face mirrors along way, times the sky: what a
// ray that mirrors off it into the sky brings back.
func mirrorsBack(way maths.Vec) float64 {
	normal := maths.V(faceTilt, faceSteep, facePlane).Unit()
	return faceSky * glossBlack().Reflects(-normal.Dot(way)).R
}

// TestAGrazingMirrorStaysAboveItsFace is the witness: a ray falling at
// 2.9 degrees onto a face whose smooth normal leans 1.7 degrees along it
// mirrors off below the face. Kept above it, the mirror brings back the
// sky; on 733bdc72 it brought back the black under the face.
func TestAGrazingMirrorStaysAboveItsFace(test *testing.T) {
	way := maths.V(faceSide, -faceGraze, facePlane).Unit()
	ray := maths.Ray{At: maths.V(-faceSide, faceGraze, faceAside), Dir: way}
	want := mirrorsBack(way)
	got := lone(tiltedFace(test), ray, faceMoment).Light()
	if !(got.R > want*faceShare) {
		test.Errorf("a grazing mirror brought back %v, want near %v",
			got.R, want)
	}
}

// TestASteepMirrorIsAsItWas is the anti-test: a ray at 45 degrees onto
// the same face mirrors off well above it, and brings back the sky by
// the share it mirrors, as it always did.
func TestASteepMirrorIsAsItWas(test *testing.T) {
	way := maths.V(faceSteep, -faceSteep, facePlane).Unit()
	ray := maths.Ray{At: maths.V(-faceSide, faceSide, faceAside), Dir: way}
	want := mirrorsBack(way)
	got := lone(tiltedFace(test), ray, faceMoment).Light()
	if !within(got.R, want, want*faceRounding) {
		test.Errorf("a steep mirror brought back %v, want %v", got.R,
			want)
	}
}
