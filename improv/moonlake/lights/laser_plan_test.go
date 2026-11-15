package lights

import (
	"math"
	"testing"
)

func TestPlannedLasersMatchCPU(t *testing.T) {
	width, height := 8, 6
	open := make([]bool, width*height)
	for index := range open {
		open[index] = true
	}
	camera := NewCamera(width, height, CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150})
	config := LaserConfig{Azimuth: 109.65, EyeOverLake: 53.6, WaterZ: -53.6, Reach: 20, FarTree: Toward(109.65, 0).Scale(1000), ShoreLeft: Vec{X: 100, Y: 100, Z: 10}, ShoreRight: Vec{X: 120, Y: 100, Z: 10}}
	input := LaserInput{Open: open, Rim: []int{3, 3, 3, 3, 3, 3, 3, 3}, Camera: camera, Moon: [3]float64{1, 1, 1}, ShoreRow: 3, Cues: LaserCues{Rig: 1, Gate: 1, GateLeft: 1, GateRight: 1, Beams: 1, Parity: -1, Figure: "beam"}}
	cpuSky := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	cpuFront := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	AddLasers(cpuSky, cpuFront, config, input)
	planned, err := PlanLasers(width, height, config, input)
	if err != nil {
		t.Fatal(err)
	}
	gotSky := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	gotFront := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	for _, splat := range planned {
		Splat(gotSky, gotFront, splat.X, splat.Y, splat.Sigma, splat.Amount, splat.Colour, input.ShoreRow)
	}
	for index := range cpuSky.Light {
		if math.Abs(cpuSky.Light[index]-gotSky.Light[index]) > 1e-12 || math.Abs(cpuFront.Light[index]-gotFront.Light[index]) > 1e-12 {
			t.Fatalf("channel %d differs", index)
		}
	}
}

func TestPlanLasersRejectsInvalidReach(t *testing.T) {
	if _, err := PlanLasers(1, 1, LaserConfig{}, LaserInput{Open: []bool{true}, Rim: []int{0}}); err == nil {
		t.Fatal("zero reach accepted")
	}
}
