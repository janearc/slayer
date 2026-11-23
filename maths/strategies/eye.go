// Package strategies is the one place for the maths that rendering
// strategies share, worked out once: what a camera sees of a box, and how a
// set of lights is sorted into a tree.
//
// A strategy is pure maths over boxes, rays and numbers. It never imports a
// scene, a set, the session, the tracer or the gpu.
package strategies

import (
	"errors"
	"math"

	"github.com/janearc/slayer/maths"
)

// Eye is the camera a thing is judged by: where it stands and which way it
// looks, how wide its lens is, in radians across, and how many pixels the
// film is across and down.
type Eye struct {
	Place        maths.Subspace
	Wide         float64
	Across, Down int
}

// Seen is how an eye sees a thing: whether any pixel's ray can meet it,
// and how many pixels across the film it spans, corner to corner.
type Seen struct {
	Visible bool
	Pixels  float64
}

// nearest is the least distance, in metres, a thing is taken to be from
// the eye, so a thing the eye stands in spans the film and no more.
const nearest = 0.01

// widestLens is the widest a lens may be, in radians: half a turn, twice
// the most a side of the view may lean from straight ahead.
const widestLens = math.Pi

// ErrEye is the refusal of a camera no picture is taken through.
var ErrEye = errors.New("strategies: an eye needs a lens wider than " +
	"nothing and narrower than a half turn, and a film of some pixels")

// Check refuses an eye that sees nothing.
func (eye Eye) Check() error {
	if !(eye.Wide > 0 && eye.Wide < widestLens) || eye.Across < 1 ||
		eye.Down < 1 {
		return ErrEye
	}
	return nil
}

// Focal is how many pixels across a metre spans at a metre from the eye.
func (eye Eye) Focal() float64 {
	return float64(eye.Across) / (2 * math.Tan(eye.Wide/2))
}

// Span is how many pixels across the film a thing spans: its extent seen
// from the eye at the near side of the ball that holds it.
func (eye Eye) Span(at maths.Vec, extent float64) float64 {
	away := at.Sub(eye.Place.Origin).Len() - extent/2
	return extent * eye.Focal() / math.Max(away, nearest)
}

// Sees is whether any of a box is in front of the eye and inside its
// view, taking the box as the ball that holds it.
func (eye Eye) Sees(box maths.Box) bool {
	middle := box.Min.Add(box.Max).Scale(0.5)
	return eye.SeesBall(middle, box.Max.Sub(box.Min).Len()/2)
}

// SeesBall is whether any of a ball is inside the view.
func (eye Eye) SeesBall(middle maths.Vec, radius float64) bool {
	way := middle.Sub(eye.Place.Origin)
	axes := eye.Place.Orientation
	ahead := -way.Dot(axes.Z)
	if ahead < -radius {
		return false
	}
	across := math.Tan(eye.Wide / 2)
	down := across * float64(eye.Down) / float64(eye.Across)
	return inView(way.Dot(axes.X), ahead, across, radius) &&
		inView(way.Dot(axes.Y), ahead, down, radius)
}

// inView is whether a ball so far to one side, so far ahead, is inside
// a side of the view that leans out by the tangent given.
func inView(side, ahead, tangent, radius float64) bool {
	slack := radius * math.Sqrt(1+tangent*tangent)
	return math.Abs(side) <= ahead*tangent+slack
}

// Cover is how the eye sees a box that is already in the eye's own world:
// whether it is in view, and its corner-to-corner length in pixels. An
// infinite or unordered box is taken as seen, and as filling the film, since
// nothing can be said to be out of its reach.
func (eye Eye) Cover(box maths.Box) Seen {
	reach := box.Max.Sub(box.Min).Len()
	if math.IsNaN(reach) || math.IsInf(reach, 0) {
		return Seen{Visible: true, Pixels: float64(eye.Across)}
	}
	middle := box.Min.Add(box.Max).Scale(0.5)
	return Seen{Visible: eye.Sees(box), Pixels: eye.Span(middle, reach)}
}
