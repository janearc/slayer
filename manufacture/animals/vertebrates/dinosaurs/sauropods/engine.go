package sauropods

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// Result holds a static reconstruction in local metres and its conservative extent.
type Result struct {
	Parts  []thing.Thing
	Bounds maths.Box
}

// Extent bounds the reconstruction without building its mesh.
func Extent(p Profile) (maths.Box, error) {
	if err := p.Validate(); err != nil {
		return maths.Box{}, err
	}
	minX, maxX := p.Stations[0].X, p.Stations[len(p.Stations)-1].X
	maxY, maxH, maxW, maxGap := 0., 0., 0., 0.
	for i, s := range p.Stations {
		maxY = math.Max(maxY, s.Y)
		maxH = math.Max(maxH, s.Height)
		maxW = math.Max(maxW, s.Width)
		if i > 0 {
			maxGap = math.Max(maxGap, s.X-p.Stations[i-1].X)
		}
	}
	maxFoot, maxZ := 0., maxW
	for _, l := range []Limb{p.Fore, p.Hind} {
		minX = math.Min(minX, l.FootX)
		maxX = math.Max(maxX, l.FootX)
		maxFoot = math.Max(maxFoot, l.FootLength/2)
		maxZ = math.Max(maxZ, l.FootZ+l.FootWidth/2)
	}
	eye := p.Head.EyeRadius
	pad := maxGap + 2*maxH + maxFoot + eye
	return maths.Box{Min: maths.V(minX-pad, -eye, -maxZ-maxW-eye), Max: maths.V(maxX+pad, maxY+maxH+eye, maxZ+maxW+eye)}, nil
}

const around = 32
const steps = 4

type section struct {
	center, up    maths.Vec
	width, height float64
}

// centreY uses bounded monotone cubic interpolation in each X interval.
// It passes through each authored station, shares its slope across adjacent
// spans, and never extends beyond that span's authored vertical range.
func centreY(stations []Station, i int, t float64) (float64, float64) {
	secant := func(j int) float64 { a, b := stations[j], stations[j+1]; return (b.Y - a.Y) / (b.X - a.X) }
	slope := func(j int) float64 {
		if j == 0 {
			return secant(0)
		}
		if j == len(stations)-1 {
			return secant(j - 1)
		}
		a, b := secant(j-1), secant(j)
		if a == 0 || b == 0 || math.Signbit(a) != math.Signbit(b) {
			return 0
		}
		return 2 * a * b / (a + b)
	}
	a, b := stations[i], stations[i+1]
	dx := b.X - a.X
	ma, mb := slope(i), slope(i+1)
	t2, t3 := t*t, t*t*t
	y := (2*t3-3*t2+1)*a.Y + (t3-2*t2+t)*dx*ma + (-2*t3+3*t2)*b.Y + (t3-t2)*dx*mb
	dy := ((6*t2-6*t)*a.Y + (3*t2-4*t+1)*dx*ma + (-6*t2+6*t)*b.Y + (3*t2-2*t)*dx*mb) / dx
	return y, dy
}

func sections(stations []Station) []section {
	out := make([]section, 0, (len(stations)-1)*steps+1)
	for i := 0; i < len(stations)-1; i++ {
		a, b := stations[i], stations[i+1]
		for j := 0; j < steps; j++ {
			t := float64(j) / steps
			y, dy := centreY(stations, i, t)
			v := maths.V(1, dy, 0).Unit()
			out = append(out, section{maths.V(a.X+(b.X-a.X)*t, y, 0), maths.V(-v.Y, v.X, 0), a.Width + (b.Width-a.Width)*t, a.Height + (b.Height-a.Height)*t})
		}
	}
	i := len(stations) - 2
	s := stations[i+1]
	_, dy := centreY(stations, i, 1)
	v := maths.V(1, dy, 0).Unit()
	return append(out, section{maths.V(s.X, s.Y, 0), maths.V(-v.Y, v.X, 0), s.Width, s.Height})
}

