package manufacture

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/geom"
	"github.com/janearc/slayer/maths"
)

// turned is a point of ninja's space, where up is z, in the world's, where up
// is y: a quarter turn about x. it is a turn and not a mirror, so a face
// wound outward there is wound outward here.
func turned(point geom.Vec3) maths.Vec {
	return maths.V(point[0], point[2], -point[1])
}

// back is turned, undone.
func back(point maths.Vec) geom.Vec3 {
	return geom.Vec3{point.X, -point.Z, point.Y}
}

// close is how near two corners must be to be the same corner: ninja's tube
// works its last ring corner out at a full turn, where the sine is not quite
// zero, and the tube here uses the first corner again.
const close = 1e-12

// areaOf is the area of every face of a surface together.
func areaOf(skin surface) float64 {
	var area float64
	for _, face := range skin {
		twice := face[1].Sub(face[0]).Cross(face[2].Sub(face[0]))
		area += twice.Len() / 2
	}
	return area
}

// volume is how much a closed surface holds, in the cube of its
// space's unit: the cones from one of its own corners to each face,
// added up. it lives with the tests, which are all that ask it since
// Volume was cut from what a made thing answers: it is how a tube is
// held to the frustum's formula and a stone to its ellipsoid's.
func (skin surface) volume() float64 {
	if len(skin) == 0 {
		return 0
	}
	from := skin[0][0]
	var sixfold float64
	for _, one := range skin {
		first, second := one[0].Sub(from), one[1].Sub(from)
		sixfold += first.Dot(second.Cross(one[2].Sub(from)))
	}
	return sixfold / tetrahedronsInABox
}

// tetrahedronsInABox is how many tetrahedrons on three edges fill the
// box on the same three: the one sixth in a cone's volume.
const tetrahedronsInABox = 6

