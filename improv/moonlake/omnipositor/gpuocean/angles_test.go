//go:build draw

package gpuocean

import (
	"math"
	"testing"

	"github.com/janearc/slayer/oceantex"
)

func TestMarchRaysCalculatesAdjacentAngles(t *testing.T) {
	camera := RayCamera{Width: 2, Height: 1}
	rays := []oceantex.Ray{{Dir: oceantex.Vec{Z: -1}}, {Dir: oceantex.Vec{X: 1, Z: 0}}}
	got, err := MarchRays(camera, rays)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got[0].PixelAngle-math.Pi/2) > 1e-12 || got[0].PixelAngle != got[1].PixelAngle {
		t.Fatalf("angles: %+v", got)
	}
}

func TestMarchRaysRefusesMismatchedGrid(t *testing.T) {
	if _, err := MarchRays(RayCamera{Width: 2, Height: 1}, nil); err == nil {
		t.Fatal("mismatched rays accepted")
	}
}
