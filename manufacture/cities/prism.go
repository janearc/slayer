package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
)

// prism is a closed solid joining one outline to another above it, edge
// by edge. The bottom stands on the origin's ground plane and the top at
// its height, so a prism of two equal outlines is straight and one of a
// wide and a narrow outline is a taper. Both outlines must have as many
// corners, and be convex: the caps are fans.
type prism struct {
	Bottom, Top outline
	Height      float64
}

// Mesh makes the prism's walls and caps, wound outward. It refuses a
// prism with no height, outlines of different corner counts, and an
// outline that is concave.
func (prism prism) Mesh(level Detail) (*Mesh, error) {
	if !(prism.Height > 0) {
		return nil, fmt.Errorf("cities: a prism needs a positive height, "+
			"not %v", prism.Height)
	}
	bottom, err := ring(prism.Bottom, level, 0)
	if err != nil {
		return nil, err
	}
	top, err := ring(prism.Top, level, prism.Height)
	if err != nil {
		return nil, err
	}
	if len(bottom) != len(top) {
		return nil, fmt.Errorf("cities: a prism joins corner to corner, "+
			"and its bottom has %d and its top %d", len(bottom), len(top))
	}
	mesh := &Mesh{Triangles: walls(bottom, top)}
	mesh.Triangles = append(mesh.Triangles,
		fill(bottom, maths.YAxis.Neg())...)
	mesh.Triangles = append(mesh.Triangles, fill(top, maths.YAxis)...)
	return mesh, nil
}

// ring is an outline's corners lifted to a height, or why they cannot
// make a prism's edge.
func ring(outline outline, level Detail, height float64) ([]maths.Vec, error) {
	corners, err := outline.Points(level)
	if err != nil {
		return nil, err
	}
	if !convex(corners) {
		return nil, fmt.Errorf("cities: a prism's cap is a fan, which an " +
			"outline that turns both ways is not filled by: it is concave")
	}
	lifted := make([]maths.Vec, len(corners))
	for index, corner := range corners {
		lifted[index] = maths.V(corner.X, height, corner.Z)
	}
	return lifted, nil
}

// walls joins each edge of the bottom ring to the same edge of the top
// as two triangles, facing away from the axis through the bottom ring's
// middle.
func walls(bottom, top []maths.Vec) []Triangle {
	axis := middle(bottom)
	var wall []Triangle
	for index := range bottom {
		next := (index + 1) % len(bottom)
		lowNear, lowFar := bottom[index], bottom[next]
		highFar, highNear := top[next], top[index]
		centre := middle([]maths.Vec{lowNear, lowFar, highFar, highNear})
		across := centre.Sub(axis)
		away := maths.V(across.X, 0, across.Z)
		wall = append(wall,
			outward(Triangle{lowNear, lowFar, highFar}, away),
			outward(Triangle{lowNear, highFar, highNear}, away))
	}
	return wall
}

// fill fills a ring as a fan from its first corner, facing the way given.
func fill(ring []maths.Vec, away maths.Vec) []Triangle {
	apex, rest := ring[0], ring[1:]
	var fan []Triangle
	for index := 0; index+1 < len(rest); index++ {
		fan = append(fan, outward(Triangle{apex, rest[index],
			rest[index+1]}, away))
	}
	return fan
}

// outward is the triangle wound so that its normal, by the cross of its
// edges, points the way given and not against it.
func outward(triangle Triangle, away maths.Vec) Triangle {
	first, second := triangle[1].Sub(triangle[0]), triangle[2].Sub(triangle[0])
	if first.Cross(second).Dot(away) < 0 {
		triangle[1], triangle[2] = triangle[2], triangle[1]
	}
	return triangle
}

// middle is the mean of some points.
func middle(points []maths.Vec) maths.Vec {
	var sum maths.Vec
	for _, point := range points {
		sum = sum.Add(point)
	}
	return sum.Scale(1 / float64(len(points)))
}