func mustTube(t *testing.T, length, foot, tip float64) tube {
	t.Helper()
	made, err := newTube(measure.Metres(length), measure.Metres(foot),
		measure.Metres(tip))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func mustCut(t *testing.T, around int) cut {
	t.Helper()
	fine, err := cutInto(around)
	if err != nil {
		t.Fatal(err)
	}
	return fine
}

// the truth for a tube's wall is ninja's orientedTube: the same faces
// in the same order, wherever the tube runs.
func TestATubesWallIsNinjasOrientedTube(t *testing.T) {
	type at = geom.Vec3
	runs := []struct {
		name      string
		from, to  geom.Vec3
		foot, tip float64
		around    int
	}{
		{"upright", at{0, 0, 0}, at{0, 0, 3}, 0.4, 0.3, 8},
		{"a cone", at{1, 2, 3}, at{1, 2, 5}, 0.5, 0, 5},
		{"leaning", at{1, -2, 0.5}, at{2, 1, 4}, 0.2, 0.1, 6},
		{"drooping", at{0, 0, 9}, at{2, 1, 8}, 0.3, 0.2, 5},
		{"nearly along x", at{0, 0, 1}, at{5, 0.2, 1.1}, 0.25, 0.25, 7},
		{"along x", at{-1, 0, 0}, at{4, 0, 0}, 0.1, 0.3, 3},
		// 0.92 of this run lies along x: just past where the side
		// changes
		{"past the change", at{0, 0, 1}, at{4.6, 1.5, 2.2}, 0.2, 0.15,
			6},
	}
	for _, run := range runs {
		want := orientedTube(run.from, run.to, run.foot, run.tip,
			run.around, barkMaterial)
		length := run.to.Sub(run.from).Length()
		space, err := spaceAlong(turned(run.from),
			turned(run.to).Sub(turned(run.from)))
		if err != nil {
			t.Fatalf("%s: %v", run.name, err)
		}
		made := mustTube(t, length, run.foot, run.tip)
		got := made.wall(mustCut(t, run.around)).placed(space)
		if differs := unlike(got, want); differs != "" {
			t.Fatalf("%s: %s", run.name, differs)
		}
	}
}

// unlike is how a surface differs from ninja's triangles, face for
// face in their order, or nothing when it does not.
func unlike(got surface, want []geom.Triangle) string {
	if len(got) != len(want) {
		return fmt.Sprintf("%d faces, ninja's has %d", len(got),
			len(want))
	}
	for index, face := range got {
		for corner, point := range face {
			truth := turned(want[index].V[corner])
			if !point.Near(truth, close) {
				return fmt.Sprintf("face %d corner %d is %v, "+
					"ninja's is %v", index, corner, point,
					truth)
			}
		}
	}
	return ""
}

// the judge of a closed surface is slayer's mesh builder (mesh.New), which
// refuses a face with no area, neighbours wound against each other and
// a closed piece wound inside out.
func TestATubeIsClosedAndWoundOutward(t *testing.T) {
	shapes := map[string]tube{
		"a trunk": mustTube(t, 10, 0.32, 0.1),
		"a cone":  mustTube(t, 2, 0.5, 0),
		"a can":   mustTube(t, 0.122, 0.033, 0.033),
		"a flare": mustTube(t, 1, 0.1, 0.4),
	}
	for name, made := range shapes {
		for _, around := range []int{3, 8, 31} {
			skin := made.skin(mustCut(t, around))
			if flaw := unsound(skin); flaw != "" {
				t.Errorf("%s, %d round: %s", name, around, flaw)
			}
		}
	}
}

// unsound is what is wrong with a surface that should be closed, wound
// outward and holding something, or nothing when it is sound.
func unsound(skin surface) string {
	mesh, err := skin.meshed()
	if err != nil {
		return err.Error()
	}
	if !mesh.Closed() {
		return "not closed"
	}
	if skin.volume() <= 0 {
		return fmt.Sprintf("it holds %v", skin.volume())
	}
	return ""
}

// a tube bulges nowhere inward, so every face of it looks away from a
// point inside it. a cap fanned about anywhere but the middle of its
// ring still closes and still holds as much, and only this notices.
func TestEveryFaceOfATubeLooksAwayFromItsMiddle(t *testing.T) {
	shapes := map[string]tube{
		"a trunk": mustTube(t, 10, 0.32, 0.1),
		"a cone":  mustTube(t, 2, 0.5, 0),
		"a flare": mustTube(t, 1, 0.1, 0.4),
	}
	for name, made := range shapes {
		middle := maths.V(0, inScene(made.length)/2, 0)
		skin := made.skin(mustCut(t, 7))
		if index := inward(skin, middle); index >= 0 {
			t.Errorf("%s: face %d looks inward", name, index)
		}
	}
}

// inward is the first face of a surface that does not look away from
// a point, or minus one when every face does.
func inward(skin surface, from maths.Vec) int {
	for index, face := range skin {
		out := face[1].Sub(face[0]).Cross(face[2].Sub(face[0]))
		thrice := face[0].Add(face[1]).Add(face[2])
		if out.Dot(thrice.Sub(from.Scale(3))) <= 0 {
			return index
		}
	}
	return -1
}

// the mesh is given three corners a face and no more, with flat
// faces: a corner of its own would move the mesh's bounds, and the gpu
// refuses a smooth mesh.
func TestASurfaceMeshedIsItsFacesAndNoMore(t *testing.T) {
	skin := mustTube(t, 1, 0.5, 0.25).skin(mustCut(t, 5))
	mesh, err := skin.meshed()
	if err != nil {
		t.Fatal(err)
	}
	corners, faces := len(mesh.Points()), mesh.FaceCount()
	if corners != 3*len(skin) || faces != len(skin) {
		t.Errorf("%d corners and %d faces for a skin of %d", corners,
			faces, len(skin))
	}
	if mesh.Smooth() {
		t.Error("the mesh is smooth")
	}
}

func TestASurfaceOfNoFacesHoldsNothing(t *testing.T) {
	if held := (surface{}).volume(); held != 0 {
		t.Errorf("no faces hold %v", held)
	}
}

// the truth for a tube's volume is the frustum's formula. with flat
// sides the rings are polygons, of area n/2 sin(2 pi/n) r squared, so
// the cut tube is exactly that share of the round one, and nears it as
// the cut gets finer.
func TestATubesVolumeIsTheFrustums(t *testing.T) {
	length, foot, tip := 3.0, 0.5, 0.2
	round := math.Pi * length * (foot*foot + foot*tip + tip*tip) / 3
	made := mustTube(t, length, foot, tip)
	short := math.Inf(1)
	for _, around := range []int{3, 6, 12, 24, 48} {
		sides := float64(around)
		share := sides / 2 * math.Sin(2*math.Pi/sides) / math.Pi
		got := made.skin(mustCut(t, around)).volume()
		if math.Abs(got-round*share) > 1e-12 {
			t.Errorf("%d around: volume %v, a frustum of %d sides "+
				"holds %v", around, got, around, round*share)
		}
		if round-got >= short || got >= round {
			t.Errorf("%d around: %v short of the round frustum, "+
				"no nearer than the coarser cut's %v", around,
				round-got, short)
		}
		short = round - got
	}
}

// a blade is a flat tube: one strip, lying in one plane and holding
// nothing, which a ray meets from either side.
func TestABladeIsOneStrip(t *testing.T) {
	blades := []struct {
		name              string
		length, foot, tip float64
		faces             int
	}{
		{"a leaflet", 0.04, 0.008, 0.002, 2},
		{"a blade of grass", 0.25, 0.0075, 0, 1},
	}
	for _, blade := range blades {
		made, err := newBlade(measure.Metres(blade.length),
			measure.Metres(blade.foot), measure.Metres(blade.tip))
		if err != nil {
			t.Fatal(err)
		}
		// however finely it is asked to be cut, it has two sides
		skin := made.skin(mustCut(t, 12))
		if len(skin) != blade.faces {
			t.Fatalf("%s: %d faces, not %d", blade.name, len(skin),
				blade.faces)
		}
		for index, face := range skin {
			if face[0].Z != 0 || face[1].Z != 0 || face[2].Z != 0 {
				t.Fatalf("%s: face %d leaves the plane: %v",
					blade.name, index, face)
			}
		}
		if skin.volume() != 0 {
			t.Errorf("%s: holds %v", blade.name, skin.volume())
		}
		strip := (blade.foot + blade.tip) * blade.length
		if math.Abs(areaOf(skin)-strip) > 1e-15 {
			t.Errorf("%s: area %v, the strip is %v", blade.name,
				areaOf(skin), strip)
		}
		// the mesh takes it wherever it is put, not only lying flat
		space, err := spaceAlong(maths.V(3, 1, 2), maths.V(1, 2, -3))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := skin.placed(space).meshed(); err != nil {
			t.Errorf("%s: %v", blade.name, err)
		}
	}
}

func TestATubeThatCannotBeIsRefused(t *testing.T) {
	nan, endless := math.NaN(), math.Inf(1)
	cases := []struct {
		name              string
		length, foot, tip float64
		want              error
	}{
		{"no length", 0, 1, 1, errTubeLength},
		{"a length below nought", -2, 1, 1, errTubeLength},
		{"a length that is no number", nan, 1, 1, errTubeLength},
		{"an endless length", endless, 1, 1, errTubeLength},
		{"no foot", 1, 0, 1, errTubeFoot},
		{"a foot below nought", 1, -1, 1, errTubeFoot},
		{"a foot that is no number", 1, nan, 1, errTubeFoot},
		{"an endless foot", 1, endless, 1, errTubeFoot},
		{"a tip below nought", 1, 1, -0.1, errTubeTip},
		{"a tip that is no number", 1, 1, nan, errTubeTip},
		{"an endless tip", 1, 1, endless, errTubeTip},
	}
	for _, refused := range cases {
		doors := map[string]func(length, foot,
			tip measure.Length) (tube, error){
			"tube": newTube, "blade": newBlade}
		for door, open := range doors {
			_, err := open(measure.Metres(refused.length),
				measure.Metres(refused.foot),
				measure.Metres(refused.tip))
			if !errors.Is(err, refused.want) {
				t.Errorf("a %s with %s: %v, not %v", door,
					refused.name, err, refused.want)
			}
		}
	}
	if _, err := newTube(measure.Metres(1), measure.Metres(1),
		measure.Metres(0)); err != nil {
		t.Errorf("a cone was refused: %v", err)
	}
}

func TestACutTooCoarseOrTooFineIsRefused(t *testing.T) {
	for _, around := range []int{-1, 0, 2, mostAround + 1} {
		if _, err := cutInto(around); !errors.Is(err, errCut) {
			t.Errorf("a cut of %d around: %v, not %v", around, err,
				errCut)
		}
	}
	for _, around := range []int{3, mostAround} {
		if _, err := cutInto(around); err != nil {
			t.Errorf("a cut of %d around was refused: %v", around,
				err)
		}
	}
}

// spaceAlong's axes are one unit long, square to each other and turn the
// right way, whichever way it runs; and it runs somewhere.
func TestSpaceAlongIsSquareAndRightHanded(t *testing.T) {
	runs := []maths.Vec{maths.V(0, 1, 0), maths.V(0, -3, 0),
		maths.V(1, 0, 0), maths.V(-5, 0.1, 0.2), maths.V(1, 1, 1),
		maths.V(0, 0, -2), maths.V(0.9, 0.3, -0.3)}
	for _, run := range runs {
		space, err := spaceAlong(maths.V(1, 2, 3), run)
		if err != nil {
			t.Fatalf("along %v: %v", run, err)
		}
		axes := space.Orientation
		if !axes.Y.Near(run.Unit(), close) {
			t.Errorf("along %v: y is %v", run, axes.Y)
		}
		if math.Abs(axes.X.Len()-1) > close ||
			math.Abs(axes.X.Dot(axes.Y)) > close {
			t.Errorf("along %v: x is %v, not a unit square to y",
				run, axes.X)
		}
		if !axes.X.Cross(axes.Y).Near(axes.Z, close) {
			t.Errorf("along %v: z is %v, not x cross y", run,
				axes.Z)
		}
		if space.Origin != maths.V(1, 2, 3) {
			t.Errorf("along %v: starts at %v", run, space.Origin)
		}
	}
	for _, nowhere := range []maths.Vec{{}, maths.V(math.NaN(), 0, 0),
		maths.V(0, math.Inf(1), 0)} {
		if _, err := spaceAlong(maths.Vec{}, nowhere); !errors.Is(err,
			errNoRun) {
			t.Errorf("along %v: %v, not %v", nowhere, err, errNoRun)
		}
	}
}

// back is used by later tests; this keeps it honest.
func TestBackUndoesTurned(t *testing.T) {
	point := geom.Vec3{1, 2, 3}
	if back(turned(point)) != point {
		t.Errorf("back of turned %v is %v", point, back(turned(point)))
	}
}
