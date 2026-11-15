package hillside

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/janearc/slayer/render/finish"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/progress"
	"github.com/janearc/slayer/scene/thing"
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
	drawn, err := session.glimpsed(ctx, session.times[index], given)
	if err != nil {
		return Drawn{}, err
	}
	told(ask.sink, drawn.lit.drawn)
	return drawn, nil
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
			return Drawn{}, err
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
	return given.rigSized(ask.height)
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

// stillFramer draws a set as the still path does: its staged world, as
// drawMeasured draws it, through a rig's lens where it has one.
type stillFramer struct {
	world *thing.World
	made  tally
	// plan is the job's camera path and zoom, where it says them
	plan sequencePlan
}

// light is the still's world drawn at its time as asked, from the job's
// camera path where it says one. A rig's lens is focused again from that
// eye, and told the angle one ray sees at the width and rays asked.
func (still stillFramer) light(at time.Duration,
	given asked) (drawnLight, error) {
	given.at = at
	if err := still.plan.exposing(&given, at); err != nil {
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
		still.world.Pixel = given.glass.Pixel(given.width * given.samples)
	}
	given.footprint(still.world)
	drawn, profile, err := drawMeasured(still.world, given,
		still.made.across)
	return drawnLight{drawn: drawn, profile: profile, made: still.made,
		world: still.world, given: given}, err
}

// timedFramer draws a retained scene as a film does: the scene at each
// frame's time, through the job's camera path and zoom, kept from the
// last frame where its film keeps sections.
type timedFramer struct {
	plan sequencePlan
}

// light is the retained scene at a time, drawn as asked.
func (timed timedFramer) light(at time.Duration,
	given asked) (drawnLight, error) {
	return sequenceLight(timed.plan, given, at)
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
