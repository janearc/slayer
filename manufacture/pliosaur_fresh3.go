package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PliosaurFresh3 builds a fresh controlled-mass body and curved chains of
// flipper sections; noncranial dimensions and stroke are inferred.
func PliosaurFresh3(phase float64) ([]thing.Thing, error) {
	ell := func(p, c, r maths.Vec) float64 {
		x := (p.X - c.X) / r.X
		y := (p.Y - c.Y) / r.Y
		z := (p.Z - c.Z) / r.Z
		return (math.Sqrt(x*x+y*y+z*z) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	blend := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	clamp := func(v, a, b float64) float64 { return math.Max(a, math.Min(b, v)) }
	field := func(p maths.Vec) float64 {
		d := ell(p, maths.V(-.25, 0, 0), maths.V(.77, .26, .34))
		d = blend(d, ell(p, maths.V(-1.08, 0, 0), maths.V(.47, .09, .09)), .06)
		d = blend(d, ell(p, maths.V(.46, 0, 0), maths.V(.28, .17, .19)), .07)
		d = blend(d, ell(p, maths.V(.72, .005, 0), maths.V(.18, .13, .135)), .04)
		d = blend(d, ell(p, maths.V(1.04, .005, 0), maths.V(.30, .067, .073)), .032)
		d = blend(d, ell(p, maths.V(1.29, .002, 0), maths.V(.075, .045, .052)), .018)
		for _, s := range []float64{-1, 1} {
			for _, root := range []float64{.18, -.72} {
				for j := 0; j < 7; j++ {
					t := float64(j) / 6
					cx := root - .20*t*t
					cz := s * (.23 + .61*t)
					rx := .19 * (1 - .68*t)
					ry := .048 * (1 - .58*t)
					rz := .12 * (1 - .25*t)
					d = blend(d, ell(p, maths.V(cx, 0, cz), maths.V(rx, ry, rz)), .027)
				}
			}
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		root, offset := .18, 0.
		if math.Abs(p.X+.72) < math.Abs(p.X-.18) {
			root, offset = -.72, .55
		}
		u := clamp((math.Abs(p.Z)-.17)/.37, 0, 1)
		u = u * u * (3 - 2*u)
		weight := u * math.Exp(-math.Pow((p.X-root)/.48, 2))
		a := .40 * math.Sin(phase+offset) * weight
		c, s := math.Cos(a), math.Sin(a)
		side := 1.
		if p.Z < 0 {
			side = -1
		}
		dy, dz := p.Y, math.Abs(p.Z)-.18
		p.Y = dy*c - dz*s
		p.Z = side * (.18 + dy*s + dz*c)
		return p
	}
	skin, e := bioformFieldMeshFine(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 3)
	coat := &thing.Material{Name: "pliosaur clay", Texture: pliosaurTexture3{}, Gloss: .14, Satin: .5}
	dark := &thing.Material{Name: "eye", Colour: maths.RGB(.025, .027, .023), Gloss: .28}
	ivory := &thing.Material{Name: "inferred teeth", Colour: maths.RGB(.7, .67, .56), Gloss: .1}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, side := range []float64{-1, 1} {
		lo, hi := 0., .24
		for j := 0; j < 28; j++ {
			z := (lo + hi) / 2
			if field(maths.V(.87, .025, side*z)) < 0 {
				lo = z
			} else {
				hi = z
			}
		}
		at := warp(maths.V(.87, .025, side*((lo+hi)/2+.002)))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .012}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		for j := 0; j < 8; j++ {
			x := .96 + .043*float64(j)
			lo, hi = 0, .18
			for k := 0; k < 25; k++ {
				z := (lo + hi) / 2
				if field(maths.V(x, -.025, side*z)) < 0 {
					lo = z
				} else {
					hi = z
				}
			}
			root := warp(maths.V(x, -.027, side*((lo+hi)/2-.003)))
			tooth, e := pliosaurTooth3(root, .006, .033)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type pliosaurTexture3 struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and a grey-green elsewhere.
func (pliosaurTexture3) At(p maths.Vec) maths.Light {
	if p.X > .73 && p.X < 1.33 && p.Y < -.022 && p.Y > -.028 && math.Abs(p.Z) > .03 {
		return maths.RGB(.055, .065, .06)
	}
	return maths.RGB(.31, .35, .32)
}

// pliosaurTooth3 is the mesh of one pointed tooth of a radius and length,
// set at a root point.
func pliosaurTooth3(root maths.Vec, radius, length float64) (*shape.Mesh, error) {
	const n = 12
	v := []maths.Vec{}
	f := [][3]int{}
	for j := 0; j < n; j++ {
		a := 2 * math.Pi * float64(j) / n
		v = append(v, root.Add(maths.V(radius*math.Cos(a), 0, radius*math.Sin(a))))
	}
	base := len(v)
	v = append(v, root)
	tip := len(v)
	v = append(v, root.Add(maths.V(.005, -length, 0)))
	for j := 0; j < n; j++ {
		k := (j + 1) % n
		f = append(f, [3]int{j, k, tip}, [3]int{base, k, j})
	}
	return shape.NewMesh(v, f, nil, false)
}