// crossingSections refuses a concrete class of folded neck or tail profiles.
// It checks the sampled upper and lower axial contours; this is not a proof
// that every possible three-dimensional surface is intersection-free.
func crossingSections(ss []section) bool {
	top, bottom := make([]maths.Vec, len(ss)), make([]maths.Vec, len(ss))
	for i, s := range ss {
		top[i] = s.center.Add(s.up.Scale(s.height))
		bottom[i] = s.center.Sub(s.up.Scale(s.height))
	}
	cross := func(a, b, c maths.Vec) float64 { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }
	on := func(a, b, p maths.Vec) bool {
		return p.X >= math.Min(a.X, b.X) && p.X <= math.Max(a.X, b.X) && p.Y >= math.Min(a.Y, b.Y) && p.Y <= math.Max(a.Y, b.Y)
	}
	intersects := func(a, b, c, d maths.Vec) bool {
		x, y, z, w := cross(a, b, c), cross(a, b, d), cross(c, d, a), cross(c, d, b)
		if x == 0 && on(a, b, c) || y == 0 && on(a, b, d) || z == 0 && on(c, d, a) || w == 0 && on(c, d, b) {
			return true
		}
		return (x < 0) != (y < 0) && (z < 0) != (w < 0)
	}
	for i := 0; i < len(ss)-1; i++ {
		for j := i + 2; j < len(ss)-1; j++ {
			if intersects(top[i], top[i+1], top[j], top[j+1]) || intersects(bottom[i], bottom[i+1], bottom[j], bottom[j+1]) {
				return true
			}
		}
		for j := 0; j < len(ss)-1; j++ {
			if intersects(top[i], top[i+1], bottom[j], bottom[j+1]) {
				return true
			}
		}
	}
	return false
}

type opening struct {
	limb           Limb
	side           float64
	r0, r1, c0, c1 int
}

