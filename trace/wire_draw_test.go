//go:build draw

package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// the wire tests' numbers
const (
	// wireSide is the test pictures' pixels each way
	wireSide = 64
	// wireClose is how near a pixel's light is to the light wanted,
	// past the float32 a frame keeps
	wireClose = 1e-6
	// leftEdge, lowEdge and slantEdge place the test triangle in
	// pixels: its upright edge at x = leftEdge, its level edge at
	// y = lowEdge, and its slant where x - y is slantEdge. Each is a
	// quarter pixel off the pixel centres, so a centre is never on one.
	leftEdge  = 15.75
	lowEdge   = 48.25
	slantEdge = 0.25 * math.Sqrt2
)

// wireLens sees a right angle across, so the plane at depth one spans
// from minus one to one.
var wireLens = film.Pinhole{Wide: math.Pi / 2}

// wireEye stands at the origin and looks along minus z.
var wireEye = maths.Subspace{Orientation: maths.Identity}

// onScreen is the point at a depth that a place in the picture, in
// pixels with centres on whole numbers, looks through.
func onScreen(across, down, depth float64) maths.Vec {
	u := 2*(across+0.5)/wireSide - 1
	v := 1 - 2*(down+0.5)/wireSide
	return maths.V(u*depth, v*depth, -depth)
}

