package titan

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/janearc/celestial/grissom/mesh"
)

// The places and sizes the gap tests use.
const (
	// gapLon and gapLat are the place the gap tests ask about, in
	// degrees. picked: no corner of the mesh, and on land in the chart.
	gapLon, gapLat = 12.3, 4.5
	// gapDetail is detail no finer than the chart's samples, in metres.
	// picked: ten samples, so the whole walk is measured.
	gapDetail = 1000.0
	// gapFine is detail finer than the chart's samples, in metres.
	// picked: a hundred times finer, so the last depths are made.
	gapFine = 1.0
	// gapHole is how far round a centre the map has no height, in
	// metres. picked: smaller than the last triangle's edges at gapDetail.
	gapHole = 1.0
	// gapWide is a hole round the place asked, in metres. picked: wider
	// than the last triangle at gapDetail, which is under a kilometre.
	gapWide = 2000.0
	// gapAbsurd is a height no chart makes, in metres. picked.
	gapAbsurd = 1e6
)

// gappy is a chart with holes in it: places within a radius of a centre
// where the map has no height, or has the height nought, or fails.
type gappy struct {
	*chart
	holes  []vector // centres of the holes
	radius float64  // metres round each centre
	zero   bool     // whether the holes answer nought instead of unknown
	absurd bool     // whether the holes answer gapAbsurd
	broken []vector // centres of places where Height fails
	fails  error    // what Height fails with there

	tally  sync.Mutex
	inside int // how many times Height was asked inside a hole
}

// near is whether a direction is within the radius of any centre.
func (from *gappy) near(at vector, centres []vector) bool {
	for _, centre := range centres {
		away := at.minus(centre)
		if away.dot(away) < from.radius*from.radius/(earth*earth) {
			return true
		}
	}
	return false
}

func (from *gappy) Height(at mesh.Point) (float64, bool, error) {
	way := vectorOf(at)
	if from.near(way, from.broken) {
		return 0, false, from.fails
	}
	if from.near(way, from.holes) {
		from.tally.Lock()
		from.inside++
		from.tally.Unlock()
		return gapHeight(from)
	}
	return from.chart.Height(at)
}

// gapHeight is what a hole answers.
func gapHeight(from *gappy) (float64, bool, error) {
	switch {
	case from.zero:
		return 0, true, nil
	case from.absurd:
		return gapAbsurd, true, nil
	}
	return 0, false, nil
}

// asked is how many times the map was asked inside its holes.
func (from *gappy) asked() int {
	from.tally.Lock()
	defer from.tally.Unlock()
	return from.inside
}

// gapChart is the chart the gap tests lay holes in.
func gapChart() *chart { return &chart{finest: 100, relief: 300, share: 0.6} }

