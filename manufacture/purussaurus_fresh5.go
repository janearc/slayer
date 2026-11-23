package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PurussaurusFresh5 builds an original tapered-segment skin and weighted rig.
// The body, soft tissue, colors, posture and motion are scientific conjecture.
func PurussaurusFresh5(phase, yaw, posture float64) ([]thing.Thing, error) {
	type axial struct{ a, b, ry0, ry1, rz0, rz1 float64 }
	bones := []axial{{-1.70, -1.28, .018, .095, .012, .055}, {-1.28, -.80, .095, .17, .055, .12}, {-.80, -.40, .17, .215, .12, .18}, {-.40, .12, .215, .24, .18, .34}, {.12, .60, .24, .20, .34, .28}, {.60, .90, .20, .105, .28, .155}, {.90, 1.10, .105, .09, .155, .147}, {1.10, 1.37, .09, .075, .147, .13}}
	clamp := func(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
	blend := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	axialField := func(p maths.Vec, b axial) float64 {
		t := clamp((p.X-b.a)/(b.b-b.a), 0, 1)
		x := b.a + (b.b-b.a)*t
		ry := b.ry0*(1-t) + b.ry1*t
		rz := b.rz0*(1-t) + b.rz1*t
		rx := math.Min(.08, math.Max(.014, rz))
		dx := (p.X - x) / rx
		return (math.Sqrt(dx*dx+(p.Y/ry)*(p.Y/ry)+(p.Z/rz)*(p.Z/rz)) - 1) * math.Min(rx, math.Min(ry, rz))
	}
	segment := func(p, a, b maths.Vec, ra, rb float64) float64 {
		ab := b.Sub(a)
		t := clamp(p.Sub(a).Dot(ab)/ab.Dot(ab), 0, 1)
		return p.Sub(a.Add(ab.Scale(t))).Len() - (ra*(1-t) + rb*t)
	}
	field := func(p maths.Vec) float64 {
		d := 1.
		for _, b := range bones {
			d = blend(d, axialField(p, b), .028)
		}
		for _, s := range []float64{-1, 1} {
			for _, x := range []float64{.43, -.40} {
				a := maths.V(x, -.075, s*.26)
				b := maths.V(x-.04, -.16, s*.40)
				c := maths.V(x+.03, -.23, s*.52)
				d = blend(d, segment(p, a, b, .10, .08), .04)
				d = blend(d, segment(p, b, c, .08, .06), .035)
				for toe := 0; toe < 4; toe++ {
					z := s * (.46 + .052*float64(toe))
					from := maths.V(x+.01, -.245, z)
					to := maths.V(x+.14, -.26, z+s*.02)
					d = blend(d, segment(p, from, to, .035, .023), .05)
				}
			}
		}
		// Low relief is speculative. The procedural color marks also approximate
		// living caiman scutes; neither is a fossil plate reconstruction.
		if p.Y > .15 && p.X > -.38 && p.X < .72 {
			a := math.Cos((p.X + .38) * 36)
			b := math.Cos(p.Z * 23)
			d -= .008 * a * a * a * a * b * b * b * b * clamp((p.Y-.15)/.07, 0, 1)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		w := clamp((-.40-p.X)/1.30, 0, 1)
		shift := .085*w*w*math.Sin(phase-4*w) + .17*yaw*w*w
		grad := (-1. / 1.30) * (.17*w*math.Sin(phase-4*w) - .34*w*w*math.Cos(phase-4*w) + .34*yaw*w)
		a := math.Atan(grad)
		p.X, p.Z = p.X-p.Z*math.Sin(a), shift+p.Z*math.Cos(a)
		side := 1.
		if p.Z < 0 {
			side = -1
		}
		z := math.Abs(p.Z)
		hipX := .43
		if math.Abs(p.X+.4) < math.Abs(p.X-.43) {
			hipX = -.4
		}
		weight := clamp((z-.28)/.27, 0, 1) * math.Exp(-math.Pow((p.X-hipX)/.22, 2))
		theta := 0.
		if posture > 0 {
			theta = .12
		}
		if posture < 0 {
			theta = -.15
		}
		dy, dz := p.Y+.075, z-.26
		c, s := math.Cos(theta), math.Sin(theta)
		p.Y += weight * (dy*c - dz*s - dy)
		p.Z += side * weight * (dy*s + dz*c - dz)
		return p
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 2.8)
	coat := &thing.Material{Name: "procedural caiman", Texture: purussaurusSkin5{}, Gloss: .08, Satin: .35}
	dark := &thing.Material{Name: "eye and nostril", Colour: maths.RGB(.022, .02, .01), Gloss: .20}
	ivory := &thing.Material{Name: "hypothetical tooth", Colour: maths.RGB(.67, .63, .49), Gloss: .10}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, s := range []float64{-1, 1} {
		for _, a := range []struct{ x, z, r float64 }{{1.00, .115, .010}, {1.31, .045, .006}} {
			lo, hi := 0., .25
			for i := 0; i < 28; i++ {
				y := (lo + hi) / 2
				if field(maths.V(a.x, y, s*a.z)) < 0 {
					lo = y
				} else {
					hi = y
				}
			}
			at := warp(maths.V(a.x, (lo+hi)/2+.001, s*a.z))
			parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: a.r}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		}
		for j := 0; j < 5; j++ {
			x := 1.06 + .065*float64(j)
			lo, hi := 0., .25
			for i := 0; i < 25; i++ {
				z := (lo + hi) / 2
				if field(maths.V(x, -.035, s*z)) < 0 {
					lo = z
				} else {
					hi = z
				}
			}
			root := warp(maths.V(x, -.045, s*((lo+hi)/2-.003)))
			tooth, e := purussaurusTooth5(root, .006, .035)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type purussaurusSkin5 struct{}

// At is the skin's colour at a point: dark marks on the head, a different
// tone along the back, and green elsewhere.
func (purussaurusSkin5) At(p maths.Vec) maths.Light {
	if p.X > .96 && p.X < 1.37 && p.Y > -.035 && p.Y < -.026 && math.Abs(p.Z) > .09 {
		return maths.RGB(.042, .044, .022)
	}
	if p.X > -.65 && p.X < .75 && p.Y > .13 {
		a := math.Abs(math.Sin(30 * p.X))
		b := math.Abs(math.Sin(22 * p.Z))
		if a < .15 || b < .15 {
			return maths.RGB(.13, .16, .085)
		}
	}
	return maths.RGB(.19, .21, .12)
}

// purussaurusTooth5 is the mesh of one pointed tooth of a radius and
// length, set at a root point.
func purussaurusTooth5(root maths.Vec, radius, length float64) (*shape.Mesh, error) {
	const n = 10
	v := make([]maths.Vec, 0, n+2)
	f := make([][3]int, 0, 2*n)
	for j := 0; j < n; j++ {
		a := 2 * math.Pi * float64(j) / n
		v = append(v, root.Add(maths.V(radius*math.Cos(a), 0, radius*math.Sin(a))))
	}
	base := len(v)
	v = append(v, root)
	tip := len(v)
	v = append(v, root.Add(maths.V(0, -length, 0)))
	for j := 0; j < n; j++ {
		k := (j + 1) % n
		f = append(f, [3]int{j, k, tip}, [3]int{base, k, j})
	}
	return shape.NewMesh(v, f, nil, false)
}
