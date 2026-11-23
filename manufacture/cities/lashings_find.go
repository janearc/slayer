package cities

import (
	"math"
	"slices"
)

// The axes of a Volume's corners, as an index into them: x runs across a
// street that runs along z, and z across one that runs along x. The one
// that is not across is along, and the two always add to two.
const (
	crossingX  = 0
	heightAxis = 1
	crossingZ  = 2
	bothAxes   = 2
)

// street is a stretch of the ground between two walls that face each
// other, with room over it to string cables across. The walls are faces
// of two volumes of the group, and the cables run between them.
type street struct {
	// Across is the axis the cables run along: crossingX or crossingZ.
	Across int
	// Near and Far are where the two walls stand on that axis, the
	// near one the lower, and Faces the indexes in the group of the
	// two volumes they belong to, near first.
	Near, Far float64
	Faces     [2]int
	// Start and End are the stretch the walls face each other along the
	// street, on the other horizontal axis, in order.
	Start, End float64
	// Floor and Top are the lowest and the highest a cable may be
	// anchored: above the retail level, and under the lower roof.
	Floor, Top float64
}

// along is the horizontal axis the street runs along, the one that is
// not across it.
func (street street) along() int { return bothAxes - street.Across }

// width is how far the walls stand apart, in metres.
func (street street) width() float64 { return street.Far - street.Near }

// length is how far the walls face each other along the street, in
// metres.
func (street street) length() float64 { return street.End - street.Start }

// streets is every stretch of the group where two walls face across a
// gap the culture strings, in the order of the group: for each volume,
// those it faces on the way up each axis. A pair with another volume
// standing between its faces in the way of the cables is not a street.
func streets(group []Volume, drawing *drawer) []street {
	var found []street
	for index, near := range group {
		for other, far := range group {
			for _, across := range []int{crossingX, crossingZ} {
				one, faces := facing(near, far, across, drawing)
				one.Faces = [2]int{index, other}
				if faces && drawing.trouble == nil &&
					!streetBlocked(group, one) {
					found = append(found, one)
				}
			}
		}
	}
	return found
}

// facing is the street between the far face of one volume and the near
// face of another across an axis, and whether there is one: the first
// stands below the second on the axis, the gap between is one the
// culture strings across, they face each other along the street for as
// far as it needs, and there is room between the retail level and the
// lower roof.
func facing(near, far Volume, across int, drawing *drawer) (street, bool) {
	along := bothAxes - across
	one := street{Across: across, Near: near.High[across],
		Far:   far.Low[across],
		Start: math.Max(near.Low[along], far.Low[along]),
		End:   math.Min(near.High[along], far.High[along]),
		Floor: drawing.fixed("clear-floor"),
		Top: math.Min(near.High[heightAxis], far.High[heightAxis]) -
			drawing.fixed("top-margin")}
	return one, one.width() >= drawing.fixed("span-least") &&
		one.width() <= drawing.fixed("span-most") &&
		one.length() >= drawing.fixed("run-least") &&
		one.Top-one.Floor >= drawing.fixed("hang-least")
}

// streetBlocked is whether a volume of the group other than the street's
// two stands in the air over the street where cables would hang.
func streetBlocked(group []Volume, one street) bool {
	for index, other := range group {
		if !slices.Contains(one.Faces[:], index) && other.reaches(one) {
			return true
		}
	}
	return false
}

// reaches is whether a volume has some of its own in the air a street's
// cables hang in: between the walls, along the stretch they face and
// between the floor and the top.
func (volume Volume) reaches(one street) bool {
	return overlaps(volume, one.Across, one.Near, one.Far) &&
		overlaps(volume, one.along(), one.Start, one.End) &&
		overlaps(volume, heightAxis, one.Floor, one.Top)
}

// overlaps is whether a volume holds some of the stretch of an axis
// between two points, not counting a touch at an end.
func overlaps(volume Volume, axis int, from, to float64) bool {
	return volume.Low[axis] < to && volume.High[axis] > from
}
