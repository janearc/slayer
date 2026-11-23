package mesh

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// edge is a way along a face's side, from one corner to the next.
// corners are matched by place, so faces that meet at the same points
// share an edge whether or not they number the corners alike.
type edge struct{ from, to maths.Vec }

// back is the edge the other way.
func (side edge) back() edge { return edge{from: side.to, to: side.from} }

// sides is a face's three edges, in its turn.
func sides(points []maths.Vec, at face) [triangleCorners]edge {
	one, two, three := points[at[first]], points[at[second]],
		points[at[third]]
	return [triangleCorners]edge{{from: one, to: two},
		{from: two, to: three}, {from: three, to: one}}
}

// place is the number of a point among the distinct points a mesh's
// faces name: corners equal in every part, to the bit, share one place,
// and so do a nought and a negative nought, which are equal.
type place = int32

// places is each face's three corners as places, in the face's turn.
type places [][triangleCorners]place

// edgeKey is the way along an edge from one place to the next, packed
// into one number: the place it leaves in the high half, the one it
// reaches in the low half.
type edgeKey = uint64

// halfWord is how far the high place of an edge key is shifted: a place
// is an int32, half the key.
const halfWord = 32 // defined by int32: its width in bits

// keyOf is the key of the edge from one place to another.
func keyOf(from, to place) edgeKey {
	return uint64(uint32(from))<<halfWord | uint64(uint32(to))
}

// unplaced marks a point of the mesh that no face has been seen to name.
const unplaced place = -1 // defined: below every place, which start at nought

// bitsOf is a point's parts as the bits that tell points apart: a
// negative nought is a nought, as it is when points are compared.
func bitsOf(point maths.Vec) [axes]uint64 {
	var bits [axes]uint64
	for part, value := range [axes]float64{point.X, point.Y, point.Z} {
		if value != nought {
			bits[part] = math.Float64bits(value)
		}
	}
	return bits
}

// placesOf is every corner of every face as a place. Each point a face
// names is looked up once, by its bits; points no face names are not
// looked at.
func placesOf(points []maths.Vec, faces []face) places {
	known := make(map[[axes]uint64]place, len(points))
	named := make([]place, len(points))
	for i := range named {
		named[i] = unplaced
	}
	made := make(places, len(faces))
	for i, at := range faces {
		for corner, index := range at {
			if named[index] == unplaced {
				bits := bitsOf(points[index])
				found, seen := known[bits]
				if !seen {
					found = place(len(known))
					known[bits] = found
				}
				named[index] = found
			}
			made[i][corner] = named[index]
		}
	}
	return made
}

// runners is an empty table of edges to the face that runs each, for
// so many faces.
func runners(faces int) map[edgeKey]int {
	return make(map[edgeKey]int, triangleCorners*faces)
}

// keysOf is a face's three edge keys, in its turn.
func keysOf(at [triangleCorners]place) [triangleCorners]edgeKey {
	return [triangleCorners]edgeKey{keyOf(at[first], at[second]),
		keyOf(at[second], at[third]), keyOf(at[third], at[first])}
}

// piece is faces joined by the edges they share, and whether every edge
// of it is shared, so that it has an inside.
type piece struct {
	faces  []int
	closed bool
}

// wound is whether the mesh is closed, or why it cannot be made: an
// edge two faces run the same way, an edge more than two faces share,
// or a closed piece whose faces turn it inside out.
func wound(points []maths.Vec, faces []face) (bool, error) {
	named := placesOf(points, faces)
	runs := runners(len(faces))
	for i := range faces {
		if err := running(runs, named, points, faces, i); err != nil {
			return false, err
		}
	}
	closed := true
	for _, each := range piecesOf(named, runs) {
		closed = closed && each.closed
		if each.closed && volume(points, faces, each.faces) < nought {
			return false, insideOut(each.faces[first])
		}
	}
	return closed, nil
}

// running puts face i's edges in the table, or is the error of an edge
// another face already runs the same way.
func running(runs map[edgeKey]int, named places, points []maths.Vec,
	faces []face, i int) error {
	for turn, key := range keysOf(named[i]) {
		if other, seen := runs[key]; seen {
			side := sides(points, faces[i])[turn]
			return refusal(points, faces, side, other, i)
		}
		runs[key] = i
	}
	return nil
}

// refusal is the error of face i running an edge, side, that face other
// runs the same way. If no more than the two faces are on the edge,
// whichever way they run it, they are wound against each other; if more
// are, the edge is shared by faces of more than one solid, or is a fin,
// and no winding of two faces mends it.
func refusal(points []maths.Vec, faces []face, side edge, other,
	i int) error {
	on := facesOn(points, faces, side)
	if on > pair {
		return fmt.Errorf("%w: %d faces run the edge %v to %v either way, "+
			"faces %d and %d the same way", errCrowded, on, side.from,
			side.to, other, i)
	}
	return fmt.Errorf("%w: faces %d and %d, %v to %v", errSameWay, other,
		i, side.from, side.to)
}

// facesOn is how many faces run an edge, either way.
func facesOn(points []maths.Vec, faces []face, side edge) int {
	var on int
	for _, at := range faces {
		for _, each := range sides(points, at) {
			if each == side || each == side.back() {
				on++
			}
		}
	}
	return on
}

// insideOut is the error of a closed piece whose faces turn inward.
func insideOut(face int) error {
	return fmt.Errorf("%w: the piece with face %d", errInsideOut, face)
}

// piecesOf is the faces gathered into pieces by the edges they share,
// each piece in the order of its first face.
func piecesOf(named places, runs map[edgeKey]int) []piece {
	joins := newJoins(len(named))
	open := make([]bool, len(named))
	for i, at := range named {
		for turn := range triangleCorners {
			back := keyOf(at[(turn+unity)%triangleCorners], at[turn])
			other, shared := runs[back]
			if !shared {
				open[i] = true
				continue
			}
			joins.join(i, other)
		}
	}
	which := make([]int, len(named))
	for i := range which {
		which[i] = nought - unity
	}
	var made []piece
	for i := range named {
		root := joins.root(i)
		at := which[root]
		if at < nought {
			at = len(made)
			which[root] = at
			made = append(made, piece{closed: true})
		}
		made[at].faces = append(made[at].faces, i)
		made[at].closed = made[at].closed && !open[i]
	}
	return made
}

// volume is six times the space a closed piece holds, by the signed volumes of
// the tetrahedra its faces make with its first corner: greater than zero when
// its faces turn out.
func volume(points []maths.Vec, faces []face, piece []int) float64 {
	origin := points[faces[piece[first]][first]]
	var sum float64
	for _, i := range piece {
		at := faces[i]
		one := points[at[first]].Sub(origin)
		two := points[at[second]].Sub(origin)
		three := points[at[third]].Sub(origin)
		sum += one.Dot(two.Cross(three))
	}
	return sum
}

// joins is which faces are joined, as a forest of parents: two faces
// are joined when they have the same root.
type joins struct{ parents []int }

// newJoins is so many faces, none joined yet.
func newJoins(faces int) joins {
	made := joins{parents: make([]int, faces)}
	for i := range made.parents {
		made.parents[i] = i
	}
	return made
}

// root is the face that stands for every face joined to face i.
func (joined joins) root(i int) int {
	for joined.parents[i] != i {
		joined.parents[i] = joined.parents[joined.parents[i]]
		i = joined.parents[i]
	}
	return i
}

// join joins two faces and everything joined to each.
func (joined joins) join(one, other int) {
	joined.parents[joined.root(one)] = joined.root(other)
}
