package main

import (
	"github.com/janearc/slayer/maths"
)

// refine subdivides the supplied original polygons before their display
// triangulation. Boundary vertices follow the boundary crease rule.
func refine(g geom) geom {
	type edge struct {
		a, b, count, index int
		sum                maths.Vec
	}
	pair := func(a, b int) [2]int { return [2]int{min(a, b), max(a, b)} }
	edges := map[[2]int]*edge{}
	var order []*edge
	centers := make([]maths.Vec, len(g.polys))
	fSum := make([]maths.Vec, len(g.pts))
	rSum := make([]maths.Vec, len(g.pts))
	val := make([]int, len(g.pts))
	boundary := make([][]int, len(g.pts))
	for fi, poly := range g.polys {
		var center maths.Vec
		for _, i := range poly {
			center = center.Add(g.pts[i])
		}
		center = center.Scale(1 / float64(len(poly)))
		centers[fi] = center
		for i, a := range poly {
			b := poly[(i+1)%len(poly)]
			key := pair(a, b)
			e := edges[key]
			if e == nil {
				e = &edge{a: a, b: b, index: len(g.pts) + len(order)}
				edges[key] = e
				order = append(order, e)
			}
			e.sum = e.sum.Add(center)
			e.count++
			fSum[a] = fSum[a].Add(center)
			val[a]++
		}
	}
	out := geom{pts: make([]maths.Vec, len(g.pts)+len(order)+len(centers))}
	for _, e := range order {
		a, b := e.a, e.b
		mid := g.pts[a].Add(g.pts[b]).Scale(.5)
		rSum[a] = rSum[a].Add(mid)
		rSum[b] = rSum[b].Add(mid)
		if e.count == 2 {
			out.pts[e.index] = mid.Scale(.5).Add(e.sum.Scale(.25))
		} else {
			out.pts[e.index] = mid
			boundary[a] = append(boundary[a], b)
			boundary[b] = append(boundary[b], a)
		}
	}
	for i, p := range g.pts {
		q := p
		n := float64(val[i])
		if len(boundary[i]) == 2 {
			a, b := boundary[i][0], boundary[i][1]
			q = p.Scale(.75).Add(g.pts[a].Add(g.pts[b]).Scale(.125))
		} else if n >= 3 {
			q = p.Scale((n - 3) / n).Add(fSum[i].Add(rSum[i].Scale(2)).Scale(1 / (n * n)))
		}
		out.pts[i] = q
	}
	offset := len(g.pts) + len(order)
	copy(out.pts[offset:], centers)
	for fi, poly := range g.polys {
		for i, a := range poly {
			next, prev := poly[(i+1)%len(poly)], poly[(i+len(poly)-1)%len(poly)]
			east := edges[pair(a, next)].index
			west := edges[pair(prev, a)].index
			c := offset + fi
			out.faces = append(out.faces, [3]int{a, east, c}, [3]int{a, c, west})
		}
	}
	return out
}
