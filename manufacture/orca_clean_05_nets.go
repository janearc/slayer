package manufacture

import (
	"sync"

	"github.com/janearc/slayer/maths"
)

// orcaWorkers is how many goroutines sample the field at once. It is
// chosen to leave the machine's other work its share.
const orcaWorkers = 4

// orcaLattice is the regular grid of sample points the field is read on:
// its lowest corner, the spacing, and the points along each axis.
type orcaLattice struct {
	low        maths.Vec
	step       float64
	nx, ny, nz int
}

// orcaCorner lists a cell's eight corners as offsets, numbered so that
// bit zero is x, bit one is y and bit two is z.
var orcaCorner = [8][3]int{
	{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0},
	{0, 0, 1}, {1, 0, 1}, {0, 1, 1}, {1, 1, 1},
}

// orcaCellEdge lists a cell's twelve edges as pairs of corners.
var orcaCellEdge = [12][2]int{
	{0, 1}, {2, 3}, {4, 5}, {6, 7},
	{0, 2}, {1, 3}, {4, 6}, {5, 7},
	{0, 4}, {1, 5}, {2, 6}, {3, 7},
}

// index is the place of lattice point (i, j, k) in the sample array.
func (l orcaLattice) index(i, j, k int) int { return (k*l.ny+j)*l.nx + i }

// cell is the place of cell (i, j, k), the cell whose lowest corner is
// that lattice point, in the array of cell vertices.
func (l orcaLattice) cell(i, j, k int) int {
	return (k*(l.ny-1)+j)*(l.nx-1) + i
}

// sample reads the field at every lattice point, in parallel by layers.
func (l orcaLattice) sample(field func(maths.Vec) float64) []float32 {
	values := make([]float32, l.nx*l.ny*l.nz)
	var group sync.WaitGroup
	for worker := 0; worker < orcaWorkers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for k := worker; k < l.nz; k += orcaWorkers {
				l.fill(values, field, k)
			}
		}()
	}
	group.Wait()
	return values
}

// fill reads the field over layer k of the lattice.
func (l orcaLattice) fill(values []float32, field func(maths.Vec) float64,
	k int,
) {
	for j := 0; j < l.ny; j++ {
		for i := 0; i < l.nx; i++ {
			at := l.low.Add(maths.V(float64(i), float64(j), float64(k)).
				Scale(l.step))
			values[l.index(i, j, k)] = float32(field(at))
		}
	}
}

// vertex places the one surface vertex of cell (i, j, k): the mean of
// where the surface crosses the cell's edges. It reports false when the
// cell holds no surface.
func (l orcaLattice) vertex(values []float32, i, j, k int) (maths.Vec, bool) {
	var corner [8]float64
	inside := 0
	for c, offset := range orcaCorner {
		corner[c] = float64(values[l.index(i+offset[0], j+offset[1], k+offset[2])])
		if corner[c] < 0 {
			inside++
		}
	}
	if inside == 0 || inside == len(corner) {
		return maths.Vec{}, false
	}
	var sum maths.Vec
	crossings := 0.
	for _, edge := range orcaCellEdge {
		a, b := corner[edge[0]], corner[edge[1]]
		if (a < 0) == (b < 0) {
			continue
		}
		t := a / (a - b)
		from, to := orcaCorner[edge[0]], orcaCorner[edge[1]]
		sum = sum.Add(maths.V(
			float64(from[0])+t*float64(to[0]-from[0]),
			float64(from[1])+t*float64(to[1]-from[1]),
			float64(from[2])+t*float64(to[2]-from[2])))
		crossings++
	}
	mean := sum.Scale(1 / crossings)
	return l.low.Add(maths.V(float64(i)+mean.X, float64(j)+mean.Y,
		float64(k)+mean.Z).Scale(l.step)), true
}

// vertices places a vertex in every cell the surface crosses. cells
// names each cell's vertex, or is minus one where there is none.
func (l orcaLattice) vertices(values []float32) ([]maths.Vec, []int32) {
	cells := make([]int32, (l.nx-1)*(l.ny-1)*(l.nz-1))
	var points []maths.Vec
	for k := 0; k < l.nz-1; k++ {
		for j := 0; j < l.ny-1; j++ {
			for i := 0; i < l.nx-1; i++ {
				at, found := l.vertex(values, i, j, k)
				if !found {
					cells[l.cell(i, j, k)] = -1
					continue
				}
				cells[l.cell(i, j, k)] = int32(len(points))
				points = append(points, at)
			}
		}
	}
	return points, cells
}

// orcaQuad turns the four cells round one lattice edge into two triangles,
// split along the shorter diagonal. The cells are given counterclockwise
// about the edge's direction; reverse says the inside lies at the far
// end of the edge, so the order is turned round.
func orcaQuad(points []maths.Vec, ring [4]int32, reverse bool) [][3]int {
	a, b, c, d := int(ring[0]), int(ring[1]), int(ring[2]), int(ring[3])
	if reverse {
		a, b, c, d = d, c, b, a
	}
	across := points[a].Sub(points[c]).Len()
	other := points[b].Sub(points[d]).Len()
	if across <= other {
		return [][3]int{{a, b, c}, {a, c, d}}
	}
	return [][3]int{{a, b, d}, {b, c, d}}
}

