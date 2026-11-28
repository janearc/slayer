package primates

import (
	"github.com/janearc/slayer/maths"
	"math"
)

// A socket replaces a rectangular patch of lateral trunk faces. Its perimeter
// is the first ring of the upper arm, so torso and shoulder share vertex IDs.
type socket struct {
	lo, hi, bottom, top, around int
	sign                        float64
}

func shoulderSockets(around, rows int) [2]socket {
	row := func(y float64) int {
		return int(math.Round((y - cage[0].y) / (cage[len(cage)-1].y - cage[0].y) * float64(rows)))
	}
	half := max(3, int(math.Round(.104*float64(around))))
	return [2]socket{{around/4 - half, around/4 + half, row(1.264), row(1.369), around, 1}, {3*around/4 - half, 3*around/4 + half, row(1.264), row(1.369), around, -1}}
}
func (s socket) covers(i, j int) bool { return i >= s.lo && i < s.hi && j >= s.bottom && j < s.top }
func (s socket) rim() []int {
	var loop []int
	at := func(i, j int) { loop = append(loop, j*s.around+i) }
	for i := s.lo; i < s.hi; i++ {
		at(i, s.bottom)
	}
	for j := s.bottom; j < s.top; j++ {
		at(s.hi, j)
	}
	for i := s.hi; i > s.lo; i-- {
		at(i, s.top)
	}
	for j := s.top; j > s.bottom; j-- {
		at(s.lo, j)
	}
	return loop
}
func (s socket) grow(pts, rest *[]maths.Vec, faces *[][3]int) {
	loop := s.rim()
	first := append([]int(nil), loop...)
	var centre maths.Vec
	for _, k := range loop {
		centre = centre.Add((*pts)[k])
	}
	centre = centre.Scale(1 / float64(len(loop)))
	// Shoulder-to-brachium centreline. Its tangent rotates the upper edge of
	// the glenoid-level opening into the lateral deltoid and the lower edge into
	// the medial arm. The open armpit remains between the trunk and the arm.
	const steps = 45
	// Parameterise an ideal cross-section. The original opening's noncircular
	// offsets and inward surface tangent are boundary conditions on that tube.
	section := func(t, sn, cs float64) maths.Vec {
		cx := .080*t + .030*math.Sin(math.Pi*t)
		cy := -.242*t*t + .004*math.Sin(math.Pi*t)
		tx := .080 + .030*math.Pi*math.Cos(math.Pi*t)
		ty := -.484*t + .004*math.Pi*math.Cos(math.Pi*t)
		axis := maths.V(-ty, tx, 0).Unit()
		radius := .038 + .015*bell((t-.34)/.25) - .003*t
		depth := .038 + .012*bell((t-.38)/.28) - .003*t
		target := centre.Add(maths.V(s.sign*cx, cy, -.014*t))
		p := target.Add(maths.V(s.sign*axis.X, axis.Y, 0).Scale(radius * sn)).Add(maths.ZAxis.Scale(depth * cs))
		p.Z += .004 * bell((t-.74)/.25) * cs * math.Abs(cs)
		return p
	}
	type boundary struct {
		sn, cs        float64
		offset, slope maths.Vec
	}
	conditions := make([]boundary, len(first))
	grids := [][]int{first}
	for i, k := range first {
		root := (*pts)[k]
		delta := root.Sub(centre)
		a := math.Atan2(delta.Y/.052, delta.Z/.067)
		sn, cs := math.Sincos(a)
		c, r := k%s.around, k/s.around
		var inward maths.Vec
		if c == s.lo {
			inward = inward.Add(root.Sub((*pts)[k-1]).Unit())
		}
		if c == s.hi {
			inward = inward.Add(root.Sub((*pts)[k+1]).Unit())
		}
		if r == s.bottom {
			inward = inward.Add(root.Sub((*pts)[k-s.around]).Unit())
		}
		if r == s.top {
			inward = inward.Add(root.Sub((*pts)[k+s.around]).Unit())
		}
		start := section(0, sn, cs)
		tangent := section(.0001, sn, cs).Sub(start).Scale(10000)
		conditions[i] = boundary{sn, cs, root.Sub(start), inward.Unit().Scale(.118).Sub(tangent)}
	}
	for j := 1; j <= steps; j++ {
		t := float64(j) / steps
		ring := make([]int, len(loop))
		for i, c := range conditions {
			p := section(t, c.sn, c.cs).Add(c.offset.Scale(1 - maths.Smooth(0, .55, t))).Add(c.slope.Scale(t * (1 - maths.Smooth(0, .39, t))))
			ring[i] = len(*pts)
			*pts = append(*pts, p)
			*rest = append(*rest, p)
		}
		for i, a := range loop {
			n := (i + 1) % len(loop)
			*faces = append(*faces, [3]int{a, loop[n], ring[n]}, [3]int{a, ring[n], ring[i]})
		}
		loop = ring
		grids = append(grids, ring)
	}
	s.parameterize(pts, rest, grids)
	var end maths.Vec
	for _, i := range loop {
		end = end.Add((*pts)[i])
	}
	end = end.Scale(1 / float64(len(loop)))
	k := len(*pts)
	*pts = append(*pts, end)
	*rest = append(*rest, end)
	for i, a := range loop {
		*faces = append(*faces, [3]int{k, a, loop[(i+1)%len(loop)]})
	}
}

