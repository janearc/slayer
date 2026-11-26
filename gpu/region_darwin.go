//go:build darwin

package gpu

import (
	"image"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// sampledRegion draws a bounded region with the software Metal backend.
func sampledRegion(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, side int, region image.Rectangle,
) (*frame.Frame, Profile, error) {
	started := time.Now()
	var spent timing
	if _, ok := pinholeOf(lens); !ok || len(world.Fields) != 0 || workCounting || onlyWindow != "" {
		return nil, Profile{}, ErrRegionUnavailable
	}
	fineWidth, fineHeight, err := checkedFineSize(width, height, side)
	if err != nil {
		return nil, Profile{}, err
	}
	encoded, _, err := prepared(world, nil, eye, lens, at,
		[2]int{fineWidth, fineHeight}, softwareTraversal, cellsFor(softwareTraversal), nil, &spent)
	if err != nil {
		return nil, Profile{}, err
	}
	if encoded.looked || encoded.opticalState {
		return nil, Profile{}, ErrRegionUnavailable
	}
	encoded.spreadOver(world, side)
	encoded.sideFor(lens, side)
	start := time.Now()
	kernel, err := metal.Open(encoded.source(softwareTraversal), "trace")
	spent.compile = time.Since(start)
	if err != nil {
		return nil, Profile{}, err
	}
	defer kernel.Close()
	kernel.FixBands(bandRows)
	spent.device = kernel.Device()
	inputs := encoded.inputs()
	if err := encoded.lit(kernel, copied(kernel, inputs), softwareTraversal, &spent); err != nil {
		return nil, Profile{}, err
	}
	for _, input := range inputs {
		spent.inputBytes += int64(len(input))
	}
	band := newBand(world, encoded.paints, encoded.records(), eye, lens,
		[2]int{fineWidth, fineHeight}, region.Dx(), region.Dy(), side)
	start = time.Now()
	err = kernel.RegionBandsSent(region.Min.X*side, region.Min.Y*side,
		region.Dx()*side, region.Dy()*side, rows, encoded.perSample(), inputs,
		encoded.releasePacked(inputs), band.consume)
	spent.banded(&band, time.Since(start), 0)
	spent.trace = traceDeviceTime(kernel.GPUTime(), spent.accelerationGPU,
		spent.lighting, spent.environment)
	if err != nil {
		return nil, spent.profile(time.Since(started)), err
	}
	return band.picture, spent.profile(time.Since(started)), nil
}
