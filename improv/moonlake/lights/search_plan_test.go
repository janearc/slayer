package lights

import (
	"math"
	"testing"
)

func TestPlannedSearchlightsMatchCPU(t *testing.T) {
	width, height := 8, 6
	open := make([]bool, width*height)
	for index := range open {
		open[index] = true
	}
	input := SearchInput{Open: open, Camera: NewCamera(width, height, CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150}), Moon: [3]float64{1, 1, 1}, FarTree: Toward(109.65, 0).Scale(1000), Azimuth: 109.65, ShoreRow: 3, Level: 1, Gate: 1, Phase: 0.2, MovePhase: 0.4, Move: "fan"}
	config := SearchConfig{Gain: 0.1, Count: 2}
	cpuSky := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	cpuFront := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	AddSearchlights(cpuSky, cpuFront, config, input)
	planned, err := PlanSearchlights(width, height, config, input)
	if err != nil {
		t.Fatal(err)
	}
	gotSky := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	gotFront := Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	for _, splat := range planned {
		if splat.SkyOnly {
			splatSky(gotSky, open, splat.X, splat.Y, splat.Sigma, splat.Amount, splat.Colour)
		} else {
			Splat(gotSky, gotFront, splat.X, splat.Y, splat.Sigma, splat.Amount, splat.Colour, input.ShoreRow)
		}
	}
	for index := range cpuSky.Light {
		if math.Abs(cpuSky.Light[index]-gotSky.Light[index]) > 1e-12 || math.Abs(cpuFront.Light[index]-gotFront.Light[index]) > 1e-12 {
			t.Fatalf("channel %d differs", index)
		}
	}
}

func TestPlanSearchlightsRejectsZeroCount(t *testing.T) {
	if _, err := PlanSearchlights(1, 1, SearchConfig{}, SearchInput{Open: []bool{true}}); err == nil {
		t.Fatal("zero searchlights accepted")
	}
}
