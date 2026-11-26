package session

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/internal/observability"
	"github.com/janearc/slayer/render/finish"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/progress"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/trace"
)

// errNotDrawn is the refusal of a hud for a Drawn no session drew.
var errNotDrawn = errors.New("hillside: this Drawn holds no frame; a " +
	"session's Frame makes one")

// frame is Frame: refused when the session is closed, the context is
// done or there is no such frame; and otherwise drawn as asked, and the
// ask's sink told of the whole frame once it is done.
func (session *Session) frame(ctx context.Context, index int,
	ask Ask) (Drawn, error) {
	if session.closed {
		return Drawn{}, errClosed
	}
	if err := ctx.Err(); err != nil {
		return Drawn{}, err
	}
	if index < nothing || index >= len(session.times) {
		return Drawn{}, fmt.Errorf("hillside: no frame %d; the job has "+
			"%d, counted from zero", index, len(session.times))
	}
	given, err := session.asking(ask)
	if err != nil {
		return Drawn{}, err
	}
	keeping := session.keeping(ctx, ask)
	given.keeping = &keeping
	if given.onGPU {
		// the gpu stops on the context between bands, and says its rows
		// where the ask says, as the processor's still says its own
		gpu.Watched(gpu.NewWatch(ctx, ask.said))
		defer gpu.Unwatched()
	}
	drawn, err := session.glimpsed(ctx, session.times[index], given)
	if err != nil {
		return Drawn{}, err
	}
	told(ask.sink, drawn.lit.drawn)
	return drawn, nil
}

// keeping is what a frame's draw on the processor stops on and keeps its
// rows in: the context always, and the ask's rows where the frame has one
// glimpse, since rows are kept for a frame and not for each glimpse.
func (session *Session) keeping(ctx context.Context,
	ask Ask) trace.Keeping {
	if len(session.offsets) > singleRay {
		return trace.NewKeeping(ctx, nil)
	}
	return trace.NewKeeping(ctx, ask.rows)
}

// glimpsed is a frame at a time drawn once for each glimpse of the
// shutter, each at its own time, and their mean where there are more
// than one.
func (session *Session) glimpsed(ctx context.Context, at time.Duration,
	given asked) (Drawn, error) {
	offsets := session.offsets
	if len(offsets) == nothing {
		offsets = unshuttered
	}
	drawn := Drawn{at: at}
	given.optics.share = session.share
	exposing, rigged := session.source.(exposer)
	for place, offset := range offsets {
		if err := ctx.Err(); err != nil {
			return Drawn{}, err
		}
		made, err := session.source.light(at+offset, given)
		if err != nil {
			if observability.Enabled() {
				observability.Add("slayer_glimpse_failures_total", 1)
			}
			return Drawn{}, err
		}
		if observability.Enabled() {
			observability.Add("slayer_glimpses_total", 1)
			observeGPUProfile(made.profile)
		}
		if place == nothing {
			drawn.lit = made
		}
		drawn.glimpses = append(drawn.glimpses, made.drawn)
	}
	var err error
	switch {
	case rigged:
		drawn.mean, err = exposing.exposed(at, given, drawn.glimpses)
	case len(drawn.glimpses) > singleRay:
		drawn.mean, err = finish.Mean(drawn.glimpses)
	}
	return drawn, err
}

// exposer is a source whose frame's light is not the mean of its
// glimpses but their exposure through a lens: a rig's.
type exposer interface {
	exposed(at time.Duration, given asked,
		glimpses []*frame.Frame) (*frame.Frame, error)
}

// asking is what was asked as staged, with the ask's size and rays
// standing over it, the angle across worked out for the frame's shape
// where the job says tall, and a rig's rows for the width.
func (session *Session) asking(ask Ask) (asked, error) {
	given := session.given
	if given.exactSamples > 0 && ask.rays > 0 {
		return asked{}, errors.New("exact samples: a rays-per-side frame override is not supported")
	}
	for _, each := range []struct {
		value int
		into  *int
	}{
		{ask.width, &given.width},
		{ask.height, &given.height},
		{ask.rays, &given.samples},
	} {
		if each.value > nothing {
			*each.into = each.value
		}
	}
	if wide, said := session.lens.WideFor(given.width,
		given.height); said {
		given.wide = wide
	}
	resized, err := given.rigSized(ask.height)
	if err != nil {
		return asked{}, err
	}
	if err := regionRefusal(resized); err != nil {
		return asked{}, err
	}
	return resized, nil
}

