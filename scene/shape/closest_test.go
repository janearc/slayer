package shape

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// closestEdges is the csv file of the points asked from, and the weights
// of the nearest point on the face, each by its name.
const closestEdges = "testdata/closest-edges.csv"

// the nearest point on a face, worked by hand on the triangle with legs of
// four along x and y. just past A along each edge, by a quarter, the
// nearest is on that edge, a sixteenth of its way along; a unit out from
// either edge, a quarter of its way.
func TestTheNearestPointOnAnEdgeIsOnIt(t *testing.T) {
	leg := picked(t, "closest-leg")
	first := maths.V(nought, nought, nought)
	second := maths.V(leg, nought, nought)
	third := maths.V(nought, leg, nought)
	for _, row := range readSheet(t, closestEdges) {
		point := meshtreeMoreVector(t, row, "point")
		want := [cornersOfAFace]float64{cell(t, row, "first-weight"),
			cell(t, row, "second-weight"), cell(t, row, "third-weight")}
		if got := closest(point, first, second, third); got != want {
			t.Errorf("from %v the nearest is %v, want %v", point, got, want)
		}
	}
}
