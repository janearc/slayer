package shape

import (
	"math"
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/janearc/slayer/maths"
)

// The files of rows the triangle tests read, and the places they ask for.
const (
	triangleFacesFile   = "testdata/triangle-faces.csv"
	triangleStreamsFile = "testdata/triangle-streams.csv"
	triangleOutsideFile = "testdata/triangle-outside-points.csv"
	triangleRisesFile   = "testdata/triangle-fan-rises.csv"
	triangleOnlyFace    = 0 // defined: the first face of a mesh of one face
)

// forever is a far with no end.
var forever = math.Inf(unity)

// triangleLookup is the number called name in numbers.yaml, or the reason
// it cannot be had.
func triangleLookup(name string) (float64, error) {
	chosen, err := chosenNumbers()
	if err != nil {
		return nought, err
	}
	number, err := chosen.Get(name)
	if err != nil {
		return nought, err
	}
	return number.Value, nil
}

// triangleChosen is that number for a helper that holds no test to stop:
// a name the book lacks panics.
func triangleChosen(name string) float64 {
	value, err := triangleLookup(name)
	if err != nil {
		panic(err)
	}
	return value
}

// trianglePicked is that number for a test or a benchmark, which a name
// the book lacks stops.
func trianglePicked(tester testing.TB, name string) float64 {
	tester.Helper()
	value, err := triangleLookup(name)
	if err != nil {
		tester.Fatal(err)
	}
	return value
}

// triangleRows are the rows of a csv file, or a stop of tester's.
func triangleRows(tester testing.TB, file string) sheet {
	tester.Helper()
	rows, err := loadSheet(file)
	if err != nil {
		tester.Fatal(err)
	}
	return rows
}

// triangleCell is the number in a column of a row, or a stop.
func triangleCell(tester testing.TB, row map[string]string,
	column string) float64 {
	tester.Helper()
	got, err := strconv.ParseFloat(row[column], floatBits)
	if err != nil {
		tester.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return got
}

// triangleWhole is the whole number in a column of a row, or a stop.
func triangleWhole(tester testing.TB, row map[string]string,
	column string) int {
	tester.Helper()
	got, err := strconv.ParseInt(row[column], prefixBase, floatBits)
	if err != nil {
		tester.Fatalf("%s of %q: %v", column, row["name"], err)
	}
	return int(got)
}

// triangleRow is the row of a csv file whose name column is name, or a
// stop.
func triangleRow(tester testing.TB, file, name string) map[string]string {
	tester.Helper()
	for _, each := range triangleRows(tester, file) {
		if each["name"] == name {
			return each
		}
	}
	tester.Fatalf("%s: no row called %q", file, name)
	return nil
}

// trianglePoint is the vector in the x, y and z columns of a row.
func trianglePoint(tester testing.TB, row map[string]string) maths.Vec {
	tester.Helper()
	return maths.V(triangleCell(tester, row, "x"),
		triangleCell(tester, row, "y"), triangleCell(tester, row, "z"))
}

// triangleMeshCorners are the corners of the mesh called named in a csv
// file with x, y and z columns, in the order of its rows.
func triangleMeshCorners(tester testing.TB, file, named string) []maths.Vec {
	tester.Helper()
	var corners []maths.Vec
	for _, row := range triangleRows(tester, file) {
		if row["mesh"] == named {
			corners = append(corners, trianglePoint(tester, row))
		}
	}
	return corners
}

// triangleFaces are the faces of the mesh called named in a csv file with
// first, second and third columns, in the order of its rows.
func triangleFaces(tester testing.TB, file,
	named string) []triangleCorners {
	tester.Helper()
	var faces []triangleCorners
	for _, row := range triangleRows(tester, file) {
		if row["mesh"] == named {
			faces = append(faces, triangleCorners{
				triangleWhole(tester, row, "first"),
				triangleWhole(tester, row, "second"),
				triangleWhole(tester, row, "third")})
		}
	}
	return faces
}

// triangleStream is the random stream seeded by the row called name in a
// csv file with high and low columns.
func triangleStream(tester testing.TB, file, name string) *rand.Rand {
	tester.Helper()
	row := triangleRow(tester, file, name)
	high := uint64(triangleWhole(tester, row, "high"))
	low := uint64(triangleWhole(tester, row, "low"))
	return rand.New(rand.NewPCG(high, low))
}

// triangleRay is the ray from at, going the way of way.
func triangleRay(at, way maths.Vec) maths.Ray {
	return maths.Ray{At: at, Dir: way}
}

// made is a mesh that must be right.
func made(tester testing.TB, vs []maths.Vec, fs []triangleCorners,
	ns []maths.Vec, smooth bool) *Mesh {
	tester.Helper()
	built, err := NewMesh(vs, fs, ns, smooth)
	if err != nil {
		tester.Fatal(err)
	}
	return built
}

// lone is the triangle on the floor of x and y, from the zero out one
// unit along each, turning counterclockwise seen from above.
func lone(tester testing.TB) *Mesh {
	corners := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(nought, unity, nought)}
	faces := triangleFaces(tester, triangleFacesFile, "lone")
	return made(tester, corners, faces, nil, false)
}

