//go:build darwin

package gpu

import (
	"fmt"
	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
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
