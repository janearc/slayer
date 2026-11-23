package mesh

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files of this file's fixtures, each row named in its first
// column.
const (
	meetCornersFile      = "testdata/meet-corners.csv"
	meetPointsFile       = "testdata/meet-points.csv"
	meetHeightsFile      = "testdata/meet-heights.csv"
	meetRaysFile         = "testdata/meet-rays.csv"
	meetPlaneCornersFile = "testdata/meet-plane-corners.csv"
	meetPlaneRayFile     = "testdata/meet-plane-ray.csv"
)

// meetHitName is the row of the rays file whose ray meets the mesh.
const meetHitName = "hit"

// meetHalvings is the name of the number of times the ball is halved,
// in numbers-meet.yaml.
const meetHalvings = "meet-ball-halvings"

// meetArea is twice the signed area of the triangle that two corners
// make with the point (x, y), in the plane z=0.
func meetArea(one, two maths.Vec, x, y float64) float64 {
	return (two.X-one.X)*(y-one.Y) - (two.Y-one.Y)*(x-one.X)
}

// barycentric is the weights of a point over a triangle's corners in
// the plane z=0, by the areas of the three triangles it makes with
// them: each corner's weight is the area opposite it, over the whole.
func barycentric(corners [triangleCorners]maths.Vec, x, y float64) weights {
	whole := meetArea(corners[first], corners[second], corners[third].X,
		corners[third].Y)
	return weights{
		meetArea(corners[second], corners[third], x, y) / whole,
		meetArea(corners[third], corners[first], x, y) / whole,
		meetArea(corners[first], corners[second], x, y) / whole}
}

// a ray down onto a face, or up from under it, meets it where it is
// over it, in its middle, on a side exactly, and on a corner exactly,
// with the corners given in each turn and the face given both ways
// round, and not where it is beside it, and the weights are the point's
// own over the corners. the face is (0,0), (2,0), (0,2) in the plane z=0.
func TestARayMeetsAFaceWhereItIsOverIt(t *testing.T) {
	corners := vectorsIn(t, meetCornersFile)
	for turn := range triangleCorners {
		var turned [triangleCorners]maths.Vec
		for place := range turned {
			turned[place] = corners[(place+turn)%triangleCorners]
		}
		for _, backward := range []bool{false, true} {
			face := theOneFace()[nought]
			if backward {
				face[second], face[third] = face[third], face[second]
			}
			made := mustMesh(t, turned[:],
				[][triangleCorners]int{face}, nil, false)
			meetFromBothSides(t, made, turned, face)
		}
	}
}

// meetFromBothSides holds a ray from above and a ray from below, at
// each point of the points file, to meeting the face or not.
func meetFromBothSides(t *testing.T, made *Mesh,
	turned [triangleCorners]maths.Vec, face [triangleCorners]int) {
	ordered := [triangleCorners]maths.Vec{turned[face[first]],
		turned[face[second]], turned[face[third]]}
	for _, side := range read(t, meetHeightsFile) {
		for _, each := range read(t, meetPointsFile) {
			x, y := float(t, each, "x"), float(t, each, "y")
			checkMeets(t, made, float(t, side, "height"), x, y,
				whole(t, each, "met") == unity, barycentric(ordered, x, y),
				face)
		}
	}
}

// checkMeets holds one ray to meeting a face or not, one away, with the
// weights over the corners in the order of the mesh's face.
func checkMeets(t *testing.T, made *Mesh, height, x, y float64, met bool,
	over weights, face [triangleCorners]int) {
	t.Helper()
	ray := maths.NewRay(maths.V(x, y, height),
		maths.V(nought, nought, -height))
	look := aimed(ray)
	along, got, ok := look.meets(made, nought, nought,
		math.Inf(plusInfinity))
	if ok != met {
		t.Errorf("from %v at (%v, %v) face %v: met %v", height, x, y, face,
			ok)
		return
	}
	if ok && (along != unity/math.Abs(height) ||
		!sameWeights(t, got, over)) {
		t.Errorf("from %v at (%v, %v): along %v weights %v, not %v",
			height, x, y, along, got, over)
	}
}

