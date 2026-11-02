package typography

import (
	"errors"
	"math"
	"slices"
	"sort"
)

// solid is one piece of a letter: an outer ring turning counterclockwise
// and the holes straight inside it, each turning clockwise, so that the
// letter is always on a ring's left whichever way the font wound it.
type solid struct {
	outer ring
	holes []ring
}

// rings is every ring in the solid, the outer first.
func (one solid) rings() []ring {
	return append([]ring{one.outer}, one.holes...)
}

// errCrossed is a glyph whose contours cross or touch, which no solid
// can be made of.
var errCrossed = errors.New("its contours cross or touch")

// cleaned is a ring without the points that add nothing to it: one at
// the same place as the point before it, and one on the line between
// its neighbours, to a part in a million million.
//
// they go one at a time, each against the neighbours left to it, so of
// two points at one place one stays. a ring left with fewer than three
// points is nil.
func cleaned(r ring) ring {
	r = slices.Clone(r)
	for changed := true; changed && len(r) >= 3; {
		changed = false
		for i := 0; i < len(r) && len(r) >= 3; i++ {
			if idle(r.at(i-1), r[i], r.at(i+1)) {
				r, changed = slices.Delete(r, i, i+1), true
				i--
			}
		}
	}
	if len(r) < 3 {
		return nil
	}
	return r
}

// idle is whether p, between a and c, turns the ring by nothing: it is
// at a's place, or on the line through a and c to rounding, which is
// also true of a spike that goes out and comes back.
func idle(a, p, c point) bool {
	in, out := p.sub(a), c.sub(p)
	if in == (point{}) || out == (point{}) {
		return true
	}
	return math.Abs(in.cross(out)) <= 1e-12*in.length()*out.length()
}

// nested sorts a glyph's rings into solids by what holds what: a ring
// inside an even number of others is an outer, and one inside an odd
// number is a hole in the smallest ring that holds it.
//
// the winding the font gave them is not asked, so truetype's clockwise
// outers and cff's counterclockwise ones come out alike.
func nested(rings []ring) []solid {
	parent, depth := holders(rings)
	order := make([]int, len(rings))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return depth[order[a]] < depth[order[b]]
	})
	solids, of := []solid{}, map[int]int{}
	for _, i := range order {
		if depth[i]%2 == 0 {
			of[i] = len(solids)
			outer := turned(rings[i], 1)
			solids = append(solids, solid{outer: outer})
			continue
		}
		in := &solids[of[parent[i]]]
		in.holes = append(in.holes, turned(rings[i], -1))
	}
	return solids
}

// holders is, for each ring, the smallest other ring that holds it, or
// minus one, and how many hold it.
func holders(rings []ring) (parent, depth []int) {
	parent, depth = make([]int, len(rings)), make([]int, len(rings))
	for i := range rings {
		parent[i] = -1
		for j := range rings {
			if i == j || !rings[j].holds(rings[i][0]) {
				continue
			}
			depth[i]++
			if parent[i] < 0 || math.Abs(rings[j].area()) <
				math.Abs(rings[parent[i]].area()) {
				parent[i] = j
			}
		}
	}
	return parent, depth
}

// turned is the ring wound counterclockwise for way one and clockwise
// for way minus one.
func turned(r ring, way float64) ring {
	if r.area()*way < 0 {
		return r.reversed()
	}
	return r
}

// crossed is whether any two steps of the rings meet, other than one
// step and the next of the same ring at the point they share.
func crossed(rings []ring) bool {
	type step struct {
		of, at int
		a, b   point
	}
	var steps []step
	for of, r := range rings {
		for at, a := range r {
			steps = append(steps, step{of, at, a, r.at(at + 1)})
		}
	}
	for i, one := range steps {
		for _, two := range steps[i+1:] {
			n := len(rings[one.of])
			if one.of == two.of && next(one.at, two.at, n) {
				continue
			}
			if touch(one.a, one.b, two.a, two.b) {
				return true
			}
		}
	}
	return false
}

// next is whether steps i and j of a ring of n steps follow one another.
func next(i, j, n int) bool { return (i+1)%n == j || (j+1)%n == i }
