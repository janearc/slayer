package manufactory_test

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// edge is one side of a face, from one corner to the next, the corners
// named by where they are.
type edge struct{ from, to maths.Vec }

// skin is a mesh's corners and faces, copied out of it and read by
// where its corners are, so that faces which share a corner share it
// whether or not the mesh says so. closed is what the mesh says of
// itself, which the reading checks and never trusts.
type skin struct {
	points []maths.Vec
	faces  [][3]int
	closed bool
}

// corners is a face's three corners, in its winding's order.
func (read skin) corners(face [3]int) [3]maths.Vec {
	return [3]maths.Vec{read.points[face[0]], read.points[face[1]],
		read.points[face[2]]}
}

// edges counts each side of every face, by the way it runs.
func (read skin) edges() map[edge]int {
	counted := map[edge]int{}
	for _, face := range read.faces {
		corners := read.corners(face)
		for i := range corners {
			counted[edge{corners[i], corners[(i+1)%3]}]++
		}
	}
	return counted
}

// unclosed is why a skin is not closed and wound one way, or nothing:
// a closed skin has every side run once each way, by the face either
// side of it.
func (read skin) unclosed() error {
	counted := read.edges()
	for side, times := range counted {
		back := counted[edge{side.to, side.from}]
		if times != 1 || back != 1 {
			return fmt.Errorf("the side %v to %v runs %d times that way "+
				"and %d back", side.from, side.to, times, back)
		}
	}
	return nil
}

// volume is the space a closed skin holds, by the divergence theorem: the sum
// over faces of the cone each makes with the origin. it is greater than zero
// when the faces face out.
func (read skin) volume() float64 {
	sum := 0.0
	for _, face := range read.faces {
		corners := read.corners(face)
		sum += corners[0].Dot(cross(corners[1], corners[2]))
	}
	return sum / 6
}

// cross is the vector square to two others, by the right hand.
func cross(one, other maths.Vec) maths.Vec {
	return maths.V(one.Y*other.Z-one.Z*other.Y,
		one.Z*other.X-one.X*other.Z, one.X*other.Y-one.Y*other.X)
}

// reversed is a skin with every face wound the other way.
func (read skin) reversed() skin {
	faces := make([][3]int, len(read.faces))
	for i, face := range read.faces {
		faces[i] = [3]int{face[0], face[2], face[1]}
	}
	return skin{points: read.points, faces: faces}
}
