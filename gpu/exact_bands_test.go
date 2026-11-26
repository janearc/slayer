//go:build darwin

package gpu

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

func TestExactBandAccumulatesSevenSamplesPerPixel(t *testing.T) {
	band := bandFinisher{finish: finishing{world: &thing.World{}}, picture: frame.New(2, 1), side: 1, exact: 7}
	samples := make([]float32, 2*7*per)
	for column := 0; column < 2; column++ {
		for i := 0; i < 7; i++ {
			at := (column*7 + i) * per
			samples[at] = float32(column*10 + i)
			samples[at+3] = 1
			samples[at+7] = float32(7 - i)
		}
	}
	if err := band.valid(0, 1, samples); err != nil {
		t.Fatal(err)
	}
	band.rows(0, 1, samples)
	for column, want := range []float64{3, 13} {
		got := band.picture.At(column, 0)
		if math.Abs(float64(got.R)-want) > 1e-5 || math.Abs(float64(got.Cover)-1) > 1e-5 || got.Depth != 1 {
			t.Fatalf("pixel %d = %+v, want red %v, cover 1, depth 1", column, got, want)
		}
	}
	if err := band.valid(0, 1, samples[:len(samples)-per]); err == nil {
		t.Fatal("short sample band was accepted")
	}
}
