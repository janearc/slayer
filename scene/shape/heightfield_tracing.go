package shape

import (
	"github.com/janearc/slayer/maths"
)

// The numbers a tracer's hit on a height field is held to. Each is a share
// of a length, so a field a thousand times as big is held to the same.
const (
	// departure is how far past the start of a search a hit must lie to
	// count, as a share of the field's reach, which is the larger of its
	// extent, its height range and how far the ray starts from the field's
	// zero. A ray that leaves the surface it stands on has that surface as
	// its first root, at the start or a rounding from it, and it is not a hit.
	//
	// It was chosen: well above the few parts in 1e12 that a root of a
	// grazing ray is off by in double precision, and well below a
	// tracer's own lift off a surface, a millionth of its reach, so a ray
	// lifted off the surface never has a true hit cut short by it.
	departure = 1e-9
	// slabSlack is how far outside the field's lowest and highest samples
	// the search for a hit is kept open, as a share of the field's reach, so
	// that a hit on the highest or the lowest sample is not rounded out of
	// the range the search is clipped to. It was chosen: as departure was.
	slabSlack = 1e-9
	// seamSlack is how far past the end of a cell, or before its start, a
	// root of the cell's own equation is still taken as lying on the cell,
	// as a share of a cell's width. A hit exactly on the seam between two
	// cells is then found by whichever of them the rounding favours, and a
	// root so taken is moved to the cell's own edge, so cells are still met
	// in the ray's order.
	//
	// It was chosen: far above the few parts in 1e14 a
	// root is off by away from tangent, and far below what a picture shows.
	seamSlack = 1e-9
)

// Hit is the distance along r, counted in lengths of r.Dir, at which it first
// meets the surface between near and far, found exactly and not by marching:
// a ray at any angle, however shallow, and a ray that skims a crest, is met
// where it is, or is a miss. Both ends of the interval are exclusive.
//
// A ray may start on the surface and leave it, as one does that bounces or
// that casts a shadow. A hit within departure of the start of the interval is
// the surface it is leaving, and is not found. A ray may start outside the
// square, or above the highest sample, and meets the surface where it comes
// to it; past the square's edge there is no surface, and it is a miss.
func (field *HeightField) Hit(r maths.Ray, near, far float64) (float64, bool) {
	low, high, ok := field.searchRange(r, near, far)
	if !ok {
		return 0, false
	}
	return field.IntersectExact(r, low, high)
}

// searchRange is the part of near to far where a hit can lie: from
// departure past near to far, and narrowed to where the ray is between the
// field's lowest and highest sample, so a ray above the field is not walked
// across it. It says whether any is left.
func (field *HeightField) searchRange(r maths.Ray, near,
	far float64) (low, high float64, ok bool) {
	if field == nil || field.samples < leastSamples || !(near < far) ||
		!finiteHeightfieldVec(r.At) || !finiteHeightfieldVec(r.Dir) ||
		r.Dir == (maths.Vec{}) {
		return 0, 0, false
	}
	reach := max(field.extent, field.top-field.bottom, r.At.Len())
	low = near + departure*reach/r.Dir.Len()
	high = far
	slack := slabSlack * reach
	if !clipHeightfieldAxis(r.At.Y, r.Dir.Y, field.bottom-slack,
		field.top+slack, &low, &high) {
		return 0, 0, false
	}
	return low, high, low < high
}
