package island

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/footprint"
	"github.com/janearc/slayer/scene/frame"
	"github.com/janearc/slayer/terrain"
)

// world is the fixed-frame position of a lattice key at a height, worked
// from the four degree region: longitude steps a quarter degree a column
// and latitude falls a quarter degree a row.
func world(t *testing.T, column, row int, height float64) maths.Vec {
	t.Helper()
	longitude := -2 + float64(column)/16*4
	latitude := 2 - float64(row)/16*4
	point, err := testSphere(t).Position(frame.Location{
		Longitude: measure.Degrees(longitude), Latitude: measure.Degrees(latitude),
		Height: measure.Metres(height)})
	if err != nil {
		t.Fatal(err)
	}
	return point
}

// near reports whether two points agree to a tolerance in metres.
func near(one, other maths.Vec, tolerance float64) bool {
	return one.Sub(other).Len() <= tolerance
}

// rangeHeights is a source that also reports a height range.
type rangeHeights struct {
	heightFunc
	low, high float64
	has       bool
	err       error
	calls     *int
}

// HeightRange returns the configured answer.
func (source rangeHeights) HeightRange(float64, float64, float64, float64) (float64, float64, bool, error) {
	if source.calls != nil {
		*source.calls++
	}
	return source.low, source.high, source.has, source.err
}

// TestNewRefusesBadInputs gives New each kind of bad frame, region and
// detail and expects a refusal for every one.
func TestNewRefusesBadInputs(t *testing.T) {
	good := testDetail()
	sphere := testSphere(t)
	cases := []struct {
		name   string
		sphere frame.Sphere
		from   Heights
		region Region
		edit   func(*Detail)
	}{
		{"no source", sphere, nil, testRegion, nil},
		{"zero sphere", frame.Sphere{}, flat(0), testRegion, nil},
		{"west past east", sphere, flat(0), Region{West: 1, South: -1, East: 0, North: 1}, nil},
		{"south past north", sphere, flat(0), Region{West: -1, South: 1, East: 1, North: 0}, nil},
		{"west of the world", sphere, flat(0), Region{West: -181, South: -1, East: 1, North: 1}, nil},
		{"north of the pole", sphere, flat(0), Region{West: -1, South: -1, East: 1, North: 91}, nil},
		{"not a number", sphere, flat(0), Region{West: math.NaN(), South: -1, East: 1, North: 1}, nil},
		{"side two", sphere, flat(0), testRegion, func(d *Detail) { d.Side = 2 }},
		{"side four", sphere, flat(0), testRegion, func(d *Detail) { d.Side = 4 }},
		{"side sixty-seven", sphere, flat(0), testRegion, func(d *Detail) { d.Side = 67 }},
		{"negative depth", sphere, flat(0), testRegion, func(d *Detail) { d.Depth = -1 }},
		{"deep depth", sphere, flat(0), testRegion, func(d *Detail) { d.Depth = 21 }},
		{"no patches", sphere, flat(0), testRegion, func(d *Detail) { d.Patches = 0 }},
		{"no evaluations", sphere, flat(0), testRegion, func(d *Detail) { d.Evaluations = 0 }},
		{"points under a side", sphere, flat(0), testRegion, func(d *Detail) { d.Points = 2 }},
		{"points under a lattice", sphere, flat(0), testRegion, func(d *Detail) { d.Points = 8 }},
		{"no cell pixels", sphere, flat(0), testRegion, func(d *Detail) { d.CellPixels = 0 }},
		{"infinite height", sphere, flat(0), testRegion, func(d *Detail) { d.MaximumHeight = math.Inf(1) }},
		{"flat shading step on one axis", sphere, flat(0), testRegion, func(d *Detail) { d.NormalLongitude = 1 }},
		{"negative step", sphere, flat(0), testRegion, func(d *Detail) { d.NormalLongitude, d.NormalLatitude = -1, -1 }},
		{"step past the world", sphere, flat(0), testRegion, func(d *Detail) { d.NormalLongitude, d.NormalLatitude = 181, 1 }},
		{"step not a number", sphere, flat(0), testRegion, func(d *Detail) { d.NormalLongitude, d.NormalLatitude = math.NaN(), 1 }},
		{"envelope upside down", sphere, flat(0), testRegion, func(d *Detail) { d.MinimumHeight, d.MaximumHeight = 5, -5 }},
		{"ground below the centre", sphere, flat(0), testRegion, func(d *Detail) { d.MinimumHeight = -2e6 }},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			detail := good
			if each.edit != nil {
				each.edit(&detail)
			}
			ground, err := New(each.sphere, each.from, each.region, detail)
			if err == nil || ground != nil || !strings.HasPrefix(err.Error(), "island: ") {
				t.Fatalf("got %v, %v; wanted a refusal", ground, err)
			}
		})
	}
}