// meetSphere draws points and directions at random.
type meetSphere struct{ random *rand.Rand }

// unit is a point on a sphere of the given radius about the origin, as
// a random direction scaled.
func (sphere meetSphere) unit(scale float64) maths.Vec {
	direction := maths.V(sphere.random.NormFloat64(),
		sphere.random.NormFloat64(), sphere.random.NormFloat64())
	return oneLong(direction).Scale(scale)
}

// inside is the given reach scaled by the root of a random share, so
// that points at that distance fill a ball evenly.
func (sphere meetSphere) inside(reach float64) float64 {
	return math.Sqrt(sphere.random.Float64()) * reach
}

// meetTally is what the walks of many rays have cost: the most nodes
// and faces for one ray, and the nodes and faces for all.
type meetTally struct{ most, tried, total, all int }

// count walks one ray through the mesh and adds what it cost.
func (tally *meetTally) count(made *Mesh, ray maths.Ray) {
	walking := meetWalk{made: made, ray: ray, near: nought,
		far: math.Inf(plusInfinity)}
	nodes, faces := work(walking.run)
	tally.most, tally.tried = max(tally.most, nodes), max(tally.tried, faces)
	tally.total += nodes
	tally.all += faces
}

// over is whether any of what it cost is more than the limit's.
func (tally meetTally) over(limit meetTally) bool {
	return tally.most > limit.most || tally.tried > limit.tried ||
		tally.total > limit.total || tally.all > limit.all
}

// meetWalk is a ray to be met against a mesh between two distances.
type meetWalk struct {
	made      *Mesh
	ray       maths.Ray
	near, far float64
}

// run meets the ray and drops what it finds.
func (walk meetWalk) run() {
	walk.made.Meet(walk.ray, walk.near, walk.far)
}

// meetWalkOf is the walk of the ray in a row of the rays file, between
// the row's near and far.
func meetWalkOf(t *testing.T, made *Mesh, row map[string]string) meetWalk {
	at := maths.V(float(t, row, "atx"), float(t, row, "aty"),
		float(t, row, "atz"))
	toward := maths.V(float(t, row, "dirx"), float(t, row, "diry"),
		float(t, row, "dirz"))
	ray := maths.NewRay(at, toward)
	return meetWalk{made: made, ray: ray, near: float(t, row, "near"),
		far: float(t, row, "far")}
}

// meetOutsideBounds is the most the rays from outside may cost.
func meetOutsideBounds(t *testing.T) meetTally {
	return meetTally{most: pickedCount(t, "meet-outside-most"),
		tried: pickedCount(t, "meet-outside-tried"),
		total: pickedCount(t, "meet-outside-total"),
		all:   pickedCount(t, "meet-outside-all")}
}

// meetInsideBounds is the most the rays from inside may cost.
func meetInsideBounds(t *testing.T) meetTally {
	return meetTally{most: pickedCount(t, "meet-inside-most"),
		tried: pickedCount(t, "meet-inside-tried"),
		total: pickedCount(t, "meet-inside-total"),
		all:   pickedCount(t, "meet-inside-all")}
}

// meetFromOutside is what the walks of the rays cost that start outside
// the ball and are aimed at points within a share of its middle.
func meetFromOutside(t *testing.T, made *Mesh, sphere meetSphere) meetTally {
	var tally meetTally
	for range pickedCount(t, "meet-rays") {
		from := sphere.unit(picked(t, "meet-outside-radius"))
		reach := sphere.inside(picked(t, "meet-target-reach"))
		toward := oneLong(sphere.unit(reach).Sub(from))
		tally.count(made, maths.NewRay(from, toward))
	}
	return tally
}

