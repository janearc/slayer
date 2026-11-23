package over

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// growth is how much coarser the spacing is made, each time, until the
// stretch fits the faces it may have. It is picked: a tenth.
const growth = 1.1

// axis is one way across the stretch: its cells each cut into so many
// parts, and the lines between the parts numbered from the first.
type axis struct {
	cell  []int     // which cell of the stretch each line is in
	part  []float64 // how far across that cell it is, nought to one
	start []int     // the first line of each cell, and then the last line
}

// newAxis is an axis whose cells are cut into the given counts of parts.
// The last line is the far side of the last cell.
func newAxis(counts []int) axis {
	var made axis
	for cell, count := range counts {
		made.start = append(made.start, len(made.cell))
		for part := range count {
			made.cell = append(made.cell, cell)
			made.part = append(made.part, float64(part)/float64(count))
		}
	}
	made.start = append(made.start, len(made.cell))
	made.cell = append(made.cell, len(counts)-1)
	made.part = append(made.part, 1)
	return made
}

// surveyed is whether a line of the axis is where the survey has a sample:
// on the edge of a cell.
func (way axis) surveyed(line int) bool {
	return way.part[line] == 0 || way.part[line] == 1
}

// extra is a place on the crease between the two triangles of a cell, where
// a line of the grid meets it. The grid's faces end there, so that none
// of them lies across the crease. It says which cell, and how far across and
// down it, nought to one; the two are the same.
type extra struct {
	column, row  int
	across, down float64
}

// grid is the stretch cut finer: its lines running east and south, the
// spacing they are cut to, and the faces between. Places of the grid are
// numbered row by row along its lines and then by the extras.
type grid struct {
	stretch             Stretch
	eastward, southward axis
	spacing             float64
	extras              []extra
	faces               [][3]int
}

// plan cuts the stretch as fine as the rules say, or the coarsest that
// fits the faces they allow. Each cell is cut so that no edge of it, along
// either way, is longer than the spacing, and every cell in a row or a
// column is cut the same number of parts, so that a cell meets its
// neighbour corner to corner.
//
// The faces are counted from the cuts, before any is made, so the cap
// bounds the work and the memory as well as what is kept.
func plan(read *shape, stretch Stretch, rules Rules) grid {
	for spacing := rules.Spacing; ; spacing *= growth {
		east := read.cuts(stretch, spacing, [2]int{1, 0})
		south := read.cuts(stretch, spacing, [2]int{0, 1})
		if mostFaces(east, south) > rules.Faces {
			continue
		}
		made := grid{stretch: stretch, spacing: spacing,
			eastward: newAxis(east), southward: newAxis(south)}
		made.cut()
		return made
	}
}

// mostFaces is the most faces cells cut into these parts east and south
// make: two a square, and in a cell cut differently each way two more
// for each square the crease between its triangles crosses, which is at
// most its parts across and down together. A cell cut the same each way
// has its crease through its squares' corners, and no more faces.
func mostFaces(east, south []int) int {
	var faces int
	for _, across := range east {
		for _, down := range south {
			faces += facesPerSquare * across * down
			if across != down {
				faces += facesPerSquare * (across + down)
			}
		}
	}
	return faces
}

// cut makes the grid's faces, cell by cell.
func (made *grid) cut() {
	made.faces = make([][3]int, 0, 2*(len(made.eastward.cell)-1)*
		(len(made.southward.cell)-1))
	for row := range made.stretch.Down {
		for column := range made.stretch.Across {
			made.cell(column, row)
		}
	}
}

// cell makes the faces of one cell of the stretch. Where a square of the grid
// lies wholly on one side of the crease between the cell's two triangles
// it is two faces. Where the crease crosses it, the square is cut along the
// crease and each side is made faces of its own.
func (made *grid) cell(column, row int) {
	across := made.eastward.start[column+1] - made.eastward.start[column]
	down := made.southward.start[row+1] - made.southward.start[row]
	crossings := map[[2]int]int{}
	for part := range down {
		for along := range across {
			made.square(column, row, [2]int{along, part}, [2]int{across, down},
				crossings)
		}
	}
}

// square makes the faces of one square of the grid in a cell: at the given
// part along and down, in a cell cut into the given parts each way.
func (made *grid) square(column, row int, at, cuts [2]int,
	crossings map[[2]int]int) {
	wide := len(made.eastward.cell)
	first := (made.southward.start[row]+at[1])*wide +
		made.eastward.start[column] + at[0]
	// corners turn so that the faces face up: near, far, far and one more,
	// near and one more
	corners := [4]int{first, first + wide, first + wide + 1, first + 1}
	parts := [4][2]int{{at[0], at[1]}, {at[0], at[1] + 1},
		{at[0] + 1, at[1] + 1}, {at[0] + 1, at[1]}}
	var sides [4]int
	for i, part := range parts {
		sides[i] = part[0]*cuts[1] - part[1]*cuts[0]
	}
	if least, most := slicesRange(sides); least >= 0 || most <= 0 {
		made.faces = append(made.faces, [3]int{corners[0], corners[2],
			corners[3]}, [3]int{corners[0], corners[1], corners[2]})
		return
	}
	for _, way := range []int{1, -1} {
		made.piece(column, row, corners, parts, sides, way, cuts, crossings)
	}
}

// slicesRange is the least and the most of four numbers.
func slicesRange(values [4]int) (least, most int) {
	least, most = values[0], values[0]
	for _, each := range values {
		least, most = min(least, each), max(most, each)
	}
	return least, most
}

