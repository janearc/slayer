package shape

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The counts and places the mesh tests make with, each true by definition.
const (
	meshCorners    = 3 // defined: the corners of a triangle
	meshAxes       = 3 // defined: the axes of space
	meshCubeSides  = 6 // defined: the sides of a cube
	meshKinds      = 4 // defined: the kinds of point cubePoint draws
	meshFirst      = 0 // defined: the first place in a list
	meshSecond     = 1 // defined: the second place in a list
	meshThird      = 2 // defined: the third place in a list
	meshFourth     = 3 // defined: the fourth place in a list
	meshGoldenRoot = 5 // defined: the golden ratio is (1 + sqrt 5) / 2
)

// The kinds of point cubePoint draws.
const (
	meshKindCorner = 0 // defined: a corner of the cube
	meshKindEdge   = 1 // defined: a place on an edge of the cube
	meshKindLine   = 2 // defined: a place on a line a side is cut along
)

// The tables the mesh tests read.
const (
	meshCornersFile  = "testdata/mesh-icosahedron-corners.csv"
	meshFacesFile    = "testdata/mesh-icosahedron-faces.csv"
	meshOverlapsFile = "testdata/mesh-overlap-points.csv"
)

// meshFace is a triangle, as the places of its three corners.
type meshFace = [meshCorners]int

// meshSide is one side of the cube: the way out of it, then the two ways
// across it, u by v out.
type meshSide struct{ out, across, along maths.Vec }

// meshSides are the six sides of the cube, by the way each faces.
func meshSides() []meshSide {
	x, y, z := maths.XAxis, maths.YAxis, maths.ZAxis
	return []meshSide{
		{x, y, z}, {x.Neg(), z, y}, {y, z, x},
		{y.Neg(), x, z}, {z, x, y}, {z.Neg(), y, x},
	}
}

// meshCornerSet is the corners of a cube so far. With split, each side
// has corners of its own, at the same places as its neighbours'.
type meshCornerSet struct {
	vertices []maths.Vec
	at       map[maths.Vec]int
	split    bool
}

// corner is the place of a corner at point, made if it is not there yet.
func (set *meshCornerSet) corner(point maths.Vec) int {
	if place, found := set.at[point]; found && !set.split {
		return place
	}
	set.vertices = append(set.vertices, point)
	place := len(set.vertices) - unity
	set.at[point] = place
	return place
}

// meshCut is the place of the cut-th line across a side cut count ways,
// from -1 to 1.
func meshCut(count, cut int) float64 {
	return -unity + pair*float64(cut)/float64(count)
}

// meshGrid is the corners of one side cut count by count, as their places
// in the set.
func meshGrid(set *meshCornerSet, side meshSide, count int) [][]int {
	grid := make([][]int, count+unity)
	for row := range grid {
		grid[row] = make([]int, count+unity)
		for column := range grid[row] {
			across := side.across.Scale(meshCut(count, row))
			along := side.along.Scale(meshCut(count, column))
			point := side.out.Add(across).Add(along)
			grid[row][column] = set.corner(point)
		}
	}
	return grid
}

// meshSquares is the two faces of each square of a grid, corner to corner.
func meshSquares(grid [][]int) []meshFace {
	var faces []meshFace
	count := len(grid) - unity
	for row := range count {
		for column := range count {
			origin, forward := grid[row][column], grid[row+unity][column]
			opposite := grid[row+unity][column+unity]
			sideways := grid[row][column+unity]
			faces = append(faces, meshFace{origin, forward, opposite},
				meshFace{origin, opposite, sideways})
		}
	}
	return faces
}

// cube is the cube from -1 to 1 on each axis, each side cut count by
// count into squares and each square into two faces, corner to corner.
// With split, each side has corners of its own, at the same places as its
// neighbours'.
func cube(count int, split bool) ([]maths.Vec, [][meshCorners]int) {
	set := &meshCornerSet{at: map[maths.Vec]int{}, split: split}
	var faces []meshFace
	for _, side := range meshSides() {
		grid := meshGrid(set, side, count)
		faces = append(faces, meshSquares(grid)...)
	}
	return set.vertices, faces
}

// meshIcosahedron is the corners of the icosahedron in the unit sphere,
// read from the table: each has a plain place of one or minus one on one
// axis and a golden place on another.
func meshIcosahedron(t *testing.T) []maths.Vec {
	t.Helper()
	golden := (unity + math.Sqrt(meshGoldenRoot)) / pair
	var vertices []maths.Vec
	for _, row := range readSheet(t, meshCornersFile) {
		var place [meshAxes]float64
		place[cellCount(t, row, "plain-axis")] = cell(t, row, "plain-sign")
		place[cellCount(t, row, "gold-axis")] = cell(t, row, "gold-sign") *
			golden
		vertex := maths.V(place[meshFirst], place[meshSecond],
			place[meshThird])
		vertices = append(vertices, vertex.Unit())
	}
	return vertices
}