// TestNewSpansTheLatticeAtItsDepth checks that depth three of a three by
// three lattice makes a root patch sixteen columns and rows across.
func TestNewSpansTheLatticeAtItsDepth(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	if want := (terrain.Patch{MaxColumn: 16, MaxRow: 16}); ground.root != want {
		t.Fatalf("root %+v, wanted %+v", ground.root, want)
	}
}

// TestLocationIsAnchoredToTheRegion checks worked longitudes and latitudes
// at the corners and the middle of the four degree region.
func TestLocationIsAnchoredToTheRegion(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	for _, each := range []struct{ column, row, longitude, latitude float64 }{
		{0, 0, -2, 2}, {16, 16, 2, -2}, {8, 4, 0, 1}, {4, 12, -1, -1},
	} {
		longitude, latitude := ground.location(each.column, each.row)
		if longitude != each.longitude || latitude != each.latitude {
			t.Errorf("location(%g, %g) = %g, %g; wanted %g, %g", each.column, each.row,
				longitude, latitude, each.longitude, each.latitude)
		}
	}
}

// TestPrepareFromFarAwayIsOneFlatPatch checks that a distant camera gets
// the root patch, sampled at its nine lattice points.
func TestPrepareFromFarAwayIsOneFlatPatch(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	origin := originAt(t, maths.V(0, 2e7, 0))
	prepared, err := ground.Prepare(context.Background(), lookDown(t, 2e7), origin)
	if err != nil {
		t.Fatal(err)
	}
	if want := (terrain.Patch{MaxColumn: 16, MaxRow: 16}); len(prepared.Patches) != 1 || prepared.Patches[0] != want {
		t.Fatalf("patches %+v", prepared.Patches)
	}
	if prepared.Samples != 9 || prepared.Limited || prepared.Transition != nil ||
		prepared.NormalStencils != 0 || prepared.Origin != (maths.V(0, 2e7, 0)) {
		t.Fatalf("prepared %+v", prepared)
	}
	surface := prepared.Surfaces[0]
	if surface.FaceCount() != 8 || surface.Smooth() {
		t.Fatalf("%d faces, smooth %v", surface.FaceCount(), surface.Smooth())
	}
	// The first corner is the region's north-west corner at sea level, seen
	// from the render origin.
	got := surface.Points()[0]
	want := world(t, 0, 0, 0).Sub(maths.V(0, 2e7, 0))
	if !near(got, want, 1e-6) {
		t.Fatalf("first corner %v, wanted %v", got, want)
	}
}

