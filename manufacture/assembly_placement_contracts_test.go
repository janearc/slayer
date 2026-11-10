package manufacture

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestFireLightQuadratureAcceptsBothDocumentedEndpoints(t *testing.T) {
	for _, cells := range []int{3, 8} {
		spec := burnerFixture()
		spec.LightCells = cells
		made, err := Burn(spec)
		if err != nil {
			t.Fatalf("%d cells refused: %v", cells, err)
		}
		volumes, lamps, err := made.Place(maths.Subspace{}, 1, 0)
		if err != nil || len(volumes) == 0 || len(lamps) == 0 {
			t.Fatalf("%d cells lost fire or lighting: volumes %d lamps %d error %v", cells, len(volumes), len(lamps), err)
		}
	}
}

func TestFirePlacementRejectsPositiveHandedButNonorthonormalBases(t *testing.T) {
	made, err := Burn(burnerFixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, basis := range []maths.Orientation{
		{X: maths.XAxis.Scale(2), Y: maths.YAxis, Z: maths.ZAxis},
		{X: maths.XAxis, Y: maths.YAxis.Scale(2), Z: maths.ZAxis},
		{X: maths.XAxis, Y: maths.YAxis, Z: maths.ZAxis.Scale(2)},
		{X: maths.XAxis, Y: maths.XAxis.Add(maths.YAxis).Unit(), Z: maths.ZAxis},
		{X: maths.XAxis, Y: maths.YAxis, Z: maths.XAxis.Add(maths.ZAxis).Unit()},
		{X: maths.XAxis, Y: maths.YAxis, Z: maths.YAxis.Add(maths.ZAxis).Unit()},
		{X: maths.XAxis, Y: maths.YAxis, Z: maths.ZAxis.Neg()},
	} {
		volumes, lamps, err := made.Place(maths.Subspace{Orientation: basis}, 1, 0)
		if err == nil || !strings.Contains(err.Error(), "invalid fire placement") || volumes != nil || lamps != nil {
			t.Fatalf("invalid basis returned partial fire: basis %v volumes %d lamps %d error %v", basis, len(volumes), len(lamps), err)
		}
	}
}

func TestFirePlacementHoldsEachOrthogonalityBoundary(t *testing.T) {
	made, err := Burn(burnerFixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range []float64{math.Nextafter(1e-9, 0), 1e-9, math.Nextafter(1e-9, math.Inf(1))} {
		for axis := range 3 {
			basis := maths.Orientation{X: maths.XAxis, Y: maths.YAxis, Z: maths.ZAxis}
			switch axis {
			case 0:
				basis.Y.X = delta
			case 1:
				basis.Z.X = delta
			case 2:
				basis.Z.Y = delta
			}
			_, _, err := made.Place(maths.Subspace{Orientation: basis}, 1, 0)
			if (err == nil) != (delta < 1e-9) {
				t.Fatalf("dot pair %d at %.18g: %v", axis, delta, err)
			}
		}
	}
}
