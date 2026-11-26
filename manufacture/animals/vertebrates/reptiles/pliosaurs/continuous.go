package pliosaurs

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// NewContinuous builds this profile on its authored family scaffold.
func NewContinuous(profile reptiles.Profile, phase float64) ([]thing.Thing, error) {
	clamp := func(v, a, b float64) float64 { return math.Max(a, math.Min(b, v)) }
	mass := func(p, c, r maths.Vec) float64 {
		x := (p.X - c.X) / r.X
		y := (p.Y - c.Y) / r.Y
		z := (p.Z - c.Z) / r.Z
		return (math.Sqrt(x*x+y*y+z*z) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	field := func(p maths.Vec) float64 {
		m := profile.Masses
		d := mass(p, maths.V(m[0][0], m[0][1], m[0][2]), maths.V(m[0][3], m[0][4], m[0][5]))
		d = geometry.SmoothMin(d, mass(p, maths.V(m[1][0], m[1][1], m[1][2]), maths.V(m[1][3], m[1][4], m[1][5])), .06)
		d = geometry.SmoothMin(d, mass(p, maths.V(m[2][0], m[2][1], m[2][2]), maths.V(m[2][3], m[2][4], m[2][5])), .07)
		headSpec := profile.ContinuousHead
		t := clamp((p.X-headSpec[0])/(headSpec[1]-headSpec[0]), 0, 1)
		w := headSpec[2] + headSpec[3]*math.Pow(1-t, 2) + headSpec[4]*math.Exp(-math.Pow((t-.90)/.10, 2))
		h := headSpec[5]*(1-t) + headSpec[6]*t
		head := (math.Sqrt((p.Y/h)*(p.Y/h)+(p.Z/w)*(p.Z/w)) - 1) * math.Min(h, w)
		head = math.Max(head, math.Max(headSpec[0]-p.X, p.X-headSpec[1]))
		d = geometry.SmoothMin(d, head, .035)
		fin := profile.ContinuousFins
		for _, root := range fin[:2] {
			u := clamp((math.Abs(p.Z)-fin[2])/(fin[3]-fin[2]), 0, 1)
			leading := root + fin[4]*math.Pow(1-u, .6) - .11*u*u
			trailing := root - fin[5]*math.Pow(1-u, .8) - .11*u*u
			cx := (leading + trailing) / 2
			half := math.Max(.008, (leading-trailing)/2)
			thick := .040*math.Pow(math.Max(.02, 1-u), .45) + .005
			wing := (math.Sqrt(math.Pow((p.X-cx)/half, 2)+math.Pow(p.Y/thick, 2)) - 1) * math.Min(half, thick)
			wing = math.Max(wing, math.Max(fin[2]-math.Abs(p.Z), math.Abs(p.Z)-fin[3]))
			d = geometry.SmoothMin(d, wing, .05)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		u := clamp((math.Abs(p.Z)-.18)/.34, 0, 1)
		u = u * u * (3 - 2*u)
		wf := math.Exp(-math.Pow((p.X-profile.ContinuousFins[0])/.31, 4))
		wh := math.Exp(-math.Pow((p.X-profile.ContinuousFins[1])/.31, 4))
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
	skin, e := geometry.FieldMeshFine(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, profile.Scale)
	coat := &thing.Material{Name: "pliosaur clay", Texture: pliosaurContinuousTexture{}, Gloss: .14, Satin: .5}
	dark := &thing.Material{Name: "eye", Colour: maths.RGB(.025, .027, .023), Gloss: .28}
	ivory := &thing.Material{Name: "inferred teeth", Colour: maths.RGB(.7, .67, .56), Gloss: .1}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, side := range []float64{-1, 1} {
		lo, hi := 0., .24
		lo, hi = maths.Bisect(lo, hi, 28, func(z float64) bool { return field(maths.V(.87, .025, side*z)) < 0 })
		at := warp(maths.V(.87, .025, side*((lo+hi)/2+.002)))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .012}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		for j := 0; j < 8; j++ {
			x := .95 + .044*float64(j)
			lo, hi = 0, .18
			lo, hi = maths.Bisect(lo, hi, 25, func(z float64) bool { return field(maths.V(x, -.025, side*z)) < 0 })
			root := warp(maths.V(x, -.027, side*((lo+hi)/2-.003)))
			tooth, e := reptiles.Tooth(root, .006, .033, 12, .004)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type pliosaurContinuousTexture struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and a grey-green elsewhere.
func (pliosaurContinuousTexture) At(p maths.Vec) maths.Light {
	if p.X > .73 && p.X < 1.33 && p.Y < -.022 && p.Y > -.028 && math.Abs(p.Z) > .025 {
		return maths.RGB(.055, .065, .06)
	}
	return maths.RGB(.31, .35, .32)
}
