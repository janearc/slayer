//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestBackHazeMatchesCPU(t *testing.T) {
	front := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
	trees := lights.Image{Width: 2, Height: 2, Light: []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1}}
	mask := []bool{false, false, false, true}
	camera := lights.NewCamera(2, 2, lights.CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150})
	position := lights.Toward(109.65, 0).Scale(1000)
	input := lights.BackHazeInput{Camera: camera, Moon: [3]float64{1, 1, 1}, Positions: [2]lights.Vec{position, position}, Levels: [2]float64{1, 0}, Gain: 0.6, Seconds: 1}
	want := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
	lights.AddBackHaze(want, trees, input)
	got, err := BackHaze(front, mask, input)
	if err != nil {
		t.Fatal(err)
	}
	for index := range got.Light {
		if math.Abs(got.Light[index]-want.Light[index]) > 1e-4 {
			t.Fatalf("channel %d: GPU %v CPU %v", index, got.Light[index], want.Light[index])
		}
	}
}

func TestBackHazeRejectsMissingMask(t *testing.T) {
	front := lights.Image{Width: 1, Height: 1, Light: []float64{0, 0, 0}}
	if _, err := BackHaze(front, nil, lights.BackHazeInput{}); err == nil {
		t.Fatal("missing mask accepted")
	}
}