// rigSized is what was asked with a rig's rows for its width: a rig's
// frame says the height from the width, so its pixels are square. A
// height asked of a rig that is not those rows is refused, by name.
func (given asked) rigSized(height int) (asked, error) {
	if given.frame == nil {
		return given, nil
	}
	rows := given.frame.Rows(given.width)
	if height != nothing && height != rows {
		return given, fmt.Errorf("%s: a film with a rig is %d rows at "+
			"%d across, not %d; ask its width alone", given.set, rows,
			given.width, height)
	}
	given.height = rows
	return given, nil
}

// told tells a sink, where there is one, of a frame done: every row of
// it, as one band.
func told(sink progress.Sink, drawn *frame.Frame) {
	if sink == nil {
		return
	}
	// a frame drawn is at least a pixel, so its band is never refused
	band, err := progress.NewBand(drawn.W, drawn.H, nothing, drawn.H)
	if err == nil {
		sink(band)
	}
}

// stillFramer draws a staged world at a scene time, as drawMeasured
// draws it, through a rig's lens where a still has one.
type stillFramer struct {
	world *thing.World
	made  tally
	given asked
	// plan is the job's camera path and zoom, where it says them
	plan sequencePlan
}

// inspect exposes the still's world at a time, along its camera path.
func (still stillFramer) inspect(at time.Duration) (*thing.World, error) {
	if err := still.plan.exposing(&still.given, at); err != nil {
		return nil, err
	}
	return still.worldAt(at)
}

// eyed is what the still is asked at its time, seen from the job's
// camera path where it says one: the steps light takes before drawing.
func (still stillFramer) eyed(at time.Duration, given asked) (asked,
	error) {
	given.at = at
	err := still.plan.exposing(&given, at)
	return given, err
}

// light is the still's world drawn at its time as asked, from the job's
// camera path where it says one.
func (still stillFramer) light(at time.Duration,
	given asked) (drawnLight, error) {
	made, err := still.staged(at, given)
	if err != nil {
		return drawnLight{}, err
	}
	made.drawn, made.profile, err = drawMeasured(made.world, made.given,
		made.made.across)
	return made, err
}

// staged is the still's world at its time and what is asked of it
// there, before it is drawn. A rig's lens is focused again from the
// eye, and told the angle one ray sees at the width and rays asked.
func (still stillFramer) staged(at time.Duration,
	given asked) (drawnLight, error) {
	given, err := still.eyed(at, given)
	if err != nil {
		return drawnLight{}, err
	}
	world, err := still.worldAt(at)
	if err != nil {
		return drawnLight{}, err
	}
	if given.lensed != nil {
		glass, err := given.refocused(*given.eye)
		if err != nil {
			return drawnLight{}, err
		}
		given.glass = &glass
	}
	if given.glass != nil {
		world.Pixel = given.glass.Pixel(given.width * given.samples)
	}
	given.footprint(world)
	return drawnLight{made: still.made, world: world, given: given}, nil
}

// timedFramer draws a retained scene as a film does: the scene at each
// frame's time, through the job's camera path and zoom, kept from the
// last frame where its film keeps sections.
type timedFramer struct {
	plan  sequencePlan
	given asked
}

// inspect returns the timed scene's world at a time, from its retained
// frames.
func (timed timedFramer) inspect(at time.Duration) (*thing.World, error) {
	if timed.given.live == nil {
		return nil, fmt.Errorf("hillside: this timed scene has no retained world to inspect")
	}
	world, _, err := timed.given.live.frameAt(at)
	if err != nil {
		return nil, err
	}
	return sceneClockOceanWorldAt(world, at)
}

// light is the retained scene at a time, drawn as asked.
func (timed timedFramer) light(at time.Duration,
	given asked) (drawnLight, error) {
	return sequenceLight(timed.plan, given, at)
}

// staged is the retained scene at a time and what is asked of it there,
// before it is drawn.
func (timed timedFramer) staged(at time.Duration,
	given asked) (drawnLight, error) {
	return sequenceStaged(timed.plan, given, at)
}

// overlay draws the set's hud, where it has one, on a developed picture
// of the frame. It refuses a picture of another size than the frame.
func (made drawnLight) overlay(picture *image.RGBA) error {
	if made.drawn == nil {
		return errNotDrawn
	}
	want := image.Pt(made.drawn.W, made.drawn.H)
	if picture == nil || picture.Bounds().Size() != want {
		return fmt.Errorf("hillside: the hud is drawn on a picture of "+
			"the frame, %d by %d, and this one is not", want.X, want.Y)
	}
	return overdraw(picture, made.given, made.world, made.made.across)
}
