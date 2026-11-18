package hillside

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/lens"
	"github.com/janearc/slayer/render/carry"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/take"
	"github.com/janearc/slayer/trace"
)

// rigged is what was asked with the set's rig in it: the picture's rows
// from the frame, so its pixels are square; the shutter and its
// glimpses; and the light at one to a watt a square metre a steradian,
// so the raw is joules a square metre.
//
// the lens is made once the eye is known, by focused.
func rigged(rig set.Rig, given asked) (asked, error) {
	if given.said.exposure || given.said.wide {
		return given, fmt.Errorf("%s: a film with a rig takes its "+
			"exposure and its angle from the rig; the job's exposure, "+
			"wide or tall cannot stand over it", given.set)
	}
	width, height := rig.Frame()
	made, err := lens.FrameOf(width, height)
	if err != nil {
		return given, fmt.Errorf("%s: %w", given.set, err)
	}
	if err := framedAs(made, width, height, given); err != nil {
		return given, err
	}
	given.frame = &made
	given.height = made.Rows(given.width)
	given.exposure = 1
	seconds, glimpses := rig.Shutter()
	given.open = time.Duration(math.Round(seconds * float64(time.Second)))
	given.glimpses = glimpses
	return given, nil
}

// framedAs refuses a height said beside a rig whose frame has another
// shape: -h, or a job's film height or preset. A size of the frame's
// shape, its rows within a pixel of the frame's for its width, is taken,
// and the height is the frame's.
func framedAs(made lens.Frame, width, height float64, given asked) error {
	if !given.said.height {
		return nil
	}
	rows := made.Rows(given.width)
	if math.Abs(float64(given.height-rows)) <= rowSlack {
		return nil
	}
	return fmt.Errorf("%s: -h, or a job's film height or preset, "+
		"%d by %d, is %s, and a film with a rig takes its height from "+
		"its frame, %g by %g mm, which is %s; at %d across it is %d rows",
		given.set, given.width, given.height,
		shaped(float64(given.width), float64(given.height)), width,
		height, shaped(width, height), given.width, rows)
}

// rowSlack is defined as the pixel a size may be from a frame's shape and
// be taken as it: a preset's rows are whole, a frame's are rounded.
const rowSlack = 1

// shaped is a picture's shape in a word: wide, tall or square.
func shaped(across, down float64) string {
	for _, each := range []struct {
		is   bool
		word string
	}{{across > down, "wide"}, {across < down, "tall"}} {
		if each.is {
			return each.word
		}
	}
	return "square"
}

// focusing is what a rig's lens is made of and focused on: the point the
// set names, the focal length in millimetres and the f-number. A film
// focuses the lens again from each eye with it.
type focusing struct {
	on          maths.Vec
	focal, stop float64
}

// refocused is the rig's lens focused on its point from an eye: the
// distance from the eye to the point, as focused first made it.
func (given asked) refocused(eye maths.Subspace) (lens.Thin, error) {
	if given.frame == nil || given.lensed == nil {
		return lens.Thin{}, fmt.Errorf("%s: a lens is focused only "+
			"through a rig", given.set)
	}
	made := given.lensed
	glass, err := lens.NewThin(*given.frame, made.focal, made.stop,
		made.on.Sub(eye.Origin).Len())
	if err != nil {
		return lens.Thin{}, fmt.Errorf("%s: %w", given.set, err)
	}
	return glass, nil
}

// focused is the rig's lens, focused on the thing the set names at its
// distance from the eye; and the world told the angle one ray of the
// picture sees, since rays from across the aperture do not say it.
func (stage *staging) focused(rig set.Rig, eye maths.Subspace) error {
	focal, stop, focus := rig.Lens()
	at, found := stage.places[focus]
	if !found {
		return fmt.Errorf("%s: the lens is focused on %q, which "+
			"is not on the set", stage.given.set, focus)
	}
	stage.given.lensed = &focusing{on: at, focal: focal, stop: stop}
	glass, err := stage.given.refocused(eye)
	if err != nil {
		return err
	}
	stage.given.glass = &glass
	fine := stage.given.width * stage.given.samples
	stage.world.Pixel = glass.Pixel(fine)
	return nil
}

