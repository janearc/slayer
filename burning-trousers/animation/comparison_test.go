//go:build darwin

package animation_test

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
)

func TestComparisonCountsTheLastPixelAndIndependentAlpha(t *testing.T) {
	one, two := image.NewNRGBA(image.Rect(0, 0, 10, 10)), image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for _, img := range []*image.NRGBA{one, two} {
		draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{A: 255}), image.Point{}, draw.Src)
	}
	two.SetNRGBA(9, 9, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	two.SetNRGBA(3, 2, color.NRGBA{A: 128})
	result := comparePNGs(t, one, two)
	if result.Pixels != 100 || result.BeyondOneJND != 1 || result.AlphaDifferences != 1 || result.WorstAt != [2]int{9, 9} {
		t.Fatalf("lost pixels, alpha or worst location: %+v", result)
	}
	// Ideal black and white have lightness zero and one in OKLab.
	if math.Abs(result.WorstJND-1/ok.Eye) > 1e-4 {
		t.Fatalf("black-white difference: %g JND", result.WorstJND)
	}
}

func TestNativeMeasurementKeepsFallbacksDistinctFromRefinements(t *testing.T) {
	counts := readCounts(t, "native closest: 100 queries, 80 candidates, 77 refined, 3 software fallbacks\n")
	if *counts != (nativeCounts{Queries: 100, Candidates: 80, Refined: 77, Fallbacks: 3}) {
		t.Fatalf("native outcomes were conflated: %+v", counts)
	}
}
