package hexes

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// A hex lattice on a plane, pointy side up, rows along one axis and
// columns along the other, with odd rows pushed half a tile along. Flat
// lays it on the x z plane and Band wraps it round a cylinder.
//
// Both do their arithmetic here, in the lattice's own two coordinates,
// along a row and across the rows.

// lattice is a hex lattice with a step between neighbouring middles.
type lattice struct{ step float64 }

// root3 is the square root of three, which is the shape of a hexagon.
var root3 = math.Sqrt(3)

// centre is the middle of a column and row.
func (grid lattice) centre(column, row int) (along, across float64) {
	along = grid.step * (float64(column) + 0.5*float64(row&1))
	across = grid.step * root3 / 2 * float64(row)
	return along, across
}

// nearest is the column and row a point is in: the lattice's axial
// coordinates, rounded as cube coordinates, then made offset.
func (grid lattice) nearest(along, across float64) (column, row int) {
	size := grid.step / root3
	looseRow := across * 2 / (3 * size)
	looseColumn := along/(size*root3) - looseRow/2
	axialColumn, axialRow := roundHex(looseColumn, looseRow)
	return axialColumn + (axialRow-(axialRow&1))/2, axialRow
}

// roundHex rounds axial coordinates to the nearest hex, through the three
// cube coordinates, fixing whichever one rounding moved the most.
func roundHex(looseColumn, looseRow float64) (column, row int) {
	looseThird := -looseColumn - looseRow
	roundedColumn := math.Round(looseColumn)
	roundedRow := math.Round(looseRow)
	roundedThird := math.Round(looseThird)
	movedColumn := math.Abs(roundedColumn - looseColumn)
	movedRow := math.Abs(roundedRow - looseRow)
	movedThird := math.Abs(roundedThird - looseThird)
	switch {
	case movedColumn > movedRow && movedColumn > movedThird:
		roundedColumn = -roundedRow - roundedThird
	case movedRow > movedThird:
		roundedRow = -roundedColumn - roundedThird
	}
	return int(roundedColumn), int(roundedRow)
}

// steps6 is the six neighbours of a tile in offset coordinates, for even
// rows and for odd.
var steps6 = [2][6][2]int{
	{{-1, 0}, {1, 0}, {-1, -1}, {0, -1}, {-1, 1}, {0, 1}},
	{{-1, 0}, {1, 0}, {0, -1}, {1, -1}, {0, 1}, {1, 1}},
}

// around is the six neighbours of a column and row, before any wrap or
// edge is applied.
func around(column, row int) [6][2]int {
	var neighbours [6][2]int
	for place, shift := range steps6[row&1] {
		neighbours[place] = [2]int{column + shift[0], row + shift[1]}
	}
	return neighbours
}

// fromEdge is how far a point is from the edge of a tile, given the
// tile's middle and its neighbours' middles: the nearest of the
// half-planes between the middle and each neighbour.
func fromEdge(along, across, midAlong, midAcross float64,
	near [][2]float64) float64 {
	best := math.Inf(1)
	for _, neighbour := range near {
		gapAlong := midAlong - neighbour[0]
		gapAcross := midAcross - neighbour[1]
		length := math.Hypot(gapAlong, gapAcross)
		betweenAlong := (midAlong + neighbour[0]) / 2
		betweenAcross := (midAcross + neighbour[1]) / 2
		best = math.Min(best, ((along-betweenAlong)*gapAlong+
			(across-betweenAcross)*gapAcross)/length)
	}
	return best
}

// Flat is a hex lattice on the x z plane, y up, with Step between
// neighbouring middles: a landscape. Across and Down are how many columns
// and rows it holds, from the origin toward plus x and plus z; a point
// beyond them is still in a tile, but Tiles does not list it.
type Flat struct {
	//ninja:vow gate a value, whose fields are what it is
	Step float64
	//ninja:vow gate a value, whose fields are what it is
	Across, Down int
}

// spacing is the step, and one when the step is zero or below it.
func (flat Flat) spacing() float64 {
	if flat.Step <= 0 {
		return 1
	}
	return flat.Step
}

// lat is the lattice.
func (flat Flat) lat() lattice { return lattice{flat.spacing()} }

// Nearest is the tile under a point, ignoring its height.
func (flat Flat) Nearest(point maths.Vec) Tile {
	column, row := flat.lat().nearest(point.X, point.Z)
	return Tile{0, column, row}
}

// Centre is the middle of a tile, on the plane.
func (flat Flat) Centre(tile Tile) maths.Vec {
	along, across := flat.lat().centre(tile.Column, tile.Row)
	return maths.V(along, 0, across)
}

// Neighbours is the six tiles round this one, fewer at the edge of the
// extent, in order of ID.
func (flat Flat) Neighbours(tile Tile) []Tile {
	var kept []Tile
	for _, next := range around(tile.Column, tile.Row) {
		if flat.holds(next[0], next[1]) {
			kept = append(kept, Tile{0, next[0], next[1]})
		}
	}
	sort.Slice(kept, func(left, right int) bool {
		return flat.ID(kept[left]) < flat.ID(kept[right])
	})
	return kept
}

