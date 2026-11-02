package typography

import (
	"errors"
	"math"
	"slices"
	"sort"
)

// errNoEar is a face that ear clipping could not finish: no corner left
// could be cut off without taking some of the outline with it.
var errNoEar = errors.New("no ear left to cut")

// faces is a solid cut into triangles, each turning counterclockwise,
// that cover its outer ring less its holes: the holes are joined to the
// outer by bridges, and the one ring that makes is cut by ear clipping,
// as eberly's "triangulation by ear clipping" (2002) does it.
func faces(one solid) ([][3]point, error) {
	return clipped(bridged(one))
}

// bridged is the solid's rings as one: each hole, rightmost first,
// joined to what is made so far by a bridge out from its rightmost
// point and back, so the ring runs round the hole on the way.
func bridged(one solid) ring {
	made := append(ring(nil), one.outer...)
	holes := slices.Clone(one.holes)
	sort.SliceStable(holes, func(a, b int) bool {
		return holes[a][rightmost(holes[a])].x >
			holes[b][rightmost(holes[b])].x
	})
	for _, hole := range holes {
		made = bridge(made, hole)
	}
	return made
}

// rightmost is the place in the ring of its point furthest toward plus
// x, the first of them when several are.
func rightmost(r ring) int {
	best := 0
	for i, p := range r {
		if p.x > r[best].x {
			best = i
		}
	}
	return best
}

// bridge is the ring with the hole spliced in: from the point that the
// hole's rightmost can see, across to it, round the hole, and back.
func bridge(made, hole ring) ring {
	start := rightmost(hole)
	to := seen(made, hole[start])
	joined := make(ring, 0, len(made)+len(hole)+2)
	joined = append(joined, made[:to+1]...)
	for k := range len(hole) + 1 {
		joined = append(joined, hole.at(start+k))
	}
	return append(joined, made[to:]...)
}

// seen is the place in the ring of a point that m sees, so a bridge to
// it crosses nothing: the nearer end of the first step a ray from m
// toward plus x meets, unless a corner that turns inward stands in the
// triangle between, when the one nearest the ray's way is taken.
func seen(made ring, m point) int {
	step, along := -1, math.Inf(1)
	for i, a := range made {
		b := made.at(i + 1)
		if math.Min(a.y, b.y) > m.y || math.Max(a.y, b.y) < m.y {
			continue
		}
		x := meets(a, b, m.y)
		if x >= m.x && x < along {
			step, along = i, x
		}
	}
	if step < 0 {
		return 0 // nothing to its right: m is outside the ring
	}
	hit, end := point{along, m.y}, step
	if made.at(step+1).x > made[step].x {
		end = (step + 1) % len(made)
	}
	switch hit {
	case made[step]:
		return opening(made, step, m)
	case made.at(step + 1):
		return opening(made, (step+1)%len(made), m)
	}
	return opening(made, inward(made, m, hit, end), m)
}

// meets is where the step from a to b is at height y, which it reaches:
// an end exactly when the end is at that height, the nearer end toward
// minus x of a step that runs along it, and otherwise between.
func meets(a, b point, y float64) float64 {
	switch {
	case a.y == y && b.y == y:
		return math.Min(a.x, b.x)
	case a.y == y:
		return a.x
	case b.y == y:
		return b.x
	}
	return a.x + (y-a.y)/(b.y-a.y)*(b.x-a.x)
}

// inward is the corner of the ring that turns inward and stands in the
// triangle from m to hit to the corner at end, nearest in angle to the
// ray from m toward plus x, and nearest m of those; or end, if none.
func inward(made ring, m, hit point, end int) int {
	best, angle, near := end, math.Inf(1), math.Inf(1)
	corner := made[end]
	for i, p := range made {
		in := inside(m, hit, corner, p) || inside(m, corner, hit, p)
		if i == end || p == corner || !reflex(made, i) || !in {
			continue
		}
		way := p.sub(m)
		turn := math.Abs(math.Atan2(way.y, way.x))
		if turn < angle || turn == angle && way.length() < near {
			best, angle, near = i, turn, way.length()
		}
	}
	return best
}

// reflex is whether the ring turns clockwise at its point i: inward,
// for a ring that turns counterclockwise.
func reflex(r ring, i int) bool {
	return r.at(i).sub(r.at(i-1)).cross(r.at(i+1).sub(r.at(i))) < 0
}

// opening is, of the places in the ring with the same point as place i,
// the one whose corner opens toward m: a ring already bridged passes
// through a bridge's end twice, and only one of the two sees m.
func opening(made ring, i int, m point) int {
	for j, p := range made {
		if p == made[i] && opens(made.at(j-1), p, made.at(j+1), m) {
			return j
		}
	}
	return i
}

// opens is whether m is inside the corner at p, between the step in
// from a and the step out to c, for a ring that turns counterclockwise.
func opens(a, p, c, m point) bool {
	left := side(a, p, m) > 0
	right := side(p, c, m) > 0
	if p.sub(a).cross(c.sub(p)) >= 0 {
		return left && right
	}
	return left || right
}

// inside is whether p is in the counterclockwise triangle a, b, c, or
// on its edge.
func inside(a, b, c, p point) bool {
	return b.sub(a).cross(p.sub(a)) >= 0 && c.sub(b).cross(p.sub(b)) >= 0 &&
		a.sub(c).cross(p.sub(c)) >= 0
}

// clipped is a ring, which may pass through a point twice where it is
// bridged, cut into triangles by taking off one ear at a time: a corner
// that turns counterclockwise, whose triangle holds no other point of
// what is left. the search for the next ear starts where the last was.
func clipped(r ring) ([][3]point, error) {
	left := make([]int, len(r))
	for i := range left {
		left[i] = i
	}
	var made [][3]point
	from := 0
	for len(left) > 3 {
		cut := earAt(r, left, from)
		if cut < 0 {
			return nil, errNoEar
		}
		made = append(made, corner(r, left, cut))
		left = slices.Delete(left, cut, cut+1)
		from = cut % len(left)
	}
	last := corner(r, left, 1)
	if last[1].sub(last[0]).cross(last[2].sub(last[1])) <= 0 {
		return nil, errNoEar
	}
	return append(made, last), nil
}

// corner is the triangle at place k of what is left: the point there
// and its two neighbours, in the ring's turn.
func corner(r ring, left []int, k int) [3]point {
	n := len(left)
	return [3]point{r[left[(k+n-1)%n]], r[left[k]], r[left[(k+1)%n]]}
}

// earAt is the first place from from, round, that is an ear, or minus
// one if there is none.
func earAt(r ring, left []int, from int) int {
	for k := range left {
		at := (from + k) % len(left)
		if ear(r, left, at) {
			return at
		}
	}
	return -1
}

// ear is whether the corner at place k of what is left can be cut off.
func ear(r ring, left []int, k int) bool {
	tri := corner(r, left, k)
	a, b, c := tri[0], tri[1], tri[2]
	if b.sub(a).cross(c.sub(b)) <= 0 {
		return false
	}
	for _, i := range left {
		p := r[i]
		if p != a && p != b && p != c && inside(a, b, c, p) {
			return false
		}
	}
	return true
}
