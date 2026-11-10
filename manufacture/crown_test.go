package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// These are visual regression floors, not botanical canopy measurements.
// A branch can have correct needles yet leave a whole tree nearly bare.
// Shoot rays across the foliage bounds from three sides, excluding lighting.
func TestCrownFoliageCoverage(t *testing.T) {
	row, err := KindNamed("coast-redwood")
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []uint64{5, 11} {
		made, err := MakeWithComplexity(row, Sow(seed), "quick")
		if err != nil {
			t.Fatal(err)
		}
		group := made.groups[leaf]
		box := group.Bounds()
		for view := range 3 {
			hits := 0
			for x := range 64 {
				for y := range 64 {
					u, v := (float64(x)+.37)/64, (float64(y)+.61)/64
					at := maths.V(box.Min.X+(box.Max.X-box.Min.X)*u, box.Min.Y+(box.Max.Y-box.Min.Y)*v, box.Max.Z+1)
					dir := maths.ZAxis.Neg()
					if view == 1 {
						at = maths.V(box.Min.X+(box.Max.X-box.Min.X)*u, box.Max.Y+1, box.Min.Z+(box.Max.Z-box.Min.Z)*v)
						dir = maths.YAxis.Neg()
					}
					if view == 2 {
						at = maths.V(box.Max.X+1, box.Min.Y+(box.Max.Y-box.Min.Y)*u, box.Min.Z+(box.Max.Z-box.Min.Z)*v)
						dir = maths.XAxis.Neg()
					}
					if _, hit := group.Hit(maths.Ray{At: at, Dir: dir}, 0, 1000); hit {
						hits++
					}
				}
			}
			minimum := 409 // Ten percent of the enclosing rectangle from either side.
			if view == 1 {
				minimum = 1024
			} // One quarter from above; gaps remain visible.
			t.Logf("seed %d, view %d: %d/4096 foliage rays hit", seed, view, hits)
			if hits < minimum {
				t.Errorf("seed %d, view %d: sparse crown, %d hits; need %d", seed, view, hits, minimum)
			}
		}
	}
}
