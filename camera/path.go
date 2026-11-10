// Package camera describes repeatable camera moves in scene space.
// Paths are sampled from elapsed time, independent of render order or speed.
package camera

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/maths"
)

// Move is checked travel between two points, or a timed hold. Its coordinates
// cannot change after construction. The path supplies the look target unless
// the move says its own.
type Move struct {
	from, to maths.Vec
	duration time.Duration
	speed    float64
	ease     string
	// looks is where the move looks at its start and at its end, when
	// it says; the target travels between them as the camera does.
	looks *[2]maths.Vec
}

// Looking is the move with its own look: a target that travels from one
// point to another over the move, eased as the move is. A hold that looks
// from one target to another is a pan.
func (move Move) Looking(from, to maths.Vec) (Move, error) {
	if !point(from) || !point(to) || !point(to.Sub(from)) {
		return move, fmt.Errorf("need finite look targets")
	}
	move.looks = &[2]maths.Vec{from, to}
	return move, nil
}

// looking is where the move looks at its start and at its end: its own
// look, or the path's when it says none.
func (move Move) looking(path maths.Vec) (start, end maths.Vec) {
	if move.looks == nil {
		return path, path
	}
	return move.looks[0], move.looks[1]
}

// Travel moves at a constant speed in metres per second.
func Travel(from, to maths.Vec, speed float64) (Move, error) {
	move := Move{from: from, to: to, speed: speed}
	_, err := move.seconds()
	return move, err
}

// Timed travels over a duration with linear or smooth easing. Equal endpoints
// make a hold. An empty ease means linear.
func Timed(from, to maths.Vec, duration time.Duration,
	ease string) (Move, error) {
	move := Move{from: from, to: to, duration: duration, ease: ease}
	_, err := move.seconds()
	return move, err
}

// Path is a checked, continuous sequence of moves with a look target, the
// path's own or each move's, continuous too. Its geometry is immutable
// after construction and safe to sample concurrently.
type Path struct {
	legs []leg
	look maths.Vec
	up   maths.Vec
}

type leg struct {
	move       Move
	start, end float64
}

// New checks every move before any frame is drawn. Points use scene axes;
// no coordinate conversion or terrain-height lookup is hidden here.
func New(moves []Move, look, up maths.Vec) (*Path, error) {
	if len(moves) == nothing || !point(look) ||
		!point(up) || up.Len() == nothing {
		return nil, fmt.Errorf("camera: need moves, " +
			"a finite target and a nonzero up")
	}
	path := &Path{look: look, up: up.Unit()}
	var elapsed float64
	for index, move := range moves {
		seconds, err := move.seconds()
		if err != nil {
			return nil, fmt.Errorf("camera move %d: %w",
				index+whole, err)
		}
		if index > nothing && moves[index-whole].to != move.from {
			return nil, fmt.Errorf("camera move %d: "+
				"does not start at the previous endpoint",
				index+whole)
		}
		lookFrom, lookTo := move.looking(look)
		if index > nothing {
			_, ended := moves[index-whole].looking(look)
			if ended != lookFrom {
				return nil, fmt.Errorf("camera move %d: "+
					"does not start looking where the "+
					"previous move ended", index+whole)
			}
		}
		// the camera and its target travel together, so the camera
		// reaches the target when their difference passes nought. The
		// difference, at both ends and in its travel, is held within
		// reach first, so the crossing test's products stay finite.
		apart := Move{from: move.from.Sub(lookFrom),
			to: move.to.Sub(lookTo)}
		if !inReach(apart.from) || !inReach(apart.to) ||
			!inReach(apart.to.Sub(apart.from)) {
			return nil, fmt.Errorf("camera move %d: "+
				"target direction exceeds the numeric range",
				index+whole)
		}
		if crossesTarget(apart, maths.Vec{}) {
			return nil, fmt.Errorf("camera move %d: "+
				"reaches its look target", index+whole)
		}
		end := elapsed + seconds
		if !number(end) || end > maxDuration.Seconds() ||
			end <= elapsed {
			return nil, fmt.Errorf("camera: path exceeds " +
				"the duration range")
		}
		path.legs = append(path.legs, leg{move, elapsed, end})
		elapsed = end
	}
	return path, nil
}