// meshIcosahedronFaces is the faces of the icosahedron, read from the
// table.
func meshIcosahedronFaces(t *testing.T) []meshFace {
	t.Helper()
	var faces []meshFace
	for _, row := range readSheet(t, meshFacesFile) {
		faces = append(faces, meshFace{cellCount(t, row, "first"),
			cellCount(t, row, "second"), cellCount(t, row, "third")})
	}
	return faces
}

// icosphere is the icosahedron in the unit sphere, each face split in
// four level times, every new corner pushed out to the sphere.
func icosphere(t *testing.T, level int) ([]maths.Vec, [][meshCorners]int) {
	t.Helper()
	vertices, faces := meshIcosahedron(t), meshIcosahedronFaces(t)
	for range level {
		faces = quarter(&vertices, faces)
	}
	return vertices, faces
}

// meshMiddles are the middles of the edges made so far, by the two corners
// each lies between.
type meshMiddles struct {
	vertices *[]maths.Vec
	made     map[[pair]int]int
}

// between is the place of the middle of the edge from first to second,
// pushed out to the unit sphere, and made when it is first asked for.
func (middles meshMiddles) between(first, second int) int {
	key := [pair]int{min(first, second), max(first, second)}
	if place, found := middles.made[key]; found {
		return place
	}
	vertices := *middles.vertices
	middle := vertices[first].Add(vertices[second]).Unit()
	*middles.vertices = append(vertices, middle)
	place := len(*middles.vertices) - unity
	middles.made[key] = place
	return place
}

// quarter splits each face in four at its edges' middles, pushed out to
// the unit sphere, each new face turning as the old one did.
func quarter(vs *[]maths.Vec, fs [][meshCorners]int) [][meshCorners]int {
	middles := meshMiddles{vertices: vs, made: map[[pair]int]int{}}
	var next [][meshCorners]int
	for _, face := range fs {
		first, second := face[meshFirst], face[meshSecond]
		third := face[meshThird]
		firstSecond := middles.between(first, second)
		secondThird := middles.between(second, third)
		thirdFirst := middles.between(third, first)
		next = append(next, meshFace{first, firstSecond, thirdFirst},
			meshFace{second, secondThird, firstSecond},
			meshFace{third, thirdFirst, secondThird},
			meshFace{firstSecond, secondThird, thirdFirst})
	}
	return next
}

// anyway is a direction, any one alike.
func anyway(rng *rand.Rand) maths.Vec {
	return maths.V(rng.NormFloat64(), rng.NormFloat64(),
		rng.NormFloat64()).Unit()
}

// meshSpread is a place anywhere from -reach to reach.
func meshSpread(rng *rand.Rand, reach float64) float64 {
	return reach * (pair*rng.Float64() - unity)
}

// within is a point anywhere in the cube from -reach to reach.
func within(rng *rand.Rand, reach float64) maths.Vec {
	return maths.V(meshSpread(rng, reach), meshSpread(rng, reach),
		meshSpread(rng, reach))
}

// inBall is a point anywhere in the ball of radius reach.
func inBall(rng *rand.Rand, reach float64) maths.Vec {
	for {
		if point := within(rng, reach); point.Len() < reach {
			return point
		}
	}
}

// aimed is a ray from at toward to.
func aimed(at, to maths.Vec) maths.Ray {
	return maths.Ray{At: at, Dir: to.Sub(at).Unit()}
}

// angle is between two directions, true however small.
func angle(first, second maths.Vec) float64 {
	return math.Atan2(first.Cross(second).Len(), first.Dot(second))
}

// passes is how near the middle the ray's line comes.
func passes(ray maths.Ray) float64 {
	along := ray.At.Dot(ray.Dir)
	foot := ray.Dir.Scale(along)
	return ray.At.Sub(foot).Len()
}

// meshStream is a random stream seeded by the two numbers whose names
// start with stem.
func meshStream(t *testing.T, stem string) *rand.Rand {
	t.Helper()
	high := uint64(pickedCount(t, stem+"-high"))
	low := uint64(pickedCount(t, stem+"-low"))
	return rand.New(rand.NewPCG(high, low))
}

// meshFault is a mesh made wrongly, and the words that name the fault.
type meshFault struct {
	why      string
	vertices []maths.Vec
	faces    []meshFace
	normals  []maths.Vec
	want     string
}

// meshSquare is the unit square in the floor, corner by corner.
func meshSquare() []maths.Vec {
	zero := maths.V(nought, nought, nought)
	return []maths.Vec{zero, maths.V(unity, nought, nought),
		maths.V(unity, unity, nought), maths.V(nought, unity, nought)}
}

