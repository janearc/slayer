package shape

import "testing"

// edgeSignPoints is the csv file of the points a zero-length edge is put at.
const edgeSignPoints = "testdata/edge-sign-points.csv"

// TestCoincidentEdgeHasNoOwnership says that an edge from a point to itself,
// with a determinant of nought, takes neither sign.
func TestCoincidentEdgeHasNoOwnership(t *testing.T) {
	for _, row := range readSheet(t, edgeSignPoints) {
		across, up := cell(t, row, "x"), cell(t, row, "y")
		if side(nought, across, up, across, up) != nought {
			t.Fatalf("zero-length edge at %v, %v acquired a side", across,
				up)
		}
	}
}