// down is a ray from above (x, y), straight down.
func down(x, y float64) maths.Ray {
	return triangleRay(maths.V(x, y, unity), maths.V(nought, nought, -unity))
}

// met is how many of a mesh's faces the ray meets, after near.
func met(built *Mesh, ray maths.Ray, near float64) int {
	asking, count := ready(ray), nought
	for i := range built.tris {
		if _, _, ok := asking.meet(&built.tris[i], near, forever); ok {
			count++
		}
	}
	return count
}

// TestATriangleIsMetInsideAndMissedOutside says that a ray down through the
// face meets it at one, and that rays outside it, or with the face out of
// the near and far asked, do not.
func TestATriangleIsMetInsideAndMissedOutside(t *testing.T) {
	built := lone(t)
	inside := picked(t, "triangle-inside-point")
	got, ok := built.Hit(down(inside, inside), nought, forever)
	if !ok || got != unity {
		t.Errorf("inside: hit at %v, %v; want 1", got, ok)
	}
	for _, row := range readSheet(t, triangleOutsideFile) {
		across, up := cell(t, row, "x"), cell(t, row, "y")
		got, ok := built.Hit(down(across, up), nought, forever)
		if ok {
			t.Errorf("outside at %s: hit at %v", row["name"], got)
		}
	}
	got, ok = built.Hit(down(inside, inside), unity, forever)
	if ok {
		t.Errorf("a hit no further than near counts: %v", got)
	}
	got, ok = built.Hit(down(inside, inside), nought, unity)
	if ok {
		t.Errorf("a hit no nearer than far counts: %v", got)
	}
}

// Both faces count: a ray from beneath meets the triangle as one from
// above does, and the test says which side it met.
func TestATriangleIsMetFromBehind(t *testing.T) {
	built := lone(t)
	inside := picked(t, "triangle-inside-point")
	under := triangleRay(maths.V(inside, inside, -unity),
		maths.V(nought, nought, unity))
	if got, ok := built.Hit(under, nought, forever); !ok || got != unity {
		t.Fatalf("from behind: hit at %v, %v; want 1", got, ok)
	}
	fromBeneath, fromAbove := ready(under), ready(down(inside, inside))
	face := &built.tris[triangleOnlyFace]
	if _, front, _ := fromBeneath.meet(face, nought, forever); front {
		t.Error("from beneath, the ray met the outside")
	}
	if _, front, _ := fromAbove.meet(face, nought, forever); !front {
		t.Error("from above, the ray met the inside")
	}
}

// A ray parallel to the face never meets it: above it, or in its own
// plane, where it sees the face edge on, with no area to meet.
func TestARayParallelToATriangleMissesIt(t *testing.T) {
	built := lone(t)
	across := picked(t, "triangle-parallel-across")
	diagonal := maths.V(unity, unity, nought).Unit()
	for _, ray := range []maths.Ray{
		triangleRay(maths.V(-unity, across, half), maths.XAxis),
		triangleRay(maths.V(-unity, across, nought), maths.XAxis),
		triangleRay(maths.V(-unity, -unity, nought), diagonal),
	} {
		if got, ok := built.Hit(ray, nought, forever); ok {
			t.Errorf("%v met it at %v", ray, got)
		}
	}
}

