package searching

import (
	"github.com/janearc/slayer/maths"
	"image"
	"image/color"
	"math"
	"testing"
)

func TestSlayerFilmPreservesShotLensAtMusicalCut(t *testing.T) {
	edit := &Edit{Plan: Plan{Shots: []ShotPlan{{Stage: &StagePlan{Wide: 55}}, {Wide: 90}}}, ShotSpans: [][2]float64{{0, 4}, {4, 8}}}
	for _, sample := range []struct{ at, wide float64 }{{0, 55}, {3.999, 55}, {4, 90}, {7.999, 90}, {1, 55}} {
		if got := screenplayWide(edit, sample.at); got != sample.wide {
			t.Fatalf("lens at %g s: %g, wanted %g", sample.at, got, sample.wide)
		}
	}
}

func TestSlayerFilmSpectrogramPanelRetainsMeasuredPixelColour(t *testing.T) {
	plate := image.NewRGBA(image.Rect(0, 0, 480, 270))
	plate.SetRGBA(240, 135, color.RGBA{128, 64, 0, 255})
	got := (spectrumPanel{plate, 60}).At(maths.Vec{})
	if math.Abs(got.R-.21586050011389926) > 1e-12 || math.Abs(got.G-.05126945837404324) > 1e-12 || got.B != 0 {
		t.Fatalf("panel changed measured sRGB light: %+v", got)
	}
}