// Drop the interior vertices of each removed socket patch before export.
func referenced(pts, rest []maths.Vec, faces [][3]int) ([]maths.Vec, []maths.Vec) {
	mapped := make([]int, len(pts))
	var p, r []maths.Vec
	for i, f := range faces {
		for j, k := range f {
			if mapped[k] == 0 {
				p = append(p, pts[k])
				r = append(r, rest[k])
				mapped[k] = len(p)
			}
			faces[i][j] = mapped[k] - 1
		}
	}
	return p, r
}

// parameterize recomputes the interior front/back charts in the coronal
// plane. The source socket boundary, distal section and both silhouette rails
// stay fixed. A positive projected Jacobian prevents a skin sheet from
// folding over itself inside the upper arm.
func (s socket) parameterize(pts, rest *[]maths.Vec, grid [][]int) {
	n, m := len(grid[0]), len(grid)
	top, bottom := 0, 0
	for i, k := range grid[0] {
		row, col := k/s.around, k%s.around
		if col == (s.lo+s.hi)/2 {
			if row == s.top {
				top = i
			}
			if row == s.bottom {
				bottom = i
			}
		}
	}
	fixed := func(j, i int) bool { return j == 0 || j == m-1 || i == top || i == bottom }
	at := func(j, i int) *maths.Vec { return &(*pts)[grid[j][(i+n)%n]] }
	// Position a nearly harmonic grid between the measured torso edge and
	// arm rails, interpolating depth with coronal position. The orientation
	// correction below changes only the coronal coordinates.
	for sweep := 0; sweep < 32; sweep++ {
		for j := 1; j < m-1; j++ {
			for i := 0; i < n; i++ {
				if fixed(j, i) {
					continue
				}
				p := at(j, i)
				a, b, c, d := at(j, i-1), at(j, i+1), at(j-1, i), at(j+1, i)
				p.X = .50*p.X + .15*(a.X+b.X) + .10*(c.X+d.X)
				p.Y = .50*p.Y + .15*(a.Y+b.Y) + .10*(c.Y+d.Y)
				p.Z = .50*p.Z + .15*(a.Z+b.Z) + .10*(c.Z+d.Z)
			}
		}
	}
	for sweep := 0; sweep < 120; sweep++ {
		for j := 0; j < m-1; j++ {
			for i := 0; i < n; i++ {
				k := (i + 1) % n
				// Direction follows the wound socket rim. The anterior half of the
				// root is identified in the rest chart rather than by a posed normal.
				src := grid[0]
				sign := s.sign
				if src[i]%s.around+src[k]%s.around > s.lo+s.hi {
					sign = -sign
				}

				for _, corners := range [2][3][2]int{{{j, i}, {j, k}, {j + 1, k}}, {{j, i}, {j + 1, k}, {j + 1, i}}} {
					var v [3]*maths.Vec
					for l, pair := range corners {
						v[l] = at(pair[0], pair[1])
					}
					area := sign * ((v[1].X-v[0].X)*(v[2].Y-v[0].Y) - (v[2].X-v[0].X)*(v[1].Y-v[0].Y))
					if area >= 2e-9 {
						continue
					}
					var g [3]maths.Vec
					den := 0.
					for l, pair := range corners {
						if fixed(pair[0], pair[1]) {
							continue
						}
						a, b := v[(l+1)%3], v[(l+2)%3]
						g[l] = maths.V(a.Y-b.Y, b.X-a.X, 0)
						den += g[l].Dot(g[l])
					}
					if den > 0 {
						for l := range v {
							*v[l] = v[l].Add(g[l].Scale(sign * (2e-9 - area) / den))
						}
					}
				}
			}
		}
	}
	for j := 1; j < m; j++ {
		for _, k := range grid[j] {
			(*rest)[k] = (*pts)[k]
		}
	}
}