// TestPrepareRefinesAsTheCameraDescends checks the sample counts worked by
// hand: four level-one patches share a five by five lattice of twenty-five
// samples, and a patch budget of sixteen leaves the closer view limited.
func TestPrepareRefinesAsTheCameraDescends(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	medium, err := ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if len(medium.Patches) != 4 || len(medium.Surfaces) != 4 || medium.Samples != 25 || medium.Limited {
		t.Fatalf("medium: %d patches, %d samples, limited %v", len(medium.Patches), medium.Samples, medium.Limited)
	}
	for _, patch := range medium.Patches {
		if patch.Level != 1 || patch.MaxColumn-patch.MinColumn != 8 {
			t.Fatalf("patch %+v is not a level-one quarter", patch)
		}
	}
	close, err := ground.Prepare(context.Background(), lookDown(t, 3e5), originAt(t, maths.V(0, 3e5, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if len(close.Patches) != 16 || !close.Limited || close.Samples != 81 {
		t.Fatalf("close: %d patches, %d samples, limited %v", len(close.Patches), close.Samples, close.Limited)
	}
}

// TestPrepareCountsNormalStencils checks that smooth shading reads four
// heights for each of the nine samples and shades the mesh smooth.
func TestPrepareCountsNormalStencils(t *testing.T) {
	detail := testDetail()
	detail.NormalLongitude, detail.NormalLatitude = 0.1, 0.1
	ground := testTerrain(t, flat(0), detail)
	prepared, err := ground.Prepare(context.Background(), lookDown(t, 2e7), originAt(t, maths.V(0, 2e7, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.NormalStencils != 9 || prepared.NormalHeightQueries != 36 || !prepared.Surfaces[0].Smooth() {
		t.Fatalf("stencils %d, queries %d", prepared.NormalStencils, prepared.NormalHeightQueries)
	}
}

// TestPrepareOfAViewLookingAwayIsEmpty checks that a camera facing away
// from the region selects nothing and reports no error.
func TestPrepareOfAViewLookingAwayIsEmpty(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	away := maths.Subspace{Origin: maths.V(0, 1e6, 0), Orientation: maths.Orientation{
		X: maths.V(1, 0, 0), Y: maths.V(0, 0, 1), Z: maths.V(0, -1, 0)}}
	view, err := footprintFor(away)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := ground.Prepare(context.Background(), view, originAt(t, maths.V(0, 1e6, 0)))
	if err != nil || len(prepared.Patches) != 0 || len(prepared.Surfaces) != 0 || prepared.Samples != 0 {
		t.Fatalf("prepared %+v, %v", prepared, err)
	}
}

// TestPrepareRefusals checks each way a view of the terrain is refused.
func TestPrepareRefusals(t *testing.T) {
	sentinel := errors.New("source failed")
	view := lookDown(t, 1e6)
	origin := originAt(t, maths.V(0, 1e6, 0))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	var none context.Context
	small := testDetail()
	small.Points = 9
	cases := []struct {
		name   string
		ground *Terrain
		ctx    context.Context
		want   string
	}{
		{"no terrain", nil, context.Background(), "no prepared terrain source"},
		{"no source", &Terrain{}, context.Background(), "no prepared terrain source"},
		{"no context", testTerrain(t, flat(0), testDetail()), none, "no prepared terrain source"},
		{"cancelled", testTerrain(t, flat(0), testDetail()), cancelled, "context canceled"},
		{"source error", testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) { return 0, false, sentinel }), testDetail()), context.Background(), "source failed"},
		{"absent sample", testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) { return 0, false, nil }), testDetail()), context.Background(), "no source height at"},
		{"height above the envelope", testTerrain(t, flat(6000), testDetail()), context.Background(), "outside its declared envelope"},
		{"height below the envelope", testTerrain(t, flat(-1001), testDetail()), context.Background(), "outside its declared envelope"},
		{"height not a number", testTerrain(t, flat(math.NaN()), testDetail()), context.Background(), "outside its declared envelope"},
		{"point budget", testTerrain(t, flat(0), small), context.Background(), "point budget exhausted"},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			prepared, err := each.ground.Prepare(each.ctx, view, origin)
			if err == nil || !strings.Contains(err.Error(), each.want) {
				t.Fatalf("error %v, wanted one containing %q", err, each.want)
			}
			if len(prepared.Surfaces) != 0 || len(prepared.Patches) != 0 {
				t.Fatalf("a refusal returned %+v", prepared)
			}
		})
	}
}

// TestPrepareRefusesAnUncheckedView checks that a zero perspective is
// refused when the root box is assessed.
func TestPrepareRefusesAnUncheckedView(t *testing.T) {
	ground := testTerrain(t, flat(0), testDetail())
	_, err := ground.Prepare(context.Background(), footprint.Perspective{}, originAt(t, maths.Vec{}))
	if err == nil || !strings.Contains(err.Error(), "footprint:") {
		t.Fatalf("error %v", err)
	}
}

// TestPrepareStopsWhenTheContextEndsWhileSampling cancels the context from
// inside the first source read and expects the second patch to be refused.
func TestPrepareStopsWhenTheContextEndsWhileSampling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ground := testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) {
		cancel()
		return 0, true, nil
	}), testDetail())
	_, err := ground.Prepare(ctx, lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
}

// TestPrepareStopsWhenTheContextEndsWhileSelecting cancels the context from
// inside a height range query, which runs during selection.
func TestPrepareStopsWhenTheContextEndsWhileSelecting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := rangeHeights{heightFunc: flat(0), low: 0, high: 0}
	ground := testTerrain(t, cancelingRange{source, cancel}, testDetail())
	_, err := ground.Prepare(ctx, lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
}

// cancelingRange cancels a context the first time its range is asked for.
type cancelingRange struct {
	rangeHeights
	cancel func()
}

// HeightRange cancels, then answers without a range.
func (source cancelingRange) HeightRange(float64, float64, float64, float64) (float64, float64, bool, error) {
	source.cancel()
	return 0, 0, false, nil
}

