package typography

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
)

// shape is how a glyph's solids are made a mesh: scale is scene units a
// font unit, depth is front to back, bevel is how far the front and
// back are stepped in and back from the walls, and smooth is the angle
// in radians below which two walls meeting at a corner blend.
type shape struct {
	scale, depth, bevel, smooth float64
}

// builder is a mesh in the making: corners, a normal at each, and
// faces, each three corners.
type builder struct {
	points, normals []maths.Vec
	faces           [][3]int
}

// triangle adds a face of three corners and the normals at them.
func (made *builder) triangle(p, n [3]maths.Vec) {
	at := len(made.points)
	made.points = append(made.points, p[:]...)
	made.normals = append(made.normals, n[:]...)
	made.faces = append(made.faces, [3]int{at, at + 1, at + 2})
}

// quad adds four corners as two faces, from the first corner to each
// of the next two and the next two after.
func (made *builder) quad(p, n [4]maths.Vec) {
	made.triangle([3]maths.Vec{p[0], p[1], p[2]},
		[3]maths.Vec{n[0], n[1], n[2]})
	made.triangle([3]maths.Vec{p[0], p[2], p[3]},
		[3]maths.Vec{n[0], n[2], n[3]})
}

// at is a point of the glyph's plane at a depth, in scene units.
func (s shape) at(p point, z float64) maths.Vec {
	return maths.V(p.x*s.scale, p.y*s.scale, z)
}

// letter is a glyph's solids made one mesh, closed, every face turning
// counterclockwise seen from outside: the front facing plus z at half
// the depth, the back facing minus z, and walls between.
func (s shape) letter(solids []solid) (*mesh.Mesh, error) {
	var made builder
	for _, one := range solids {
		if err := s.solid(&made, one); err != nil {
			return nil, err
		}
	}
	return mesh.New(made.points, made.faces, made.normals, true)
}

// solid adds one solid's faces, walls and, if it has one, its bevel.
func (s shape) solid(made *builder, one solid) error {
	half, face := s.depth/2, one
	if s.bevel > 0 {
		in, to, err := inset(one, s.bevel/s.scale)
		if err != nil {
			return err
		}
		for k, r := range one.rings() {
			s.band(made, r, in.rings()[k], to[k], half)
		}
		face = in
	}
	cut, err := faces(face)
	if err != nil {
		return err
	}
	front, back := maths.V(0, 0, 1), maths.V(0, 0, -1)
	for _, t := range cut {
		made.triangle([3]maths.Vec{s.at(t[0], half), s.at(t[1], half),
			s.at(t[2], half)}, [3]maths.Vec{front, front, front})
		made.triangle([3]maths.Vec{s.at(t[2], -half), s.at(t[1], -half),
			s.at(t[0], -half)}, [3]maths.Vec{back, back, back})
	}
	for _, r := range one.rings() {
		s.walls(made, r, half-s.bevel)
	}
	return nil
}

// walls adds a wall along every step of the ring, from h in front to h
// behind, facing out of the letter.
func (s shape) walls(made *builder, r ring, h float64) {
	start, end := s.ends(r)
	for k, a := range r {
		b := r.at(k + 1)
		na, nb := maths.V(start[k].x, start[k].y, 0),
			maths.V(end[k].x, end[k].y, 0)
		made.quad([4]maths.Vec{s.at(a, h), s.at(a, -h), s.at(b, -h),
			s.at(b, h)}, [4]maths.Vec{na, na, nb, nb})
	}
}

// band adds the bevel between a ring and its inset, at the front and
// at the back: from the ring at h, where the walls end, to the inset at
// the full half depth. to is where in the inset each of the ring's
// points moved.
//
// a step whose line moved in is a slope of forty-five degrees, facing
// out and toward the front or the back. a step that vanished on the way
// in is a triangle up to the corner its two ends moved to, and slopes
// more steeply.
func (s shape) band(made *builder, r, in ring, to []int, half float64) {
	start, end := s.ends(r)
	h := half - s.bevel
	for k := range r {
		a, b := r[k], r.at(k+1)
		ia, ib := in[to[k]], in[to[(k+1)%len(r)]]
		for _, z := range [2]float64{1, -1} {
			na, nb := tilted(start[k], z), tilted(end[k], z)
			p := [4]maths.Vec{s.at(ia, z*half), s.at(a, z*h),
				s.at(b, z*h), s.at(ib, z*half)}
			if z < 0 {
				p = [4]maths.Vec{p[3], p[2], p[1], p[0]}
				na, nb = nb, na
			}
			if ia == ib {
				made.triangle([3]maths.Vec{p[0], p[1], p[2]},
					[3]maths.Vec{na, na, nb})
				continue
			}
			made.quad(p, [4]maths.Vec{na, na, nb, nb})
		}
	}
}

// tilted is a way out of the ring in the plane, tipped forty-five
// degrees toward the front for z one or the back for z minus one.
func tilted(out point, z float64) maths.Vec {
	return maths.V(out.x, out.y, z).Unit()
}

// ends is the normal at each end of each step of the ring, in the
// plane: the step's own way out where the corner it meets is sharper
// than the smoothing angle, and the two steps' ways out blended where
// it is gentler.
//
// so a curve flattened into steps shades as a curve, and a corner
// stays a corner.
func (s shape) ends(r ring) (start, end []point) {
	n := len(r)
	start, end = make([]point, n), make([]point, n)
	for k := range r {
		before, after := r.at(k).sub(r.at(k-1)).right(),
			r.at(k+1).sub(r.at(k)).right()
		start[k], end[(k+n-1)%n] = after, before
		if before.dot(after) >= math.Cos(s.smooth) {
			mid := before.add(after)
			mid = mid.scale(1 / mid.length())
			start[k], end[(k+n-1)%n] = mid, mid
		}
	}
	return start, end
}
