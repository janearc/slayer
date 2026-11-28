package noise

import "math"

// Nearest is what Cellular finds at a point: the distance to the nearest
// of the field's points, the distance to the next nearest, how far the
// point is from the edge of the nearest point's cell, and a number from
// zero up to one that belongs to the nearest point and to nothing else.
type Nearest struct {
	First, Second, Border, Identity float64
}

// Edge is the second distance less the first, nought where two cells meet
// and growing into either. It does not grow evenly: where the two points
// are near each other it grows slowly, and the cells' edge looks wide.
// Border is the distance from the edge itself, and does.
func (found Nearest) Edge() float64 { return found.Second - found.First }

// step is a vector in space, from a point to another.
type step [3]float64

// long is how long a step is.
func (by step) long() float64 {
	return math.Sqrt(by[0]*by[0] + by[1]*by[1] + by[2]*by[2])
}

// midway is how far along the way to a point, from the origin, the plane
// midway between the origin and the point is crossed, as the distance of
// the origin from that plane, when the origin is at the step from the
// nearest point: the half of the two steps' sum, taken along the line
// between them.
func (by step) midway(other step) float64 {
	var along, apart float64
	for axis := range by {
		difference := other[axis] - by[axis]
		along += 0.5 * (by[axis] + other[axis]) * difference
		apart += difference * difference
	}
	return along / math.Sqrt(apart)
}

// neighbourhood is how many lattice cells are looked at about a point: its
// own and the twenty six round it, three to a side.
const neighbourhood = 27

// side is how many lattice cells are looked at to a side.
const side = 3

// Cellular is cellular noise (Worley 1996, "A cellular texture basis
// function", SIGGRAPH; from memory, to shelve) in three dimensions: each
// cell of the integer lattice holds one point, put in it by the seed's
// hash, and a point of space is asked how far the nearest two of them are.
// Space falls into irregular polyhedral cells, one to each point, which
// meet along the planes midway between points. Border is the distance of
// the point from the nearest of those planes of its own cell, worked as
// Quilez does (iquilezles.org, "Voronoi edges", from memory, to shelve):
// the distance to the plane midway between the nearest point and each
// other, the least of them.
//
// Only the point's own lattice cell and the twenty six round it are
// looked at: the nearest point is always among them. The rest is right
// at all but a few in a thousand places, where the Second is a little too
// far and the Border a little too wide.
//
// A coordinate that is NaN or infinite is no place, and every number is
// NaN there, as Fractal's is.
func Cellular(seed uint64, x, y, z float64) Nearest {
	if !finite(x) || !finite(y) || !finite(z) {
		return Nearest{First: math.NaN(), Second: math.NaN(),
			Border: math.NaN(), Identity: math.NaN()}
	}
	home := [3]int64{int64(math.Floor(x)), int64(math.Floor(y)),
		int64(math.Floor(z))}
	var toward [neighbourhood]step
	var cells [neighbourhood][3]int64
	nearest := 0
	for index := range toward {
		cells[index] = [3]int64{home[0] + int64(index%side) - 1,
			home[1] + int64(index/side%side) - 1,
			home[2] + int64(index/(side*side)) - 1}
		toward[index] = pointOf(seed, cells[index], [3]float64{x, y, z})
		if toward[index].long() < toward[nearest].long() {
			nearest = index
		}
	}
	cell := cells[nearest]
	found := Nearest{First: toward[nearest].long(), Second: math.Inf(1),
		Border: math.Inf(1), Identity: Hash3(seed+pointStreamIdentity,
			cell[0], cell[1], cell[2])}
	for index := range toward {
		if index != nearest {
			found.Second = math.Min(found.Second, toward[index].long())
			found.Border = math.Min(found.Border,
				toward[nearest].midway(toward[index]))
		}
	}
	return found
}

// pointOf is the step from a place to the point of a lattice cell.
func pointOf(seed uint64, cell [3]int64, at [3]float64) step {
	var toward step
	for axis, stream := range [3]uint64{0, pointStreamY, pointStreamZ} {
		toward[axis] = float64(cell[axis]) +
			Hash3(seed+stream, cell[0], cell[1], cell[2]) - at[axis]
	}
	return toward
}

// The streams a cell's point takes its three coordinates and its identity
// from, offsets of the seed, so that each is its own field. Picked: any
// four different numbers do.
const (
	pointStreamY        = 101
	pointStreamZ        = 202
	pointStreamIdentity = 303
)
