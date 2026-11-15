//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestStrobesMatchCPUWhenForced(t *testing.T) {
	sky := lights.Image{Width: 4, Height: 3, Light: make([]float64, 36)}
	open := make([]bool, 12)
	near := make([]bool, 12)
	for index := range open {
		open[index] = true
	}
	input := lights.StrobeInput{Open: open, Near: near, Rim: []int{2, 2, 2, 2}, Camera: lights.NewCamera(4, 3, lights.CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150}), Moon: [3]float64{1, 1, 1}, FarTree: lights.Toward(109.65, 0).Scale(1000), Azimuth: 109.65, Top: 3, Bottom: -3, Shore: 0, Seconds: 1, White: 1, Magenta: 1}
	want := lights.Image{Width: sky.Width, Height: sky.Height, Light: append([]float64(nil), sky.Light...)}
	level := lights.AddStrobes(want, input)
	got, actual, err := Strobes(sky, input)
	if err != nil {
		t.Fatal(err)
	}
	if actual != level {
		t.Fatalf("level %v want %v", actual, level)
	}
	for index := range got.Light {
		if math.Abs(got.Light[index]-want.Light[index]) > 1e-3 {
			t.Fatalf("channel %d: GPU %v CPU %v", index, got.Light[index], want.Light[index])
		}
	}
}

func TestStrobesOffLeavesImage(t *testing.T) {
	sky := lights.Image{Width: 1, Height: 1, Light: []float64{1, 2, 3}}
	input := lights.StrobeInput{Open: []bool{true}, Near: []bool{false}, Rim: []int{1}, Top: 1, Bottom: -1, Shore: 0, White: 0, Magenta: 0}
	got, level, err := Strobes(sky, input)
	if err != nil {
		t.Fatal(err)
	}
	if level != 0 || got.Light[0] != 1 || got.Light[1] != 2 || got.Light[2] != 3 {
		t.Fatalf("off: %v %v", got.Light, level)
	}
}