// instants is when the rig's shutter takes each glimpse, from the
// picture's moment: slayer's take, one frame cranked once over the time
// the shutter is open, its disc open for the whole turn.
func instants(given asked, eye maths.Subspace) ([]time.Duration, error) {
	pace, err := take.Cranked(1/given.open.Seconds(), 2*math.Pi,
		given.glimpses)
	if err != nil {
		return nil, err
	}
	shot, err := take.Take(thing.Still(eye), pace, given.open)
	if err != nil {
		return nil, err
	}
	var moments []time.Duration
	for _, caught := range shot.Glimpses(0) {
		moments = append(moments, given.at+caught.At())
	}
	return moments, nil
}

// exposed is the picture the rig takes: the world drawn at each instant
// through the lens, on the cpu or the gpu as asked, and the lens's
// exposure of them. With it comes what the device did for the last
// instant drawn, which for a rig of one glimpse is the whole picture; on
// the cpu it is empty.
func exposed(world *thing.World, given asked, eye maths.Subspace) (
	*frame.Frame, gpu.Profile, error) {
	if err := refuseGPUInitialMediaAperture(given); err != nil {
		return nil, gpu.Profile{}, err
	}
	moments, err := instants(given, eye)
	if err != nil {
		return nil, gpu.Profile{}, err
	}
	var measured gpu.Profile
	glimpses := make([]*frame.Frame, 0, len(moments))
	for _, moment := range moments {
		drawn, profile, err := glimpsed(world, given, eye, moment)
		if err != nil {
			return nil, profile, err
		}
		glimpses, measured = append(glimpses, drawn), profile
	}
	raw, err := given.glass.Expose(given.open, carry.NinjaFrames(glimpses))
	return carry.Slayer(raw), measured, err
}

// glimpsed is the world drawn at one instant through the rig's lens, by
// the renderer asked for, and what the device did.
func glimpsed(world *thing.World, given asked, eye maths.Subspace,
	moment time.Duration) (*frame.Frame, gpu.Profile, error) {
	if err := refuseGPUInitialMediaAperture(given); err != nil {
		return nil, gpu.Profile{}, err
	}
	glass, width, height, side := *given.glass, given.width, given.height,
		given.samples
	switch {
	case !given.onGPU:
		drawn, err := sampled(world, given, eye, glass, moment, width, height,
			side)
		return drawn, gpu.Profile{}, err
	case len(given.initialMedia) > 0:
		return gpu.SampledWithInitialMedia(framesMode(given), world, eye,
			glass, moment, width, height, side, given.initialMedia)
	case given.nativeMeshes:
		return gpu.SampledAccelerated(world, eye, glass, moment, width,
			height, side)
	case given.nativeShadows:
		return gpu.SampledNative(world, eye, glass, moment, width, height,
			side)
	}
	return gpu.SampledMeasured(world, eye, glass, moment, width, height, side)
}

// replayed is one pixel of the rig's picture made again: the pixel at
// each instant, exposed as the whole picture's was, and every step the
// replays took, in order.
func replayed(world *thing.World, given asked, eye maths.Subspace, x,
	y int) (frame.Pixel, []trace.Step, error) {
	moments, err := instants(given, eye)
	if err != nil {
		return frame.Pixel{}, nil, err
	}
	var seen []frame.Pixel
	var steps []trace.Step
	for _, moment := range moments {
		pixel, walked, err := trace.Replay(world, eye, *given.glass,
			moment, given.width, given.height, given.samples, x, y)
		if err != nil {
			return frame.Pixel{}, nil, err
		}
		seen, steps = append(seen, pixel), append(steps, walked...)
	}
	pixel := given.glass.Exposing(x, y, given.width, given.height,
		given.open, carry.NinjaPixels(seen))
	return carry.SlayerPixel(pixel), steps, nil
}
