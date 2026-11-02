package main

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
	width, height := rig.Frame()
	made, err := lens.FrameOf(width, height)
	if err != nil {
		return given, fmt.Errorf("%s: %w", given.set, err)
	}
	given.frame = &made
	given.height = made.Rows(given.width)
	given.exposure = 1
	seconds, glimpses := rig.Shutter()
	given.open = time.Duration(math.Round(seconds * float64(time.Second)))
	given.glimpses = glimpses
	return given, nil
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
	glass, err := lens.NewThin(*stage.given.frame, focal, stop,
		at.Sub(eye.Origin).Len())
	if err != nil {
		return fmt.Errorf("%s: %w", stage.given.set, err)
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
// exposure of them.
func exposed(world *thing.World, given asked, eye maths.Subspace) (
	*frame.Frame, error) {
	moments, err := instants(given, eye)
	if err != nil {
		return nil, err
	}
	sampled := trace.Sampled
	if given.onGPU {
		sampled = gpu.Sampled
	}
	glimpses := make([]*frame.Frame, 0, len(moments))
	for _, moment := range moments {
		drawn, err := sampled(world, eye, *given.glass, moment,
			given.width, given.height, given.samples)
		if err != nil {
			return nil, err
		}
		glimpses = append(glimpses, drawn)
	}
	raw, err := given.glass.Expose(given.open, carry.NinjaFrames(glimpses))
	return carry.Slayer(raw), err
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
