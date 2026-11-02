package typography

import "math"

// piece is one stretch of a glyph's outline: a straight step of two
// points, or a quadratic curve of three or a cubic of four, from its
// first point to its last, the points between pulling it.
type piece []point

// deepest is how many times a curve is halved at most: a curve 4^16
// times rounder than the tolerance is no glyph's.
const deepest = 16

// flattened is a contour as a ring of straight steps: every curve cut
// into steps, more where it bends harder, until each step is within
// tolerance of the curve it stands for.
func flattened(contour []piece, tolerance float64) ring {
	var made ring
	for _, each := range contour {
		made = flatten(each, tolerance, deepest, made)
	}
	return made
}

// flatten is into with the points that step along a piece after its
// first: its last alone when it is straight or near enough, and
// otherwise each half of it flattened in turn.
func flatten(p piece, tolerance float64, depth int, into ring) ring {
	if len(p) == 2 || depth == 0 || bend(p) <= tolerance {
		return append(into, p[len(p)-1])
	}
	front, back := halves(p)
	into = flatten(front, tolerance, depth-1, into)
	return flatten(back, tolerance, depth-1, into)
}

// bend is as far as a curve can stray from the straight step between
// its ends: a curve lies inside the hull of its points, and the point
// of a hull furthest from a step is one of its corners, so it is the
// distance from the step of the point that pulls furthest from it.
func bend(p piece) float64 {
	worst := 0.0
	for _, pull := range p[1 : len(p)-1] {
		worst = max(worst, apart(p[0], p[len(p)-1], pull))
	}
	return worst
}

// apart is how far p is from the step from a to b.
func apart(a, b, p point) float64 {
	way := b.sub(a)
	t := 0.0
	if l := way.dot(way); l > 0 {
		t = math.Max(0, math.Min(1, p.sub(a).dot(way)/l))
	}
	return p.sub(a.add(way.scale(t))).length()
}

// halves is a curve cut at its middle into two of the same degree, by
// de casteljau's halving of each step between its points.
func halves(p piece) (piece, piece) {
	front, back := make(piece, len(p)), make(piece, len(p))
	run := append(piece(nil), p...)
	for k := range p {
		front[k] = run[0]
		back[len(p)-1-k] = run[len(run)-1]
		for i := 0; i+1 < len(run); i++ {
			run[i] = run[i].add(run[i+1]).scale(0.5)
		}
		run = run[:len(run)-1]
	}
	return front, back
}