// holds is whether a column and row are inside the extent.
func (flat Flat) holds(column, row int) bool {
	return column >= 0 && row >= 0 &&
		column < flat.Across && row < flat.Down
}

// Edge is how far a point is from the edge of a tile, on the plane.
func (flat Flat) Edge(point maths.Vec, tile Tile) float64 {
	grid := flat.lat()
	midAlong, midAcross := grid.centre(tile.Column, tile.Row)
	var near [][2]float64
	for _, next := range around(tile.Column, tile.Row) {
		along, across := grid.centre(next[0], next[1])
		near = append(near, [2]float64{along, across})
	}
	return fromEdge(point.X, point.Z, midAlong, midAcross, near)
}

// ID is the row then the column, each in its own thirty-two bits, so two
// tiles 2^32 columns apart share one; a flat extent is never so wide.
func (flat Flat) ID(tile Tile) int64 { return pair(tile.Column, tile.Row) }

// pair is a column and a row as one number.
func pair(column, row int) int64 {
	return int64(row)<<32 | int64(uint32(column))
}

// Spacing is the distance between neighbouring middles.
func (flat Flat) Spacing() float64 { return flat.spacing() }

// Tiles is every tile inside the extent, in order of ID.
func (flat Flat) Tiles() []Tile {
	kept := make([]Tile, 0, max(0, flat.Across)*max(0, flat.Down))
	for row := 0; row < flat.Down; row++ {
		for column := 0; column < flat.Across; column++ {
			kept = append(kept, Tile{0, column, row})
		}
	}
	return kept
}

// Finer is the lattice at half the spacing, twice as many columns and
// rows, with every middle of this one.
func (flat Flat) Finer() Tiling {
	return Flat{Step: flat.spacing() / 2, Across: 2 * flat.Across,
		Down: 2 * flat.Down}
}

// Band is a hex lattice wrapped once round a cylinder of Radius about the y
// axis, Around tiles round, so a row is a ring: the inside of a ring world.
// Rows run along y from minus Width/2, as many as fit.
//
// Angle is measured from plus z toward plus x, so a point on the cylinder at z
// = Radius is in column zero.
type Band struct {
	//ninja:vow gate a value, whose fields are what it is
	Radius float64
	//ninja:vow gate a value, whose fields are what it is
	Width float64
	//ninja:vow gate a value, whose fields are what it is
	Around int
}

// around is how many tiles go round, with fewer than six read as six.
func (band Band) around() int { return max(6, band.Around) }

// spacing is the distance between neighbouring middles: the
// circumference over Around.
func (band Band) spacing() float64 {
	return 2 * math.Pi * band.Radius / float64(band.around())
}

// lat is the lattice.
func (band Band) lat() lattice { return lattice{band.spacing()} }

// rows is how many rows fit across the width.
func (band Band) rows() int {
	return max(1,
		int(math.Floor(band.Width/(band.spacing()*root3/2)))+1)
}

// unwrap is a point as the lattice sees it: along the circumference,
// from zero up to it, and across along y from the band's bottom.
func (band Band) unwrap(point maths.Vec) (along, across float64) {
	angle := math.Atan2(point.X, point.Z)
	if angle < 0 {
		angle += 2 * math.Pi
	}
	return band.Radius * angle, point.Y + band.Width/2
}

// wrap keeps a column inside the band.
func (band Band) wrap(column int) int {
	count := band.around()
	return ((column % count) + count) % count
}

// Nearest is the tile a point is in, by its angle round the axis and its
// height, whatever its distance from the axis.
func (band Band) Nearest(point maths.Vec) Tile {
	along, across := band.unwrap(point)
	column, row := band.lat().nearest(along, across)
	return Tile{0, band.wrap(column), row}
}

// Centre is the middle of a tile, on the cylinder.
func (band Band) Centre(tile Tile) maths.Vec {
	along, across := band.lat().centre(tile.Column, tile.Row)
	angle := along / band.Radius
	return maths.V(band.Radius*math.Sin(angle),
		across-band.Width/2, band.Radius*math.Cos(angle))
}

// Neighbours is the six tiles round this one, wrapping round the band,
// fewer at the band's rim, in order of ID.
func (band Band) Neighbours(tile Tile) []Tile {
	var kept []Tile
	for _, next := range around(tile.Column, tile.Row) {
		if next[1] < 0 || next[1] >= band.rows() {
			continue
		}
		wrapped := Tile{0, band.wrap(next[0]), next[1]}
		dup := false
		for _, have := range kept {
			dup = dup || have == wrapped
		}
		if !dup {
			kept = append(kept, wrapped)
		}
	}
	sort.Slice(kept, func(left, right int) bool {
		return band.ID(kept[left]) < band.ID(kept[right])
	})
	return kept
}

