package island

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/terrain"
)

// slope is a height field that rises a hundred metres a degree east and
// fifty a degree north, so no two lattice points share a height.
func slope(longitude, latitude float64) (float64, bool, error) {
	return 500 + 100*longitude + 50*latitude, true, nil
}

// slopeAt is the height slope gives at a lattice key.
func slopeAt(column, row int) float64 {
	height, _, _ := slope(-2+float64(column)/16*4, 2-float64(row)/16*4)
	return height
}

// slopeWorld is the fixed-frame position of a lattice key on the slope.
func slopeWorld(t *testing.T, column, row int) maths.Vec {
	t.Helper()
	return world(t, column, row, slopeAt(column, row))
}

// midway is the point halfway between two points.
func midway(one, other maths.Vec) maths.Vec { return one.Add(other.Sub(one).Scale(0.5)) }

// bookFor is an empty vertex book over leaves, with normals if the detail
// asks for them.
func bookFor(ground *Terrain, leaves []terrain.Patch) *vertexBook {
	book := &vertexBook{terrain: ground, points: map[lattice]maths.Vec{}, edges: newEdgeIndex(leaves)}
	if ground.detail.NormalLongitude > 0 {
		book.normals = map[lattice]maths.Vec{}
	}
	return book
}

// TestCoarserPicksTheLowestLevelThenTheEarliestLeaf checks the edge index
// on a corner where leaves of two levels meet and where leaves of one level
// meet in either order.
func TestCoarserPicksTheLowestLevelThenTheEarliestLeaf(t *testing.T) {
	lowFirst := newEdgeIndex([]terrain.Patch{
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1},
		{MinColumn: 4, MaxColumn: 8, MinRow: 4, MaxRow: 8, Level: 1},
	})
	highFirst := newEdgeIndex([]terrain.Patch{
		{MinColumn: 4, MaxColumn: 8, MinRow: 4, MaxRow: 8, Level: 1},
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1},
	})
	mixed := newEdgeIndex([]terrain.Patch{
		{MinColumn: 4, MaxColumn: 8, MinRow: 4, MaxRow: 8, Level: 2},
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1},
	})
	corner := lattice{4, 4}
	for name, each := range map[string]struct {
		index edgeIndex
		want  int
	}{"earlier leaf found second": {lowFirst, 0}, "earlier leaf found first": {highFirst, 0}, "lower level wins": {mixed, 1}} {
		got := each.index.coarser(corner, 3)
		if got == nil || *got != each.index.leaves[each.want] {
			t.Errorf("%s: got %v, wanted leaf %d", name, got, each.want)
		}
	}
	if got := lowFirst.coarser(corner, 1); got != nil {
		t.Errorf("a leaf of the same level counted as coarser: %v", got)
	}
	if got := lowFirst.coarser(lattice{2, 2}, 5); got != nil {
		t.Errorf("a point on no edge found %v", got)
	}
	if got := lowFirst.coarser(lattice{4, 6}, 5); got == nil || *got != lowFirst.leaves[1] {
		t.Errorf("a point on one leaf's edge found %v", got)
	}
	if got := lowFirst.coarser(lattice{4, 9}, 5); got != nil {
		t.Errorf("a point past a leaf's end found %v", got)
	}
	if got := lowFirst.coarser(lattice{9, 4}, 5); got != nil {
		t.Errorf("a point past a leaf's end found %v", got)
	}
}

// TestConformBlendsAlongACoarseEdge checks stitched points worked by hand:
// a fine knot halfway between two coarse knots takes their midpoint, whether
// the coarse edge runs down a column or along a row, and a fine knot that is
// also a coarse knot keeps its own height.
func TestConformBlendsAlongACoarseEdge(t *testing.T) {
	ground := testTerrain(t, heightFunc(slope), testDetail())
	book := bookFor(ground, []terrain.Patch{
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1},
		{MinColumn: 4, MaxColumn: 6, MinRow: 0, MaxRow: 2, Level: 2},
		{MinColumn: 0, MaxColumn: 2, MinRow: 4, MaxRow: 6, Level: 2},
	})
	cases := []struct {
		name string
		key  lattice
		want maths.Vec
	}{
		{"down a column", lattice{4, 1}, midway(slopeWorld(t, 4, 0), slopeWorld(t, 4, 2))},
		{"along a row", lattice{1, 4}, midway(slopeWorld(t, 0, 4), slopeWorld(t, 2, 4))},
		{"on a coarse knot in a column", lattice{4, 2}, slopeWorld(t, 4, 2)},
		{"on a coarse knot in a row", lattice{2, 4}, slopeWorld(t, 2, 4)},
		{"away from every edge", lattice{5, 1}, slopeWorld(t, 5, 1)},
	}
	for _, each := range cases {
		got, err := book.conform(each.key, 2)
		if err != nil || !near(got, each.want, 1e-6) {
			t.Errorf("%s: got %v, %v; wanted %v", each.name, got, err, each.want)
		}
	}
}

