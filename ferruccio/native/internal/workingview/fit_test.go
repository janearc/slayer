package workingview

import "testing"

// TestFitLeavesTargetsWithinTheCapAlone checks a window that fits is drawn
// at its own size, the cap included.
func TestFitLeavesTargetsWithinTheCapAlone(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {1195, 747}, {largestSide, largestSide}} {
		width, height := fit(size[0], size[1])
		if width != size[0] || height != size[1] {
			t.Errorf("fit(%d, %d) = %d, %d; want it unchanged",
				size[0], size[1], width, height)
		}
	}
}

// TestFitScalesALargeWindowDownWithItsShape checks a window past the cap
// is drawn at the cap on its longer side, with its shape kept.
func TestFitScalesALargeWindowDownWithItsShape(t *testing.T) {
	for _, size := range [][2]int{{5120, 2880}, {2880, 5120}, {8192, 8192}} {
		width, height := fit(size[0], size[1])
		if max(width, height) != largestSide {
			t.Fatalf("fit(%d, %d) = %d, %d; want the longer side %d",
				size[0], size[1], width, height, largestSide)
		}
		want := float64(size[0]) / float64(size[1])
		got := float64(width) / float64(height)
		// One pixel of rounding on a side of a few thousand.
		const tolerance = 0.001
		if got/want > 1+tolerance || want/got > 1+tolerance {
			t.Errorf("fit(%d, %d) shape %.4f; want %.4f",
				size[0], size[1], got, want)
		}
	}
}

// TestFitNeverRoundsASideToNothing is the anti-test: a very thin window
// keeps a side of at least one pixel rather than becoming unrenderable.
func TestFitNeverRoundsASideToNothing(t *testing.T) {
	width, height := fit(100000, 1)
	if width != largestSide || height != 1 {
		t.Fatalf("fit(100000, 1) = %d, %d; want %d, 1", width, height, largestSide)
	}
}
