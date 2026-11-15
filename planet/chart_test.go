package planet

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// single is defined as one thing, counted.
const single = 1

// chartFace is a face as full as the giant's over a real weather, its
// body's radius, and an eye three radii off.
func chartFace(t *testing.T) (*Face, float64) {
	t.Helper()
	body := stormyBody(t, false).Flowing(realWeather(t))
	pole, star, eye := look(t, "chart pole"), look(t, "chart star"),
		look(t, "chart eye")
	return NewFace(body, pole, star, eye, wide(),
		picked(t, "chart-exposure")), body.Radius()
}

// seenPoint is the k-th of a face's points the eye sees well clear of the
// limb, by a hash.
func seenPoint(t *testing.T, face *Face, radius float64, k int64) (maths.Vec,
	bool) {
	t.Helper()
	point := pointOf(radius, pickedSeed(t, "chart-lat-seed"),
		pickedSeed(t, "chart-lon-seed"), k)
	toward := face.eye.Sub(point).Unit()
	return point, point.Unit().Dot(toward) > picked(t, "chart-clear-of-limb")
}

// limbExact is how many lit points at the limb a chart gives as its face
// does. A dark point does not count: read between dark points of a chart
// it is dark too, whether or not the chart asked the face.
func limbExact(t *testing.T, face *Face, chart *Chart, radius float64) int {
	t.Helper()
	var exact int
	grazing := picked(t, "chart-limb-grazing")
	for k := range pickedCount(t, "chart-points") {
		limb, _ := seenPoint(t, face, radius, k)
		eye := limb.Unit().Dot(face.eye.Sub(limb).Unit())
		if positive(eye) && eye < grazing {
			want, got := face.Within(limb, nothing), chart.Within(limb, nothing)
			if got == want && want != maths.Grey(nothing) {
				exact++
			}
		}
	}
	return exact
}

// a chart paints only the tiles rays land in, asks the face where a point
// about a ray is out of the eye's sight, and for a pixel wider than the
// face's own step for its fronts.
func TestAChartPaintsWhatIsAskedAndAsksTheFaceAtItsEdges(t *testing.T) {
	face, radius := chartFace(t)
	chart, _ := NewChart(face, radius, picked(t, "chart-paint-cell")*maths.Degree)
	point := face.ways.toward(picked(t, "chart-paint-down"),
		picked(t, "chart-paint-along")).Scale(radius)
	chart.Within(point, nothing)
	if painted, all := chart.tilesPainted(); painted != single ||
		all < pickedInt(t, "chart-least-tiles") {
		t.Fatalf("one ray painted %d tiles of %d", painted, all)
	}
	if limbExact(t, face, chart, radius) == nothing {
		t.Fatal("no ray at the limb was the face's own")
	}
	broad := picked(t, "chart-broad-degrees") * maths.Degree * radius
	if got, want := chart.Within(point, broad), face.Within(point, broad); got != want {
		t.Fatalf("a wide pixel reads %v, the face %v", got, want)
	}
	bare := NewFace(stormyBody(t, false), look(t, "ungridded pole"),
		look(t, "ungridded star"), look(t, "ungridded eye"), wide(),
		picked(t, "ungridded-exposure"))
	if _, ok := NewChart(bare, radius, picked(t, "chart-ungridded-cell")); ok {
		t.Fatal("a face with no flow was charted")
	}
}
