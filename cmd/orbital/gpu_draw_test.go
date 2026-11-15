//go:build draw

package main

import (
	"errors"
	"io"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/named"
	"github.com/janearc/slayer/trace"
)

// the picture the orbital GPU test draws: 160 by 90, one ray a pixel, the
// frame the command draws at 500 milliseconds, with the scene told the
// time as draw tells it.
const (
	gpuWide, gpuHigh, gpuSide = 160, 90, 1
	gpuAt                     = 500 * time.Millisecond
	gpuView                   = 90 * maths.Degree
)

// the bounds a view is held to, as the gpu tests measure a difference: the
// mean and the largest pixel. measured on 2026-11-10 at 1.3e-5 and 9.6e-5
// for the near view and 7.4e-6 and 4.4e-5 for the far; with no gas at all
// 1.6e-5 and 1.0e-4, and 6.4e-6 and 4.6e-5. The mean is over the gpu
// package's exact 1e-5 for the near view because of that, not of the gas.
// The bounds are twice or three times the measurements; before the nebula
// was asked by the kernel the views were 3.8e-2 and 3.0e-2 apart.
const (
	orbitalMean    = 3e-5
	orbitalLargest = 3e-4
)

// apart is how far the second frame is from the first, as the gpu tests
// measure it: the sum over pixels of the differences in each channel as a
// share of the first frame's light, and the largest difference in any
// one pixel as a share of its brightest. notFinite counts the pixels the
// difference of which is not a number.
type apart struct {
	mean, largest float64
	notFinite     int
}

// measure is how far drawn is from truth, as gpu's compare reads it.
func measure(truth, drawn *frame.Frame) apart {
	var found apart
	var total, brightest float64
	for row := range truth.H {
		for column := range truth.W {
			one, other := truth.At(column, row), drawn.At(column, row)
			light := float64(one.R + one.G + one.B)
			difference := math.Abs(float64(one.R-other.R)) +
				math.Abs(float64(one.G-other.G)) +
				math.Abs(float64(one.B-other.B))
			total += light
			brightest = math.Max(brightest, light)
			if math.IsNaN(difference) || math.IsInf(difference, 0) {
				found.notFinite++
			}
			found.mean += difference
			found.largest = math.Max(found.largest, difference)
		}
	}
	if total > 0 {
		found.mean /= total
		found.largest /= brightest
	}
	return found
}

// the orbital's near view and its far view each drawn on the gpu and by
// the tracer, at the nebula and the day the scene has by default. The
// gpu must draw them, and draw numbers, within the bounds below of the
// tracer's picture. The nebula is a library medium the kernel asks itself,
// as the tracer does, so it adds nothing to the difference: the same
// pictures with no gas at all are 1.6e-5 and 6.4e-6 apart on average, which
// is the bands' and the lamps' own float32 against float64. The times are
// logged so the next piece of work has a number to bring down.
func TestTheGPUDrawsOrbitalsViews(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the kernel is Metal, mac only")
	}
	views := []struct {
		name string
		with map[string]any
	}{{"near", nil}, {"far", map[string]any{"far": true}}}
	for _, view := range views {
		t.Run(view.name, func(t *testing.T) { drawnBoth(t, view.with) })
	}
}

// drawnBoth builds a view as the command does and draws it both ways.
func drawnBoth(t *testing.T, with map[string]any) {
	scene, err := named.Get("orbital")
	if err != nil {
		t.Fatal(err)
	}
	built, err := scene.Build(with, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	world, eye := built.World(), built.Eye()
	built.Set(gpuAt)
	lens := lensFor(printing{wide: gpuView}, square)
	start := time.Now()
	truth, err := trace.Sampled(world, eye, lens, gpuAt, gpuWide, gpuHigh,
		gpuSide)
	cpuWall := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	drawn, err := gpu.Sampled(world, eye, lens, gpuAt, gpuWide, gpuHigh,
		gpuSide)
	gpuWall := time.Since(start)
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatalf("the gpu refuses it: %v", err)
	}
	start = time.Now()
	if _, err := gpu.Sampled(world, eye, lens, gpuAt, gpuWide, gpuHigh,
		gpuSide); err != nil {
		t.Fatalf("the gpu refuses it the second time: %v", err)
	}
	again := time.Since(start)
	found := measure(truth, drawn)
	t.Logf("things %d, volumes %d, lamps %d", len(world.Things),
		len(world.Volumes), len(world.Lamps))
	t.Logf("mean %g, largest %g, not finite %d", found.mean,
		found.largest, found.notFinite)
	t.Logf("cpu %v, gpu %v, gpu again %v", cpuWall, gpuWall, again)
	if found.notFinite > 0 {
		t.Errorf("%d pixels of the gpu's picture are not finite",
			found.notFinite)
	}
	if !(found.mean <= orbitalMean && found.largest <= orbitalLargest) {
		t.Errorf("the gpu's picture is %g from the tracer's on average and "+
			"%g at most, want within %g and %g", found.mean, found.largest,
			orbitalMean, orbitalLargest)
	}
}
