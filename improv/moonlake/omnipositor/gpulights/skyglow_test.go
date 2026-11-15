//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestSkyGlowMatchesCPU(t *testing.T) {
	sky := lights.Image{Width: 3, Height: 3, Light: make([]float64, 27)}
	input := lights.SkyGlowInput{Open: []bool{true, true, true, true, false, true, false, false, false}, Rim: []int{2, 1, 2}, Camera: lights.NewCamera(3, 3, lights.CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150}), Moon: [3]float64{2, 3, 4}, Gain: 0.02, Azimuth: 109.65, MoonAltitude: 3, MoonScale: 6, MoonLevel: 1, Master: 0.8}
	want := lights.Image{Width: sky.Width, Height: sky.Height, Light: append([]float64(nil), sky.Light...)}
	lights.AddSkyGlow(want, input)
	got, err := SkyGlow(sky, input)
	if err != nil {
		t.Fatal(err)
	}
	for index := range want.Light {
		if math.Abs(got.Light[index]-want.Light[index]) > 1e-4 {
			t.Fatalf("channel %d: GPU %v CPU %v", index, got.Light[index], want.Light[index])
		}
	}
	if got.Light[0] == 0 {
		t.Fatal("sky glow did not affect open sky")
	}
	if got.Light[12] != 0 {
		t.Fatal("sky glow affected closed sky")
	}
}

func TestSkyGlowRejectsInvalidMasks(t *testing.T) {
	sky := lights.Image{Width: 1, Height: 1, Light: []float64{1, 1, 1}}
	if _, err := SkyGlow(sky, lights.SkyGlowInput{Open: nil, Rim: []int{0}}); err == nil {
		t.Fatal("missing sky mask accepted")
	}
}
