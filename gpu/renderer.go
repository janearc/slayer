//go:build darwin

package gpu

import (
	"fmt"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
	"sync/atomic"
	"time"
)

// Renderer owns a volume-resolution policy, independent of other renders.
// Its bound limits CPU-baked volume grids; surface rays are unchanged.
type Renderer struct {
	volumeCells int
	timing      atomic.Pointer[FrameTiming]
	seeds       fieldSeedCache
}

// FrameTiming separates host preparation from device tracing and finishing.
// Bake and Trace use Metal timestamps; other fields measure host wall time.
type FrameTiming struct {
	Encode, Compile, Bake, Trace, Finish time.Duration
	SeedsCached                          bool
}

// LastTiming reports the most recent Sampled call, or zero before rendering.
func (renderer *Renderer) LastTiming() FrameTiming {
	if value := renderer.timing.Load(); value != nil {
		return *value
	}
	return FrameTiming{}
}

// NewRenderer selects a longest-side volume grid between 2 and 192 cells.
// Lower bounds approximate volume detail for smaller output images.
func NewRenderer(volumeCells int) (*Renderer, error) {
	if volumeCells < 2 || volumeCells > 192 {
		return nil, fmt.Errorf("gpu: volume grid needs 2 to 192 cells")
	}
	return &Renderer{volumeCells: volumeCells}, nil
}

// Sampled uses this renderer's explicit volume-grid policy.
func (renderer *Renderer) Sampled(world *thing.World, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, side int) (*frame.Frame, error) {
	var picture *frame.Frame
	var spent timing
	var err error
	if len(world.Fields) == MaxGravityFields {
		ops := fieldDefaults()
		ops.seeds = renderer.cachedFieldSeeds
		picture, spent, err = fieldSampledUsing(world, eye, lens, at, width, height, side, ops, renderer.volumeCells)
	} else {
		picture, spent, err = sampledTimed(world, nil, eye, lens, at, width, height, side, renderer.volumeCells)
	}
	renderer.timing.Store(&FrameTiming{spent.encode, spent.compile, spent.bake, spent.trace, spent.average, spent.seedCached})
	return picture, err
}

// SampledWithInitialMedia is Sampled with an explicit camera medium stack.
// A gravity field cannot carry that state through its separate geodesic
// kernel, so the combination is refused before opening a device kernel.
func (renderer *Renderer) SampledWithInitialMedia(world *thing.World,
	eye maths.Subspace, lens film.Sensor, at time.Duration, width, height,
	side int, media []trace.InitialMedium) (*frame.Frame, error) {
	if err := trace.ValidateClearState(world, media); err != nil {
		return nil, err
	}
	if len(world.Fields) > 0 {
		if len(media) > 0 {
			return nil, unsupported{"initial camera media with gravity fields"}
		}
		return renderer.Sampled(world, eye, lens, at, width, height, side)
	}
	picture, spent, err := sampledBackendWithInitialMedia(world, nil, eye,
		lens, at, width, height, side, softwareTraversal, media,
		renderer.volumeCells)
	renderer.timing.Store(&FrameTiming{spent.encode, spent.compile,
		spent.bake, spent.trace, spent.average, spent.seedCached})
	return picture, err
}