// TestBoundsUseNativeHeightRanges checks that a source range narrows the
// box, an absent range keeps the declared envelope, and a range outside the
// envelope or backwards is refused.
func TestBoundsUseNativeHeightRanges(t *testing.T) {
	patch := terrain.Patch{MaxColumn: 8, MaxRow: 8, Level: 1}
	sentinel := errors.New("range failed")
	declared := testTerrain(t, flat(0), testDetail())
	wide, _, err := declared.bounds(patch, true)
	if err != nil {
		t.Fatal(err)
	}
	narrow := testTerrain(t, rangeHeights{heightFunc: flat(0), low: 0, high: 10, has: true}, testDetail())
	small, _, err := narrow.bounds(patch, true)
	if err != nil {
		t.Fatal(err)
	}
	if small.Max.X-small.Min.X >= wide.Max.X-wide.Min.X {
		t.Fatalf("a ten metre range gave box %v, no smaller than %v", small, wide)
	}
	absent := testTerrain(t, rangeHeights{heightFunc: flat(0), has: false}, testDetail())
	same, _, err := absent.bounds(patch, true)
	if err != nil || same != wide {
		t.Fatalf("absent range gave %v, %v; wanted %v", same, err, wide)
	}
	unindexed, _, err := narrow.bounds(patch, false)
	if err != nil || unindexed != wide {
		t.Fatalf("an unindexed query gave %v, %v; wanted the declared box", unindexed, err)
	}
	for name, source := range map[string]rangeHeights{
		"error":      {heightFunc: flat(0), err: sentinel},
		"too high":   {heightFunc: flat(0), low: 0, high: 5001, has: true},
		"too low":    {heightFunc: flat(0), low: -1001, high: 0, has: true},
		"backwards":  {heightFunc: flat(0), low: 10, high: 5, has: true},
		"not finite": {heightFunc: flat(0), low: math.NaN(), high: 5, has: true},
	} {
		ground := testTerrain(t, source, testDetail())
		if _, _, err := ground.bounds(patch, true); err == nil {
			t.Errorf("%s: accepted the range", name)
		}
	}
}

// TestBoundsRefuseAPositionPastTheRange uses a sphere and ground height whose
// radius overflows, and expects both bounds and a source read to refuse.
func TestBoundsRefuseAPositionPastTheRange(t *testing.T) {
	patch := terrain.Patch{MaxColumn: 8, MaxRow: 8, Level: 1}
	detail := testDetail()
	vast, err := frame.NewSphere(measure.Metres(1e295), frame.Location{})
	if err != nil {
		t.Fatal(err)
	}
	detail.MinimumHeight, detail.MaximumHeight = math.MaxFloat64, math.MaxFloat64
	ground, err := New(vast, flat(math.MaxFloat64), testRegion, detail)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ground.bounds(patch, false); err == nil || !strings.Contains(err.Error(), "invalid spherical location") {
		t.Fatalf("error %v", err)
	}
	book := vertexBook{terrain: ground, points: map[lattice]maths.Vec{}}
	if _, err := book.at(lattice{4, 4}); err == nil || len(book.points) != 0 {
		t.Fatalf("a failed position was kept: %v, %d points", err, len(book.points))
	}
}

// TestPrepareFailsWhenARangeQueryFails makes the height range query fail
// during selection and expects that error from Prepare.
func TestPrepareFailsWhenARangeQueryFails(t *testing.T) {
	sentinel := errors.New("range failed")
	ground := testTerrain(t, rangeHeights{heightFunc: flat(0), err: sentinel}, testDetail())
	_, err := ground.Prepare(context.Background(), lookDown(t, 1e6), originAt(t, maths.V(0, 1e6, 0)))
	if !errors.Is(err, sentinel) {
		t.Fatalf("error %v", err)
	}
}

// TestAtReadsEachKeyOnce checks that a repeated lattice key does not query
// the source again and returns the same point.
func TestAtReadsEachKeyOnce(t *testing.T) {
	reads := 0
	ground := testTerrain(t, heightFunc(func(float64, float64) (float64, bool, error) {
		reads++
		return 100, true, nil
	}), testDetail())
	book := vertexBook{terrain: ground, points: map[lattice]maths.Vec{}}
	first, err := book.at(lattice{4, 12})
	if err != nil {
		t.Fatal(err)
	}
	second, err := book.at(lattice{4, 12})
	if err != nil || first != second || reads != 1 {
		t.Fatalf("%v, %v, %d reads", first, err, reads)
	}
	if want := world(t, 4, 12, 100); !near(first, want, 1e-6) {
		t.Fatalf("point %v, wanted %v", first, want)
	}
}

// TestFiniteAndPoint checks the two numeric guards on their edges.
func TestFiniteAndPoint(t *testing.T) {
	nan, inf := math.NaN(), math.Inf(-1)
	if !finite(0) || !finite(-1e300) || finite(nan) || finite(inf) {
		t.Error("finite is wrong")
	}
	if !point(maths.V(1, 2, 3)) || point(maths.V(1, nan, 3)) || point(maths.V(1, 2, inf)) || point(maths.V(nan, 2, 3)) {
		t.Error("point is wrong")
	}
}
