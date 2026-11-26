package trace

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// Exact traces exactly count rays per pixel, using film's low-discrepancy
// places and averaging linear light and coverage. Unlike Sampled, count is a
// total, not the number along each side. A grid filter uses the box shape for
// this arbitrary-count operation; legacy Sampled keeps its regular grid.
// This primitive is not yet connected to the job or GPU paths.
func Exact(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, count int,
	media []InitialMedium) (*frame.Frame, error) {
	return ExactKept(world, eye, lens, moment, width, height, count, media, Keeping{})
}

// ExactKept traces a total count per pixel, stopping between samples and
// retaining complete picture rows through keeping. Kept rows must be keyed
// by the caller to the exact sample count and every other drawing input.
func ExactKept(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, count int,
	media []InitialMedium, keeping Keeping) (*frame.Frame, error) {
	seen, err := newView(world, eye, lens, moment, width, height, 1)
	if err != nil {
		return nil, err
	}
	if count < 1 || count > mostPixels/width/height {
		return nil, fmt.Errorf("trace: %d samples per pixel at %d by %d exceeds the supported ray count", count, width, height)
	}
	if lens == nil {
		lens = film.Pinhole{}
	}
	if _, ok := lens.(film.Placed); !ok && count > 1 {
		return nil, fmt.Errorf("trace: exact sampling requires a lens that can look through a place in a pixel")
	}
	seen.lens = lens
	seen, err = seen.withInitialMedia(media)
	if err != nil {
		return nil, err
	}
	if keeping.told() {
		return nil, keeping.stop.Err()
	}
	scene, err := seen.staging()
	if err != nil {
		return nil, err
	}
	wide := seen.wide()
	ramp, _ := world.Filter.Ramp()
	if scene.walked != nil {
		keeping.rows = nil
	}
	drawn, drawErr := exactRows(keeping, width, height, func(x, y int) (frame.Pixel, bool) {
		return exactMean(count, func(index int) (frame.Pixel, bool) {
			if keeping.told() {
				return frame.Pixel{}, false
			}
			across, down := film.Place(index, count, ramp)
			ray := film.LookThrough(eye, lens, x, y, across, down, width, height, maths.Band{})
			asked := look{ray: ray, pixel: wide, grain: wide}
			if seen.holding > 0 {
				asked.inside, asked.holding = seen.inside, seen.holding
			}
			return shade(scene, asked), true
		})
	}, func() bool { return scene.refusal() == nil })
	if err := scene.refusal(); err != nil {
		return nil, err
	}
	if drawErr != nil {
		return nil, drawErr
	}
	return drawn, nil
}