// meshHalves are the two triangles of the square, cut corner to corner.
func meshHalves() []meshFace {
	return []meshFace{{meshFirst, meshSecond, meshThird},
		{meshFirst, meshThird, meshFourth}}
}

// meshFaultsInFaces are meshes whose faces are made wrongly, each with the
// words NewMesh names the fault in.
func meshFaultsInFaces(t *testing.T) []meshFault {
	t.Helper()
	square, halves := meshSquare(), meshHalves()
	zero := maths.V(nought, nought, nought)
	inline := []maths.Vec{zero, maths.V(unity, unity, unity),
		maths.V(pair, pair, pair)}
	missing := pickedCount(t, "mesh-fault-missing-vertex")
	first, second, third := meshFirst, meshSecond, meshThird
	return []meshFault{
		{"no faces", square, nil, nil, "at least one triangle"},
		{"a vertex not there", square,
			[]meshFace{{first, second, third}, {first, third, missing}},
			nil, "triangle 1 names vertex 7, and there are 4"},
		{"a vertex before the first", square,
			[]meshFace{{-unity, second, third}}, nil,
			"triangle 0 names vertex -1"},
		{"a corner twice", square,
			[]meshFace{{first, second, third}, {first, third, third}},
			nil, "triangle 1 has no area"},
		{"corners in a line", inline, halves[:unity], nil,
			"triangle 0 has no area"},
	}
}

// meshFaultsInCorners are meshes whose corners or normals are made
// wrongly, each with the words NewMesh names the fault in.
func meshFaultsInCorners(t *testing.T) []meshFault {
	t.Helper()
	square, halves := meshSquare(), meshHalves()
	nowhere := append([]maths.Vec(nil), square...)
	nowhere[meshFourth] = maths.V(nought, math.NaN(), nought)
	zero := maths.V(nought, nought, nought)
	flat := []maths.Vec{maths.Up, maths.Up, zero, maths.Up}
	short := pickedCount(t, "mesh-fault-short-normals")
	return []meshFault{
		{"a corner nowhere", nowhere, halves, nil,
			"triangle 1: vertex 3 is not a finite point"},
		{"a normal short", square, halves, square[:short],
			"3 normals for 4 vertices"},
		{"a normal pointing nowhere", square, halves, flat,
			"triangle 0: the normal at vertex 2 points nowhere"},
	}
}

// meshFaultsInWinding are meshes whose faces turn the wrong way, each
// with the words NewMesh names the fault in.
func meshFaultsInWinding(t *testing.T) []meshFault {
	t.Helper()
	inside, sides := cube(unity, false)
	for index := range sides {
		sides[index][meshSecond], sides[index][meshThird] =
			sides[index][meshThird], sides[index][meshSecond]
	}
	_, twisted := cube(unity, false)
	turned := pickedCount(t, "mesh-fault-twisted-face")
	twisted[turned][meshSecond], twisted[turned][meshThird] =
		twisted[turned][meshThird], twisted[turned][meshSecond]
	apart := []meshFace{{meshFirst, meshSecond, meshThird},
		{meshFirst, meshFourth, meshThird}}
	return []meshFault{
		{"neighbours turning apart", meshSquare(), apart, nil,
			"triangles 0 and 1 both run from vertex 2 to vertex 0"},
		{"one face of a cube turned", inside, twisted, nil,
			"triangles 0 and 5 both run"},
		{"a cube inside out", inside, sides, nil,
			"the closed piece with triangle 0 in it is inside out"},
	}
}

// NewMesh names the triangle, the corner or the pair of faces at fault,
// for each way a mesh can be made wrongly.
func TestNewMeshNamesTheTriangleAtFault(t *testing.T) {
	faults := meshFaultsInFaces(t)
	faults = append(faults, meshFaultsInCorners(t)...)
	faults = append(faults, meshFaultsInWinding(t)...)
	for _, fault := range faults {
		built, err := NewMesh(fault.vertices, fault.faces, fault.normals,
			false)
		switch {
		case err == nil:
			t.Errorf("%s: made a mesh of %d faces", fault.why,
				len(built.Faces))
		case !strings.Contains(err.Error(), fault.want):
			t.Errorf("%s: %q, wanting %q", fault.why, err, fault.want)
		}
	}
}

// meshCentre is the middle of a face: the mean of its three corners.
func meshCentre(vertices []maths.Vec, face meshFace) maths.Vec {
	sum := vertices[face[meshFirst]].Add(vertices[face[meshSecond]]).
		Add(vertices[face[meshThird]])
	return sum.Scale(unity / float64(meshCorners))
}

