package shape

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Tube is a hollow cylinder about its own y axis: a ring stood on edge,
// or the shell of a ring world. Radius is to the middle of the wall, Thick
// the wall's thickness, Height its length along y.
//
// It has no exact hit, so it is marched. Its surface faces outward on the
// outside and inward on the inside, which is where a ring world's ground
// is.
//
// A tube of no length, or of a wall as thin as nothing, is still a tube: a
// whisker or a strand is built so, and it is measured as any other. Only a
// radius or a height below zero, or a measure that is not a finite number,
// is no tube: its Bounds is the one point at the origin and its Distance is
// infinite. Mesh needs more, and refuses a tube without a wall and a length.
type Tube struct {
	Radius, Thick, Height float64
}

// valid is whether the tube's measures are numbers a tube can have: finite,
// and a radius and a height of nought or more.
func (tube Tube) valid() bool {
	return tube.Radius >= nought && tube.Height >= nought &&
		finiteNumber(tube.Radius) && finiteNumber(tube.Thick) &&
		finiteNumber(tube.Height)
}

// meshable is whether the tube has a wall and a length, and an outer radius
// above zero: what a closed mesh of it needs.
func (tube Tube) meshable() bool {
	return tube.valid() && tube.Thick > nought && tube.Height > nought &&
		tube.Radius+tube.Thick*half > nought
}

// Bounds is the box the tube just fits in.
func (tube Tube) Bounds() maths.Box {
	if !tube.valid() {
		return nowhere()
	}
	reach, length := tube.Radius+tube.Thick*half, tube.Height*half
	return maths.Box{Min: maths.V(-reach, -length, -reach),
		Max: maths.V(reach, length, reach)}
}

// Distance is how far point is from the wall: the distance to a ring in the
// radial and axial plane, which is a box of half the thickness by half
// the height, centred on the radius.
func (tube Tube) Distance(point maths.Vec) float64 {
	if !tube.valid() {
		return math.Inf(unity)
	}
	radial := math.Hypot(point.X, point.Z) - tube.Radius
	acrossWall := math.Abs(radial) - tube.Thick*half
	alongAxis := math.Abs(point.Y) - tube.Height*half
	out := math.Hypot(math.Max(acrossWall, nought), math.Max(alongAxis, nought))
	return out + math.Min(math.Max(acrossWall, alongAxis), nought)
}
