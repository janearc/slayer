package typography

import "math"

// point is a place in the plane of a glyph, in the font's units with y
// up, the way the letter is read.
type point struct{ x, y float64 }

// add is the two points summed, part by part.
func (a point) add(b point) point { return point{a.x + b.x, a.y + b.y} }

// sub is the way from b to a.
func (a point) sub(b point) point { return point{a.x - b.x, a.y - b.y} }

// scale is the point s times as far from zero.
func (a point) scale(s float64) point { return point{a.x * s, a.y * s} }

// cross is how far b turns counterclockwise from a, times both their lengths:
// positive for a left turn, negative for a right, zero when they lie on one
// line.
func (a point) cross(b point) float64 { return a.x*b.y - a.y*b.x }

// dot is how far a and b go the same way, times both their lengths.
func (a point) dot(b point) float64 { return a.x*b.x + a.y*b.y }

// length is how far the point is from zero.
func (a point) length() float64 { return math.Hypot(a.x, a.y) }

// right is the way a quarter turn clockwise from a, one unit long: out
// of a ring that turns counterclockwise, on its right.
func (a point) right() point {
	l := a.length()
	return point{a.y / l, -a.x / l}
}

// ring is a closed path of straight steps, the last point stepping back
// to the first. counterclockwise, it encloses what is on its left.
type ring []point

// at is the ring's point i, counting round past either end.
func (r ring) at(i int) point { return r[((i%len(r))+len(r))%len(r)] }

// area is the ring's signed area by the shoelace: positive when it
// turns counterclockwise, negative when it turns clockwise.
func (r ring) area() float64 {
	sum := 0.0
	for i, a := range r {
		b := r.at(i + 1)
		sum += a.x*b.y - b.x*a.y
	}
	return sum / 2
}

// reversed is the ring the other way round, from the same first point.
func (r ring) reversed() ring {
	turned := make(ring, len(r))
	for i := range r {
		turned[i] = r.at(-i)
	}
	return turned
}

// holds is whether p is inside the ring, by counting how many of its
// steps a ray from p toward plus x crosses: odd is inside. a point on
// the ring may come out either way.
func (r ring) holds(p point) bool {
	inside := false
	for i, a := range r {
		b := r.at(i + 1)
		if (a.y > p.y) == (b.y > p.y) {
			continue
		}
		x := a.x + (p.y-a.y)/(b.y-a.y)*(b.x-a.x)
		if x > p.x {
			inside = !inside
		}
	}
	return inside
}

// touch is whether the steps from a to b and from c to d meet anywhere,
// ends included.
func touch(a, b, c, d point) bool {
	one, two := side(a, b, c), side(a, b, d)
	three, four := side(c, d, a), side(c, d, b)
	if one*two < 0 && three*four < 0 {
		return true
	}
	return one == 0 && within(a, b, c) || two == 0 && within(a, b, d) ||
		three == 0 && within(c, d, a) || four == 0 && within(c, d, b)
}

// side is which side of the line from a to b the point p is: one on its left,
// minus one on its right, zero on it.
func side(a, b, p point) float64 {
	turn := b.sub(a).cross(p.sub(a))
	switch {
	case turn > 0:
		return 1
	case turn < 0:
		return -1
	}
	return 0
}

// within is whether p, on the line through a and b, lies between them.
func within(a, b, p point) bool {
	return math.Min(a.x, b.x) <= p.x && p.x <= math.Max(a.x, b.x) &&
		math.Min(a.y, b.y) <= p.y && p.y <= math.Max(a.y, b.y)
}
