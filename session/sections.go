package session

import (
	"image"
	"reflect"
	"time"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/section"
	"github.com/janearc/slayer/trace"
)

// sectionsAsked is how a film draws in sections: a frame wider than over
// pixels, in sections size pixels square, kept where keep holds, with so
// much drift allowed in the light of a kept section's volumes.
type sectionsAsked struct {
	over, size int
	keep       bool
	drift      float64
}

// a keeper holds a sequence's last frame as it was drawn, and all it was
// drawn from, so the next frame can keep the sections of it that cannot
// have changed. kept and drawn count the sections of the frames since the
// first that were kept and that were drawn again.
//
// frames, where the film draws on the gpu, is the film's retained drawer,
// which keeps the scene on the device between frames; the film closes it
// at its end. generation is the last generation it was given: every frame
// of a film is a new world, so every frame is a new generation.
type keeper struct {
	world                  *thing.World
	at                     time.Duration
	eye                    maths.Subspace
	lens                   film.Sensor
	width, height, samples int
	raw                    *frame.Frame
	kept, drawn            int
	frames                 *gpu.Frames
	restir                 *gpu.Restir
	generation             uint64
}

// measured is the frame as drawMeasured draws it: by the film's retained
// drawer where it has one and the frame is a pinhole's on the gpu, or by
// drawMeasured itself.
func (keep *keeper) measured(world *thing.World, given asked,
	across float64) (*frame.Frame, gpu.Profile, error) {
	if keep != nil && keep.restir != nil && given.onGPU && given.glass == nil {
		return keep.restir.Draw(world, eyeFor(given, across), lensFor(given),
			given.at, given.width, given.height, given.samples)
	}
	if keep == nil || keep.frames == nil || !given.onGPU ||
		given.glass != nil {
		return drawMeasured(world, given, across)
	}
	keep.generation++
	return given.drawnBy(keep.frames, world, keep.generation,
		eyeFor(given, across))
}

// framesMode is the retained drawer's mode for what was asked, chosen
// in the order drawMeasured chooses the per-call function.
func framesMode(given asked) gpu.Mode {
	for _, choice := range []struct {
		asked bool
		mode  gpu.Mode
	}{{given.skyLooks, gpu.FasterLighting},
		{given.fastLighting, gpu.FastLighting},
		{given.cachedLighting, gpu.CachedLighting},
		{given.nativeMeshes, gpu.NativeMeshes},
		{given.nativeShadows, gpu.NativeShadows}} {
		if choice.asked {
			return choice.mode
		}
	}
	return gpu.Software
}

// filmKeeper is a film's keeper, with a retained drawer where the film
// draws on the gpu. The caller closes the drawer when the film ends.
func filmKeeper(given asked) (*keeper, error) {
	if err := fastRefusal(given); err != nil {
		return nil, err
	}
	if !given.onGPU {
		return &keeper{}, nil
	}
	if given.restir {
		return &keeper{restir: gpu.NewRestir()}, nil
	}
	frames, err := gpu.NewFrames(framesMode(given))
	if err != nil {
		return nil, err
	}
	return &keeper{frames: frames}, nil
}

// close lets the film's retained drawer go, where it has one.
func (keep *keeper) close() {
	if keep != nil && keep.frames != nil {
		keep.frames.Close()
	}
}

// picture is the frame of a world as drawMeasured draws it, made from the
// keeper's last frame where the film keeps sections and the camera has
// not moved. On the cpu the kept sections are copied and the rest drawn.
// The gpu draws no part of a frame yet: there the last frame is kept
// whole when every section is kept, and otherwise drawn whole.
//
// A film with a rig or a job's opened lens, a frame no wider than the
// film's sections start at, a wire picture, or no keeper, is drawn whole.
func (keep *keeper) picture(world *thing.World, given asked,
	across float64) (*frame.Frame, gpu.Profile, error) {
	if given.exactSamples > 0 {
		return drawMeasured(world, given, across)
	}
	if given.composite != nil {
		return drawMeasured(world, given, across)
	}
	eye, lens := eyeFor(given, across), lensFor(given)
	if keep == nil || len(given.initialMedia) > 0 || given.glass != nil || given.optics.opened != nil ||
		given.cachedLighting || given.fastLighting || !given.sections.keep ||
		given.width <= given.sections.over || given.wire {
		return keep.measured(world, given, across)
	}
	drawn, err := keep.next(world, given, eye, lens)
	var profile gpu.Profile
	if err == errDrawWhole {
		drawn, profile, err = keep.measured(world, given, across)
	}
	if err != nil {
		return nil, profile, err
	}
	*keep = keeper{world: world, at: given.at, eye: eye, lens: lens,
		width: given.width, height: given.height, samples: given.samples,
		raw: drawn, kept: keep.kept, drawn: keep.drawn,
		frames: keep.frames, generation: keep.generation}
	return drawn, profile, nil
}

// sightLine is how a frame's rays are looked along: from an eye through a
// lens, in a frame of width by height pixels.
type sightLine struct {
	eye           maths.Subspace
	lens          film.Sensor
	width, height int
}

// ray is the ray through the pixel at x, y.
func (from sightLine) ray(x, y int) maths.Ray {
	var visible maths.Band
	return film.Look(from.eye, from.lens, x, y, from.width, from.height,
		visible)
}

// redrawing draws again the sections a change leaves, and counts them.
type redrawing struct {
	world *thing.World
	given asked
	eye   maths.Subspace
	lens  film.Sensor
	again int
}

// draw is the sections left, drawn on the cpu. On the gpu it draws none
// of them, and says the frame is to be drawn whole when there are any.
func (redo *redrawing) draw(left []image.Rectangle) ([]*frame.Frame, error) {
	redo.again = len(left)
	if redo.given.onGPU && redo.again > nothing {
		return nil, errDrawWhole
	}
	redo.given.placing(redo.world)
	return trace.SampledEach(redo.world, redo.eye, redo.lens, redo.given.at,
		redo.given.width, redo.given.height, redo.given.samples, left)
}

// next is the frame after the keeper's, from it where it can be, or
// errDrawWhole where the frame is to be drawn whole instead.
func (keep *keeper) next(world *thing.World, given asked,
	eye maths.Subspace, lens film.Sensor) (*frame.Frame, error) {
	if keep.raw == nil || eye != keep.eye ||
		!reflect.DeepEqual(lens, keep.lens) || given.width != keep.width ||
		given.height != keep.height || given.samples != keep.samples {
		return nil, errDrawWhole
	}
	boxes := section.Split(given.width, given.height, given.sections.size)
	sights := make([]section.Sight, len(boxes))
	looking := sightLine{eye: eye, lens: lens, width: given.width,
		height: given.height}
	for at, box := range boxes {
		sights[at] = section.SightOf(looking.ray, box)
	}
	change := section.Between(keep.world, world, keep.at,
		given.at).Allowing(given.sections.drift)
	redo := &redrawing{world: world, given: given, eye: eye, lens: lens}
	later, kept, err := change.Redrawn(keep.raw, boxes, sights, redo.draw)
	if err == errDrawWhole {
		keep.drawn += len(boxes)
		return nil, errDrawWhole
	}
	keep.kept, keep.drawn = keep.kept+kept, keep.drawn+redo.again
	return later, err
}

// errDrawWhole says the frame is to be drawn whole: the gpu draws no
// part of a frame.
var errDrawWhole = errorString("draw the frame whole")

// errorString is an error that is its own text.
type errorString string

// Error is the text.
func (text errorString) Error() string { return string(text) }