// Build joins the paired legs to openings in one unposed, closed body skin.
// The profile is an illustrative static reconstruction, not a skeletal pose.
func Build(p Profile) (Result, error) {
	box, err := Extent(p)
	if err != nil {
		return Result{}, err
	}
	p.Stations = append([]Station(nil), p.Stations...)
	ss := sections(p.Stations)
	points := make([]maths.Vec, 0, len(ss)*around+256)
	for _, s := range ss {
		for j := 0; j < around; j++ {
			a := 2 * math.Pi * float64(j) / around
			points = append(points, s.center.Add(s.up.Scale(s.height*math.Cos(a))).Add(maths.V(0, 0, s.width*math.Sin(a))))
		}
	}
	openings := make([]opening, 0, 4)
	for _, l := range []Limb{p.Hind, p.Fore} {
		for _, side := range []float64{1, -1} {
			c := 10
			if side < 0 {
				c = 18
			}
			openings = append(openings, opening{l, side, l.Station*steps - 1, l.Station*steps + 1, c, c + 4})
		}
	}
	faces := make([][3]int, 0, (len(ss)-1)*around*2+400)
	for r := 0; r < len(ss)-1; r++ {
		for j := 0; j < around; j++ {
			hole := false
			for _, o := range openings {
				if r >= o.r0 && r < o.r1 && j >= o.c0 && j < o.c1 {
					hole = true
					break
				}
			}
			if hole {
				continue
			}
			a, b, c, d := r*around+j, r*around+(j+1)%around, (r+1)*around+(j+1)%around, (r+1)*around+j
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	for _, o := range openings {
		loop := make([]int, 0, 12)
		for j := o.c0; j < o.c1; j++ {
			loop = append(loop, o.r0*around+j)
		}
		for r := o.r0; r < o.r1; r++ {
			loop = append(loop, r*around+o.c1)
		}
		for j := o.c1; j > o.c0; j-- {
			loop = append(loop, o.r1*around+j)
		}
		for r := o.r1; r > o.r0; r-- {
			loop = append(loop, r*around+o.c0)
		}
		center := maths.V(0, 0, 0)
		minY := math.Inf(1)
		for _, index := range loop {
			center = center.Add(points[index])
			minY = math.Min(minY, points[index].Y)
		}
		center = center.Scale(1 / float64(len(loop)))
		rx, rz := 0., 0.
		for _, index := range loop {
			q := points[index]
			rx = math.Max(rx, math.Abs(q.X-center.X))
			rz = math.Max(rz, math.Abs(q.Z-center.Z))
		}
		if minY <= 0 || rx <= 0 || rz <= 0 {
			return Result{}, fmt.Errorf("sauropod %q: invalid limb attachment", p.Kind)
		}
		ux, uz := make([]float64, len(loop)), make([]float64, len(loop))
		for j, index := range loop {
			q := points[index]
			x, z := (q.X-center.X)/rx, (q.Z-center.Z)/rz
			d := math.Hypot(x, z)
			if d == 0 {
				return Result{}, fmt.Errorf("sauropod %q: degenerate limb attachment", p.Kind)
			}
			ux[j], uz[j] = x/d, z/d
		}
		prev := loop
		// A narrowed ankle meets a broader, planted foot. FootLength and
		// FootWidth remain the full authored ground footprint; no toes or
		// claws are inferred from the incomplete fossil evidence.
		heights := [...]float64{.78, .50, .28, .15, .07, 0}
		radii := [...]float64{.65, .65, .65, .68, .95, 1}
		for k, height := range heights {
			t := 1 - height
			next := make([]int, len(loop))
			cx, cz := center.X+(o.limb.FootX-center.X)*t, center.Z+(o.side*o.limb.FootZ-center.Z)*t
			xrad, zrad := radii[k]*o.limb.FootLength/2, radii[k]*o.limb.FootWidth/2
			if k == 0 {
				xrad, zrad = (rx+xrad)/2, (rz+zrad)/2
			}
			for j := range next {
				next[j] = len(points)
				points = append(points, maths.V(cx+ux[j]*xrad, minY*height, cz+uz[j]*zrad))
			}
			for j, a := range prev {
				n := (j + 1) % len(prev)
				faces = append(faces, [3]int{a, prev[n], next[n]}, [3]int{a, next[n], next[j]})
			}
			prev = next
		}
		pole := len(points)
		points = append(points, maths.V(o.limb.FootX, 0, o.side*o.limb.FootZ))
		for j, a := range prev {
			faces = append(faces, [3]int{pole, a, prev[(j+1)%len(prev)]})
		}
	}
	first, last := len(points), len(points)+1
	points = append(points, ss[0].center, ss[len(ss)-1].center)
	for j := 0; j < around; j++ {
		n := (j + 1) % around
		faces = append(faces, [3]int{first, n, j}, [3]int{last, (len(ss)-1)*around + j, (len(ss)-1)*around + n})
	}
	// The vertices strictly inside each removed patch have no faces. Drop
	// them before building the mesh so every vertex belongs to the skin.
	used := make([]bool, len(points))
	for _, face := range faces {
		for _, index := range face {
			used[index] = true
		}
	}
	indices := make([]int, len(points))
	compact := make([]maths.Vec, 0, len(points))
	for i, point := range points {
		if used[i] {
			indices[i] = len(compact)
			compact = append(compact, point)
		}
	}
	for i := range faces {
		for j := range faces[i] {
			faces[i][j] = indices[faces[i][j]]
		}
	}
	points = compact
	skin, err := mesh.New(points, faces, geometry.UnitNormals(points, faces), true)
	if err != nil {
		return Result{}, fmt.Errorf("sauropod %q: %w", p.Kind, err)
	}
	body := thing.Thing{Shape: skin, Material: &thing.Material{Name: p.Kind + " skin", Texture: coat{profile: p}, Satin: .2}, Motion: thing.Still(maths.Local())}
	parts := []thing.Thing{body}
	eye := &thing.Material{Name: p.Kind + " eyes", Colour: maths.RGB(.015, .012, .009), Gloss: .3, Satin: .25}
	s := ss[p.Head.EyeStation*steps]
	level := p.Head.EyeHeight
	across := math.Sqrt(1 - level*level)
	for _, side := range []float64{-1, 1} {
		surface := s.center.Add(s.up.Scale(s.height * level)).Add(maths.V(0, 0, side*s.width*across))
		normal := s.up.Scale(level / s.height).Add(maths.V(0, 0, side*across/s.width)).Unit()
		center := surface.Sub(normal.Scale(.75 * p.Head.EyeRadius))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: p.Head.EyeRadius}, Material: eye, Motion: thing.Still(maths.NewSubspace(center, maths.Identity, 1))})
	}
	return Result{Parts: parts, Bounds: box}, nil
}
