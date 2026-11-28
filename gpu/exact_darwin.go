//go:build darwin

package gpu

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies/scrooge"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// ExactMeasured dispatches exactly count samples per pixel through Metal.
func ExactMeasured(world *thing.World, eye maths.Subspace, lens film.Sensor,
	at time.Duration, width, height, count int, media []trace.InitialMedium,
) (*frame.Frame, Profile, error) {
	return ExactWithInitialMedia(Software, world, eye, lens, at, width, height, count, media)
}

// ExactWithInitialMedia draws exact primary sample totals with the chosen
// traversal. The scene encoder and native acceleration are shared with
// legacy draws, while the rays retain their exact per-pixel layout.
func ExactWithInitialMedia(mode Mode, world *thing.World, eye maths.Subspace,
	lens film.Sensor, at time.Duration, width, height, count int,
	media []trace.InitialMedium,
) (*frame.Frame, Profile, error) {
	selected, err := traversalFor(mode)
	if err != nil {
		return nil, Profile{}, err
	}
	switch mode {
	case Software, NativeMeshes, NativeShadows:
	default:
		return nil, Profile{}, unsupported{"exact samples with accelerated lighting"}
	}
	started := time.Now()
	var spent timing
	if width < 1 || height < 1 || count < 1 || count > 64 || width > math.MaxInt32/count {
		return nil, Profile{}, fmt.Errorf("gpu: invalid exact sample dimensions or count")
	}
	fineWidth := width * count
	stride := (per + perRecord*2) * 4
	if fineWidth > math.MaxInt/stride/height || height > math.MaxInt32 {
		return nil, Profile{}, fmt.Errorf("gpu: exact sample payload exceeds the supported size")
	}
	if lens == nil {
		lens = film.Pinhole{}
	}
	if _, ok := pinholeOf(lens); !ok {
		return nil, Profile{}, unsupported{"exact samples through a non-pinhole lens"}
	}
	encoded, firsts, err := preparedWithInitialMedia(world, nil, eye, lens, at,
		[2]int{fineWidth, height}, selected, cellsFor(selected), nil, media, &spent)
	if err != nil {
		return nil, spent.profile(time.Since(started)), err
	}
	ramp, _ := world.Filter.Ramp()
	encoded.looked = true
	encoded.through = &through{eye: eye, lens: lens, width: fineWidth, height: height,
		exact: count, pictureWidth: width, pictureHeight: height, ramp: ramp}
	if encoded.through.whole() {
		encoded.rays.bytes = encoded.through.rays(firstRow, height)
	}
	headerStruct.patch(encoded.header.bytes, values{"look": wholes(1, encoded.counts.beams,
		encoded.counts.fields, flag(scrooge.Asked()))})
	start := time.Now()
	kernel, err := metal.Open(encoded.source(selected), "trace")
	spent.compile = time.Since(start)
	if err != nil {
		return nil, spent.profile(time.Since(started)), err
	}
	defer kernel.Close()
	kernel.FixBands(bandRows)
	if selected != softwareTraversal {
		if err := encoded.accelerate(kernel, &spent); err != nil {
			return nil, spent.profile(time.Since(started)), err
		}
	}
	spent.device = kernel.Device()
	inputs := encoded.inputs()
	if selected.primary() {
		inputs = append(inputs, firsts)
	}
	if err = encoded.lit(kernel, copied(kernel, inputs), selected, &spent); err != nil {
		return nil, spent.profile(time.Since(started)), err
	}
	for _, input := range inputs {
		spent.inputBytes += int64(len(input))
	}
	start = time.Now()
	band := newBand(world, encoded.paints, encoded.records(), eye, lens,
		[2]int{width, height}, width, height, 1)
	band.exact = count
	setup := time.Since(start)
	start = time.Now()
	made, err := encoded.drawn(kernel, inputs, fineWidth, height, &band)
	spent.inputBytes += made
	spent.banded(&band, time.Since(start), setup)
	spent.trace = traceDeviceTime(kernel.GPUTime(), spent.accelerationGPU, spent.lighting, spent.environment)
	if err != nil {
		return nil, spent.profile(time.Since(started)), err
	}
	wantBytes := int64(width) * int64(height) * int64(count) * int64(per+perRecord*encoded.records()) * 4
	if band.bytes != wantBytes {
		return nil, spent.profile(time.Since(started)), fmt.Errorf("gpu: exact draw received %d sample bytes, expected %d", band.bytes, wantBytes)
	}
	profile := spent.profile(time.Since(started))
	profile.PrimarySamples = band.bytes / int64((per+perRecord*encoded.records())*4)
	return band.picture, profile, nil
}
