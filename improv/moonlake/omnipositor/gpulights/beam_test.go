//go:build draw

package gpulights

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/lights"
)

func TestBeamMatchesCPUForSyntheticScene(t *testing.T) {
	width, height := 4, 3
	sky := lights.Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	front := lights.Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	water := lights.Image{Width: width, Height: height, Light: make([]float64, width*height*3)}
	for index := range water.Light {
		water.Light[index] = 1
	}
	config := lights.UFOConfig{Out: 900, Radius: 90, Apex: 450, Gain: 40, Halo: 0.1, Azimuth: 109.65, WaterZ: -53.6}
	input := lights.UFOInput{Water: water, Camera: lights.NewCamera(width, height, lights.CameraSettings{Azimuth: 109.65, LookDown: -0.5, GaugeWidth: 70.41, Focal: 150}), Moon: [3]float64{1, 1, 1}, Seconds: 1, Cues: lights.UFOCues{Overhead: 1}}
	wantSky := lights.Image{Width: width, Height: height, Light: make([]float64, len(sky.Light))}
	wantFront := lights.Image{Width: width, Height: height, Light: make([]float64, len(front.Light))}
	lights.AddUFOBeam(wantSky, wantFront, config, input)
	waterMask := make([]bool, width*height)
	for index := range waterMask {
		waterMask[index] = true
	}
	gotSky, gotFront, err := Beam(sky, front, config, input, nil, waterMask)
	if err != nil {
		t.Fatal(err)
	}
	for index := range wantSky.Light {
		if math.Abs(gotSky.Light[index]-wantSky.Light[index]) > 0.01+0.01*math.Abs(wantSky.Light[index]) || math.Abs(gotFront.Light[index]-wantFront.Light[index]) > 0.01+0.01*math.Abs(wantFront.Light[index]) {
			t.Fatalf("channel %d: sky %v/%v front %v/%v", index, gotSky.Light[index], wantSky.Light[index], gotFront.Light[index], wantFront.Light[index])
		}
	}
}

func TestBeamRejectsInvalidGeometry(t *testing.T) {
	image := lights.Image{Width: 1, Height: 1, Light: []float64{0, 0, 0}}
	if _, _, err := Beam(image, image, lights.UFOConfig{}, lights.UFOInput{}, nil, nil); err == nil {
		t.Fatal("zero beam geometry accepted")
	}
}