// screenMesh is slayer's mesh of faces given by places in the picture,
// at a depth, so each lies over those places exactly.
func screenMesh(t *testing.T, depth float64, places [][2]float64,
	faces [][3]int) thing.Thing {
	t.Helper()
	corners := make([]maths.Vec, len(places))
	for i, place := range places {
		corners[i] = onScreen(place[0], place[1], depth)
	}
	made, err := mesh.New(corners, faces, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return thing.Thing{Shape: made,
		Motion: thing.Still(maths.Subspace{Orientation: maths.Identity})}
}

// triangle is the test triangle at a depth, moved some pixels right.
func triangle(t *testing.T, depth, moved float64) thing.Thing {
	return screenMesh(t, depth, [][2]float64{
		{leftEdge + moved, lowEdge},
		{slantEdge + lowEdge + moved, lowEdge},
		{leftEdge + moved, leftEdge - slantEdge}}, [][3]int{{0, 1, 2}})
}

// triangleWire is the light the wire of the unmoved triangle gives the
// pixel at x, y, and whether the triangle covers its centre: the face's
// faint light, and the edge's within a pixel of an edge, by how near.
func triangleWire(x, y int) (float64, bool) {
	across, down := float64(x), float64(y)
	gap := min(across-leftEdge, lowEdge-down,
		(slantEdge-(across-down))/math.Sqrt2)
	if gap <= 0 {
		return 0, false
	}
	return wireFace + (wireLine-wireFace)*min(max(1-gap, 0), 1), true
}

// wired is the wire picture of things, one ray a pixel.
func wired(t *testing.T, things ...thing.Thing) *frame.Frame {
	t.Helper()
	world := thing.NewWorld()
	for _, each := range things {
		world.Add(each)
	}
	drawn, err := Wire(world, wireEye, wireLens, 0, wireSide, wireSide, 1)
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// misses is how many pixels of a wire are not the unmoved triangle's.
func misses(drawn *frame.Frame) int {
	missed := 0
	for y := range wireSide {
		for x := range wireSide {
			want, covered := triangleWire(x, y)
			got := drawn.At(x, y)
			if covered != (got.Cover == 1) ||
				math.Abs(float64(got.R)-want) > wireClose {
				missed++
			}
		}
	}
	return missed
}

// TestWireDrawsATrianglesEdgesWhereTheyAre holds that every pixel of
// one triangle's wire is the light its distance to the triangle's
// edges gives, measured in pixels on the picture, and that there are
// edges: the lines lie on the triangle's edges and nowhere else.
func TestWireDrawsATrianglesEdgesWhereTheyAre(t *testing.T) {
	drawn := wired(t, triangle(t, 1, 0))
	if missed := misses(drawn); missed > 0 {
		t.Fatalf("%d pixels are not the triangle's wire", missed)
	}
	lined := 0
	for y := range wireSide {
		for x := range wireSide {
			if drawn.At(x, y).R > wireLine/2 {
				lined++
			}
		}
	}
	if lined < wireSide {
		t.Errorf("only %d pixels are edges", lined)
	}
}

// anti-test of TestWireDrawsATrianglesEdgesWhereTheyAre
// TestWireLinesMoveWithTheTriangle holds that a triangle a pixel to the
// right is not drawn where the unmoved one is: the check above can
// tell a line out of place.
func TestWireLinesMoveWithTheTriangle(t *testing.T) {
	if misses(wired(t, triangle(t, 1, 1))) < wireSide/2 {
		t.Error("a moved triangle's wire passes for the unmoved one")
	}
}

// quad is a face pair at depth one over the picture's left half, from
// x = -200 to just short of the middle, its diagonal far off the
// picture.
func quad(t *testing.T) thing.Thing {
	const left, right, top, bottom = -200, 31.75, -200, 280
	return screenMesh(t, 1, [][2]float64{{left, top}, {right, top},
		{right, bottom}, {left, bottom}}, [][3]int{{0, 1, 2}, {0, 2, 3}})
}

// hidden is the pixels of the triangle's upright edge, column 16, that
// the quad stands in front of.
func hidden() [][2]int {
	var places [][2]int
	for y := 20; y <= 46; y++ {
		places = append(places, [2]int{16, y})
	}
	return places
}

// TestWireHidesEdgesBehindANearerQuad holds that a triangle's edge
// behind a nearer quad is not drawn: the quad's face is, and the
// triangle's level edge beside the quad still is.
func TestWireHidesEdgesBehindANearerQuad(t *testing.T) {
	drawn := wired(t, triangle(t, 2, 0), quad(t))
	for _, place := range hidden() {
		got := drawn.At(place[0], place[1])
		if math.Abs(float64(got.R)-wireFace) > wireClose {
			t.Fatalf("at %v the hidden edge shows: %v", place, got.R)
		}
	}
	for x := 34; x <= 46; x++ {
		want, _ := triangleWire(x, 48)
		if got := drawn.At(x, 48).R; math.Abs(float64(got)-want) >
			wireClose {
			t.Errorf("at %d, 48 the edge beside the quad is %v, not %v",
				x, got, want)
		}
	}
}

// anti-test of TestWireHidesEdgesBehindANearerQuad
// TestWireShowsTheEdgeWithNoQuadInFront holds that the same pixels,
// with no quad, are the triangle's edge: what the quad hid was there.
func TestWireShowsTheEdgeWithNoQuadInFront(t *testing.T) {
	drawn := wired(t, triangle(t, 2, 0))
	for _, place := range hidden() {
		want, _ := triangleWire(place[0], place[1])
		got := drawn.At(place[0], place[1]).R
		if want <= wireFace || math.Abs(float64(got)-want) > wireClose {
			t.Fatalf("at %v the edge is %v, not %v", place, got, want)
		}
	}
}

// ball is a sphere of radius a half at depth two.
func ball() thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: 0.5},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -2),
			Orientation: maths.Identity})}
}

// analytics is how many pixels of a wire are covered, and how many of
// those are not the analytic tint.
func analytics(drawn *frame.Frame) (covered, other int) {
	for y := range wireSide {
		for x := range wireSide {
			got := drawn.At(x, y)
			if got.Cover == 0 {
				continue
			}
			covered++
			if got.Light() != pixelOf(wireAnalytic, 1, 0).Light() {
				other++
			}
		}
	}
	return covered, other
}

// TestWireDrawsASphereFlatWithNoEdges holds that a sphere is the
// analytic tint over every pixel it covers, with no edge made up.
func TestWireDrawsASphereFlatWithNoEdges(t *testing.T) {
	covered, other := analytics(wired(t, ball()))
	if covered < wireSide || other > 0 {
		t.Errorf("the sphere covers %d pixels, %d of them not flat",
			covered, other)
	}
}