// lastTriangle is the three corners the walk to a place ends on, as At
// walks it: the last measured triangle at gapDetail.
func lastTriangle(t *testing.T, world *Ground, at *mesh.Point,
	detail float64) [corners]corner {
	t.Helper()
	toward := vectorOf(*at)
	cell, err := mesh.Locate(at, none)
	if err != nil {
		t.Fatal(err)
	}
	first, second, third := cell.Corners()
	nameA, nameB, nameC := cell.CornerNames()
	var shape [corners]corner
	for i, each := range []mesh.Point{first, second, third} {
		shape[i], err = world.read(vectorOf(each),
			[]uint64{nameA, nameB, nameC}[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	for depth := range world.Depth(detail) {
		shape, err = world.step(shape, toward, depth+1)
		if err != nil {
			t.Fatal(err)
		}
	}
	return shape
}

// faceCorners is the directions to the corners of the face a place is in.
func faceCorners(t *testing.T, at *mesh.Point) []vector {
	t.Helper()
	var ways []vector
	for _, each := range cellCorners(t, at) {
		ways = append(ways, vectorOf(each))
	}
	return ways
}

// A hole in the map at the coarse corners of the walk, thousands of
// kilometres from the place, changes nothing: the sample is the one the
// same map without the hole gives, measured and generated alike.
func TestAHoleFarFromThePlaceChangesNothing(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	for _, detail := range []float64{gapDetail, gapFine} {
		whole := ground(t, gapChart(), 1)
		from := &gappy{chart: gapChart(), holes: faceCorners(t, at),
			radius: gapHole}
		holed := ground(t, from, 1)
		want, err := whole.At(at, detail)
		if err != nil {
			t.Fatal(err)
		}
		got, err := holed.At(at, detail)
		if err != nil {
			t.Fatalf("at %v m of detail: %v", detail, err)
		}
		if from.asked() == 0 {
			t.Fatalf("the walk never reached the hole")
		}
		if *got != *want {
			t.Errorf("at %v m of detail the holed map gives %+v, the "+
				"whole one %+v", detail, *got, *want)
		}
	}
}

// Anti: what the far hole would have held makes no difference either, and
// an unknown corner is never counted as measured: the coarse sample stays
// measured with nothing made, and the fine one stays generated.
func TestAFarHoleIsNeverCountedOrFilled(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	for _, detail := range []float64{gapDetail, gapFine} {
		whole := ground(t, gapChart(), 1)
		from := &gappy{chart: gapChart(), holes: faceCorners(t, at),
			radius: gapHole, absurd: true}
		want, _ := whole.At(at, detail)
		got, err := ground(t, from, 1).At(at, detail)
		if err != nil {
			t.Fatal(err)
		}
		if got.Height() != want.Height() || got.Made() != want.Made() ||
			got.Kind() != want.Kind() {
			t.Errorf("at %v m a far corner's height of %v m moved the "+
				"ground", detail, gapAbsurd)
		}
		if (detail == gapDetail) != (got.Kind() == Measured) {
			t.Errorf("at %v m of detail the ground is %v", detail,
				got.Kind())
		}
	}
}

// A hole at a corner of the last measured triangle round the place
// refuses it, and the words name that corner.
func TestAHoleBesideThePlaceRefuses(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	last := lastTriangle(t, ground(t, gapChart(), 1), at, gapDetail)
	for i, each := range last {
		from := &gappy{chart: gapChart(), holes: []vector{each.at},
			radius: gapHole}
		_, err := ground(t, from, 1).At(at, gapDetail)
		if !errors.Is(err, Refused()) {
			t.Fatalf("corner %d unknown: %v", i, err)
		}
		here := each.at.point()
		named := fmt.Sprintf("%.6f, %.6f", here.Lon(), here.Lat())
		if !strings.Contains(err.Error(), named) ||
			!strings.Contains(err.Error(), "beside the place") {
			t.Errorf("corner %d: %q does not name %s", i, err, named)
		}
	}
}

// Anti: the same corner with a height of nought is a height, and the
// place is not refused.
func TestAKnownNoughtBesideThePlaceIsNotAHole(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	last := lastTriangle(t, ground(t, gapChart(), 1), at, gapDetail)
	from := &gappy{chart: gapChart(), holes: []vector{last[0].at},
		radius: gapHole, zero: true}
	got, err := ground(t, from, 1).At(at, gapDetail)
	if err != nil {
		t.Fatalf("a corner of nought was refused: %v", err)
	}
	if from.asked() == 0 || got.Kind() != Measured {
		t.Errorf("the nought corner was not read, or not measured")
	}
}

// A hole exactly at the place asked, with detail no finer than the map,
// is refused as it always was.
func TestAHoleAtThePlaceRefuses(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	from := &gappy{chart: gapChart(), holes: []vector{vectorOf(*at)},
		radius: gapWide}
	_, err := ground(t, from, 1).At(at, gapDetail)
	if !errors.Is(err, Refused()) {
		t.Errorf("a place the map has no height for: %v", err)
	}
}

// Anti: the same place with the hole filled is not refused.
func TestThePlaceWithoutItsHoleIsNotRefused(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	if _, err := ground(t, gapChart(), 1).At(at, gapDetail); err != nil {
		t.Errorf("a place the map has a height for: %v", err)
	}
}

// An unknown corner is kept as unknown: a second walk through the same
// hole asks the map no more times for it than the first did.
func TestAnUnknownCornerIsKeptAsUnknown(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	last := lastTriangle(t, ground(t, gapChart(), 1), at, gapDetail)
	from := &gappy{chart: gapChart(), holes: []vector{last[1].at},
		radius: gapHole}
	world := ground(t, from, 1)
	if _, err := world.At(at, gapDetail); !errors.Is(err, Refused()) {
		t.Fatalf("the first walk: %v", err)
	}
	first := from.asked()
	if first == 0 {
		t.Fatalf("the map was never asked about the hole")
	}
	if _, err := world.At(at, gapDetail); !errors.Is(err, Refused()) {
		t.Fatalf("the second walk: %v", err)
	}
	if from.asked() != first {
		t.Errorf("the map was asked about the hole %d more times",
			from.asked()-first)
	}
}

// Anti: an unknown corner is not a corner of nought. Kept, it is still
// unknown, and a map whose hole is a height of nought is a different
// ground, which refuses nothing.
func TestAnUnknownCornerIsNotNought(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	last := lastTriangle(t, ground(t, gapChart(), 1), at, gapDetail)
	world := ground(t, &gappy{chart: gapChart(),
		holes: []vector{last[1].at}, radius: gapHole}, 1)
	kept, err := world.read(last[1].at, last[1].name)
	if err != nil {
		t.Fatal(err)
	}
	if !kept.unknown || kept.missing() == nil {
		t.Errorf("a kept corner the map has no height at is %+v", kept)
	}
	known, err := ground(t, gapChart(), 1).read(last[1].at, last[1].name)
	if err != nil || known.unknown || known.missing() != nil {
		t.Errorf("a corner the map has a height at is unknown: %v", err)
	}
}

// An error from the map passes through unchanged, at a far corner of the
// walk as at any other.
func TestAnErrorFromTheMapPassesThrough(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	broken := errors.New("the map's file is gone")
	from := &gappy{chart: gapChart(), broken: faceCorners(t, at)[:1],
		radius: gapHole, fails: broken}
	_, err := ground(t, from, 1).At(at, gapDetail)
	if err != broken {
		t.Errorf("an error from the map came back as %v", err)
	}
	if errors.Is(err, Refused()) {
		t.Errorf("an error from the map was called a refusal")
	}
}

// Anti: the same map with nothing broken gives no error at all.
func TestAMapWithNothingBrokenGivesNoError(t *testing.T) {
	at := spot(t, gapLon, gapLat)
	from := &gappy{chart: gapChart(), radius: gapHole,
		fails: errors.New("never asked")}
	if _, err := ground(t, from, 1).At(at, gapDetail); err != nil {
		t.Errorf("a map with nothing broken: %v", err)
	}
}
