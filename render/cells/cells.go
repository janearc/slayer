// Package cells renders a scene as a stream of frames. Scene implements
// terminal.Source: frame number and pixel dimensions determine a render at the
// corresponding scene time.
//
// This package does not manage terminal windows or output. The terminal package
// converts its frames to cells and writes them.
package cells

import (
	"fmt"
	"image"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// fewest is the fewest frames a second, and the fewest rays a side, that
// make sense. It is defined as one.
const fewest = 1

// none is no time at all. It is defined as zero.
const none = 0

// Pace is when each frame of a stream happens in the scene: the time of
// frame zero, and how many frames a second after it.
//
//ninja:vow practice NewPace makes one and New takes it
type Pace struct {
	start time.Duration
	rate  int
}

// NewPace is a pace that starts at start and runs rate frames a second. It
// refuses a rate under one and a start before zero.
func NewPace(start time.Duration, rate int) (Pace, error) {
	if rate < fewest || start < none {
		return Pace{}, fmt.Errorf("cells: a pace starts at nought "+
			"or later and runs at least %d frame a second, not "+
			"%v at %d", fewest, start, rate)
	}
	return Pace{start: start, rate: rate}, nil
}

// At is the scene's time at frame number.
//
//ninja:vow practice the scene asks it the time of each frame
func (pace Pace) At(number int) time.Duration {
	step := time.Duration(number) * time.Second / time.Duration(pace.rate)
	return pace.start + step
}

// Clock is told the time of each frame before it is drawn. A scene whose
// lamps keep their own time, as the lantern's beams do, needs this.
//
//ninja:vow practice New takes one, and orbital's beams is one
type Clock interface {
	// Set is the time the frame about to be drawn happens at.
	Set(at time.Duration)
}

// Scene is a world seen by an eye, drawn frame by frame.
type Scene struct {
	world *thing.World
	eye   maths.Subspace
	lens  film.Sensor
	tone  colour.Tone
	rays  int
	pace  Pace
	clock Clock
}

// New is a scene drawn through lens and developed with tone. It casts
// rays by rays through every pixel. The frames happen at pace. The clock
// may be nil.
//
// The world and the eye come later, from Place, so that a bad ask can be
// refused before a world is built. A scene with no lens or fewer than one
// ray is refused.
func New(lens film.Sensor, tone colour.Tone, rays int, pace Pace,
	clock Clock) (*Scene, error) {
	if lens == nil || rays < fewest {
		return nil, fmt.Errorf("cells: a scene needs a lens and "+
			"at least %d ray a side, not %v and %d", fewest, lens,
			rays)
	}
	return &Scene{lens: lens, tone: tone, rays: rays, pace: pace,
		clock: clock}, nil
}

// Place puts the world and the eye in the scene.
func (scene *Scene) Place(world *thing.World, eye maths.Subspace) {
	scene.world = world
	scene.eye = eye
}

// Frame is frame number, wide pixels across and high down. The clock is
// set first. The light is developed with the scene's tone. A scene with
// no world placed is refused by the tracer.
func (scene *Scene) Frame(number, wide, high int) (image.Image, error) {
	at := scene.pace.At(number)
	if scene.clock != nil {
		scene.clock.Set(at)
	}
	drawn, err := trace.Sampled(scene.world, scene.eye, scene.lens, at,
		wide, high, scene.rays)
	if err != nil {
		return nil, err
	}
	return scene.tone.Develop(drawn), nil
}