// Edge is how far a point is from the edge of a tile, measured on the
// unwrapped cylinder, with the seam crossed where a neighbour is over it.
func (band Band) Edge(point maths.Vec, tile Tile) float64 {
	grid := band.lat()
	along, across := band.unwrap(point)
	midAlong, midAcross := grid.centre(tile.Column, tile.Row)
	circumference := 2 * math.Pi * band.Radius
	// bring the point to the same side of the seam as the middle
	if along-midAlong > circumference/2 {
		along -= circumference
	} else if midAlong-along > circumference/2 {
		along += circumference
	}
	var near [][2]float64
	for _, next := range around(tile.Column, tile.Row) {
		nextAlong, nextAcross := grid.centre(next[0], next[1])
		near = append(near, [2]float64{nextAlong, nextAcross})
	}
	return fromEdge(along, across, midAlong, midAcross, near)
}

// ID is the row then the column, each in its own thirty-two bits; a band
// of more columns than that, as an orbital is, numbers its tiles row by
// row instead, so no two share one.
func (band Band) ID(tile Tile) int64 {
	if count := band.around(); count > 1<<32 {
		return int64(tile.Row)*int64(count) + int64(tile.Column)
	}
	return pair(tile.Column, tile.Row)
}

// Spacing is the distance between neighbouring middles.
func (band Band) Spacing() float64 { return band.spacing() }

// Tiles is every tile of the band, in order of ID.
func (band Band) Tiles() []Tile {
	rows, count := band.rows(), band.around()
	kept := make([]Tile, 0, rows*count)
	for row := 0; row < rows; row++ {
		for column := 0; column < count; column++ {
			kept = append(kept, Tile{0, column, row})
		}
	}
	return kept
}

// Finer is the band with twice as many tiles round, and so half the
// spacing and twice the rows, with every middle of this one.
func (band Band) Finer() Tiling {
	return Band{Radius: band.Radius, Width: band.Width,
		Around: 2 * band.around()}
}

// Distance is the straight line on the plane, ignoring height.
func (flat Flat) Distance(from, to maths.Vec) float64 {
	return math.Hypot(from.X-to.X, from.Z-to.Z)
}

// bounds is the extent's middle and its half-sizes across and down, from
// the four corner tiles' middles.
func (flat Flat) bounds() (mid maths.Vec, halfAcross, halfDown float64) {
	across, down := max(1, flat.Across), max(1, flat.Down)
	leastX, mostX := math.Inf(1), math.Inf(-1)
	leastZ, mostZ := math.Inf(1), math.Inf(-1)
	for _, corner := range []Tile{{0, 0, 0}, {0, across - 1, 0},
		{0, 0, down - 1}, {0, across - 1, down - 1}} {
		middle := flat.Centre(corner)
		leastX = math.Min(leastX, middle.X)
		mostX = math.Max(mostX, middle.X)
		leastZ = math.Min(leastZ, middle.Z)
		mostZ = math.Max(mostZ, middle.Z)
	}
	return maths.V((leastX+mostX)/2, 0, (leastZ+mostZ)/2),
		(mostX - leastX) / 2, (mostZ - leastZ) / 2
}

// Latitude is how far a point lies along north from the middle of the extent,
// as a share of the extent's reach that way: north is taken in the plane, and a
// north that points straight up gives zero.
func (flat Flat) Latitude(point, north maths.Vec) float64 {
	axis := maths.V(north.X, 0, north.Z)
	if axis.Len() < 1e-12 {
		return 0
	}
	axis = axis.Unit()
	mid, halfAcross, halfDown := flat.bounds()
	reach := halfAcross*math.Abs(axis.X) + halfDown*math.Abs(axis.Z)
	if reach <= 0 {
		return 0
	}
	return math.Min(1, math.Abs(point.Sub(mid).Dot(axis))/reach)
}

// Somewhere is a point of the extent: a share across it and a share
// down it.
func (flat Flat) Somewhere(acrossShare, _, downShare float64) maths.Vec {
	mid, halfAcross, halfDown := flat.bounds()
	return mid.Add(maths.V((acrossShare*2-1)*halfAcross, 0,
		(downShare*2-1)*halfDown))
}

// azimuth is a point's angle round the band, as Centre lays it out.
func azimuth(point maths.Vec) float64 {
	return math.Atan2(point.X, point.Z)
}

// Distance is along the cylinder: the shorter way round, times the
// radius, and the difference along the axis, together.
func (band Band) Distance(from, to maths.Vec) float64 {
	round := math.Abs(math.Remainder(azimuth(from)-azimuth(to),
		2*math.Pi))
	return math.Hypot(round*band.Radius, from.Y-to.Y)
}

// Latitude is how far a point lies from the band's mid-line toward its rims,
// which are a band's poles: zero in the middle, one at a rim. North is the
// band's own axis and the given one is not used.
func (band Band) Latitude(point, _ maths.Vec) float64 {
	if band.Width <= 0 {
		return 0
	}
	return math.Min(1, math.Abs(point.Y)/(band.Width/2))
}

// Somewhere is a point of the band: a share round it and a share
// along it.
func (band Band) Somewhere(roundShare, alongShare, _ float64) maths.Vec {
	angle := roundShare * 2 * math.Pi
	return maths.V(band.Radius*math.Sin(angle),
		(alongShare*2-1)*band.Width/2,
		band.Radius*math.Cos(angle))
}
