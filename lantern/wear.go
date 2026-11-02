package lantern

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// cells is how many cells a worn panel is cut into along each side
const cells = 32

// worn is one panel with the damage a hard life leaves: its edges bitten inward
// and abraded, small holes punched through, and the surface dented in and out.
//
// the damage is a field over the panel, metal where it is greater than zero and
// gone where it is below, and the panel is cut exactly along where it crosses
// zero, so the edges are frayed and never stepped. wear from zero to one is how
// much; id makes every panel's own.
func worn(square [4]maths.Vec, out maths.Vec, id int64,
	wear float64) ([]maths.Vec, [][3]int) {
	plate := plate{square: square, out: out, seed: uint64(id),
		wear: wear}
	var faces [][3]int
	for row := range cells {
		for column := range cells {
			first := plate.corner(column, row)
			right := plate.corner(column+1, row)
			above := plate.corner(column, row+1)
			corner := plate.corner(column+1, row+1)
			lower := plate.cut(first, right, corner)
			upper := plate.cut(first, corner, above)
			faces = append(append(faces, lower...), upper...)
		}
	}
	return plate.points, faces
}

// plate is a panel being worn: where its corners are, which way is out,
// and the points made so far, each kept once by where it is on the
// panel so faces that meet share them
type plate struct {
	square [4]maths.Vec
	out    maths.Vec
	seed   uint64
	wear   float64
	points []maths.Vec
	places [][2]float64
	index  map[[2]float64]int
}

// at is the point at a place on the panel, from (0, 0) to (1, 1),
// dented in or out by a slow noise
func (worn *plate) at(u, v float64) maths.Vec {
	across := worn.square[1].Sub(worn.square[0])
	up := worn.square[3].Sub(worn.square[0])
	depth := 0.06 * worn.wear * across.Len()
	dent := noise.Fractal(worn.seed, u*3, v*3, 0, 3) * depth
	flat := worn.square[0].Add(across.Scale(u)).Add(up.Scale(v))
	return flat.Add(worn.out.Scale(dent))
}

// corner is the index of the grid's corner at a column and row
func (worn *plate) corner(column, row int) int {
	return worn.point(float64(column)/cells, float64(row)/cells)
}

// point is the index of the point at a place, made the first time
func (worn *plate) point(u, v float64) int {
	if worn.index == nil {
		worn.index = map[[2]float64]int{}
	}
	key := [2]float64{u, v}
	if index, ok := worn.index[key]; ok {
		return index
	}
	worn.index[key] = len(worn.points)
	worn.points = append(worn.points, worn.at(u, v))
	worn.places = append(worn.places, key)
	return worn.index[key]
}
