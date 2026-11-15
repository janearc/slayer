package named

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

// The field drawn: a dipole's lines and the hoops of danger, each a thin
// tube of triangles, so the tracer and the GPU both draw them. A field
// line of a dipole whose axis is y lies in a plane through the axis, at
// distance L sin^2(theta) from the middle at colatitude theta, L being
// how far out it crosses the equator; it leaves the star where that
// distance is the star's radius.

// ErrLines is a field line or hoop that cannot be drawn.
var ErrLines = errors.New("named: the magnetar's lines")

// the tubes' make: the sides round a tube and the steps along a line or
// round a hoop, picked by the author of this scene, 2026-11-12, as the
// fewest that look round and smooth at a frame's width
const (
	tubeSides = 6
	lineSteps = 96
	hoopSteps = 256
)

// the least a line and a tube may be made of: two points make a line,
// three sides a tube, each defined
const (
	fewestPoints = 2
	fewestSides  = 3
)

// tube is a polyline drawn as a hollow tube of thick radius, its
// vertices and faces added to a mesh's.
type tube struct {
	vertices []maths.Vec
	faces    [][3]int
}

// add puts a tube along points into the mesh: a ring of sides vertices
// round each point, square to the line there, joined ring to ring; a
// closed line joins its last ring to its first.
func (made *tube) add(points []maths.Vec, thick float64, closed bool) error {
	if len(points) < fewestPoints || !(thick > 0) {
		return fmt.Errorf("%w: %d points %g thick; a line needs two "+
			"points and some thickness", ErrLines, len(points), thick)
	}
	first := len(made.vertices)
	for i, at := range points {
		along := heading(points, i, closed)
		across := maths.Upright(along)
		for side := range tubeSides {
			turn := 2 * math.Pi * float64(side) / tubeSides
			out := across.Out(maths.V(math.Cos(turn), 0, math.Sin(turn)))
			made.vertices = append(made.vertices, at.Add(out.Scale(thick)))
		}
	}
	rings := len(points)
	if !closed {
		rings--
	}
	for ring := range rings {
		made.join(first, ring, (ring+1)%len(points))
	}
	return nil
}

// join is the faces between two rings of a tube that starts at first.
func (made *tube) join(first, ring, next int) {
	for side := range tubeSides {
		round := (side + 1) % tubeSides
		a, b := first+ring*tubeSides+side, first+ring*tubeSides+round
		c, d := first+next*tubeSides+side, first+next*tubeSides+round
		made.faces = append(made.faces, [3]int{a, c, b},
			[3]int{b, c, d})
	}
}

// heading is which way a line runs at its point i: from the point
// before to the point after, where there are both.
func heading(points []maths.Vec, i int, closed bool) maths.Vec {
	last := len(points) - 1
	before, after := max(i-1, 0), min(i+1, last)
	if closed {
		before, after = (i+last)%len(points), (i+1)%len(points)
	}
	return points[after].Sub(points[before]).Unit()
}

// mesh is the tubes made one mesh, or what is wrong.
func (made *tube) mesh() (*shape.Mesh, error) {
	return shape.NewMesh(made.vertices, made.faces, nil, false)
}

// fieldLine is the dipole line that crosses the magnetic equator apex
// out, in the plane at azimuth round the axis, from where it leaves the
// star of radius to where it comes back, in steps.
func fieldLine(radius, apex, azimuth float64) ([]maths.Vec, error) {
	if !(apex > radius) || !(radius > 0) {
		return nil, fmt.Errorf("%w: a line crossing the equator %g out "+
			"from a star of %g never leaves it; ask for one further out",
			ErrLines, apex, radius)
	}
	leaves := math.Asin(math.Sqrt(radius / apex))
	span := math.Pi - 2*leaves
	points := make([]maths.Vec, lineSteps+1)
	for i := range points {
		theta := leaves + span*float64(i)/lineSteps
		out := apex * math.Pow(math.Sin(theta), 2)
		points[i] = maths.V(out*math.Sin(theta)*math.Cos(azimuth),
			out*math.Cos(theta), out*math.Sin(theta)*math.Sin(azimuth))
	}
	return points, nil
}

// circle is the circle radius out in the magnetic equator, in steps.
func circle(radius float64) []maths.Vec {
	points := make([]maths.Vec, hoopSteps)
	for i := range points {
		turn := 2 * math.Pi * float64(i) / hoopSteps
		points[i] = maths.V(radius*math.Cos(turn), 0,
			radius*math.Sin(turn))
	}
	return points
}
