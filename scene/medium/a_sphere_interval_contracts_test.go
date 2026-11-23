package medium

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// aSphereIntervalContractsFile holds the rays and the ranges they are cut to.
const aSphereIntervalContractsFile = "testdata/a-sphere-interval-contracts.csv"

// aSphereIntervalContractsRay is the ray of a row: from its place in x
// and z, straight down the z axis towards the sphere.
func aSphereIntervalContractsRay(t *testing.T, row map[string]string) maths.Ray {
	t.Helper()
	x, z := cell(t, row, "x"), cell(t, row, "z")
	return maths.Ray{At: maths.V(x, nought, z), Dir: maths.ZAxis.Neg()}
}

// TestSphericalMediumSpansContainPositiveDistanceOnly holds that a span of no
// length is refused, and a span with length is kept as it is clipped.
func TestSphericalMediumSpansContainPositiveDistanceOnly(t *testing.T) {
	cases := readSheet(t, aSphereIntervalContractsFile)
	puff := Puff{Radius: picked(t, "a-sphere-interval-contracts-radius"),
		Thick: picked(t, "a-sphere-interval-contracts-thick")}
	for _, name := range []string{"tangent", "pinched", "ends-at-entry",
		"starts-at-exit"} {
		row := cases.row(t, aSphereIntervalContractsFile,
			"a-sphere-interval-contracts-"+name)
		ray := aSphereIntervalContractsRay(t, row)
		near, far := cell(t, row, "near"), cell(t, row, "far")
		if _, _, ok := puff.Span(ray, near, far); ok {
			t.Fatalf("zero-length span accepted: %s", row["name"])
		}
	}
	row := cases.row(t, aSphereIntervalContractsFile,
		"a-sphere-interval-contracts-clipped")
	ray := aSphereIntervalContractsRay(t, row)
	near, far := cell(t, row, "near"), cell(t, row, "far")
	wantEnter, wantLeave := cell(t, row, "enter"), cell(t, row, "leave")
	enter, leave, ok := puff.Span(ray, near, far)
	if !ok || enter != wantEnter || leave != wantLeave {
		t.Fatalf("positive clipped span %g..%g, %t", enter, leave, ok)
	}
}