// meshFacesOutwardOnTheBall fails unless each face of the icosphere has a
// normal that points out, and a ray from outside meets its outside.
func meshFacesOutwardOnTheBall(t *testing.T) {
	t.Helper()
	vs, fs := icosphere(t, pickedCount(t, "mesh-turning-level"))
	ball := made(t, vs, fs, nil, false)
	if !ball.Closed() {
		t.Fatal("the icosphere is not closed")
	}
	outside := picked(t, "mesh-turning-outside")
	zero := maths.V(nought, nought, nought)
	for _, face := range fs {
		middle := meshCentre(vs, face)
		if normal := ball.Normal(middle); normal.Dot(middle) <= nought {
			t.Fatalf("at %v the normal %v points in", middle, normal)
		}
		ray := aimed(middle.Scale(outside), zero)
		_, index, ok := ball.first(ray, nought, forever)
		if !ok {
			t.Fatalf("a ray at %v missed", middle)
		}
		prepared := ready(ray)
		_, front, _ := prepared.meet(&ball.tris[index], nought, forever)
		if !front {
			t.Fatalf("a ray from outside at %v met the inside", middle)
		}
	}
}

// meshNormalsOutwardOnTheBox fails unless a flat face on each side of the
// cube has a normal that points out.
func meshNormalsOutwardOnTheBox(t *testing.T) {
	t.Helper()
	cvs, cfs := cube(pickedCount(t, "mesh-turning-box-cuts"), true)
	box := made(t, cvs, cfs, nil, false)
	tolerance := picked(t, "mesh-turning-normal-tolerance")
	acrossShare := picked(t, "mesh-turning-across")
	alongShare := picked(t, "mesh-turning-along")
	for k := range meshCubeSides {
		out := cubeSide(k)
		across := maths.V(out.Y, out.Z, out.X).Scale(acrossShare)
		along := maths.V(out.Z, out.X, out.Y).Scale(alongShare)
		at := out.Add(across).Add(along)
		if normal := box.Normal(at); !normal.Near(out, tolerance) {
			t.Errorf("the cube's side at %v faces %v", at, normal)
		}
	}
}

// Faces turn counterclockwise seen from outside, so a flat face's normal
// points out, and a ray from outside meets the outside of the face.
func TestAMeshTurnsCounterclockwiseFromOutside(t *testing.T) {
	meshFacesOutwardOnTheBall(t)
	meshNormalsOutwardOnTheBox(t)
}

// meshBoxDistance is the box from -1 to 1's own distance at point, less
// than nothing inside.
func meshBoxDistance(point maths.Vec) float64 {
	past := maths.V(math.Abs(point.X)-unity, math.Abs(point.Y)-unity,
		math.Abs(point.Z)-unity)
	gap := maths.V(math.Max(past.X, nought), math.Max(past.Y, nought),
		math.Max(past.Z, nought)).Len()
	deepest := math.Max(past.X, math.Max(past.Y, past.Z))
	return gap + math.Min(deepest, nought)
}

// Inside and Distance on the cube are the box's own: inside is within it,
// and the distance is the box's distance, less than nothing inside.
func TestInsideAndDistanceOnACube(t *testing.T) {
	vs, fs := cube(pickedCount(t, "mesh-inside-cuts"), false)
	built := made(t, vs, fs, nil, false)
	rng := meshStream(t, "mesh-inside-seed")
	reach := picked(t, "mesh-inside-reach")
	tolerance := picked(t, "mesh-inside-tolerance")
	for range pickedCount(t, "mesh-inside-rays") {
		point := within(rng, reach)
		want := meshBoxDistance(point)
		if in := built.Inside(point); in != (want < nought) {
			t.Fatalf("at %v, inside %v; want %v", point, in, want < nought)
		}
		if got := built.Distance(point); math.Abs(got-want) > tolerance {
			t.Fatalf("at %v, %v from the surface; want %v",
				point, got, want)
		}
	}
}

// Two cubes in one mesh, overlapping, are inside each other: a point in
// both is inside, as a point in either is. Counting faces crossed in
// twos would say the overlap was out.
func TestOverlappingPiecesAreInsideEachOther(t *testing.T) {
	vs, fs := cube(unity, false)
	count, faces := len(vs), len(fs)
	shift := maths.V(unity, half, picked(t, "mesh-overlap-shift-z"))
	for _, vertex := range vs[:count] {
		vs = append(vs, vertex.Add(shift))
	}
	for _, face := range fs[:faces] {
		fs = append(fs, meshFace{face[meshFirst] + count,
			face[meshSecond] + count, face[meshThird] + count})
	}
	built := made(t, vs, fs, nil, false)
	for _, row := range readSheet(t, meshOverlapsFile) {
		at := maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
		want := row["inside"] == "true"
		if got := built.Inside(at); got != want {
			t.Errorf("inside at %v: %v, want %v", at, got, want)
		}
	}
}

