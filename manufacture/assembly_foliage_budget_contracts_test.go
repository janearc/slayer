package manufacture

import (
	"github.com/janearc/celestial/measure"
	"testing"
)

func TestLegacyFoliageSpendsItsAuthoredBladeBudgetAcrossActualLeaves(t *testing.T) {
	axis, err := newTube(measure.Metres(1), measure.Metres(.1), measure.Metres(.02))
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{1, 2} {
		parts := make([]part, count)
		for i := range parts {
			parts[i] = part{of: leaf, shape: axis}
		}
		made := individualFoliage(good{parts: parts}, Sow(5), 1)
		blades, sprays := 0, 0
		for _, piece := range made.parts {
			if spray, ok := piece.shape.(foliageSpray); ok {
				blades += spray.count
				sprays++
			}
		}
		// 12000 is the picked preview budget, not a measured foliage density.
		if sprays != count || blades != 12000 {
			t.Fatalf("%d leaves got %d sprays and %d blades", count, sprays, blades)
		}
	}
}
