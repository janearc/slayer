// Package scrooge decides how much of a set a picture pays for: nothing
// is made until a picture asks for it, and nothing finer than the picture
// can show. Every object of a set is a node, a promise until asked, that
// answers from its own numbers how big it is, how much it blocks, how much
// light it gives, and, asked with a footprint, its next layer of detail
// and no finer. See doc/scrooge.md.
//
// Scrooge is a strategy: pure maths over boxes and numbers. It never
// imports a scene, a set, the manufactory, the tracer or the gpu; they
// implement its node and call it.
package scrooge

import (
	"iter"
	"math"

	"github.com/janearc/slayer/maths"
)

// Node is one object of a set as scrooge asks it: a city, a district, a
// tower, a window, a stand of trees or one tree. Until asked for detail it
// is a promise, a recipe and a seed and a place, and its aggregate comes
// from those numbers alone.
type Node interface {
	// Aggregate is what the node and everything under it come to,
	// worked out without making any of it.
	Aggregate() Aggregate
	// Coverage is how much of a cell the given width across the node
	// blocks, from nought to one. It may say less than the truth, never
	// more, or something that is seen is lost.
	Coverage(cell float64) float64
	// Detail is the node's next layer, as fine as the footprint asks and
	// no finer: its immediate children, each made only when taken. A node
	// whose children would all be finer than the footprint gives none,
	// and answers by its aggregate.
	Detail(at Footprint) iter.Seq[Node]
}

// Aggregate is what a node and everything under it come to.
type Aggregate struct {
	// Box holds everything the node could make. It may be too large,
	// never too small.
	Box maths.Box
	// Emission is the summed light of the lamps and glowing parts under
	// the node, in the set's light.
	Emission maths.Light
	// Albedo is the share of light its surfaces give back, as the mean
	// the eye would see over its covered part.
	Albedo maths.Light
	// Grain is the size, in metres, of the finest parts its next layer
	// would give: a city's districts, a tree's whorls. Nought for a node
	// with no layer under it.
	Grain float64
}

// Size is how big the node is across, corner to corner of its box.
func (made Aggregate) Size() float64 {
	return made.Box.Max.Sub(made.Box.Min).Len()
}

// Footprint is how wide a ray, or a cell of the picture, is at each
// distance from where it starts: a cone. A camera ray's is a pixel's
// angle; a shadow ray's widens toward its light; a cascade's is its
// cell.
type Footprint struct {
	// From is where the cone starts.
	From maths.Vec
	// Width is how wide it is where it starts, in metres.
	Width float64
	// Spread is how much wider it grows with each metre away, a pixel's
	// angle in radians for a camera's ray.
	Spread float64
}

// WidthAt is the footprint's width at a distance from where it starts.
func (cone Footprint) WidthAt(distance float64) float64 {
	return cone.Width + cone.Spread*math.Max(distance, 0)
}

// Nearest is the distance from where the footprint starts to the nearest
// point of a box, nought when it starts inside.
func (cone Footprint) Nearest(box maths.Box) float64 {
	gap := func(at, low, high float64) float64 {
		return math.Max(math.Max(low-at, 0), at-high)
	}
	return maths.V(gap(cone.From.X, box.Min.X, box.Max.X),
		gap(cone.From.Y, box.Min.Y, box.Max.Y),
		gap(cone.From.Z, box.Min.Z, box.Max.Z)).Len()
}
