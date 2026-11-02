package trace

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// Replay recomputes one Sampled pixel and returns its ordered calculation
// steps. It stages the same world, generates the same rays, and uses the same
// summation order so the resulting pixel matches Sampled bit for bit.
//
// Coordinates start at the top left. Out-of-bounds coordinates and requests
// rejected by Sampled return an error.
func Replay(world *thing.World, eye maths.Subspace, lens film.Sensor,
	moment time.Duration, width, height, rays, x, y int) (frame.Pixel,
	[]Step, error) {
	seen, err := newView(world, eye, lens, moment, width, height, rays)
	if err != nil {
		return frame.Pixel{}, nil, err
	}
	if x < 0 || y < 0 || x >= width || y >= height {
		return frame.Pixel{}, nil, fmt.Errorf("trace: pixel %d, %d is "+
			"not in a picture of %d by %d", x, y, width, height)
	}
	fine := seen.fine()
	scene := stage(fine.world, fine.at)
	walked := &walk{}
	scene.walked = walked
	wide := fine.wide()
	ray := func(over, down int) frame.Pixel {
		walked.ray = down*rays + over
		each := fine.through(scene, wide, x*rays+over, y*rays+down)
		walked.tell("the ray's light", "pixel", each)
		return each
	}
	var made frame.Pixel
	if rays == 1 {
		// Sampled keeps a one-ray frame as it was made, with no mean
		made = ray(0, 0)
	} else {
		made = averageOf(rays, ray)
	}
	walked.ray = -1
	walked.tell("the pixel", "pixel", made, "rays", rays*rays)
	return made, walked.steps, nil
}