// meetFromInside is what the walks of the rays cost that start inside
// the ball, in any direction.
func meetFromInside(t *testing.T, made *Mesh, sphere meetSphere) meetTally {
	var tally meetTally
	for range pickedCount(t, "meet-rays") {
		reach := sphere.inside(picked(t, "meet-inside-reach"))
		at := sphere.unit(reach)
		toward := sphere.unit(unity)
		tally.count(made, maths.NewRay(at, toward))
	}
	return tally
}

// a ray that misses the mesh's box does no work; one that meets it does
// little: a thousand rays from outside toward points within a unit
// and a fifth of the middle of a ball of 1280 faces and 1469 nodes, and
// a thousand from inside it, in any direction. measured, the most for
// one ray is 47 nodes and 21 faces from outside, and 36 and 9 from
// inside, and the thousand cost 15138 nodes and 3531 faces from outside
// and 19934 and 3384 from inside. the bounds are those and a twentieth.
// the ball's corners and the rays' directions are made one long by
// oneLong, so the counts are the same on every chip.
func TestTheWalkOfARayEntersFewNodesAndTriesFewFaces(t *testing.T) {
	made, _, _ := ball(t, pickedCount(t, meetHalvings))
	seedFirst := uint64(pickedCount(t, "meet-seed-first"))
	seedSecond := uint64(pickedCount(t, "meet-seed-second"))
	sphere := meetSphere{random: rand.New(rand.NewPCG(seedFirst, seedSecond))}
	outside := meetFromOutside(t, made, sphere)
	if outside.over(meetOutsideBounds(t)) {
		t.Errorf("from outside: %d, %d for one, %d, %d in all",
			outside.most, outside.tried, outside.total, outside.all)
	}
	inside := meetFromInside(t, made, sphere)
	if inside.over(meetInsideBounds(t)) {
		t.Errorf("from inside: %d, %d for one, %d, %d in all",
			inside.most, inside.tried, inside.total, inside.all)
	}
}

// meetMiss is a walk that must meet nothing.
type meetMiss struct {
	walk meetWalk
	test *testing.T
	name string
}

// run meets the ray and says so if it meets the mesh.
func (miss meetMiss) run() {
	walk := miss.walk
	if _, ok := walk.made.Meet(walk.ray, walk.near, walk.far); ok {
		miss.test.Errorf("%s: met", miss.name)
	}
}

// meetCheckMiss holds the ray in a row to entering no node and trying
// no face.
func meetCheckMiss(t *testing.T, made *Mesh, row map[string]string) {
	miss := meetMiss{walk: meetWalkOf(t, made, row), test: t,
		name: row["name"]}
	nodes, faces := work(miss.run)
	if nodes != nought || faces != nought {
		t.Errorf("%s: %d nodes and %d faces", miss.name, nodes, faces)
	}
}

// a ray that does not reach the mesh's box, because it is beside it,
// behind it, or stopped short, enters no node and tries no face.
func TestARayThatMissesTheBoxDoesNoWork(t *testing.T) {
	made, _, _ := ball(t, pickedCount(t, meetHalvings))
	for _, row := range read(t, meetRaysFile) {
		if row["name"] != meetHitName {
			meetCheckMiss(t, made, row)
			continue
		}
		nodes, faces := work(meetWalkOf(t, made, row).run)
		if nodes == nought || faces == nought {
			t.Errorf("a ray that hits entered %d nodes and tried %d faces",
				nodes, faces)
		}
	}
}

// a ray that runs along the plane of a face, down through the face's
// own plane, sees it edge on: its three shares are all nought, and it
// is not met.
func TestARayAlongAFacesPlaneDoesNotMeetIt(t *testing.T) {
	corners := vectorsIn(t, meetPlaneCornersFile)
	made := mustMesh(t, corners, theOneFace(), nil, false)
	along := meetWalkOf(t, made, read(t, meetPlaneRayFile)[nought])
	look := aimed(along.ray)
	reach := math.Inf(plusInfinity)
	if _, _, ok := look.meets(made, nought, nought, reach); ok {
		t.Error("a ray along a face's plane met the face")
	}
}
