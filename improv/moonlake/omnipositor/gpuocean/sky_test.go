//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/oceantex"
)

func TestSkySamplesProjectionAndFallback(t *testing.T) {
	image := ocean.RGB{Width: 2, Height: 2, Light: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	projection := SkyProjection{Forward: oceantex.Vec{X: 1}, Right: oceantex.Vec{Y: 1}, Up: oceantex.Vec{Z: 1}, Focal: 1, Normalization: 2}
	got, err := SkySamples(image, projection, []oceantex.Vec{{X: 1}, {X: -1, Z: 1}, {X: -1, Z: -1}})
	if err != nil {
		t.Fatal(err)
	}
	want := [][3]float32{{2, 2.5, 3}, {4.25, 4.75, 5.25}, {0, 0, 0}}
	for index := range want {
		for channel := range want[index] {
			if math.Abs(float64(got[index][channel]-want[index][channel])) > 1e-5 {
				t.Fatalf("sample %d: %v want %v", index, got[index], want[index])
			}
		}
	}
}

func TestSkySamplesRejectsZeroDirection(t *testing.T) {
	image := ocean.RGB{Width: 1, Height: 1, Light: []float32{1, 1, 1}}
	projection := SkyProjection{Forward: oceantex.Vec{X: 1}, Right: oceantex.Vec{Y: 1}, Up: oceantex.Vec{Z: 1}, Focal: 1, Normalization: 1}
	if _, err := SkySamples(image, projection, []oceantex.Vec{{}}); err == nil {
		t.Fatal("zero direction accepted")
	}
}
