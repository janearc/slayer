package manufacture

import "testing"

// Upper wood must remain longer than its diameter. The original
// whole-trunk radius floor made short crown branches wider than they were long.
func TestRedwoodUpperWoodTapersWithBranchLength(t *testing.T) {
	for _, name := range []string{"coast-redwood", "giant-sequoia"} {
		for number := range uint64(20) {
			made := mustMake(t, name, number)
			height := made.Bounds().Max.Y
			checked := 0
			for _, piece := range made.parts {
				wood, ok := piece.shape.(tube)
				if !ok || piece.of != body || piece.space.Origin.Y < height*0.75 {
					continue
				}
				if 2*wood.foot.InMetres() >= wood.length.InMetres() {
					t.Errorf("%s seed %d, height %.2f: upper branch is too thick: %v",
						name, number, piece.space.Origin.Y, wood)
				}
				checked++
			}
			if checked < 10 {
				t.Fatalf("only checked %d upper branches", checked)
			}
		}
	}
}
