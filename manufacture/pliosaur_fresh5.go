package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PliosaurFresh5 builds a fresh axial/head field and four asymmetric wing
// fields. Fore and hind influences are continuous and independently phased.
func PliosaurFresh5(phase float64) ([]thing.Thing, error) {
	clamp := func(v, a, b float64) float64 { return math.Max(a, math.Min(b, v)) }
	blend := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	mass := func(p, c, r maths.Vec) float64 {
		x := (p.X - c.X) / r.X
		y := (p.Y - c.Y) / r.Y
		z := (p.Z - c.Z) / r.Z
		return (math.Sqrt(x*x+y*y+z*z) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	field := func(p maths.Vec) float64 {
		d := mass(p, maths.V(-.24, 0, 0), maths.V(.79, .26, .34))
		d = blend(d, mass(p, maths.V(-1.08, 0, 0), maths.V(.43, .09, .10)), .06)
		d = blend(d, mass(p, maths.V(.49, 0, 0), maths.V(.25, .17, .18)), .07)
		t := clamp((p.X-.66)/.67, 0, 1)
		w := .035 + .085*math.Pow(1-t, 2) + .025*math.Exp(-math.Pow((t-.90)/.10, 2))
		h := .105*(1-t) + .036*t
		head := (math.Sqrt((p.Y/h)*(p.Y/h)+(p.Z/w)*(p.Z/w)) - 1) * math.Min(h, w)
		head = math.Max(head, math.Max(.66-p.X, p.X-1.33))
		d = blend(d, head, .035)
		for _, root := range []float64{.18, -.72} {
			u := clamp((math.Abs(p.Z)-.20)/.70, 0, 1)
			leading := root + .17*math.Pow(1-u, .6) - .11*u*u
			trailing := root - .23*math.Pow(1-u, .8) - .11*u*u
			cx := (leading + trailing) / 2
			half := math.Max(.008, (leading-trailing)/2)
			thick := .040*math.Pow(math.Max(.02, 1-u), .45) + .005
			wing := (math.Sqrt(math.Pow((p.X-cx)/half, 2)+math.Pow(p.Y/thick, 2)) - 1) * math.Min(half, thick)
			wing = math.Max(wing, math.Max(.20-math.Abs(p.Z), math.Abs(p.Z)-.90))
			d = blend(d, wing, .05)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		u := clamp((math.Abs(p.Z)-.18)/.34, 0, 1)
		u = u * u * (3 - 2*u)
		wf := math.Exp(-math.Pow((p.X-.18)/.31, 4))
		wh := math.Exp(-math.Pow((p.X+.72)/.31, 4))
		weight := math.Max(1, wf+wh)
		a := .40 * u * (wf*math.Sin(phase) + wh*math.Sin(phase+.55)) / weight
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
	coat := &thing.Material{Name: "pliosaur clay", Texture: pliosaurTexture5{}, Gloss: .14, Satin: .5}
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
			x := .95 + .044*float64(j)
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
			tooth, e := pliosaurTooth5(root, .006, .033)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type pliosaurTexture5 struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and a grey-green elsewhere.
func (pliosaurTexture5) At(p maths.Vec) maths.Light {
	if p.X > .73 && p.X < 1.33 && p.Y < -.022 && p.Y > -.028 && math.Abs(p.Z) > .025 {
		return maths.RGB(.055, .065, .06)
	}
	return maths.RGB(.31, .35, .32)
}

// pliosaurTooth5 is the mesh of one pointed tooth of a radius and length,
// set at a root point.
func pliosaurTooth5(root maths.Vec, radius, length float64) (*shape.Mesh, error) {
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
	v = append(v, root.Add(maths.V(.004, -length, 0)))
	for j := 0; j < n; j++ {
		k := (j + 1) % n
		f = append(f, [3]int{j, k, tip}, [3]int{base, k, j})
	}
	return shape.NewMesh(v, f, nil, false)
}
