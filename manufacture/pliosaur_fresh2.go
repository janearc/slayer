package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PliosaurFresh2 makes one original implicit skin with four separately posed
// flippers. The skull is informed by DORCM G.13,675; postcranium is inferred.
func PliosaurFresh2(phase float64) ([]thing.Thing, error) {
	clamp := func(v, a, b float64) float64 { return math.Max(a, math.Min(b, v)) }
	blend := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	ell := func(p, c, r maths.Vec) float64 {
		x := (p.X - c.X) / r.X
		y := (p.Y - c.Y) / r.Y
		z := (p.Z - c.Z) / r.Z
		return (math.Sqrt(x*x+y*y+z*z) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	field := func(p maths.Vec) float64 {
		d := ell(p, maths.V(-.25, 0, 0), maths.V(.78, .26, .34))
		d = blend(d, ell(p, maths.V(.48, 0, 0), maths.V(.26, .17, .19)), .07)
		d = blend(d, ell(p, maths.V(-1.08, 0, 0), maths.V(.55, .12, .12)), .07)
		t := clamp((p.X-.66)/.67, 0, 1)
		w := .12*(1-t) + .035*t + .022*math.Exp(-math.Pow((t-.91)/.10, 2))
		h := .105*(1-t) + .036*t
		head := (math.Sqrt((p.Y/h)*(p.Y/h)+(p.Z/w)*(p.Z/w)) - 1) * math.Min(h, w)
		head = math.Max(head, math.Max(.66-p.X, p.X-1.33))
		d = blend(d, head, .025)
		for _, root := range []float64{.18, -.72} {
			u := clamp((math.Abs(p.Z)-.22)/.68, 0, 1)
			cx := root - .16*u
			chord := .19*math.Pow(1-u, .65) + .014
			thick := .037*math.Sqrt(math.Max(.02, 1-u)) + .006
			wing := (math.Sqrt(math.Pow((p.X-cx)/chord, 2)+math.Pow(p.Y/thick, 2)) - 1) * math.Min(chord, thick)
			wing = math.Max(wing, math.Max(.22-math.Abs(p.Z), math.Abs(p.Z)-.90))
			d = blend(d, wing, .045)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		root, offset := .18, 0.
		if math.Abs(p.X+.72) < math.Abs(p.X-.18) {
			root, offset = -.72, .55
		}
		xweight := math.Exp(-math.Pow((p.X-root)/.32, 4))
		span := clamp((math.Abs(p.Z)-.22)/.22, 0, 1)
		weight := xweight * span
		angle := .42 * math.Sin(phase+offset) * weight
		c, s := math.Cos(angle), math.Sin(angle)
		side := 1.
		if p.Z < 0 {
			side = -1
		}
		dy, dz := p.Y, math.Abs(p.Z)-.22
		p.Y = dy*c - dz*s
		p.Z = side * (.22 + dy*s + dz*c)
		return p
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 3)
	coat := &thing.Material{Name: "pliosaur clay", Texture: pliosaurTexture2{}, Gloss: .14, Satin: .5}
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
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .013}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		for j := 0; j < 8; j++ {
			x := .94 + .045*float64(j)
			lo, hi = 0, .18
			for k := 0; k < 25; k++ {
				z := (lo + hi) / 2
				if field(maths.V(x, -.025, side*z)) < 0 {
					lo = z
				} else {
					hi = z
				}
			}
			root := warp(maths.V(x, -.026, side*((lo+hi)/2-.003)))
			tooth, e := pliosaurTooth2(root, .006, .032)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type pliosaurTexture2 struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and a grey-green elsewhere.
func (pliosaurTexture2) At(p maths.Vec) maths.Light {
	if p.X > .72 && p.X < 1.32 && p.Y < -.022 && p.Y > -.028 && math.Abs(p.Z) > .035 {
		return maths.RGB(.055, .065, .06)
	}
	return maths.RGB(.31, .35, .32)
}

// pliosaurTooth2 is the mesh of one pointed tooth of a radius and length,
// set at a root point.
func pliosaurTooth2(root maths.Vec, radius, length float64) (*shape.Mesh, error) {
	const n = 10
	v := []maths.Vec{}
	f := [][3]int{}
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