// A mesh with an edge of its own encloses nothing.
func TestAnOpenMeshHasNoInside(t *testing.T) {
	built := lone(t)
	if built.Closed() {
		t.Error("one triangle is closed")
	}
	offset := picked(t, "mesh-open-offset")
	point := maths.V(offset, offset, picked(t, "mesh-open-depth"))
	tolerance := picked(t, "mesh-open-tolerance")
	if in, away := built.Inside(point), built.Distance(point); in ||
		math.Abs(away-half) > tolerance {
		t.Errorf("under it: inside %v, %v away", in, away)
	}
}

// inradius is the nearest any face's plane comes to the zero: the ball
// that size is inside the icosphere, which is inside the unit ball.
func inradius(vs []maths.Vec, fs [][meshCorners]int) float64 {
	nearest := math.Inf(unity)
	for _, face := range fs {
		first, second := vs[face[meshFirst]], vs[face[meshSecond]]
		third := vs[face[meshThird]]
		normal := second.Sub(first).Cross(third.Sub(first)).Unit()
		offset := math.Abs(normal.Dot(first))
		nearest = math.Min(nearest, offset)
	}
	return nearest
}

// A tessellated sphere against the sphere, over ten thousand rays a
// level, most from outside and a fifth from within.
//
// The icosphere lies between two balls: the unit ball outside it, and
// inside it the ball of its inradius, in. A ray whose line passes nearer
// the middle than in meets both shapes, and one passing further than one
// meets neither; between in and one is the silhouette, where either may
// meet and the other not. Along a ray that meets both, the two surfaces
// are between the two balls, which are sqrt(1-b^2) - sqrt(in^2-b^2) apart
// there, b the line's distance from the middle, so the two distances agree
// that well. The band, one less in, quarters each time the faces split in
// four, and the gap with it.
func TestATessellatedSphereAgreesWithTheSphere(t *testing.T) {
	lastBand, lastGap := math.Inf(unity), math.Inf(unity)
	first := pickedCount(t, "mesh-sphere-first-level")
	last := pickedCount(t, "mesh-sphere-last-level")
	bandShrink := picked(t, "mesh-band-shrink")
	gapShrink := picked(t, "mesh-gap-shrink")
	for level := first; level <= last; level++ {
		band, gap := againstTheSphere(t, level)
		if band > lastBand/bandShrink || gap > lastGap/gapShrink {
			t.Errorf("level %d: the band %v and the gap %v did "+
				"not shrink from %v and %v", level, band, gap,
				lastBand, lastGap)
		}
		lastBand, lastGap = band, gap
	}
}

// meshSphereRay is the ray the place-th draw of the sphere test aims:
// mostly from outside toward a point near the middle, and every so often
// from within.
func meshSphereRay(t *testing.T, rng *rand.Rand, place int) maths.Ray {
	t.Helper()
	start := picked(t, "mesh-sphere-start-reach")
	reach := picked(t, "mesh-sphere-aim-reach")
	ray := aimed(anyway(rng).Scale(start), inBall(rng, reach))
	every := pickedCount(t, "mesh-sphere-inside-every")
	if place%every == pickedCount(t, "mesh-sphere-inside-phase") {
		ray = maths.Ray{At: inBall(rng, half), Dir: anyway(rng)}
	}
	return ray
}

// againstTheSphere is one level of the icosphere against the sphere: the
// band at its silhouette, and how far apart the distances were for rays
// that pass within 0.9 of the middle.
func againstTheSphere(t *testing.T, level int) (float64, float64) {
	t.Helper()
	vs, fs := icosphere(t, level)
	built := made(t, vs, fs, nil, false)
	nearest, ball := inradius(vs, fs), Sphere{Radius: unity}
	high := uint64(pickedCount(t, "mesh-sphere-seed-high"))
	seed := rand.NewPCG(high, uint64(level))
	rng := rand.New(seed)
	var worst float64
	banded := nought
	slack := picked(t, "mesh-sphere-gap-slack")
	nearMiddle := picked(t, "mesh-sphere-near-middle")
	for place := range pickedCount(t, "mesh-sphere-rays") {
		ray := meshSphereRay(t, rng, place)
		closest := passes(ray)
		sphereAt, hitSphere := ball.Hit(ray, nought, forever)
		meshAt, hitMesh := built.Hit(ray, nought, forever)
		switch {
		case closest < nearest && (!hitMesh || !hitSphere):
			t.Fatalf("level %d: %v met the sphere %v, the mesh %v",
				level, ray, hitSphere, hitMesh)
		case closest < nearest:
			gap := math.Sqrt(unity-closest*closest) -
				math.Sqrt(nearest*nearest-closest*closest)
			if apart := math.Abs(meshAt - sphereAt); apart > gap+slack {
				t.Fatalf("level %d: %v apart along %v, past %v",
					level, apart, ray, gap)
			}
			if closest < nearMiddle {
				worst = math.Max(worst, math.Abs(meshAt-sphereAt))
			}
		case closest > unity && (hitMesh || hitSphere):
			t.Fatalf("level %d: %v, past both, met the sphere %v, "+
				"the mesh %v", level, ray, hitSphere, hitMesh)
		case closest <= unity:
			banded++
			if hitMesh && !hitSphere {
				t.Fatalf("level %d: %v met the mesh and "+
					"not the sphere around it", level, ray)
			}
		}
	}
	t.Logf("level %d, %d faces: band %.2e, %d rays in it; distances "+
		"within %.2e where b < 0.9", level, len(fs), unity-nearest, banded,
		worst)
	return unity - nearest, worst
}

