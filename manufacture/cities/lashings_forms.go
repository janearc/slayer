package cities

import (
	"fmt"
	"math"
	"slices"

	"github.com/janearc/slayer/maths"
)

// The forms a lashing is hung with: a cable drawn through points, and a
// paper lantern.

// leastCablePoints is the fewest points a cable is drawn through: its two
// ends.
const leastCablePoints = 2

// cableCorners is how many corners a cable's section has: a square.
const cableCorners = 4

// leastRings is the fewest bands of latitude a lantern is cut into, so
// that even the hull level of a lantern is round enough to read as one.
const leastRings = 3

// verticalTangent is how far along the up axis a cable's tangent may run
// before its side is taken from another axis, as a share of its length.
const verticalTangent = 1 - 1e-6

// lashingCable is a cable of square section drawn through points in its holder's
// frame, the points that of a hanging cable, and so not a straight run.
// Radius is half the side of the section.
type lashingCable struct {
	Points []maths.Vec
	Radius float64
}

// Mesh makes the cable's four long faces between each pair of rings of
// corners, and its two ends, wound outward. It refuses a cable of fewer
// than two points, one of no radius, and one that doubles back on a point.
func (cable lashingCable) Mesh(Detail) (*Mesh, error) {
	if len(cable.Points) < leastCablePoints || !(cable.Radius > 0) {
		return nil, fmt.Errorf("cities: a cable needs at least %d points "+
			"and a positive radius, not %d and %v", leastCablePoints,
			len(cable.Points), cable.Radius)
	}
	rings := make([][]maths.Vec, len(cable.Points))
	for index := range rings {
		ring, err := cable.ring(index)
		if err != nil {
			return nil, err
		}
		rings[index] = ring
	}
	mesh := &Mesh{}
	for index := 0; index+1 < len(rings); index++ {
		mesh.Triangles = append(mesh.Triangles,
			links(rings[index], rings[index+1])...)
	}
	last := len(rings) - 1
	mesh.Triangles = append(mesh.Triangles,
		fill(rings[0], cable.Points[0].Sub(cable.Points[1]))...)
	mesh.Triangles = append(mesh.Triangles, fill(rings[last],
		cable.Points[last].Sub(cable.Points[last-1]))...)
	return mesh, nil
}

// ring is the four corners of the cable's section at one of its points,
// square to the way the cable runs there.
func (cable lashingCable) ring(index int) ([]maths.Vec, error) {
	before, after := max(index-1, 0), min(index+1, len(cable.Points)-1)
	run := cable.Points[after].Sub(cable.Points[before])
	if !(run.Len() > 0) {
		return nil, fmt.Errorf("cities: a cable doubles back on its point "+
			"%d, which is at %v", index, cable.Points[index])
	}
	tangent := run.Unit()
	across := maths.YAxis
	if math.Abs(tangent.Dot(maths.YAxis)) > verticalTangent {
		across = maths.XAxis
	}
	side := tangent.Cross(across).Unit()
	lift := side.Cross(tangent).Unit()
	at := cable.Points[index]
	corner := func(toSide, toLift float64) maths.Vec {
		return at.Add(side.Scale(toSide * cable.Radius)).Add(
			lift.Scale(toLift * cable.Radius))
	}
	return []maths.Vec{corner(1, 1), corner(-1, 1), corner(-1, -1),
		corner(1, -1)}, nil
}

// links is the four faces between one ring of a cable and the next, two
// triangles each, facing away from the cable's middle there.
func links(near, far []maths.Vec) []Triangle {
	inside := middle(slices.Concat(near, far))
	var faces []Triangle
	for index := range cableCorners {
		next := (index + 1) % cableCorners
		quad := []maths.Vec{near[index], near[next], far[next], far[index]}
		away := middle(quad).Sub(inside)
		faces = append(faces,
			outward(Triangle{quad[0], quad[1], quad[2]}, away),
			outward(Triangle{quad[0], quad[2], quad[3]}, away))
	}
	return faces
}

// lashingLantern is a round paper lantern: a spheroid about its own origin, as
// wide as its radius across and as tall as its stretch times that.
type lashingLantern struct{ Radius, Stretch float64 }

// Mesh makes the lantern as bands of latitude, as many corners round as
// the detail level cuts a circle into, wound outward. It refuses a
// lantern with no radius or no stretch.
func (lantern lashingLantern) Mesh(level Detail) (*Mesh, error) {
	if !(lantern.Radius > 0 && lantern.Stretch > 0) {
		return nil, fmt.Errorf("cities: a lantern needs a positive radius "+
			"and stretch, not %v", lantern)
	}
	sides, err := sidesAt(level)
	if err != nil {
		return nil, err
	}
	return bandMesh(max(sides/2, leastRings), sides, lantern.latitude), nil
}

// bandMesh is a closed surface cut into bands of latitude, each ring of
// it given by a function of its number from the top, the rings there are
// and the corners round, wound outward from the origin. A ring that is a
// point at a pole makes no face of its own.
func bandMesh(rings, sides int,
	latitude func(band, rings, sides int) []maths.Vec) *Mesh {
	mesh := &Mesh{}
	for band := range rings {
		upper, lower := latitude(band, rings, sides),
			latitude(band+1, rings, sides)
		for index := range sides {
			next := (index + 1) % sides
			for _, face := range []Triangle{
				{upper[index], lower[index], lower[next]},
				{upper[index], lower[next], upper[next]}} {
				if triangleHolds(face) {
					mesh.Triangles = append(mesh.Triangles,
						outward(face, middle(face[:])))
				}
			}
		}
	}
	return mesh
}

// latitude is the corners of the circle a number of bands down from the
// top of the lantern, of the rings it has in all. At the top and at the
// bottom the circle is a point, repeated for every corner.
func (lantern lashingLantern) latitude(band, rings, sides int) []maths.Vec {
	angle := math.Pi * float64(band) / float64(rings)
	round, height := lantern.Radius*math.Sin(angle),
		lantern.Radius*lantern.Stretch*math.Cos(angle)
	if band == rings {
		round = 0
	}
	corners := make([]maths.Vec, sides)
	for index := range corners {
		turn := fullTurn * float64(index) / float64(sides)
		corners[index] = maths.V(round*math.Cos(turn), height,
			round*math.Sin(turn))
	}
	return corners
}

// triangleHolds is whether a triangle has an area: two of its corners
// that are the same one, as at a pole, make none. The corners are compared
// themselves, as a cross of equal edges is not nought to the last digit on
// a machine that fuses multiplies with adds.
func triangleHolds(face Triangle) bool {
	return face[0] != face[1] && face[1] != face[2] && face[0] != face[2]
}
