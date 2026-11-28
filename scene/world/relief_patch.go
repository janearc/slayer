package world

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// ReliefFace names one side of the cube used to address a spherical
// terrain patch. These are body-space axes, not camera directions.
type ReliefFace uint8

const (
	ReliefEast ReliefFace = iota
	ReliefWest
	ReliefNorth
	ReliefSouth
	ReliefFront
	ReliefBack
	reliefFaces
)

// reliefDeepest bounds integer coordinates and work requested by callers.
// It does not promise that a mesh at that depth will fit in memory.
const reliefDeepest = 16

// ReliefPatch is an address in one face's quadtree. It does not contain
// geometry, a visibility decision, or an error estimate.
type ReliefPatch struct {
	face  ReliefFace
	level uint8
	x, y  int
}

// NewReliefPatch makes a patch address. Level zero has one patch per face;
// each level doubles the number of patches along each axis.
func NewReliefPatch(face ReliefFace, level, x, y int) (ReliefPatch, error) {
	if face >= reliefFaces || level < 0 || level > reliefDeepest {
		return ReliefPatch{}, fmt.Errorf("world: invalid relief face or level")
	}
	width := 1 << level
	if x < 0 || x >= width || y < 0 || y >= width {
		return ReliefPatch{}, fmt.Errorf("world: relief patch lies outside its face")
	}
	return ReliefPatch{face: face, level: uint8(level), x: x, y: y}, nil
}

// Address gives the patch's face, depth and face-local grid coordinates.
func (p ReliefPatch) Address() (ReliefFace, int, int, int) {
	return p.face, int(p.level), p.x, p.y
}

// Children returns the four addressed quarters in row-major order. It
// refuses at the depth limit; it does not choose which quarters are visible.
func (p ReliefPatch) Children() ([4]ReliefPatch, error) {
	if int(p.level) >= reliefDeepest {
		return [4]ReliefPatch{}, fmt.Errorf("world: relief patch is at its deepest level")
	}
	next := p.level + 1
	x, y := p.x*2, p.y*2
	return [4]ReliefPatch{{p.face, next, x, y}, {p.face, next, x + 1, y},
		{p.face, next, x, y + 1}, {p.face, next, x + 1, y + 1}}, nil
}

// ReliefVertex is an integer cube direction reduced to lowest terms. The
// same cube edge point from either adjacent face or a finer patch has the
// same key; applying Direction projects that point onto the unit sphere.
type ReliefVertex struct{ X, Y, Z int }

// Direction projects a canonical cube vertex onto the unit sphere.
func (v ReliefVertex) Direction() (maths.Vec, error) {
	if v.X == 0 && v.Y == 0 && v.Z == 0 {
		return maths.Vec{}, fmt.Errorf("world: a relief vertex needs a direction")
	}
	return maths.V(float64(v.X), float64(v.Y), float64(v.Z)).Unit(), nil
}

// Corners returns the patch's four keyed corners, row-major. Corner order
// is not a triangle winding: a mesh builder must choose outward winding.
func (p ReliefPatch) Corners() [4]ReliefVertex {
	width := 1 << p.level
	return [4]ReliefVertex{reliefVertex(p.face, width, p.x, p.y),
		reliefVertex(p.face, width, p.x+1, p.y),
		reliefVertex(p.face, width, p.x, p.y+1),
		reliefVertex(p.face, width, p.x+1, p.y+1)}
}

func reliefVertex(face ReliefFace, width, x, y int) ReliefVertex {
	u, v := 2*x-width, 2*y-width
	var made ReliefVertex
	switch face {
	case ReliefEast:
		made = ReliefVertex{width, u, v}
	case ReliefWest:
		made = ReliefVertex{-width, u, v}
	case ReliefNorth:
		made = ReliefVertex{u, width, v}
	case ReliefSouth:
		made = ReliefVertex{u, -width, v}
	case ReliefFront:
		made = ReliefVertex{u, v, width}
	case ReliefBack:
		made = ReliefVertex{u, v, -width}
	}
	g := reliefGCD(reliefGCD(made.X, made.Y), made.Z)
	return ReliefVertex{made.X / g, made.Y / g, made.Z / g}
}

func reliefGCD(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// Point evaluates one vertex of the fixed relief in metres from the body
// centre. If sea is true, heights below sea level are clamped to it; this
// is a choice of surface, not a separate ocean or sea-bed representation.
func (relief *Relief) Point(vertex ReliefVertex, sea bool) (maths.Vec, error) {
	direction, err := vertex.Direction()
	if err != nil {
		return maths.Vec{}, err
	}
	height, err := relief.Height(direction)
	if err != nil {
		return maths.Vec{}, err
	}
	if sea {
		height = math.Max(0, height)
	}
	radius := relief.radius + height
	if !(radius > 0) || math.IsInf(radius, 0) {
		return maths.Vec{}, fmt.Errorf("world: relief vertex reaches the planet's centre or beyond finite space")
	}
	return direction.Scale(radius), nil
}
