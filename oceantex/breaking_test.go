package oceantex

import "testing"

func TestBreakingRatioIsMcCowans(t *testing.T) {
	if BreakingRatio != 0.78 {
		t.Fatalf("the breaking ratio is %g, want McCowan's 0.78",
			BreakingRatio)
	}
}

func TestBreakingRatioIsUnderOne(t *testing.T) {
	if !(BreakingRatio > 0 && BreakingRatio < 1) {
		t.Fatalf("a wave of %g of the depth stands taller than the water",
			BreakingRatio)
	}
}
