//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestShadeFlatOceanIsFiniteAndRespondsToSky(t *testing.T) {
	scene := oceantex.Scene{Sea: oceantex.Sea{Wind: 8, Fetch: 1500}, Sun: oceantex.Vec{Z: 1}, Celsius: 15, Chlorophyll: 2}
	constants, err := scene.LightingConstants()
	if err != nil {
		t.Fatal(err)
	}
	samples := []ShadeSample{{Eye: oceantex.Vec{Z: 1}}, {Eye: oceantex.Vec{Z: 1}, Sky: [3]float32{1, 1, 1}}}
	got, err := Shade(constants, samples)
	if err != nil {
		t.Fatal(err)
	}
	for index := range got {
		for _, value := range got[index] {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				t.Fatal("nonfinite shade")
			}
		}
	}
	if got[1][0] <= got[0][0] {
		t.Fatalf("sky did not affect reflected light: %v", got)
	}
}

func TestShadeRejectsInvalidEye(t *testing.T) {
	scene := oceantex.Scene{Sea: oceantex.Sea{Wind: 8, Fetch: 1500}, Sun: oceantex.Vec{Z: 1}, Celsius: 15, Chlorophyll: 2}
	constants, err := scene.LightingConstants()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Shade(constants, []ShadeSample{{}}); err == nil {
		t.Fatal("zero eye accepted")
	}
}