// A ray grazing the long edge a hair inside meets the triangle, and a
// hair outside does not; a ray that runs along the edge, in the plane,
// sees it edge on and misses.
func TestARayGrazingAnEdge(t *testing.T) {
	built := lone(t)
	hair := picked(t, "triangle-hair")
	step := picked(t, "triangle-graze-step")
	for where := step; where < unity; where += step {
		in := down(where-hair, unity-where-hair)
		out := down(where+hair, unity-where+hair)
		if _, ok := built.Hit(in, nought, forever); !ok {
			t.Errorf("a hair inside at %v missed", where)
		}
		if _, ok := built.Hit(out, nought, forever); ok {
			t.Errorf("a hair outside at %v met it", where)
		}
	}
	edgeway := maths.V(unity, -unity, nought).Unit()
	edge := triangleRay(maths.V(-unity, pair, nought), edgeway)
	if got, ok := built.Hit(edge, nought, forever); ok {
		t.Errorf("along the edge, met at %v", got)
	}
}

// slant is a way down at any slant up to about 35 degrees, or straight
// down for the first of many.
func slant(rng *rand.Rand, i int) maths.Vec {
	if i == nought {
		return maths.V(nought, nought, -unity)
	}
	return maths.V(rng.Float64()-half, rng.Float64()-half, -unity).Unit()
}

// The two halves of a square share its diagonal. A ray through any point
// of the diagonal, from either side and at any slant, meets one of them:
// never both and never neither.
func TestARayThroughASharedEdgeMeetsOneFace(t *testing.T) {
	corners := []maths.Vec{maths.V(nought, nought, nought),
		maths.V(unity, nought, nought), maths.V(unity, unity, nought),
		maths.V(nought, unity, nought)}
	faces := triangleFaces(t, triangleFacesFile, "square")
	built := made(t, corners, faces, nil, false)
	rng := triangleStream(t, triangleStreamsFile, "edge")
	randomFrom := pickedCount(t, "triangle-edge-random-from")
	cycle := pickedCount(t, "triangle-edge-cycle")
	divisor := picked(t, "triangle-edge-divisor")
	slants := pickedCount(t, "triangle-edge-slants")
	standoff := picked(t, "triangle-edge-standoff")
	for i := range pickedCount(t, "triangle-edge-rays") {
		// on the diagonal exactly, mostly
		share := float64(i%cycle+unity) / divisor
		if i >= randomFrom {
			share = rng.Float64()
		}
		on, way := maths.V(share, share, nought), slant(rng, i%slants)
		if i%pair == unity {
			way = way.Neg()
		}
		start := on.Sub(way.Scale(standoff))
		ray := triangleRay(start, way)
		if count := met(built, ray, nought); count != unity {
			t.Fatalf("a ray through %v going %v met %d faces",
				on, way, count)
		}
	}
}

// fan is six faces about a corner at the zero, raised by rise above its
// ring, turning counterclockwise seen from above; spokes are the ring.
func fan(t *testing.T, rise float64) (*Mesh, []maths.Vec) {
	t.Helper()
	faces := pickedCount(t, "triangle-fan-faces")
	divisor := picked(t, "triangle-fan-turn-divisor")
	vs := []maths.Vec{maths.V(nought, nought, rise)}
	var fs []triangleCorners
	for k := range faces {
		turn := float64(k) * math.Pi / divisor
		corner := maths.V(math.Cos(turn), math.Sin(turn), nought)
		vs = append(vs, corner)
		fs = append(fs, triangleCorners{nought, unity + k,
			unity + (k+unity)%faces})
	}
	return made(t, vs, fs, nil, false), vs[unity:]
}

// A ray through the corner six faces share meets exactly one of them, as
// does a ray through one of the edges between them.
func TestARayThroughASharedCornerMeetsOneFace(t *testing.T) {
	rng := triangleStream(t, triangleStreamsFile, "corner")
	spokeCount := pickedCount(t, "triangle-fan-faces")
	standoff := picked(t, "triangle-corner-standoff")
	for _, row := range readSheet(t, triangleRisesFile) {
		rise := cell(t, row, "rise")
		built, spokes := fan(t, rise)
		corner := maths.V(nought, nought, rise)
		for i := range pickedCount(t, "triangle-corner-rays") {
			at := corner
			if i%pair == unity { // a way along a spoke
				end := spokes[rng.IntN(spokeCount)].Sub(corner)
				share := rng.Float64()
				at = corner.Add(end.Scale(share))
			}
			way := slant(rng, i)
			start := at.Sub(way.Scale(standoff))
			ray := triangleRay(start, way)
			if count := met(built, ray, nought); count != unity {
				t.Fatalf("rise %v: a ray through %v going %v "+
					"met %d faces", rise, at, way, count)
			}
		}
	}
}