// TestConformFollowsCoarserEdgesInTurn checks a fine knot whose coarse
// neighbour's own end knot lies on a still coarser edge, so the blend is
// taken twice: half of the midpoint of two level-zero knots and a level-one
// knot.
func TestConformFollowsCoarserEdgesInTurn(t *testing.T) {
	ground := testTerrain(t, heightFunc(slope), testDetail())
	book := bookFor(ground, []terrain.Patch{
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 8, Level: 0},
		{MinColumn: 4, MaxColumn: 8, MinRow: 1, MaxRow: 5, Level: 1},
		{MinColumn: 4, MaxColumn: 6, MinRow: 1, MaxRow: 3, Level: 2},
	})
	edge := midway(slopeWorld(t, 4, 0), slopeWorld(t, 4, 2))
	want := midway(edge, slopeWorld(t, 6, 1))
	got, err := book.conform(lattice{5, 1}, 2)
	if err != nil || !near(got, want, 1e-6) {
		t.Fatalf("got %v, %v; wanted %v", got, err, want)
	}
}

// TestConformReportsASourceFailureAtEitherEnd makes the source fail at the
// first and then at the last knot of a coarse segment.
func TestConformReportsASourceFailureAtEitherEnd(t *testing.T) {
	sentinel := errors.New("no height here")
	leaves := []terrain.Patch{
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1},
		{MinColumn: 4, MaxColumn: 6, MinRow: 0, MaxRow: 2, Level: 2},
	}
	for name, badRow := range map[string]int{"first knot": 0, "last knot": 2} {
		failing := heightFunc(func(longitude, latitude float64) (float64, bool, error) {
			if longitude == -1 && latitude == 2-float64(badRow)/4 {
				return 0, false, sentinel
			}
			return slope(longitude, latitude)
		})
		book := bookFor(testTerrain(t, failing, testDetail()), leaves)
		if _, err := book.conform(lattice{4, 1}, 2); !errors.Is(err, sentinel) {
			t.Errorf("%s: error %v", name, err)
		}
	}
}

// TestSurfaceStitchesAFinePatchToACoarseOne checks the mesh of a fine patch
// beside a coarse one: its middle knot on the shared edge lies halfway
// between the coarse knots and the rest are plain source points, all taken
// from the render origin.
func TestSurfaceStitchesAFinePatchToACoarseOne(t *testing.T) {
	ground := testTerrain(t, heightFunc(slope), testDetail())
	fine := terrain.Patch{MinColumn: 4, MaxColumn: 6, MinRow: 0, MaxRow: 2, Level: 2}
	book := bookFor(ground, []terrain.Patch{
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1}, fine})
	origin := maths.V(10, 20, 30)
	surface, err := book.surface(fine, originAt(t, origin))
	if err != nil {
		t.Fatal(err)
	}
	want := []maths.Vec{
		slopeWorld(t, 4, 0), slopeWorld(t, 5, 0), slopeWorld(t, 6, 0),
		midway(slopeWorld(t, 4, 0), slopeWorld(t, 4, 2)), slopeWorld(t, 5, 1), slopeWorld(t, 6, 1),
		slopeWorld(t, 4, 2), slopeWorld(t, 5, 2), slopeWorld(t, 6, 2),
	}
	got := surface.Points()
	if len(got) != len(want) || surface.FaceCount() != 8 || surface.Smooth() {
		t.Fatalf("%d points, %d faces, smooth %v", len(got), surface.FaceCount(), surface.Smooth())
	}
	for index := range want {
		if !near(got[index], want[index].Sub(origin), 1e-6) {
			t.Errorf("point %d: %v, wanted %v", index, got[index], want[index].Sub(origin))
		}
	}
	if faces := surface.Faces(); faces[0] != [3]int{0, 4, 1} || faces[1] != [3]int{0, 3, 4} {
		t.Errorf("first faces %v", faces[:2])
	}
}

// TestSurfaceBlendsShadingNormalsAcrossAStitch checks, on flat sea level
// where each normal points straight away from the sphere's centre, that the
// stitched knot takes the direction halfway between its two coarse knots.
func TestSurfaceBlendsShadingNormalsAcrossAStitch(t *testing.T) {
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	ground := testTerrain(t, flat(0), detail)
	fine := terrain.Patch{MinColumn: 4, MaxColumn: 6, MinRow: 0, MaxRow: 2, Level: 2}
	book := bookFor(ground, []terrain.Patch{
		{MinColumn: 0, MaxColumn: 4, MinRow: 0, MaxRow: 4, Level: 1}, fine})
	surface, err := book.surface(fine, originAt(t, maths.Vec{}))
	if err != nil {
		t.Fatal(err)
	}
	if !surface.Smooth() {
		t.Fatal("a mesh with normals was shaded flat")
	}
	centre := maths.V(0, -1e6, 0)
	outward := func(column, row int) maths.Vec { return world(t, column, row, 0).Sub(centre).Unit() }
	want := midway(outward(4, 0), outward(4, 2)).Unit()
	// Face 1 runs near, far, far-plus-one, so its second corner is index 3,
	// the stitched knot.
	normals, smooth := surface.CornerNormals(1)
	if !smooth || !near(normals[1], want, 1e-6) {
		t.Fatalf("stitched normal %v, wanted %v", normals[1], want)
	}
	if !near(normals[0], outward(4, 0), 1e-6) {
		t.Fatalf("corner normal %v, wanted %v", normals[0], outward(4, 0))
	}
}

