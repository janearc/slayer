//go:build draw

package omnipositor

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpulights"
)

func TestGPULightBaseMatchesCPUAtMoonMaskEdges(t *testing.T) {
	plate := lights.Image{Width: 2, Height: 3, Light: []float64{1, 2, 3, 2, 1, 0, 3, 2, 1, 4, 5, 6, 2, 3, 4, 3, 4, 5}}
	moon := lights.Image{Width: 2, Height: 3, Light: []float64{2, 1, 0, 4, 3, 2, 1, 2, 3, 2, 3, 4, 5, 6, 7, 3, 2, 1}}
	open := []bool{true, false, true, true, false, true}
	want := lights.Image{Width: plate.Width, Height: plate.Height, Light: append([]float64(nil), plate.Light...)}
	for index := range want.Light {
		want.Light[index] *= 0.5
	}
	if err := lights.AddMoonDisc(want, moon, open, lights.MoonDiscConfig{Shift: 0.5, Level: 0.75}); err != nil {
		t.Fatal(err)
	}
	renderer, err := gpulights.New(plate.Width, plate.Height)
	if err != nil {
		t.Fatal(err)
	}
	got, err := (gpuLightBase{renderer: renderer}).Base(plate, &moon, open, 0.5, 0.5, 0.75)
	if err != nil {
		t.Fatal(err)
	}
	for index := range want.Light {
		if math.Abs(got.Light[index]-want.Light[index]) > 1e-5 {
			t.Fatalf("channel %d: GPU %v CPU %v", index, got.Light[index], want.Light[index])
		}
	}
}

func TestGPULightSkyGlowPreservesClosedPixels(t *testing.T) {
	renderer, err := gpulights.New(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	sky := lights.Image{Width: 2, Height: 1, Light: []float64{1, 1, 1, 2, 2, 2}}
	input := lights.SkyGlowInput{Open: []bool{true, false}, Rim: []int{1, 1}, Camera: lights.NewCamera(2, 1, lights.CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150}), Moon: [3]float64{1, 1, 1}, Gain: 0.02, Azimuth: 109.65, MoonAltitude: 3, MoonScale: 6, MoonLevel: 1, Master: 1}
	got, err := (gpuLightBase{renderer: renderer}).SkyGlow(sky, input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Light[0] <= sky.Light[0] || got.Light[3] != sky.Light[3] {
		t.Fatalf("unexpected sky glow: %v", got.Light)
	}
}

func TestGPULightFixturesKeepDistinctPasses(t *testing.T) {
	sky := lights.Image{Width: 1, Height: 1, Light: []float64{1, 2, 3}}
	passes := map[string]lights.Image{
		"first":  {Width: 1, Height: 1, Light: []float64{1, 2, 4}},
		"second": {Width: 1, Height: 1, Light: []float64{8, 16, 32}},
	}
	layers := []lights.FixtureLayer{
		{Name: "first", Amount: 0.5, Colour: [3]float64{1, 1, 1}},
		{Name: "second", Amount: 0.25, Colour: [3]float64{1, 1, 1}},
	}
	want := lights.Image{Width: 1, Height: 1, Light: append([]float64(nil), sky.Light...)}
	if err := lights.AddFixtures(want, layers, passes); err != nil {
		t.Fatal(err)
	}
	renderer, err := gpulights.New(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := (gpuLightBase{renderer: renderer}).Fixtures(sky, layers, passes)
	if err != nil {
		t.Fatal(err)
	}
	for index := range want.Light {
		if math.Abs(got.Light[index]-want.Light[index]) > 1e-5 {
			t.Fatalf("channel %d: GPU %v CPU %v", index, got.Light[index], want.Light[index])
		}
	}
}
