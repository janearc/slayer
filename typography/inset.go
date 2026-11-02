package typography

import (
	"errors"
	"math"
	"slices"
)

// errBevel is a bevel the glyph cannot take: a stroke narrower than two
// bevels, so its outline stepped in would cross itself.
var errBevel = errors.New("the bevel is too wide for it")

// increments is how many steps a ring is moved in by, a bevel's share
// at a time, so that of the steps that vanish on the way the first to
// vanish goes first.
const increments = 16

// edge is a step of a ring as a line moved in: where the step starts,
// moved, and which way it runs.
type edge struct{ at, way point }

// moved is the step's line moved in by d, toward the letter.
func (e edge) moved(d float64) edge {
	return edge{e.at.sub(e.way.right().scale(d)), e.way}
}

// meet is where two lines cross, and whether they do: lines that run
// the same way, or near enough, meet nowhere.
func meet(one, two edge) (point, bool) {
	turn := one.way.cross(two.way)
	if math.Abs(turn) <= 1e-9*one.way.length()*two.way.length() {
		return point{}, false
	}
	t := two.at.sub(one.at).cross(two.way) / turn
	return one.at.add(one.way.scale(t)), true
}

// inset is the solid with every ring moved in toward the letter by d
// font units, and for each ring, the place in its inset that each of
// its points moved to. rings that cross once moved are errBevel.
func inset(one solid, d float64) (solid, [][]int, error) {
	var made solid
	var to [][]int
	for k, r := range one.rings() {
		in, places, err := insetRing(r, d)
		if err != nil {
			return solid{}, nil, err
		}
		to = append(to, places)
		if k == 0 {
			made.outer = in
		} else {
			made.holes = append(made.holes, in)
		}
	}
	if crossed(made.rings()) || !kept(made) {
		return solid{}, nil, errBevel
	}
	return made, to, nil
}

// kept is whether a solid moved in is still one: its outer turning
// counterclockwise, and each hole turning clockwise inside it. a hole
// grown past an outer shrunk holds the outer, and crosses nothing.
func kept(one solid) bool {
	if !(one.outer.area() > 0) {
		return false
	}
	for _, hole := range one.holes {
		if !(hole.area() < 0) || !one.outer.holds(hole[0]) {
			return false
		}
	}
	return true
}

// insetRing is a ring moved in by d: each step's line moved d toward
// the letter, and each corner where its two steps' lines meet.
//
// a step whose ends cross on the way, which happens where the steps
// beside it turn toward each other faster than it is long, vanishes,
// and its neighbours meet without it.
//
// to is, for each point of the ring, the place in the inset of the
// corner it moved to: one shared by the two ends of a step that
// vanished.
func insetRing(r ring, d float64) (in ring, to []int, err error) {
	edges := make([]edge, len(r))
	for k, p := range r {
		edges[k] = edge{p, r.at(k + 1).sub(p)}
	}
	alive := make([]int, len(r))
	for k := range alive {
		alive[k] = k
	}
	for step := 1; step <= increments; step++ {
		far := d * float64(step) / increments
		if alive, in, err = vanish(edges, alive, far); err != nil {
			return nil, nil, err
		}
	}
	return in, places(len(r), alive), nil
}

// vanish is the steps left alive once every one whose ends cross, when
// moved in by d, is gone, the one most turned back first, and the
// corners of those left.
func vanish(edges []edge, alive []int, d float64) ([]int, ring, error) {
	for {
		if len(alive) < 3 {
			return nil, nil, errBevel
		}
		corners, err := cornersOf(edges, alive, d)
		if err != nil {
			return nil, nil, err
		}
		worst, back := -1, 0.0
		for i, k := range alive {
			run := corners[(i+1)%len(alive)].sub(corners[i])
			if turned := run.dot(edges[k].way) / edges[k].way.dot(
				edges[k].way); turned <= back {
				worst, back = i, turned
			}
		}
		if worst < 0 {
			return alive, corners, nil
		}
		alive = slices.Delete(alive, worst, worst+1)
	}
}

// cornersOf is where each step alive starts once moved in by d: where
// its line meets the line of the step alive before it.
func cornersOf(edges []edge, alive []int, d float64) (ring, error) {
	corners := make(ring, len(alive))
	for i, k := range alive {
		before := edges[alive[(i+len(alive)-1)%len(alive)]].moved(d)
		at, ok := meet(before, edges[k].moved(d))
		if !ok {
			return nil, errBevel
		}
		corners[i] = at
	}
	return corners, nil
}

// places is, for each of n points, the place among the steps alive of
// the corner it moved to: its own step's start if that step is alive,
// and otherwise the start of the next step alive after it.
func places(n int, alive []int) []int {
	to := make([]int, n)
	for i, k := range alive {
		before := alive[(i+len(alive)-1)%len(alive)]
		for j := k; j != before; j = (j + n - 1) % n {
			to[j] = i
		}
	}
	return to
}
