package skeleton

import (
	"errors"

	"github.com/janearc/slayer/maths"
)

// A fin's web: the thin skin stretched between a fin's rays. Each ray is a
// line from its base to its tip, carried by the bone that ray is, so the
// web moves as its rays do. The web is thickest at its base and comes to
// nothing at its free edge and at its first and last rays, a closed solid
// whose two faces meet along the rim.

// webFewest is the fewest rays a web is stretched between: with two the
// web would have no ray inside it to hold its faces apart.
const webFewest = 3

var errWebRays = errors.New("skeleton: a web needs three rays or more " +
	"and a step or more along each")

// WebRay is one ray of a web: its base and its tip at rest, and the
// motion of the bone that carries it.
type WebRay struct {
	base, tip maths.Vec
	carry     Frame
}

// NewWebRay is the ray from base to tip, carried by carry.
func NewWebRay(base, tip maths.Vec, carry Frame) WebRay {
	return WebRay{base: base, tip: tip, carry: carry}
}

// Sheet is a mesh being made with the rest place of each of its points
// beside the posed one, so that paint can follow the motion.
type Sheet struct {
	points, rest []maths.Vec
	faces        [][3]int
}

// NewSheet is a sheet with nothing on it.
func NewSheet() *Sheet { return &Sheet{} }

// Points is the posed points.
func (sheet *Sheet) Points() []maths.Vec { return sheet.points }

// Rest is each point's place at rest.
func (sheet *Sheet) Rest() []maths.Vec { return sheet.rest }

// Faces is the triangles, wound outward.
func (sheet *Sheet) Faces() [][3]int { return sheet.faces }

// add puts a point on the sheet, at rest and as carry moves it, and is
// its place.
func (sheet *Sheet) add(rest maths.Vec, carry Frame) int {
	sheet.rest = append(sheet.rest, rest)
	sheet.points = append(sheet.points, carry.Apply(rest))
	return len(sheet.points) - 1
}

// webGrid is the places of a web's points: for each ray and each step
// along it, the point on its front face and the one on its back, the
// same place where the web has no thickness.
type webGrid [][][2]int

// Web adds the web between rays, in order, steps steps along each, thick
// thick at its base, its faces toward facing and away from it.
func (sheet *Sheet) Web(rays []WebRay, steps int, thick float64,
	facing maths.Vec) error {
	if len(rays) < webFewest || steps < 1 {
		return errWebRays
	}
	firstFace := len(sheet.faces)
	grid := sheet.webPoints(rays, steps, thick, facing.Unit())
	for i := 0; i+1 < len(rays); i++ {
		for j := 0; j < steps; j++ {
			sheet.webQuad(grid, i, j)
		}
		sheet.webBase(grid[i][0], grid[i+1][0])
	}
	sheet.faces = outward(sheet.points, sheet.faces, firstFace)
	return nil
}

// webPoints puts the web's points on the sheet and is their places.
func (sheet *Sheet) webPoints(rays []WebRay, steps int, thick float64,
	facing maths.Vec) webGrid {
	grid := make(webGrid, len(rays))
	last := len(rays) - 1
	for i, ray := range rays {
		grid[i] = make([][2]int, steps+1)
		for j := 0; j <= steps; j++ {
			share := float64(j) / float64(steps)
			middle := ray.base.Add(ray.tip.Sub(ray.base).Scale(share))
			half := thick * (1 - share) / 2
			if i == 0 || i == last || j == steps {
				at := sheet.add(middle, ray.carry)
				grid[i][j] = [2]int{at, at}
				continue
			}
			frontPoint := middle.Add(facing.Scale(half))
			front := sheet.add(frontPoint, ray.carry)
			backPoint := middle.Sub(facing.Scale(half))
			back := sheet.add(backPoint, ray.carry)
			grid[i][j] = [2]int{front, back}
		}
	}
	return grid
}

// webQuad adds the four triangles, two on each face, between rays i and
// i+1 at steps j and j+1.
func (sheet *Sheet) webQuad(grid webGrid, i, j int) {
	a, b := grid[i][j], grid[i+1][j]
	c, d := grid[i+1][j+1], grid[i][j+1]
	sheet.faced(a, b, c)
	sheet.faced(a, c, d)
}

// faced adds a triangle to the front face and its mirror to the back,
// unless all three corners lie where the web has no thickness: there
// the two would be one triangle wound both ways, a flap of nothing.
func (sheet *Sheet) faced(a, b, c [2]int) {
	if a[0] == a[1] && b[0] == b[1] && c[0] == c[1] {
		return
	}
	sheet.triangle(a[0], b[0], c[0])
	sheet.triangle(a[1], c[1], b[1])
}

// webBase closes the web along its base between two rays' first points.
func (sheet *Sheet) webBase(a, b [2]int) {
	sheet.triangle(a[0], a[1], b[1])
	sheet.triangle(a[0], b[1], b[0])
}

// triangle adds a triangle, unless two of its corners are one point.
func (sheet *Sheet) triangle(a, b, c int) {
	if a == b || b == c || a == c {
		return
	}
	sheet.faces = append(sheet.faces, [3]int{a, b, c})
}