// A closed cube lets no ray through. Rays are aimed at its corners, at
// its edges, at the lines its sides are cut along, corners where six
// faces meet among them, and anywhere on its sides, from outside and
// from within, with corners shared by index and with corners only at
// the same place. Each meets the cube where it was aimed, and a ray from
// outside crosses the surface exactly twice, in and out.
func TestACubeLetsNoRayThrough(t *testing.T) {
	for _, split := range []bool{false, true} {
		vs, fs := cube(pickedCount(t, "mesh-through-cuts"), split)
		built := made(t, vs, fs, nil, false)
		rng := meshStream(t, "mesh-through-seed")
		for place := range pickedCount(t, "mesh-through-rays") {
			meshThroughOne(t, built, rng, place)
		}
		for k := range meshCubeSides {
			straightAtASide(t, built, cubeSide(k))
		}
	}
}

// meshThroughOne aims the place-th ray at a point of the cube: mostly from
// outside, and every so often from within.
func meshThroughOne(t *testing.T, built *Mesh, rng *rand.Rand, place int) {
	t.Helper()
	aim := cubePoint(t, rng, place%meshKinds)
	spread := picked(t, "mesh-through-spread")
	out := outward(t, rng, aim).Scale(unity + spread*rng.Float64())
	every := pickedCount(t, "mesh-through-inside-every")
	if place%every == pickedCount(t, "mesh-through-inside-phase") {
		reach := picked(t, "mesh-through-inside-reach")
		throughCube(t, built, within(rng, reach), aim, false)
	} else {
		throughCube(t, built, aim.Add(out), aim, true)
	}
}

// meshGridShare is where the place-th line of a side cut into quarters
// falls, from the middle of the side: a share of the way to its edge.
func meshGridShare(t *testing.T, place int) float64 {
	t.Helper()
	quarters := pickedCount(t, "mesh-side-grid-quarters")
	return float64(place-quarters) / float64(quarters)
}

// straightAtASide aims rays along an axis at the side of the cube facing
// out, at every corner and line it is cut along and between, where the
// edges' determinants are exactly zero.
func straightAtASide(t *testing.T, built *Mesh, out maths.Vec) {
	t.Helper()
	across := maths.V(out.Y, out.Z, out.X)
	along := maths.V(out.Z, out.X, out.Y)
	limit := pickedCount(t, "mesh-side-grid-limit")
	standoff := picked(t, "mesh-side-standoff")
	for row := unity; row < limit; row++ {
		acrossAt := across.Scale(meshGridShare(t, row))
		for column := unity; column < limit; column++ {
			alongAt := along.Scale(meshGridShare(t, column))
			aim := out.Add(acrossAt).Add(alongAt)
			from := aim.Add(out.Scale(standoff))
			throughCube(t, built, from, aim, true)
		}
	}
}

// cubeSide is the cube's kth side, by which way it faces.
func cubeSide(k int) maths.Vec {
	sign := float64(unity - pair*(k%pair))
	sides := [meshAxes]maths.Vec{maths.V(sign, nought, nought),
		maths.V(nought, sign, nought), maths.V(nought, nought, sign)}
	return sides[k/pair]
}

// meshDraws are the random draws that place a point on the cube, with the
// counts they are made by.
type meshDraws struct {
	rng           *rand.Rand
	gridLines     int
	edgeChance    int
	edgeLines     int
	diagonalLines int
	ways          int
}

// meshDrawsOf is the draws from rng, with the counts from the book.
func meshDrawsOf(t *testing.T, rng *rand.Rand) meshDraws {
	t.Helper()
	return meshDraws{
		rng:           rng,
		gridLines:     pickedCount(t, "mesh-grid-lines"),
		edgeChance:    pickedCount(t, "mesh-edge-grid-chance"),
		edgeLines:     pickedCount(t, "mesh-edge-grid-lines"),
		diagonalLines: pickedCount(t, "mesh-diagonal-lines"),
		ways:          pickedCount(t, "mesh-diagonal-ways"),
	}
}

