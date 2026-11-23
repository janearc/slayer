package over

import (
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// assemble is the mesh: the map's own corners in the order the map gives
// them, then the stretch's made corners; the map's plain cells outside the
// stretch and the stretch's finer faces inside it.
func (made laid) assemble(read *shape) (*mesh.Mesh, Receipt, error) {
	points, normals := read.corners()
	ids := made.identify(read, &points, &normals)
	faces := read.plain(made.grid.stretch)
	for _, triangle := range made.grid.faces {
		faces = append(faces, [3]int{ids[triangle[0]], ids[triangle[1]],
			ids[triangle[2]]})
	}
	built, err := mesh.New(points, faces, normals, true)
	if err != nil {
		return nil, Receipt{}, err
	}
	return built, made.receipt(read, built), nil
}

// corners are the map's samples as points of the mesh, x east, z south, y
// over the floor, and the normals the map shades them by.
func (read *shape) corners() ([]maths.Vec, []maths.Vec) {
	points := make([]maths.Vec, 0, len(read.height))
	for row := range read.down {
		for column := range read.across {
			points = append(points, maths.V(float64(column)*read.step,
				read.at(column, row), float64(row)*read.step))
		}
	}
	return points, append([]maths.Vec(nil), read.normal...)
}

// plain is the faces of every cell of the map outside the stretch, two to
// a cell, each facing up.
func (read *shape) plain(stretch Stretch) [][3]int {
	faces := make([][3]int, 0, 2*(read.across-1)*(read.down-1))
	for row := range read.down - 1 {
		for column := range read.across - 1 {
			if column >= stretch.Column && column < stretch.Column+
				stretch.Across && row >= stretch.Row &&
				row < stretch.Row+stretch.Down {
				continue
			}
			near := row*read.across + column
			far := near + read.across
			faces = append(faces, [3]int{near, far + 1, near + 1},
				[3]int{near, far, far + 1})
		}
	}
	return faces
}

// identify is the mesh's number for each place of the grid. A sample of the
// map is the map's own corner, and takes the normal the grid gives it unless
// it is on the stretch's edge, where the plain ground shades it. A made
// place is appended to the points and normals.
func (made laid) identify(read *shape, points, normals *[]maths.Vec) []int {
	shading := made.normals()
	wide := len(made.grid.eastward.cell)
	ids := make([]int, len(made.corners))
	for place, each := range made.corners {
		if !made.grid.surveyed(place) {
			ids[place] = len(*points)
			*points = append(*points, each.point)
			*normals = append(*normals, shading[place])
			continue
		}
		column := made.grid.stretch.Column +
			made.grid.eastward.cell[place%wide] +
			int(made.grid.eastward.part[place%wide])
		row := made.grid.stretch.Row + made.grid.southward.cell[place/wide] +
			int(made.grid.southward.part[place/wide])
		ids[place] = row*read.across + column
		if !made.grid.onEdge(place) {
			(*normals)[ids[place]] = shading[place]
		}
	}
	return ids
}

// normals are the shading normals of the grid's places: the shape's own,
// changed by how the made faces lean differently from the faces before
// detail. Shaded so, the faces without detail shade as the map's do, smooth
// across its samples, and detail shows in the light.
func (made laid) normals() []maths.Vec {
	before := made.gather(func(each corner) maths.Vec { return each.base })
	after := made.gather(func(each corner) maths.Vec { return each.point })
	shading := make([]maths.Vec, len(made.corners))
	for place, each := range made.corners {
		shading[place] = each.normal
		if made.grid.onEdge(place) {
			continue
		}
		shading[place] = each.normal.Add(after[place].Sub(before[place])).Unit()
	}
	return shading
}

// gather is each place's normal from the faces round it, taken one long,
// where the places are as pick says.
func (made laid) gather(pick func(corner) maths.Vec) []maths.Vec {
	sums := make([]maths.Vec, len(made.corners))
	for _, triangle := range made.grid.faces {
		first := pick(made.corners[triangle[0]])
		second := pick(made.corners[triangle[1]])
		third := pick(made.corners[triangle[2]])
		across := second.Sub(first).Cross(third.Sub(first))
		for _, place := range triangle {
			sums[place] = sums[place].Add(across)
		}
	}
	for place := range sums {
		sums[place] = sums[place].Unit()
	}
	return sums
}

// receipt says what was laid, measured from the grid and the mesh built.
func (made laid) receipt(read *shape, built *mesh.Mesh) Receipt {
	surveyed := read.across * read.down
	receipt := Receipt{Surveyed: surveyed,
		Made:  len(built.Points()) - surveyed,
		Faces: len(made.grid.faces), Bound: made.detail.bound,
		Sag: made.sag(read), Moved: read.movedIn(built)}
	receipt.Finest, receipt.Coarsest = made.edges()
	made.spread(&receipt)
	return receipt
}

// spread fills in how far the made corners moved: the median, the 90th and
// 99th share of them, the most, and the share within a tenth of the bound.
func (made laid) spread(receipt *Receipt) {
	var moves []float64
	for place, each := range made.corners {
		if !made.grid.surveyed(place) {
			moves = append(moves, math.Abs(each.moved))
		}
	}
	spreadOf(moves, receipt)
}

// spreadOf fills in a receipt's spread from how far each made corner
// moved, Lay's and Seen's alike. Where none was made there is no spread,
// and the receipt says nought.
func spreadOf(moves []float64, receipt *Receipt) {
	moves = slices.Sorted(slices.Values(moves))
	if len(moves) == 0 {
		return
	}
	of := func(share float64) float64 {
		return moves[min(len(moves)-1, int(share*float64(len(moves))))]
	}
	receipt.Median, receipt.Ninetieth, receipt.Ninetyninth = of(.5), of(.9),
		of(.99)
	receipt.Largest = moves[len(moves)-1]
	near, _ := slices.BinarySearch(moves, receipt.Bound*(1-nearBound))
	receipt.Near = float64(len(moves)-near) / float64(len(moves))
}

// nearBound is how near the bound a corner is to count as at it: within a
// tenth of it.
const nearBound = 0.1

// edges are the shortest and longest edges along the map's ways between
// places of the grid's lines, after detail.
func (made laid) edges() (finest, coarsest float64) {
	wide := len(made.grid.eastward.cell)
	finest = math.Inf(1)
	for place := range made.grid.lines() {
		for _, next := range []int{place + 1, place + wide} {
			if next >= made.grid.lines() || (next == place+1 &&
				next%wide == 0) {
				continue
			}
			length := made.corners[next].point.
				Sub(made.corners[place].point).Len()
			finest, coarsest = min(finest, length), max(coarsest, length)
		}
	}
	return finest, coarsest
}

// sag is the most a made face's middle stands, vertically, from the map's
// own surface there. Every face lies within one of the map's triangles, so
// it is only the arithmetic's rounding.
func (made laid) sag(read *shape) float64 {
	var most float64
	for _, triangle := range made.grid.faces {
		var east, height, south float64
		for _, place := range triangle {
			base := made.corners[place].base
			east, height, south = east+base.X, height+base.Y, south+base.Z
		}
		east, height, south = east/3, height/3, south/3
		stand := math.Abs(height - read.surface(read.locate(east, south)))
		most = max(most, stand)
	}
	return most
}

// movedIn is the most any sample of the map stands in the mesh from where
// the map has it, metres.
func (read *shape) movedIn(built *mesh.Mesh) float64 {
	var most float64
	points := built.Points()
	for row := range read.down {
		for column := range read.across {
			want := maths.V(float64(column)*read.step, read.at(column, row),
				float64(row)*read.step)
			most = max(most, points[row*read.across+column].Sub(want).Len())
		}
	}
	return most
}