// anti-test of TestWireDrawsASphereFlatWithNoEdges
// TestWireDoesNotDrawAMeshAsAnalytic holds that a mesh is never the
// analytic tint: the check above tells the two apart.
func TestWireDoesNotDrawAMeshAsAnalytic(t *testing.T) {
	covered, other := analytics(wired(t, triangle(t, 1, 0)))
	if covered == 0 || other != covered {
		t.Errorf("of %d mesh pixels, only %d are not analytic", covered,
			other)
	}
}

// field is a flat height field of three samples a side, two across, at
// depth two, turned to face the eye: its cells' edges are at x and y of
// minus one, nought and one.
func field(t *testing.T) thing.Thing {
	t.Helper()
	flat, err := shape.NewHeightField(make([]float64, 9), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	return thing.Thing{Shape: flat, Motion: thing.Still(maths.Subspace{
		Origin:      maths.V(0, 0, -2),
		Orientation: maths.About(maths.XAxis, math.Pi/2)})}
}

// TestWireDrawsAHeightFieldsCells holds that a height field's middle
// cell edge, half a pixel from the centres either side of it, is half
// drawn on each, and that a pixel inside a cell is the analytic tint.
func TestWireDrawsAHeightFieldsCells(t *testing.T) {
	drawn := wired(t, field(t))
	half := wireAnalytic.Scale(0.5).Add(wireCell.Scale(0.5))
	for _, x := range []int{31, 32} {
		got := drawn.At(x, 23).Light()
		apart := got.Add(half.Scale(-1))
		if max(math.Abs(apart.R), math.Abs(apart.G),
			math.Abs(apart.B)) > wireClose {
			t.Errorf("at %d, 23 the cell edge is %v, not %v", x, got, half)
		}
	}
	inside := drawn.At(40, 23)
	if inside.Light() != pixelOf(wireAnalytic, 1, 0).Light() {
		t.Errorf("inside a cell the field is %v", inside.Light())
	}
}

// anti-test of TestWireDrawsAHeightFieldsCells
// TestWireDrawsNoCellsOnASphere holds that the same pixels of a sphere
// as wide as the field have no cell edge: the edges are the field's.
func TestWireDrawsNoCellsOnASphere(t *testing.T) {
	drawn := wired(t, thing.Thing{Shape: shape.Sphere{Radius: 1.5},
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -3),
			Orientation: maths.Identity})})
	for _, x := range []int{31, 32} {
		if got := drawn.At(x, 23); got.Light() !=
			pixelOf(wireAnalytic, 1, 0).Light() {
			t.Errorf("at %d, 23 the sphere is %v", x, got.Light())
		}
	}
}

// TestWireRefusesRaysAFieldBends holds that a wire whose rays enter a
// gravitational field before any surface is refused, not drawn straight.
func TestWireRefusesRaysAFieldBends(t *testing.T) {
	world := thing.NewWorld()
	world.Add(ball())
	world.Fields = []*thing.Field{{Mass: 0.01, Reach: 1,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -1.2),
			Orientation: maths.Identity})}}
	_, err := Wire(world, wireEye, wireLens, 0, wireSide, wireSide, 1)
	if err != errWireBent {
		t.Fatalf("a bent wire gave %v", err)
	}
}

// anti-test of TestWireRefusesRaysAFieldBends
// TestWireDrawsAFieldTheRaysMissOrMeetBehind holds that a field behind
// every surface its rays meet bends nothing the wire draws, and the
// wire is drawn.
func TestWireDrawsAFieldTheRaysMissOrMeetBehind(t *testing.T) {
	world := thing.NewWorld()
	world.Add(triangle(t, 1, 0))
	world.Fields = []*thing.Field{{Mass: 0.0001, Reach: 0.01,
		Motion: thing.Still(maths.Subspace{Origin: maths.V(0, 0, -50),
			Orientation: maths.Identity})}}
	if _, err := Wire(world, wireEye, wireLens, 0, wireSide, wireSide,
		1); err != nil {
		t.Fatal(err)
	}
}