// grid is a place on one of the lines a side is cut along.
func (draws meshDraws) grid() float64 {
	return float64(draws.rng.IntN(draws.gridLines)-pair) / pair
}

// anywhere is a place anywhere across a side.
func (draws meshDraws) anywhere() float64 {
	return pair*draws.rng.Float64() - unity
}

// sign is one or minus one.
func (draws meshDraws) sign() float64 {
	return float64(unity - pair*draws.rng.IntN(pair))
}

// diagonal is a place on a line of the squares' diagonals, share along.
func (draws meshDraws) diagonal(share float64) float64 {
	return float64(draws.rng.IntN(draws.diagonalLines)-pair)/pair + share
}

// corner is the place of a corner of the cube.
func (draws meshDraws) corner() [meshAxes]float64 {
	return [meshAxes]float64{draws.sign(), draws.sign(), draws.sign()}
}

// edge is a place on an edge of the cube, sometimes on a line it is cut
// along.
func (draws meshDraws) edge() [meshAxes]float64 {
	place := [meshAxes]float64{draws.sign(), draws.sign(), draws.anywhere()}
	if draws.rng.IntN(draws.edgeChance) == nought {
		cut := float64(draws.rng.IntN(draws.edgeLines) - unity)
		place[meshThird] = cut / pair
	}
	return place
}

// line is a place on a line a side is cut along, or on a square's
// diagonal.
func (draws meshDraws) line() [meshAxes]float64 {
	place := [meshAxes]float64{draws.sign(), draws.grid(), draws.anywhere()}
	switch draws.rng.IntN(draws.ways) {
	case meshFirst:
		place[meshThird] = draws.grid()
	case meshSecond:
		share := draws.rng.Float64() / pair
		place[meshSecond] = draws.diagonal(share)
		place[meshThird] = draws.diagonal(share)
	}
	return place
}

// side is a place anywhere on a side.
func (draws meshDraws) side() [meshAxes]float64 {
	return [meshAxes]float64{draws.sign(), draws.anywhere(),
		draws.anywhere()}
}

// cubePoint is a point on the cube's surface: of kind 0, a corner; 1, on
// an edge; 2, on a line a side is cut along, or a square's diagonal; 3,
// anywhere on a side.
func cubePoint(t *testing.T, rng *rand.Rand, kind int) maths.Vec {
	t.Helper()
	draws := meshDrawsOf(t, rng)
	var place [meshAxes]float64
	switch kind {
	case meshKindCorner:
		place = draws.corner()
	case meshKindEdge:
		place = draws.edge()
	case meshKindLine:
		place = draws.line()
	default:
		place = draws.side()
	}
	axis := rng.IntN(meshAxes) // turn it to any side
	return maths.V(place[axis], place[(axis+unity)%meshAxes],
		place[(axis+pair)%meshAxes])
}

// meshPart is one part of a way out of the cube: out across a side the
// point is on, and any way along the others.
func meshPart(rng *rand.Rand, margin, coordinate float64) float64 {
	if math.Abs(coordinate) == unity {
		return coordinate * (margin + rng.Float64())
	}
	return pair*rng.Float64() - unity
}

// outward is a way out of the cube from point on its surface: out across
// every side point is on, and any way along the others.
func outward(t *testing.T, rng *rand.Rand, point maths.Vec) maths.Vec {
	t.Helper()
	margin := picked(t, "mesh-outward-margin")
	return maths.V(meshPart(rng, margin, point.X),
		meshPart(rng, margin, point.Y), meshPart(rng, margin, point.Z)).Unit()
}

// throughCube aims a ray from from at aim on the cube, and fails unless
// it meets the cube there; from outside it must cross twice, from inside
// once.
func throughCube(t *testing.T, built *Mesh, from, aim maths.Vec,
	outside bool) {
	t.Helper()
	ray, far := aimed(from, aim), aim.Sub(from).Len()
	got, ok := built.Hit(ray, nought, forever)
	if !ok || math.Abs(got-far) > picked(t, "mesh-through-tolerance") {
		t.Fatalf("aimed at %v from %v: met at %v, %v; want %v",
			aim, from, got, ok, far)
	}
	want := unity
	if outside {
		want = pair
	}
	if crossed := met(built, ray, nought); crossed != want {
		t.Fatalf("aimed at %v from %v: crossed %d times, not %d",
			aim, from, crossed, want)
	}
}