// TestSurfaceReportsASourceFailure fails the source at one knot and expects
// the surface to report that failure, for both a plain read and a stencil.
func TestSurfaceReportsASourceFailure(t *testing.T) {
	sentinel := errors.New("source gone")
	patch := terrain.Patch{MinColumn: 4, MaxColumn: 6, MinRow: 0, MaxRow: 2, Level: 2}
	failing := heightFunc(func(longitude, latitude float64) (float64, bool, error) {
		return 0, false, sentinel
	})
	book := bookFor(testTerrain(t, failing, testDetail()), []terrain.Patch{patch})
	if _, err := book.surface(patch, originAt(t, maths.Vec{})); !errors.Is(err, sentinel) {
		t.Fatalf("plain read: error %v", err)
	}
	// Lattice latitudes are quarter degrees; the stencil steps a tenth, so
	// only the stencil's north and south reads fail.
	offLattice := heightFunc(func(longitude, latitude float64) (float64, bool, error) {
		if math.Abs(latitude*4-math.Round(latitude*4)) > 1e-9 {
			return 0, false, sentinel
		}
		return 0, true, nil
	})
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	book = bookFor(testTerrain(t, offLattice, detail), []terrain.Patch{patch})
	if _, err := book.surface(patch, originAt(t, maths.Vec{})); !errors.Is(err, sentinel) {
		t.Fatalf("stencil: error %v", err)
	}
}

// TestNormalAtPointsAwayFromTheCentre checks that over flat sea level at
// the anchor the normal is straight up, and that a second ask for the same
// key reads no more heights.
func TestNormalAtPointsAwayFromTheCentre(t *testing.T) {
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	ground := testTerrain(t, flat(0), detail)
	book := bookFor(ground, nil)
	normal, err := book.normalAt(lattice{8, 8})
	if err != nil || !near(normal, maths.V(0, 1, 0), 1e-9) {
		t.Fatalf("normal %v, %v", normal, err)
	}
	if book.normalQueries != 4 {
		t.Fatalf("%d queries for one stencil", book.normalQueries)
	}
	again, err := book.normalAt(lattice{8, 8})
	if err != nil || again != normal || book.normalQueries != 4 {
		t.Fatalf("second ask: %v, %v, %d queries", again, err, book.normalQueries)
	}
	north, err := book.normalAt(lattice{8, 0})
	want := world(t, 8, 0, 0).Sub(maths.V(0, -1e6, 0)).Unit()
	if err != nil || !near(north, want, 1e-9) {
		t.Fatalf("normal at lattice latitude two: %v, %v; wanted %v", north, err, want)
	}
}

// TestNormalAtRefusals checks each way a stencil can be refused.
func TestNormalAtRefusals(t *testing.T) {
	sentinel := errors.New("stencil read failed")
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	source := func(height float64, has bool, err error) Heights {
		return heightFunc(func(float64, float64) (float64, bool, error) { return height, has, err })
	}
	tiny := detail
	tiny.NormalLongitude, tiny.NormalLatitude = 1e-320, 1e-320
	wide := detail
	wide.NormalLongitude = 179
	short := detail
	short.Points = 9
	cases := []struct {
		name   string
		from   Heights
		detail Detail
		key    lattice
		want   string
	}{
		{"source error", source(0, false, sentinel), detail, lattice{8, 8}, "stencil read failed"},
		{"absent height", source(0, false, nil), detail, lattice{8, 8}, "finite source heights"},
		{"height above the envelope", source(5001, true, nil), detail, lattice{8, 8}, "finite source heights"},
		{"height below the envelope", source(-1001, true, nil), detail, lattice{8, 8}, "finite source heights"},
		{"height not a number", source(math.NaN(), true, nil), detail, lattice{8, 8}, "finite source heights"},
		{"step past the antimeridian", source(0, true, nil), wide, lattice{16, 8}, "invalid spherical location"},
		{"step too small to move", source(0, true, nil), tiny, lattice{8, 8}, "degenerate source normal"},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			book := bookFor(testTerrain(t, each.from, each.detail), nil)
			if _, err := book.normalAt(each.key); err == nil || !strings.Contains(err.Error(), each.want) {
				t.Fatalf("error %v, wanted one containing %q", err, each.want)
			}
			if len(book.normals) != 0 {
				t.Fatalf("a refused stencil was kept")
			}
		})
	}
	book := bookFor(testTerrain(t, flat(0), short), nil)
	for column := range 9 {
		if _, err := book.normalAt(lattice{column, 8}); err != nil {
			t.Fatalf("stencil %d: %v", column, err)
		}
	}
	if _, err := book.normalAt(lattice{9, 8}); err == nil || !strings.Contains(err.Error(), "budget exhausted") {
		t.Fatalf("tenth stencil: error %v", err)
	}
}