// Duration includes moves and holds, rounded up to a whole nanosecond.
func (path *Path) Duration() time.Duration {
	if path == nil || len(path.legs) == nothing {
		return nothing
	}
	end := path.legs[len(path.legs)-whole].end
	nanoseconds := end * float64(time.Second)
	return time.Duration(math.Ceil(nanoseconds))
}

// At samples a camera, holding the first or last position outside the path.
// At a join, the next move supplies its exact starting coordinates.
// A zero path is refused rather than supplying an unrelated camera.
func (path *Path) At(elapsed time.Duration) (*maths.Subspace, error) {
	if path == nil || len(path.legs) == nothing {
		return nil, fmt.Errorf("camera: no checked path")
	}
	seconds := elapsed.Seconds()
	chosen := path.legs[len(path.legs)-whole]
	for _, step := range path.legs {
		if seconds < step.end {
			chosen = step
			break
		}
	}
	portion := (seconds - chosen.start) / (chosen.end - chosen.start)
	share := math.Max(nothing, math.Min(whole, portion))
	if chosen.move.ease == "smooth" {
		share = maths.Smooth(nothing, whole, share)
	}
	at := between(chosen.move.from, chosen.move.to, share)
	lookFrom, lookTo := chosen.move.looking(path.look)
	target := between(lookFrom, lookTo, share)
	return maths.View(at, target, path.up)
}

// between is a share of the way from one point to another, keeping the
// exact endpoints at nought and one, however short the way.
func between(from, to maths.Vec, share float64) maths.Vec {
	switch share {
	case nothing:
		return from
	case whole:
		return to
	}
	return from.Add(to.Sub(from).Scale(share))
}

// maxDuration leaves a second for floating conversion and rounding.
// Its Seconds conversion is the same one used for timed moves.
const maxDuration = time.Duration(math.MaxInt64) - time.Second

// seconds checks the exclusive timing choice and converts it to seconds.
func (move Move) seconds() (float64, error) {
	if !point(move.from) || !point(move.to) ||
		!point(move.to.Sub(move.from)) || !number(move.speed) ||
		move.speed < nothing || move.duration < nothing ||
		(move.speed == nothing) == (move.duration == nothing) {
		return nothing, fmt.Errorf("need finite endpoints " +
			"and exactly one positive speed or duration")
	}
	if move.ease != "" && move.ease != "linear" && move.ease != "smooth" {
		return nothing, fmt.Errorf("ease must be linear or smooth")
	}
	if move.speed > nothing {
		seconds := move.to.Sub(move.from).Len() / move.speed
		if !number(seconds) || seconds < time.Nanosecond.Seconds() {
			return nothing, fmt.Errorf("speed move needs " +
				"measurable travel and duration")
		}
		return seconds, nil
	}
	return move.duration.Seconds(), nil
}

// crossesTarget checks the whole segment, not just its endpoint views.
func crossesTarget(move Move, target maths.Vec) bool {
	delta := move.to.Sub(move.from)
	distance := delta.Dot(delta)
	var share float64
	if distance > nothing {
		projection := target.Sub(move.from).Dot(delta) / distance
		share = math.Max(nothing, math.Min(whole, projection))
	}
	return move.from.Add(delta.Scale(share)).Sub(target).Len() == nothing
}

// number rejects non-finite scalars before they enter path arithmetic.
func number(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, nothing)
}

// reach is the longest a camera's difference from its target may be:
// half the square root of the largest float64. The dot product of any
// two such vectors, and their sum, stays finite, so crossesTarget's
// arithmetic cannot overflow. It's defined by float64.
func reach() float64 { return math.Sqrt(math.MaxFloat64) / 2 }

// inReach is whether a vector is finite and no longer than reach.
func inReach(value maths.Vec) bool {
	return point(value) && value.Len() <= reach()
}

// point rejects non-finite coordinates and lengths outside Vec.Len's range.
// Any non-finite coordinate also makes Len non-finite.
func point(value maths.Vec) bool {
	return number(value.Len())
}