// meshSmoothDraws are the numbers the smooth-normal test draws its rays
// by.
type meshSmoothDraws struct {
	start, aim, wayOut, slack float64
}

// meshSmoothDrawsOf reads the numbers the smooth-normal test draws by.
func meshSmoothDrawsOf(t *testing.T) meshSmoothDraws {
	t.Helper()
	return meshSmoothDraws{
		start:  picked(t, "mesh-smooth-start-reach"),
		aim:    picked(t, "mesh-smooth-aim-reach"),
		wayOut: picked(t, "mesh-smooth-way-out-tolerance"),
		slack:  picked(t, "mesh-smooth-bound-slack"),
	}
}

// Smooth normals on the icosphere, given each corner's way out: blended,
// they point out through the point exactly, as the corners do, since the
// corners are on the sphere; and the point is near the sphere, so they
// are near the sphere's own normal where the same ray meets the sphere.
//
// The two points are on the ray, as far apart as the distances are,
// which is at most (1-in^2)/sqrt(1-b^2); seen from the middle, at least in
// away, they are at most twice the arcsine of that over 2in apart. For
// 5,120 faces and rays that pass within 0.9 of the middle that is under
// three tenths of a degree, where flat faces are off by degrees.
func TestSmoothNormalsFollowTheSphere(t *testing.T) {
	vs, fs := icosphere(t, pickedCount(t, "mesh-smooth-level"))
	smooth := made(t, vs, fs, vs, true)
	flat := made(t, vs, fs, vs, false)
	nearest, ball := inradius(vs, fs), Sphere{Radius: unity}
	rng := meshStream(t, "mesh-smooth-seed")
	draws := meshSmoothDrawsOf(t)
	var worst, worstFlat, most float64
	for range pickedCount(t, "mesh-smooth-rays") {
		ray := aimed(anyway(rng).Scale(draws.start), inBall(rng, draws.aim))
		got, ok := smooth.Hit(ray, nought, forever)
		sphereAt, _ := ball.Hit(ray, nought, forever)
		if !ok {
			t.Fatalf("%v missed", ray)
		}
		point := ray.Point(got)
		want := ball.Normal(ray.Point(sphereAt))
		normal := smooth.Normal(point)
		if wayOff := angle(normal, point); wayOff > draws.wayOut {
			t.Fatalf("at %v the blend is %v off the way out", point, wayOff)
		}
		closest := passes(ray)
		apart := (unity - nearest*nearest) / math.Sqrt(unity-closest*closest)
		bound := pair * math.Asin(apart/(pair*nearest))
		off := angle(normal, want)
		if off > bound+draws.slack {
			t.Fatalf("at %v the blend is %v off the sphere's, "+
				"past %v", point, off, bound)
		}
		flatNormal := flat.Normal(point)
		worst = math.Max(worst, off)
		worstFlat = math.Max(worstFlat, angle(flatNormal, want))
		most = math.Max(most, bound)
	}
	t.Logf("smooth within %.3f°, bound %.3f°; flat within %.3f°",
		worst/maths.Degree, most/maths.Degree, worstFlat/maths.Degree)
	if worst > 0.3*maths.Degree || worstFlat < unity*maths.Degree {
		t.Errorf("smooth within %v°, flat %v°", worst/maths.Degree,
			worstFlat/maths.Degree)
	}
}

// A mesh is a surface: it answers rays and normals exactly, and says
// nothing of where a ray leaves it, which is for media.
func TestAMeshIsASurfaceOnly(t *testing.T) {
	var solid any = lone(t)
	if _, ok := solid.(Shape); !ok {
		t.Error("a mesh is not a shape")
	}
	if _, ok := solid.(Hitter); !ok {
		t.Error("a mesh is not hit exactly")
	}
	if _, ok := solid.(normaler); !ok {
		t.Error("a mesh does not know its normals")
	}
	if _, ok := solid.(Spanner); ok {
		t.Error("a mesh spans, as a medium does")
	}
}

// meshExpectPanic fails the test unless a panic is under way, so it is
// deferred around a call that must panic.
func meshExpectPanic(t *testing.T) {
	t.Helper()
	if recover() == nil {
		t.Error("a mesh not built answered a ray")
	}
}

// A Mesh not made by NewMesh has nothing built, and says so rather than
// answering wrongly.
func TestAMeshNotMadeByNewMeshSaysSo(t *testing.T) {
	defer meshExpectPanic(t)
	zero := maths.V(nought, nought, nought)
	unbuilt := &Mesh{Vertices: []maths.Vec{zero, maths.XAxis, maths.Up},
		Faces: [][meshCorners]int{{meshFirst, meshSecond, meshThird}}}
	offset := picked(t, "mesh-unbuilt-offset")
	unbuilt.Hit(down(offset, offset), nought, forever)
}
