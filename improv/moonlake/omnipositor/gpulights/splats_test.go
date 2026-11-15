//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestSplatsMatchCPUAndPreserveOrder(t *testing.T) {
	sky := lights.Image{Width: 3, Height: 3, Light: make([]float64, 27)}
	front := lights.Image{Width: 3, Height: 3, Light: make([]float64, 27)}
	wantSky := lights.Image{Width: 3, Height: 3, Light: make([]float64, 27)}
	wantFront := lights.Image{Width: 3, Height: 3, Light: make([]float64, 27)}
	open := []bool{true, true, true, true, true, true, true, true, true}
	splats := []Splat{{X: 1, Y: 1, Sigma: 1, Amount: 2, Colour: [3]float64{1, 0.5, 0.25}}, {X: 1.5, Y: 2, Sigma: 0.8, Amount: 1, Colour: [3]float64{0.2, 1, 0.3}}}
	for _, splat := range splats {
		lights.Splat(wantSky, wantFront, splat.X, splat.Y, splat.Sigma, splat.Amount, splat.Colour, 1.5)
	}
	gotSky, gotFront, err := Splats(sky, front, open, 1.5, splats)
	if err != nil {
		t.Fatal(err)
	}
	for index := range wantSky.Light {
		if math.Abs(gotSky.Light[index]-wantSky.Light[index]) > 1e-5 || math.Abs(gotFront.Light[index]-wantFront.Light[index]) > 1e-5 {
			t.Fatalf("channel %d: sky %v/%v front %v/%v", index, gotSky.Light[index], wantSky.Light[index], gotFront.Light[index], wantFront.Light[index])
		}
	}
}

func TestSplatsRejectInvalidSigma(t *testing.T) {
	image := lights.Image{Width: 1, Height: 1, Light: []float64{0, 0, 0}}
	if _, _, err := Splats(image, image, []bool{true}, 0, []Splat{{Sigma: 0}}); err == nil {
		t.Fatal("zero sigma accepted")
	}
}
