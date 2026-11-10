//go:build darwin

package gpu

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/scene/thing"
)

// Measurement must keep every returned float, including coverage and depth.
func TestMeasuredRenderingKeepsTheFrameAndAccountsForItsCosts(t *testing.T) {
	world, eye := hall(t)
	plain, err := Sampled(world, eye, film.Pinhole{}, 0, 16, 12, 2)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	measured, profile, err := SampledMeasured(world, eye, film.Pinhole{}, 0, 16, 12, 2)
	if err != nil {
		t.Fatal(err)
	}
	covered := 0
	for i, pixel := range plain.Pix {
		if pixel.Cover > 0 && !math.IsInf(float64(pixel.Depth), 1) {
			covered++
		}
		one := []float32{pixel.R, pixel.G, pixel.B, pixel.Cover, pixel.Depth}
		other := measured.Pix[i]
		two := []float32{other.R, other.G, other.B, other.Cover, other.Depth}
		for field, value := range one {
			if math.Float32bits(value) != math.Float32bits(two[field]) {
				t.Fatalf("pixel %d field %d changed", i, field)
			}
		}
	}
	if covered == 0 {
		t.Fatal("the comparison did not exercise any surface hits")
	}
	if profile.Device == "" || profile.InputBytes <= 0 {
		t.Fatal("missing device or input accounting")
	}
	if profile.SampleBytes != int64(32*24*per*4) {
		t.Fatalf("sample bytes: %d", profile.SampleBytes)
	}
	if profile.TraceGPU <= 0 || profile.BakeGPU <= 0 {
		t.Fatal("missing gpu times")
	}
	wall := profile.Encode + profile.Compile + profile.BakeWall + profile.DispatchWall + profile.Finish
	if wall > profile.Total || profile.TraceGPU > profile.DispatchWall || profile.BakeGPU > profile.BakeWall {
		t.Fatalf("inconsistent nested timing: %+v", profile)
	}
}

// A failed render still fails when measured, before any kernel is opened.
func TestMeasuredRenderingPreservesConstructionFailures(t *testing.T) {
	world := thing.NewWorld()
	world.Haze.Density = 1
	picture, profile, err := SampledMeasured(world, maths.Subspace{}, nil, 0, 16, 12, 1)
	var unsupportedFeature unsupported
	if !errors.As(err, &unsupportedFeature) || picture != nil {
		t.Fatalf("lost backend refusal: %v", err)
	}
	if profile.Encode <= 0 || profile.Total <= 0 || profile.Compile != 0 || profile.TraceGPU != 0 {
		t.Fatalf("failed construction reports work it did not do: %+v", profile)
	}
}
