package session

// A shutter, as a session draws one.
//
// Design. A job's shutter opens at each frame's time and stays open for
// its angle's share of the frame's interval: 180 degrees is half of it.
// The open interval runs from the frame's time toward the next frame's;
// it never begins before the frame's time.
//
// The frame is drawn once for each glimpse, at the midpoints of equal
// shares of the open interval, as package take places them for a rig's
// shutter. Each glimpse is the scene at its own time: the retained scene
// advanced to it, a named scene told it, and the camera path sampled at
// it, so moving things and a moving camera both blur.
//
// The frame's interval is in scene time: one over the rate, or, where the
// job stretches scene time with a scene duration, that span over the
// frames. A frame of G glimpses costs G draws.
//
// On the gpu the eye also moves inside each glimpse, across the glimpse's
// own share of the open interval (session_exposure.go), so a camera's
// move is blurred evenly and one glimpse is enough for it.

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/take"
)

// unshuttered is a frame with no shutter: one glimpse, at the frame's own
// time.
var unshuttered = []time.Duration{nothing}

// shutterOffsets is how far after each frame's time its glimpses are, as
// take places them: one, at none, where the job says no shutter. With
// them is how long each glimpse's share of the open interval is, which
// its offset is the middle of; none where the job says no shutter.
func shutterOffsets(wanted job.Job) ([]time.Duration, time.Duration,
	error) {
	open, said := wanted.Shutter()
	if !said {
		return unshuttered, nothing, nil
	}
	clock := wanted.Clock()
	rate := framesASecond(clock)
	pace, err := take.Cranked(rate, open.Angle()*maths.Degree,
		open.Glimpses())
	if err != nil {
		return nil, nothing, fmt.Errorf("hillside: the job's shutter: %w",
			err)
	}
	interval := time.Duration(float64(time.Second) / rate)
	// the camera of the take is held still: the session samples the
	// job's own camera path at each glimpse's time
	shot, err := take.Take(thing.Still(maths.Subspace{}), pace, interval)
	if err != nil {
		return nil, nothing, fmt.Errorf("hillside: the job's shutter: %w",
			err)
	}
	var offsets []time.Duration
	for _, caught := range shot.Glimpses(nothing) {
		offsets = append(offsets, caught.At())
	}
	// a glimpse is the middle of its share, so the first is half a share in
	return offsets, offsets[nothing] * shareHalves, nil
}

// shareHalves is defined as the two halves of a glimpse's share of the
// shutter that lie either side of the glimpse.
const shareHalves = 2

// framesASecond is how many frames a second of scene time holds: the
// rate, or the frames over the scene duration where the job stretches it.
func framesASecond(clock job.Clock) float64 {
	span, stretched := clock.SceneDuration()
	if stretched {
		return float64(clock.Frames()) / span.Seconds()
	}
	rate, _ := clock.Rate()
	return float64(rate)
}
