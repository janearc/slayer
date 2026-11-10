package manufacture

import (
	"math"
	"strings"
	"testing"
)

func TestBurnRejectsOverflowInOnlyOneExtent(t *testing.T) {
	spec := burnerFixture()
	spec.Smoke = nil
	spec.Flame.Height = 1e155
	body, err := makeBurningField(spec, 1)
	if err != nil {
		t.Fatal(err)
	}
	bounds := body.Bounds()
	if math.IsInf(bounds.Min.Len(), 0) || !math.IsInf(bounds.Max.Len(), 1) {
		t.Fatal("fixture must overflow only its upper extent norm")
	}
	if _, err := Burn(spec); err == nil || !strings.Contains(err.Error(), "nonfinite fire extent") {
		t.Fatalf("single overflowing extent accepted: %v", err)
	}
}
