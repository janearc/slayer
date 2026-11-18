package hillside

// A job's lens and its shutter's share, as a session draws them.
//
// Design. A job's film may say an aperture and a focus, both in the
// scene's own unit. The session makes a thin lens of them for each frame:
// one that sees the frame's angle across the frame's own shape, open as
// wide as the aperture and focused as far off as the focus. A film
// through it is drawn on the cpu or the gpu; on the gpu the kernel makes
// its rays (gpu/exposure.go).
//
// A set's rig is another lens, with its own frame, exposure and shutter
// (session_rig.go). A job's aperture over a rig is refused by name. On
// the gpu a rig's thin lens is made by the same kernel, a draw at a time.
//
// A job's shutter is drawn in glimpses (session_shutter.go). On the gpu
// each glimpse also holds the shutter open over its own share of the open
// interval: the eye is the camera path's at the share's start, and where
// the shutter closes it is the path's at the share's end. The kernel
// gives each sample its own time between the two.
//
// So a camera's move blurs evenly across the whole open interval at one
// draw a glimpse, and one glimpse is enough for it. The scene is still
// drawn at each glimpse's own time, its middle: things that move in the
// scene are blurred by glimpses, as before, and by nothing else.
//
// On the cpu nothing changes: each glimpse is drawn from the path at its
// own time, and the eye does not move inside it.

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
)

// refuseGPUInitialMediaAperture keeps an explicit camera stack from being
// applied to thin-lens rays whose origins are spread across an aperture.
// A point-camera stack does not prove that every aperture origin is inside
// the named media. CPU tracing retains its existing lens behavior.
func refuseGPUInitialMediaAperture(given asked) error {
	if !given.onGPU || len(given.initialMedia) == 0 {
		return nil
	}
	if given.glass == nil && given.optics.aperture == 0 &&
		given.optics.opened == nil {
		return nil
	}
	return fmt.Errorf("%s: GPU initial media currently require a pinhole camera; rig and thin-lens aperture rays may start outside the declared medium", given.set)
}

// optics is a job's lens and shutter as one draw takes them.
type optics struct {
	// aperture is how wide the job's lens is open, across, and focus how
	// far off it is focused, both in the scene's unit; zero is a pinhole
	aperture, focus float64
	// opened is the job's lens made for this draw's angle and shape
	opened *lens.Thin
	// share is how long this draw's share of the open shutter is, and
	// closing the eye where it ends, on the gpu; nil is a closed shutter
	share   time.Duration
	closing *maths.Subspace
}

// perMetre is defined as the millimetres in a metre, which lens.FrameOf
// and lens.NewThin take their frame and focal length in.
const perMetre = 1000

// exposing is withCamera, then the job's optics for a draw at a time:
// its lens made for the angle and shape the draw has, and on the gpu the
// eye at each end of the draw's share of the shutter, the time its middle.
func (plan sequencePlan) exposing(given *asked, at time.Duration) error {
	if err := plan.withCamera(given, at); err != nil {
		return err
	}
	if err := given.openLens(); err != nil {
		return err
	}
	share := given.optics.share
	if plan.camera == nil || !given.onGPU || share <= nothing {
		return nil
	}
	elapsed := at - plan.start
	opens, err := plan.camera.At(elapsed - share/shareHalves)
	if err != nil {
		return fmt.Errorf("sequence: %w", err)
	}
	closes, err := plan.camera.At(elapsed + share/shareHalves)
	if err != nil {
		return fmt.Errorf("sequence: %w", err)
	}
	given.eye, given.optics.closing = opens, closes
	return nil
}

// openLens makes the job's lens for what is asked now, where the job says an
// aperture: a thin lens that sees the angle asked across a frame of the
// picture's shape. Its film stands as far behind it as its focus is in
// front, so its focal length is half the focus distance; any lens of the
// same aperture, focus and angle makes the same rays.
//
// It refuses an aperture over a set's rig, by name.
func (given *asked) openLens() error {
	wanted := given.optics
	if wanted.aperture == nothing {
		return nil
	}
	if given.glass != nil {
		return fmt.Errorf("%s: a film with a rig takes its lens from the "+
			"rig; the job's aperture and focus cannot stand over it",
			given.set)
	}
	image := wanted.focus
	across := shareHalves * image * math.Tan(given.wide*maths.Degree/shareHalves)
	up := across * float64(given.height) / float64(given.width)
	made, err := lens.FrameOf(across*perMetre, up*perMetre)
	if err != nil {
		return fmt.Errorf("hillside: the job's lens: %w", err)
	}
	focal := wanted.focus / shareHalves
	glass, err := lens.NewThin(made, focal*perMetre, focal/wanted.aperture,
		wanted.focus)
	if err != nil {
		return fmt.Errorf("hillside: the job's lens, open %g and focused "+
			"at %g: %w", wanted.aperture, wanted.focus, err)
	}
	given.optics.opened = &glass
	return nil
}

// footprint tells a world the angle one ray of the picture sees, where
// the job's lens is open: rays from across an aperture do not say it.
func (given asked) footprint(world *thing.World) {
	if given.optics.opened != nil {
		world.Pixel = given.optics.opened.Pixel(given.width *
			given.samples)
	}
}

// sightFor is the lens a hud's labels are placed through. The job's
// opened lens places them as the pinhole of its angle does, through its
// middle, where the thing is sharpest; any other lens is its own.
func sightFor(given asked) film.Sensor {
	if given.glass == nil && given.optics.opened != nil {
		return film.Pinhole{Wide: given.wide * maths.Degree}
	}
	return lensFor(given)
}

// drawnBy is the picture a retained drawer draws of a world from an eye
// as asked: with the shutter held open to the closing eye where the
// draw's share has one, and closed otherwise.
func (given asked) drawnBy(frames *gpu.Frames, world *thing.World,
	generation uint64, eye maths.Subspace) (*frame.Frame, gpu.Profile,
	error) {
	if err := refuseGPUInitialMediaAperture(given); err != nil {
		return nil, gpu.Profile{}, err
	}
	if given.wire {
		// a retained drawer is the gpu's
		return nil, gpu.Profile{}, errWireOnGPU
	}
	if closing := given.optics.closing; closing != nil {
		if len(given.initialMedia) > 0 {
			return frames.DrawOpenWithInitialMedia(world, generation, eye,
				*closing, lensFor(given), given.at, given.width,
				given.height, given.samples, given.initialMedia)
		}
		return frames.DrawOpen(world, generation, eye, *closing,
			lensFor(given), given.at, given.width, given.height,
			given.samples)
	}
	if len(given.initialMedia) > 0 {
		return frames.DrawWithInitialMedia(world, generation, eye,
			lensFor(given), given.at, given.width, given.height,
			given.samples, given.initialMedia)
	}
	return frames.Draw(world, generation, eye, lensFor(given), given.at,
		given.width, given.height, given.samples)
}

// drawOpen is one picture drawn with the shutter held open, by a drawer
// made for it and let go after: for a draw no retained drawer is kept
// for.
func drawOpen(world *thing.World, given asked, eye maths.Subspace) (
	*frame.Frame, gpu.Profile, error) {
	frames, err := gpu.NewFrames(framesMode(given))
	if err != nil {
		return nil, gpu.Profile{}, err
	}
	defer frames.Close()
	return given.drawnBy(frames, world, wholeGeneration, eye)
}

// wholeGeneration is defined as the generation of a world drawn once.
const wholeGeneration = 1
