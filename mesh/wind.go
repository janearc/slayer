package mesh

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// edge is a way along a face's side, from one corner to the next.
// corners are matched by place, so faces that meet at the same points
// share an edge whether or not they number the corners alike.
type edge struct{ from, to maths.Vec }

// back is the edge the other way.
func (side edge) back() edge { return edge{from: side.to, to: side.from} }

// sides is a face's three edges, in its turn.
func sides(points []maths.Vec, at face) [3]edge {
	one, two, three := points[at[0]], points[at[1]], points[at[2]]
	return [3]edge{{from: one, to: two}, {from: two, to: three},
		{from: three, to: one}}
}

// runners is an empty table of edges to the face that runs each, for
// so many faces.
func runners(faces int) map[edge]int { return make(map[edge]int, 3*faces) }

// piece is faces joined by the edges they share, and whether every edge
// of it is shared, so that it has an inside.
type piece struct {
	faces  []int
	closed bool
}

// wound is whether the mesh is closed, or why it cannot be made: an
// edge two faces run the same way, or a closed piece whose faces turn
// it inside out.
func wound(points []maths.Vec, faces []face) (bool, error) {
	runs := runners(len(faces))
	for i, at := range faces {
		if err := running(runs, points, at, i); err != nil {
			return false, err
		}
	}
	closed := true
	for _, each := range piecesOf(points, faces, runs) {
		closed = closed && each.closed
		if each.closed && volume(points, faces, each.faces) < 0 {
			return false, insideOut(each.faces[0])
		}
	}
	return closed, nil
}

// running puts face i's edges in the table, or is the error of an edge
// another face already runs the same way.
func running(runs map[edge]int, points []maths.Vec, at face, i int) error {
	for _, side := range sides(points, at) {
		if other, seen := runs[side]; seen {
			return fmt.Errorf("%w: faces %d and %d run one edge "+
				"one way, %v to %v", ErrMesh, other, i,
				side.from, side.to)
		}
		runs[side] = i
	}
	return nil
}

// insideOut is the error of a closed piece whose faces turn inward.
func insideOut(face int) error {
	return fmt.Errorf("%w: the closed piece with face %d is inside out",
		ErrMesh, face)
}

// piecesOf is the faces gathered into pieces by the edges they share,
// each piece in the order of its first face.
func piecesOf(points []maths.Vec, faces []face, runs map[edge]int) []piece {
	joins := newJoins(len(faces))
	open := make([]bool, len(faces))
	for i, at := range faces {
		for _, side := range sides(points, at) {
			other, shared := runs[side.back()]
			if !shared {
				open[i] = true
				continue
			}
			joins.join(i, other)
		}
	}
	which := make(map[int]int, 1)
	var made []piece
	for i := range faces {
		root := joins.root(i)
		at, seen := which[root]
		if !seen {
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
	origin := points[faces[piece[0]][0]]
	sum := 0.0
	for _, i := range piece {
		at := faces[i]
		one := points[at[0]].Sub(origin)
		two := points[at[1]].Sub(origin)
		three := points[at[2]].Sub(origin)
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
