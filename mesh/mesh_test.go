package mesh

import (
	"errors"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The csv files the mesh tests read.
const (
	meshSlipStartsFile    = "testdata/mesh-slip-starts.csv"
	meshAxisNormalsFile   = "testdata/mesh-axis-normals.csv"
	meshSmoothCasesFile   = "testdata/mesh-smooth-cases.csv"
	meshBlendWeightsFile  = "testdata/mesh-blend-weights.csv"
	meshBoundsCasesFile   = "testdata/mesh-bounds-cases.csv"
	meshTetrahedronPoints = "testdata/mesh-tetrahedron-corners.csv"
	meshTetrahedronFaces  = "testdata/mesh-tetrahedron-faces.csv"
	meshTwoFacesFile      = "testdata/mesh-two-faces.csv"
)

// meshThird is the weight of each corner of a face at its middle.
const meshThird = 1.0 / 3 // defined: a face's middle weighs its corners alike

// meshRandom is a random stream from two seeds, each by its name in the
// numbers file.
func meshRandom(test testing.TB, firstName, secondName string) *rand.Rand {
	test.Helper()
	firstSeed := pickedCount(test, firstName)
	secondSeed := pickedCount(test, secondName)
	source := rand.NewPCG(uint64(firstSeed), uint64(secondSeed))
	return rand.New(source)
}

// meshNormalPoint is a point whose three parts are each drawn from the
// normal distribution, x first.
func meshNormalPoint(random *rand.Rand) maths.Vec {
	return maths.V(random.NormFloat64(), random.NormFloat64(),
		random.NormFloat64())
}

// meshBoxPoint is a point whose three parts are each drawn from half a
// unit either side of nought, x first.
func meshBoxPoint(random *rand.Rand) maths.Vec {
	return maths.V(random.Float64()-boxHalf, random.Float64()-boxHalf,
		random.Float64()-boxHalf)
}

// meshBoxCorner is a corner of a box in a row: the columns that start
// with a prefix and end in x, y and z.
func meshBoxCorner(test testing.TB, row map[string]string,
	prefix string) maths.Vec {
	test.Helper()
	return maths.V(float(test, row, prefix+"x"),
		float(test, row, prefix+"y"), float(test, row, prefix+"z"))
}

// meshBlendWeights is the weights of the blend test, in the order of a
// face's corners.
func meshBlendWeights(test testing.TB) weights {
	test.Helper()
	row := read(test, meshBlendWeightsFile)[nought]
	return weights{float(test, row, "first"), float(test, row, "second"),
		float(test, row, "third")}
}

// cubeNormals is a normal at each of the cube's corners, out from its
// middle.
func cubeNormals() []maths.Vec {
	corners := cubeCorners()
	for i, corner := range corners {
		corners[i] = corner.Unit()
	}
	return corners
}

// a cube and a ball of faces are closed; one face of a cube is not.
func TestAClosedMeshKnowsItIs(t *testing.T) {
	if !cube(t).Closed() {
		t.Error("the cube is not closed")
	}
	halvings := pickedCount(t, "mesh-closed-ball-halvings")
	if made, _, _ := ball(t, halvings); !made.Closed() {
		t.Error("the ball is not closed")
	}
	if mustMesh(t, cubeCorners(), cubeFaces()[:pair], nil, false).Closed() {
		t.Error("one side of a cube is closed")
	}
}

// slabs is where a ray first meets the cube's box, by the slab
// arithmetic of a box, and whether it does: the true answer the mesh's
// faces are held to.
func slabs(ray maths.Ray) (float64, bool) {
	enter, leave := math.Inf(-plusInfinity), math.Inf(plusInfinity)
	for axis, at := range parts(ray.At) {
		way := parts(ray.Dir)[axis]
		low, high := (-boxHalf-at)/way, (boxHalf-at)/way
		enter = math.Max(enter, math.Min(low, high))
		leave = math.Min(leave, math.Max(low, high))
	}
	return enter, enter <= leave && enter > nought
}

// a ray from outside meets the cube where a box's slabs say, to a part
// in a hundred million million, facing the way of the side it lands on:
// ten thousand rays from all round toward points near it.
func TestARayMeetsTheCubeWhereItIs(t *testing.T) {
	made := cube(t)
	random := meshRandom(t, "mesh-cube-seed-first", "mesh-cube-seed-second")
	start := picked(t, "mesh-cube-start-radius")
	reach := picked(t, "mesh-cube-aim-reach")
	for range pickedCount(t, "mesh-cube-rays") {
		from := meshNormalPoint(random).Unit().Scale(start)
		toward := meshBoxPoint(random).Scale(reach)
		ray := maths.NewRay(from, toward.Sub(from).Unit())
		want, wanted := slabs(ray)
		met, got := made.Meet(ray, nought, math.Inf(plusInfinity))
		if got != wanted {
			t.Fatalf("from %v toward %v: met %v, a box %v", from, toward,
				got, wanted)
		}
		if got {
			checkCubeMeet(t, made, ray, met, want)
		}
	}
}

// checkCubeMeet holds a meeting of a ray with the cube to where the
// box has it, and to the side it lands on.
func checkCubeMeet(t *testing.T, made *Mesh, ray maths.Ray, met Met,
	want float64) {
	t.Helper()
	if !(math.Abs(met.Along()-want) <=
		picked(t, "mesh-cube-meet-tolerance")*want) {
		t.Fatalf("met at %v, the box at %v", met.Along(), want)
	}
	point := ray.Point(met.Along())
	normal := made.NormalAt(met)
	side := math.Abs(normal.Dot(point)*pair - maxPart(point)*pair)
	length := math.Abs(normal.Len() - unity)
	if !(side <= picked(t, "mesh-cube-facing-tolerance")) ||
		!(length <= picked(t, "mesh-cube-length-tolerance")) {
		t.Fatalf("at %v the cube faces %v", point, normal)
	}
}

// maxPart is the largest of a point's parts by size.
func maxPart(point maths.Vec) float64 {
	rest := math.Max(math.Abs(point.Y), math.Abs(point.Z))
	return math.Max(math.Abs(point.X), rest)
}

// slipAims is where the rays of the slip test aim: every corner, the
// middle of an edge of each face, and random points.
func slipAims(t *testing.T, random *rand.Rand, corners []maths.Vec,
	faces [][triangleCorners]int) []maths.Vec {
	t.Helper()
	var aims []maths.Vec
	aims = append(aims, corners...)
	for _, at := range faces {
		edge := corners[at[first]].Add(corners[at[second]])
		aims = append(aims, edge.Scale(boxHalf))
	}
	for range pickedCount(t, "mesh-slip-random-aims") {
		aims = append(aims, meshNormalPoint(random))
	}
	return aims
}

// no ray slips between faces: from inside a closed ball of faces,
// every ray leaves through a face, aimed exactly at a corner, at the
// middle of an edge, or anywhere at all, from the middle and from off
// it. a test that is not watertight loses some of these.
func TestNoRaySlipsBetweenFaces(t *testing.T) {
	made, corners, faces := ball(t, pickedCount(t, "mesh-slip-halvings"))
	random := meshRandom(t, "mesh-slip-seed-first", "mesh-slip-seed-second")
	starts := vectorsIn(t, meshSlipStartsFile)
	aims := slipAims(t, random, corners, faces)
	missed := nought
	for _, from := range starts {
		for _, aim := range aims {
			ray := maths.NewRay(from, aim.Sub(from).Unit())
			if _, ok := made.Hit(ray, nought, math.Inf(plusInfinity)); !ok {
				missed++
			}
		}
	}
	if missed > nought {
		t.Errorf("%d of %d rays from inside slipped out", missed,
			len(starts)*len(aims))
	}
}

// a ball of faces round the unit sphere is met from outside between
// the sphere its flattest faces reach in to and the sphere itself.
func TestABallOfFacesIsMetAtTheSphere(t *testing.T) {
	made, corners, faces := ball(t, pickedCount(t, "mesh-sphere-halvings"))
	inner := float64(unity)
	for _, at := range faces {
		middle := corners[at[first]].Add(corners[at[second]])
		middle = middle.Add(corners[at[third]])
		reach := middle.Scale(meshThird).Len()
		inner = math.Min(inner, reach)
	}
	random := meshRandom(t, "mesh-sphere-seed-first",
		"mesh-sphere-seed-second")
	start := picked(t, "mesh-sphere-start-radius")
	slack := picked(t, "mesh-sphere-tolerance")
	for range pickedCount(t, "mesh-sphere-rays") {
		from := meshNormalPoint(random).Unit().Scale(start)
		ray := maths.NewRay(from, from.Neg().Unit())
		along, ok := made.Hit(ray, nought, math.Inf(plusInfinity))
		if !ok || along < start-unity-slack || along > start-inner+slack {
			t.Fatalf("straight in from %v: %v at %v, inner %v", from, ok,
				along, inner)
		}
	}
}

// checkBadPart is the fourth corner of a cube with one part made bad,
// which is no point and must not make a mesh.
func checkBadPart(t *testing.T, axis int, bad float64) {
	t.Helper()
	broken := pickedCount(t, "mesh-finite-corner")
	corners := cubeCorners()
	part := parts(corners[broken])
	part[axis] = bad
	corners[broken] = vectorOf(part)
	_, err := New(corners, cubeFaces(), nil, false)
	if !errors.Is(err, errNotPoint) {
		t.Errorf("part %d of %v: %v", axis, bad, err)
	}
	if finite(corners[broken]) {
		t.Errorf("part %d of %v is a point", axis, bad)
	}
}

// a corner is a point only if each part of it is a finite number: not
// nowhere, and not endlessly far either way, in any one of its parts.
func TestACornerIsAPointOnlyIfEveryPartIsFinite(t *testing.T) {
	bads := []float64{math.NaN(), math.Inf(plusInfinity),
		math.Inf(-plusInfinity)}
	for axis := range axes {
		for _, bad := range bads {
			checkBadPart(t, axis, bad)
		}
	}
	far := picked(t, "mesh-far-reach")
	if !finite(maths.V(far, -far, nought)) {
		t.Error("a far point is not a point")
	}
}

// checkSmoothCase builds the mesh of a case, with the normals if the
// case has them and the ask if it makes it, and holds it to its want.
func checkSmoothCase(t *testing.T, row map[string]string,
	normals []maths.Vec) {
	t.Helper()
	var given []maths.Vec
	if whole(t, row, "normals") == unity {
		given = normals
	}
	smooth := whole(t, row, "smooth") == unity
	want := whole(t, row, "want") == unity
	made := mustMesh(t, aSquare(nought), theOneFace(), given, smooth)
	if made.Smooth() != want {
		t.Errorf("%s: smooth is %v", row["name"], made.Smooth())
	}
	if !want {
		got := made.facing(nought, meshBlendWeights(t))
		if got != maths.V(nought, nought, unity) {
			t.Errorf("%s: it faces %v", row["name"], got)
		}
	}
}

// a mesh blends its corners' normals only if it is asked to and has
// them: asked without them, or given them and not asked, it faces
// each face's own way.
func TestASmoothMeshNeedsNormalsAndTheAsk(t *testing.T) {
	normals := vectorsIn(t, meshAxisNormalsFile)
	for _, row := range read(t, meshSmoothCasesFile) {
		checkSmoothCase(t, row, normals)
	}
}

// the blend is each corner's normal by its weight, made one long: a
// fifth of the first, three tenths of the second and half the third,
// the normals along x, y and z, which is (0.2, 0.3, 0.5) over its
// length, the root of 0.38.
func TestTheBlendWeighsEachCornersNormalByItsOwnWeight(t *testing.T) {
	normals := vectorsIn(t, meshAxisNormalsFile)
	made := mustMesh(t, aSquare(nought), theOneFace(), normals, true)
	weight := meshBlendWeights(t)
	long := math.Sqrt(picked(t, "mesh-blend-length-squared"))
	want := maths.V(weight[first]/long, weight[second]/long,
		weight[third]/long)
	got := made.facing(nought, weight)
	if got.Sub(want).Len() > picked(t, "mesh-blend-tolerance") {
		t.Errorf("the blend faces %v, not %v", got, want)
	}
}

// the mesh's bounds are the box round its corners, whatever the shape
// of the tree: the cube's, a tetrahedron's, and a mesh of two faces
// that is one leaf.
func TestTheBoundsAreTheBoxRoundTheCorners(t *testing.T) {
	skew := mustMesh(t, vectorsIn(t, meshTetrahedronPoints),
		shapeFaces(meshTetrahedronFaces), nil, false)
	height := picked(t, "mesh-bounds-height")
	two := mustMesh(t, aSquare(height), shapeFaces(meshTwoFacesFile), nil,
		false)
	shapes := []*Mesh{cube(t), skew, two}
	for place, row := range read(t, meshBoundsCasesFile) {
		got := shapes[place].Bounds()
		bottom, top := meshBoxCorner(t, row, "low"),
			meshBoxCorner(t, row, "top")
		if got.Min != bottom || got.Max != top {
			t.Errorf("%s: bounds %v to %v", row["name"], got.Min, got.Max)
		}
	}
}
