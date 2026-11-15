package hillside

// A film through a set's lens rig, as a session's source.
//
// Design. A rig is a frame, a thin lens focused on a thing the set names,
// and a shutter open for so many seconds in which so many glimpses are
// drawn. A still through a rig is drawMeasured's exposed. A film through
// one is that still, frame after frame, with three things a still has
// not.
//
// Each frame is the retained scene at its own time, as a pinhole film's
// is. Each glimpse is the scene and the camera at the glimpse's time: the
// retained scene advanced to it, the job's camera path sampled there, and
// the lens focused again from that eye on the rig's point, since the
// distance to it changes as the camera moves.
//
// The glimpses are at take's times across the rig's shutter, from the
// frame's time, as a still's are (instants). The rig's shutter is the
// film's: a job that says a shutter beside a rig is refused, because the
// rig's open time sets the exposure in joules, and another would change
// how bright the frame is, not only how it blurs.
//
// The frame's light is the lens's exposure of its glimpses (Thin.Expose),
// by the lens focused from the eye at the frame's own time: a raw in
// joules a square metre of film, as the still's is, and developed by the
// same tone. Drawn's Light is that raw and its Glimpses the glimpses.
//
// A rig's frames are drawn whole. The film keeper compares one eye and
// one lens between frames, and a rig's lens changes with each, so a rig
// film does not keep sections between frames, whatever its film says.
// The gpu draws a rig's lens per call, as a rigged still does; the
// kernel makes its rays (gpu/exposure.go), and nothing is kept on the
// device between a rig's draws.

import (
	"errors"
	"fmt"
	"time"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/carry"
	"github.com/janearc/slayer/render/frame"
)

// rigFramer draws a set's retained scene through its rig, a glimpse at a
// time, and exposes a frame's glimpses through its lens.
type rigFramer struct {
	plan sequencePlan
}

// errTwoShutters is the refusal of a job's shutter beside a set's rig.
var errTwoShutters = errors.New("the set's rig says the shutter, open so " +
	"many seconds for so many glimpses, and the job says one too; a " +
	"rig's open time sets the exposure, so leave the job's shutter out")

// openRigFilm is a film of a staged set through its rig: its lighting
// retained, the job saying no shutter of its own, and the glimpses at the
// rig's shutter's times.
func openRigFilm(wanted job.Job, given asked) (*Session, error) {
	if _, said := wanted.Shutter(); said {
		return nil, fmt.Errorf("%s: %w", given.set, errTwoShutters)
	}
	if err := rigZoom(wanted, given); err != nil {
		return nil, err
	}
	// a rig's film is drawn glimpse by glimpse through the rig's glass,
	// which fast light cannot see through: refused here, as a still is
	if err := fastRefusal(given); err != nil {
		return nil, fmt.Errorf("%s: %w", given.set, err)
	}
	if given.optics.aperture != nothing {
		return nil, fmt.Errorf("%s: a film with a rig takes its lens "+
			"from the rig; the job's aperture and focus cannot stand "+
			"over it", given.set)
	}
	if given.live == nil || given.live.unsupported != "" {
		return nil, fmt.Errorf("%s: sequence cannot retain this "+
			"scene's lighting", given.set)
	}
	offsets, err := rigOffsets(given)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", given.set, err)
	}
	session := newSession(rigFramer{plan: cameraPlan(wanted)}, given,
		wanted, given.tone())
	session.offsets = offsets
	return session, nil
}

// errRigZoom is the refusal of a job's zoom on a film through a rig,
// whose lens says its own angle.
var errRigZoom = errors.New("a film with a rig sees through the rig's " +
	"lens, whose focal length says its angle; a job's zoom cannot change " +
	"it, so leave the zoom out")

// rigZoom refuses a job's zoom on a set staged through a rig.
func rigZoom(wanted job.Job, given asked) error {
	lens, _ := wanted.Camera()
	if _, zoomed := lens.Zoom(); zoomed && given.glass != nil {
		return fmt.Errorf("%s: %w", given.set, errRigZoom)
	}
	return nil
}

// rigOffsets is how far after a frame's time each glimpse of the rig's
// shutter is: take's times, as a rigged still's instants are.
func rigOffsets(given asked) ([]time.Duration, error) {
	opening := asked{open: given.open, glimpses: given.glimpses}
	return instants(opening, maths.Subspace{})
}

// light is one glimpse: the retained scene at its time, seen from the
// job's camera at that time through the rig's lens focused from there,
// drawn by the renderer asked.
func (rig rigFramer) light(at time.Duration, given asked) (drawnLight,
	error) {
	world, made, err := given.live.frameAt(at)
	if err != nil {
		return drawnLight{}, err
	}
	given.at = at
	if err = rig.plan.withCamera(&given, at); err != nil {
		return drawnLight{made: made}, err
	}
	glass, err := given.refocused(*given.eye)
	if err != nil {
		return drawnLight{made: made}, err
	}
	given.glass = &glass
	world.Pixel = glass.Pixel(given.width * given.samples)
	drawn, profile, err := glimpsed(world, given, *given.eye, at)
	return drawnLight{drawn: drawn, profile: profile, made: made,
		world: world, given: given}, err
}

// exposed is a frame's raw: its glimpses gathered through the rig's lens
// focused from the eye at the frame's own time, for the rig's open time.
func (rig rigFramer) exposed(at time.Duration, given asked,
	glimpses []*frame.Frame) (*frame.Frame, error) {
	if err := rig.plan.withCamera(&given, at); err != nil {
		return nil, err
	}
	glass, err := given.refocused(*given.eye)
	if err != nil {
		return nil, err
	}
	raw, err := glass.Expose(given.open, carry.NinjaFrames(glimpses))
	if err != nil {
		return nil, err
	}
	return carry.Slayer(raw), nil
}