// piece makes the faces of the part of a square on one side of the crease:
// way one is the side from the cell's near corner toward the east, minus one
// is the other. The part's corners are the square's that are on that side,
// or on the crease, and the places where the crease crosses its edges.
func (made *grid) piece(column, row int, corners [4]int, parts [4][2]int,
	sides [4]int, way int, cuts [2]int, crossings map[[2]int]int) {
	var ring []int
	for i := range corners {
		next := (i + 1) % 4
		if way*sides[i] >= 0 {
			ring = append(ring, corners[i])
		}
		if sides[i]*sides[next] < 0 {
			ring = append(ring, made.crossing(column, row, [2]int{i, next},
				corners, parts, cuts, crossings))
		}
	}
	for k := 1; k+1 < len(ring); k++ {
		made.faces = append(made.faces, [3]int{ring[0], ring[k], ring[k+1]})
	}
}

// crossing is the place where the crease crosses an edge of a square, as a
// number among the grid's places, made once however many squares share the
// edge. The crease is where a cell's parts across and down are the same, so a
// edge along a line across has it a share of the way down equal to its
// line's, and along a line down, a share across equal to its line's.
func (made *grid) crossing(column, row int, edge [2]int, corners [4]int,
	parts [4][2]int, cuts [2]int, crossings map[[2]int]int) int {
	key := [2]int{min(corners[edge[0]], corners[edge[1]]),
		max(corners[edge[0]], corners[edge[1]])}
	if number, found := crossings[key]; found {
		return number
	}
	share := float64(parts[edge[0]][1]) / float64(cuts[1])
	if parts[edge[0]][0] == parts[edge[1]][0] {
		share = float64(parts[edge[0]][0]) / float64(cuts[0])
	}
	number := len(made.eastward.cell)*len(made.southward.cell) +
		len(made.extras)
	made.extras = append(made.extras, extra{
		column: made.stretch.Column + column, row: made.stretch.Row + row,
		across: share, down: share})
	crossings[key] = number
	return number
}

// cuts is the parts each cell of the stretch is cut into along one way,
// east or south: as many as make the longest edge of the cell, that way,
// no longer than the spacing. Edges are measured on the shape, up its
// rises.
func (read *shape) cuts(stretch Stretch, spacing float64,
	move [2]int) []int {
	cells, lines := stretch.Across, stretch.Down+1
	if move[1] == 1 {
		cells, lines = stretch.Down, stretch.Across+1
	}
	counts := make([]int, cells)
	for cell := range counts {
		var longest float64
		for line := range lines {
			column := stretch.Column + cell*move[0] + line*move[1]
			row := stretch.Row + cell*move[1] + line*move[0]
			rise := read.at(column+move[0], row+move[1]) - read.at(column, row)
			longest = max(longest, math.Hypot(read.step, rise))
		}
		counts[cell] = max(1, int(math.Ceil(longest/spacing)))
	}
	return counts
}

// lines is how many places the grid has along its lines, before the extras.
func (made grid) lines() int {
	return len(made.eastward.cell) * len(made.southward.cell)
}

// surveyed is whether a place of the grid is a sample of the map.
func (made grid) surveyed(place int) bool {
	if place >= made.lines() {
		return false
	}
	wide := len(made.eastward.cell)
	return made.eastward.surveyed(place%wide) &&
		made.southward.surveyed(place/wide)
}

// onEdge is whether a place of the grid is on the stretch's outer edge,
// where it is shared with the plain ground.
func (made grid) onEdge(place int) bool {
	if place >= made.lines() {
		return false
	}
	wide := len(made.eastward.cell)
	return place%wide == 0 || place/wide == 0 || place%wide == wide-1 ||
		place/wide == len(made.southward.cell)-1
}

// corner is a place of the grid: where it is in the shape, where detail
// moves it to, the shape's normal there and how far it moved.
type corner struct {
	base, point, normal maths.Vec
	moved               float64
}

// laid is the grid with its detail: every place and where it landed.
type laid struct {
	grid    grid
	corners []corner // as the grid's faces number them
	detail  ground
}

// lay is the grid's places, each in the shape and then moved by the rules.
func (made grid) lay(read *shape, rules Rules) laid {
	return made.layWith(read, newGround(read, made.stretch, rules,
		made.spacing))
}

// layWith is the grid's places, each in the shape and then moved by the
// detail given.
func (made grid) layWith(read *shape, detail ground) laid {
	wide := len(made.eastward.cell)
	corners := make([]corner, 0, made.lines()+len(made.extras))
	for place := range made.lines() {
		column := float64(made.stretch.Column+made.eastward.cell[place%wide]) +
			made.eastward.part[place%wide]
		row := float64(made.stretch.Row+made.southward.cell[place/wide]) +
			made.southward.part[place/wide]
		corners = append(corners, made.corner(read, detail, column, row,
			made.surveyed(place)))
	}
	for _, each := range made.extras {
		corners = append(corners, made.corner(read, detail,
			float64(each.column)+each.across, float64(each.row)+each.down,
			false))
	}
	return laid{grid: made, corners: corners, detail: detail}
}

// corner is one place of the grid, at a column and row of the map, in cells
// and parts of one. A sample of the map does not move.
func (made grid) corner(read *shape, detail ground, column, row float64,
	surveyed bool) corner {
	at := read.locate(column*read.step, row*read.step)
	base := maths.V(column*read.step, read.surface(at), row*read.step)
	normal := read.smoothed(at)
	if surveyed {
		return corner{base: base, point: base, normal: normal}
	}
	moved := detail.displacement(at, base, normal)
	return corner{base: base, point: base.Add(normal.Scale(moved)),
		normal: normal, moved: moved}
}
