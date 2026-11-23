package stage

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// ringCorners is how many corners a cross-section has: a rectangle.
const ringCorners = 4

// twoNormals is how many unit normals a joint's bisector is made from.
// Their sum is this many times the cosine of half the turn long.
const twoNormals = 2

// leastHalfTurn is the least cosine of half a turn a joint may have: a
// right angle, which is Sqrt2 over two. A sharper joint throws its mitre
// out past every wall it joins, so the line is refused instead.
const leastHalfTurn = math.Sqrt2 / twoNormals

// tetrahedronShare is the share of the triple product of a face's three
// corners that is the volume of the tetrahedron they make with the origin.
const tetrahedronShare = 6.0

// plan is a place on the ground, as east and north in metres.
type plan = [2]float64

// section is a rectangle across a line: how far each side of it the
// rectangle reaches, the left of the line's way being positive, and how
// low and how high it stands, in metres.
type section struct {
	near, far, low, high float64
}

// place is a point of the ground at a height, in the world's space.
func place(at plan, up float64) maths.Vec {
	return maths.V(at[0], up, 0-at[1])
}

// direction is the unit way from one point to another.
func direction(from, to plan) (plan, error) {
	east, north := to[0]-from[0], to[1]-from[1]
	length := math.Hypot(east, north)
	if length == 0 {
		return plan{}, fmt.Errorf("a line stays at one point")
	}
	return plan{east / length, north / length}, nil
}

// directions are the unit ways of each segment of a line.
func directions(line []plan) ([]plan, error) {
	ways := make([]plan, len(line)-1)
	for index := range ways {
		way, err := direction(line[index], line[index+1])
		if err != nil {
			return nil, err
		}
		ways[index] = way
	}
	return ways, nil
}

// mitre is how far a joint's point moves for an offset of a distance, and
// how far the offset joint slides along the segments from where it was. It
// moves along the bisector of its two segments' left normals, so that both
// offset segments pass through it. It refuses a joint turned past a right
// angle.
func mitre(before, after plan, distance float64) (move plan, slide float64,
	err error) {
	sum := plan{0 - before[1] - after[1], before[0] + after[0]}
	length := math.Hypot(sum[0], sum[1])
	cosine := length / twoNormals
	if cosine < leastHalfTurn {
		return plan{}, 0, fmt.Errorf("a line turns by more than a right angle")
	}
	scale := distance / (cosine * length)
	slide = math.Abs(distance) * math.Sqrt(1-cosine*cosine) / cosine
	return plan{sum[0] * scale, sum[1] * scale}, slide, nil
}

// offsets is a line moved sideways by a distance, to the left of its way
// where the distance is above nought, every joint mitred. It refuses a
// line that bends so sharply, for the distance, that a joint slides past
// half of a segment beside it.
func offsets(line []plan, distance float64) ([]plan, error) {
	ways, err := directions(line)
	if err != nil {
		return nil, err
	}
	moved := make([]plan, len(line))
	for index, point := range line {
		before, after := max(index-1, 0), min(index, len(ways)-1)
		move, slide, err := mitre(ways[before], ways[after], distance)
		if err != nil {
			return nil, err
		}
		shortest := math.Min(segmentLength(line, before),
			segmentLength(line, after))
		if slide > shortest/twoNormals {
			return nil, fmt.Errorf("a line bends too sharply for an "+
				"offset of %g m at point %d", distance, index)
		}
		moved[index] = plan{point[0] + move[0], point[1] + move[1]}
	}
	return moved, nil
}

// segmentLength is the length of the segment of a line that starts at a
// point.
func segmentLength(line []plan, index int) float64 {
	return math.Hypot(line[index+1][0]-line[index][0],
		line[index+1][1]-line[index][1])
}

// sweep is the solid a section makes along a line: a ring of its four
// corners at each point of the line, mitred, joined by faces and closed by
// a face at each end.
func sweep(line []plan, across section) (*mesh.Mesh, error) {
	near, err := offsets(line, across.near)
	if err != nil {
		return nil, err
	}
	far, err := offsets(line, across.far)
	if err != nil {
		return nil, err
	}
	var corners []maths.Vec
	for index := range line {
		corners = append(corners,
			place(near[index], across.low), place(far[index], across.low),
			place(far[index], across.high), place(near[index], across.high))
	}
	faces := append(sides(len(line)), ends(len(line))...)
	if volume(corners, faces) < 0 {
		faces = turned(faces)
	}
	return mesh.New(corners, faces, nil, false)
}

// sides are the faces along a sweep of rings: each edge of each ring,
// joined to the same edge of the next ring by two triangles.
func sides(rings int) [][3]int {
	var faces [][3]int
	for ring := 0; ring+1 < rings; ring++ {
		for edge := range ringCorners {
			first := ring*ringCorners + edge
			second := ring*ringCorners + (edge+1)%ringCorners
			third := (ring+1)*ringCorners + (edge+1)%ringCorners
			fourth := (ring+1)*ringCorners + edge
			faces = append(faces, [3]int{first, second, third},
				[3]int{first, third, fourth})
		}
	}
	return faces
}

// ends are the faces that close a sweep: the first ring wound against the
// way of the sweep and the last one with it.
func ends(rings int) [][3]int {
	last := (rings - 1) * ringCorners
	return [][3]int{{0, 2, 1}, {0, 3, 2},
		{last, last + 1, last + 2}, {last, last + 2, last + 3}}
}

// volume is the signed volume a closed set of faces holds: above nought
// where the faces turn their counterclockwise side outward.
func volume(corners []maths.Vec, faces [][3]int) float64 {
	var total float64
	for _, face := range faces {
		first, second, third := corners[face[0]], corners[face[1]],
			corners[face[2]]
		total += first.Dot(second.Cross(third))
	}
	return total / tetrahedronShare
}

// turned is a set of faces each turned inside out.
func turned(faces [][3]int) [][3]int {
	back := make([][3]int, len(faces))
	for index, face := range faces {
		back[index] = [3]int{face[0], face[2], face[1]}
	}
	return back
}
