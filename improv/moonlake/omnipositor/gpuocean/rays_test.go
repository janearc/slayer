//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestRaysMatchCPUCamera(t *testing.T) {
	gauge, err := oceantex.NewGauge("test", 70.41, 39.61)
	if err != nil {
		t.Fatal(err)
	}
	eye := oceantex.Subspace{Origin: oceantex.Vec{Z: 71.7}, Orientation: oceantex.LookAt(oceantex.Vec{Z: 71.7}, oceantex.Vec{X: 1000, Z: 1})}
	camera := RayCamera{Width: 5, Height: 3, Focal: 85, GaugeWidth: 70.41, Eye: eye}
	got, err := Rays(camera)
	if err != nil {
		t.Fatal(err)
	}
	lens := oceantex.NewLens(camera.Focal, gauge, 0)
	for y := 0; y < camera.Height; y++ {
		for x := 0; x < camera.Width; x++ {
			want := oceantex.Look(eye, lens, x, y, camera.Width, camera.Height)
			ray := got[(camera.Height-1-y)*camera.Width+x]
			for _, delta := range []float64{ray.Dir.X - want.Dir.X, ray.Dir.Y - want.Dir.Y, ray.Dir.Z - want.Dir.Z} {
				if math.Abs(delta) > 1e-5 {
					t.Fatalf("pixel %d,%d: GPU %+v CPU %+v", x, y, ray, want)
				}
			}
		}
	}
}

func TestRaysRefuseInvalidCamera(t *testing.T) {
	for _, camera := range []RayCamera{{Width: 0, Height: 1, Focal: 85, GaugeWidth: 70}, {Width: 1, Height: 1, Focal: 0, GaugeWidth: 70}, {Width: 1, Height: 1, Focal: 85, GaugeWidth: math.Inf(1)}, {Width: 1, Height: 1, Focal: 85, GaugeWidth: 70, Eye: oceantex.Subspace{Orientation: oceantex.Orientation{X: oceantex.Vec{X: 1}}}}} {
		if _, err := Rays(camera); err == nil {
			t.Fatalf("accepted %+v", camera)
		}
	}
}