// faces joins the cell vertices into triangles: one quad round every
// lattice edge the surface crosses, wound so the outside is the side the
// field is positive on.
func (l orcaLattice) faces(values []float32, cells []int32,
	points []maths.Vec,
) [][3]int {
	faces := make([][3]int, 0, 2*len(points))
	at := func(i, j, k int) int32 { return cells[l.cell(i, j, k)] }
	for k := 1; k < l.nz-1; k++ {
		for j := 1; j < l.ny-1; j++ {
			for i := 1; i < l.nx-1; i++ {
				here := values[l.index(i, j, k)] < 0
				if (values[l.index(i+1, j, k)] < 0) != here {
					faces = append(faces, orcaQuad(points, [4]int32{
						at(i, j-1, k-1), at(i, j, k-1), at(i, j, k), at(i, j-1, k),
					}, !here)...)
				}
				if (values[l.index(i, j+1, k)] < 0) != here {
					faces = append(faces, orcaQuad(points, [4]int32{
						at(i-1, j, k-1), at(i-1, j, k), at(i, j, k), at(i, j, k-1),
					}, !here)...)
				}
				if (values[l.index(i, j, k+1)] < 0) != here {
					faces = append(faces, orcaQuad(points, [4]int32{
						at(i-1, j-1, k), at(i, j-1, k), at(i, j, k), at(i-1, j, k),
					}, !here)...)
				}
			}
		}
	}
	return faces
}

// orcaContour finds the surface where field crosses zero on the lattice,
// as shared points and triangles wound outward.
func orcaContour(field func(maths.Vec) float64,
	l orcaLattice,
) ([]maths.Vec, [][3]int) {
	values := l.sample(field)
	l.joinAll(values)
	points, cells := l.vertices(values)
	return orcaCompact(points, orcaDropTwins(l.faces(values, cells, points)))
}

// orcaJoinDepth is how far inside the surface a sample is put when it is
// joined to the inside, in metres. Chosen: any depth below nought will
// do, and a micrometre moves the surface by less than anything shows.
const orcaJoinDepth = 1e-6

// joinAll joins the lattice's alternating faces, pass after pass, until
// no face alternates. Putting a sample inside can make a face beside it
// alternate; each pass turns samples inside only, so the passes end.
func (l orcaLattice) joinAll(values []float32) {
	for again := true; again; {
		again = l.joined(values) > 0
	}
}

// joined puts inside the surface one sample of every lattice face whose
// corners are inside and outside by turns, as a thin tip leaves them,
// and reports how many faces it did.
//
// Such a face has two inside corners that meet only along its diagonal:
// the surface touches itself there, four quads share one edge between two
// cells, and the twins dropped for it leave a hole. Joining the inside
// across the face lets the surface pass round it once.
func (l orcaLattice) joined(values []float32) int {
	faces := 0
	for k := 0; k < l.nz; k++ {
		for j := 0; j < l.ny; j++ {
			for i := 0; i < l.nx; i++ {
				at := l.index(i, j, k)
				if i+1 < l.nx && j+1 < l.ny {
					faces += orcaJoinFace(values, at, 1, l.nx)
				}
				if i+1 < l.nx && k+1 < l.nz {
					faces += orcaJoinFace(values, at, 1, l.nx*l.ny)
				}
				if j+1 < l.ny && k+1 < l.nz {
					faces += orcaJoinFace(values, at, l.nx, l.nx*l.ny)
				}
			}
		}
	}
	return faces
}

// orcaJoinFace joins the face whose corners lie at sample at, a step
// across from it and a step up from it, if they alternate: it puts inside
// whichever of the two outside corners has the lower field, and reports
// one. Otherwise it changes nothing and reports nought.
func orcaJoinFace(values []float32, at, across, up int) int {
	a, b, c, d := at, at+across, at+up, at+across+up
	if (values[a] < 0) != (values[d] < 0) ||
		(values[b] < 0) != (values[c] < 0) ||
		(values[a] < 0) == (values[b] < 0) {
		return 0
	}
	first, second := a, d
	if values[a] < 0 {
		first, second = b, c
	}
	if values[second] < values[first] {
		first = second
	}
	values[first] = -orcaJoinDepth
	return 1
}

// orcaDropTwins removes every triangle that shares an edge, running the
// same way, with another. Where the surface is thinner than the lattice
// the contour can fold so; the fold is a pinhole, and leaving it would
// make the mesh refuse to build.
func orcaDropTwins(faces [][3]int) [][3]int {
	seen := make(map[[2]int]int, 3*len(faces))
	twin := make([]bool, len(faces))
	for n, f := range faces {
		for corner := 0; corner < 3; corner++ {
			edge := [2]int{f[corner], f[(corner+1)%3]}
			if other, again := seen[edge]; again {
				twin[n], twin[other] = true, true
				continue
			}
			seen[edge] = n
		}
	}
	kept := make([][3]int, 0, len(faces))
	for n, f := range faces {
		if !twin[n] {
			kept = append(kept, f)
		}
	}
	return kept
}

// orcaCompact renumbers the points so that only those a triangle uses
// remain, and returns them with the triangles renumbered to match.
func orcaCompact(points []maths.Vec, faces [][3]int) ([]maths.Vec, [][3]int) {
	renumber := make([]int, len(points))
	for i := range renumber {
		renumber[i] = -1
	}
	var used []maths.Vec
	for n, f := range faces {
		for corner, v := range f {
			if renumber[v] < 0 {
				renumber[v] = len(used)
				used = append(used, points[v])
			}
			faces[n][corner] = renumber[v]
		}
	}
	return used, faces
}
