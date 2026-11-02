// Package take samples a camera path during a shutter opening. A pace and
// duration define the frames; each frame contains timed camera placements,
// called glimpses. Slayer renders those instants, and ninjafilm integrates the
// resulting light into a developed frame.
package take

import (
	"errors"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/number"
	"github.com/janearc/slayer/scene/thing"
)

// turn is one full shutter rotation in radians.
const turn = 2 * math.Pi

// midway selects the midpoint of a sampling interval.
const midway = 0.5

// fewest requires at least one glimpse per frame.
const fewest = 1

// none represents zero duration.
const none = 0

// Construction errors for invalid pace and take parameters.
var (
	errPace = errors.New("take: a camera is cranked at a rate " +
		"above nought, with a shutter open for more than none of a " +
		"turn and no more than all of it, and one glimpse or more a " +
		"frame")
	errTake = errors.New("take: a take has a path, a pace from " +
		"Cranked, and a length above nought")
)

// pace stores frame rate, shutter-open fraction, and glimpses per frame.
type pace struct {
	rate     float64
	open     float64
	glimpses int
}

// Cranked constructs a pace from frames per second, shutter angle in radians,
// and glimpses per opening. Each glimpse requires a render; one freezes motion,
// while multiple samples approximate motion blur.
func Cranked(rate, shutter float64, glimpses int) (cranked pace, err error) {
	if !number.Positive(rate) || !number.Positive(shutter) ||
		shutter > turn || glimpses < fewest {
		return cranked, errPace
	}
	return pace{rate, shutter / turn, glimpses}, nil
}

// take combines a camera path, sampling pace, and duration.
type take struct {
	path   thing.Motion
	pace   pace
	length time.Duration
}

// Take constructs a timed camera path. The path returns camera subspaces
// through thing.Motion. Time begins at zero, when the first frame opens.
func Take(path thing.Motion, cranked pace, length time.Duration) (
	made take, err error) {
	if path == nil || !number.Positive(cranked.rate) || length <= none {
		return made, errTake
	}
	return take{path, cranked, length}, nil
}

// Frames returns the number of complete frames. It multiplies duration in
// nanoseconds by rate before converting to seconds, avoiding a rounding error
// that can otherwise lose the final frame.
func (shot take) Frames() int {
	frames := float64(shot.length) * shot.pace.rate / float64(time.Second)
	return int(math.Floor(frames))
}

// Glimpses returns ordered scene times and camera placements for a zero-based
// frame number. Render each sample and pass the results to the roll’s Print
// method.
func (shot take) Glimpses(frame int) []glimpse {
	var caught []glimpse
	for index := range shot.pace.glimpses {
		at := shot.opening(frame, index)
		caught = append(caught, glimpse{at, shot.path.At(at)})
	}
	return caught
}

// opening places a glimpse at the midpoint of its share of the shutter’s open
// interval, distributing samples evenly across exposure time.
func (shot take) opening(frame, index int) time.Duration {
	step := (float64(index) + midway) / float64(shot.pace.glimpses)
	frames := float64(frame) + shot.pace.open*step
	seconds := frames / shot.pace.rate
	nanoseconds := math.Round(seconds * float64(time.Second))
	return time.Duration(nanoseconds)
}

// glimpse stores a shutter sample’s time and camera transform.
type glimpse struct {
	at  time.Duration
	eye maths.Subspace
}

// At returns elapsed time since the take began. Use the same time to evaluate
// moving scene objects.
func (caught glimpse) At() time.Duration { return caught.at }

// Eye returns the camera placement and orientation for the glimpse.
func (caught glimpse) Eye() maths.Subspace { return caught.eye }
