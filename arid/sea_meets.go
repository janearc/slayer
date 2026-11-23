// This file is whether a ray meets the sea, which a ray through the whole
// slab settles without a search over open sea.

package arid

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Meets is whether the ray meets the surface between near and far: Hit's
// answer without its distance. Every ray it answers outright, Hit would
// have found the same.
func (sea *Sea) Meets(ray maths.Ray, near, far float64) bool {
	if !sea.usable(ray, near, far) {
		return false
	}
	enter, leave, inside := sea.within(ray, near, far)
	if !inside {
		return false
	}
	if sea.crossed(ray, near, far) {
		return true
	}
	_, met := sea.search(ray, near, enter, leave)
	return met
}

// crossed is whether the ray certainly meets water, with no search. A ray
// that meets the slab's first face after near and its second by far is at
// one face over the surface and at the other under it, and the surface is
// continuous and stays in the slab, so the ray crosses it between.
//
// The search steps to the first crossing and cannot pass it, and a search
// that used up its steps ends on the mean plane, which lies between the
// faces; so it finds water wherever water may stand, and a ray whose way
// is over no dry place (wetAlong) finds it. Whether it does is all this
// says; where is Hit's.
func (sea *Sea) crossed(ray maths.Ray, near, far float64) bool {
	if ray.Dir.Y == 0 {
		return false
	}
	first, second := sea.faces(ray)
	return first > near && second <= far &&
		sea.wetAlong(ray.Point(first), ray.Point(second))
}

// wetAlong is whether water may stand under every point of the line
// between two points of the ray: a sea with no coast is wet everywhere,
// and a line wholly off the coast's grid is, wet() being true there.
func (sea *Sea) wetAlong(one, other maths.Vec) bool {
	return sea.coast == nil || !sea.coast.mayBeDry(one, other)
}

// mayBeDry is whether the line between two points, east and south in the
// sea's frame, may pass over a place that is dry. It does not if it is
// wholly off the grid, by more than a step to leave the grid's edge and
// rounding clear of it. A point that is not a number may be dry.
func (shore *coast) mayBeDry(one, other maths.Vec) bool {
	lastEast := shore.east + float64(shore.across-1)*shore.step
	lastSouth := shore.south + float64(shore.down-1)*shore.step
	off := func(first, second, low, high float64) bool {
		return math.Max(first, second) < low-shore.step ||
			math.Min(first, second) > high+shore.step
	}
	return !(off(one.X, other.X, shore.east, lastEast) ||
		off(one.Z, other.Z, shore.south, lastSouth))
}
