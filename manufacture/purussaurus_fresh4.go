package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PurussaurusFresh4 constructs a fresh density-kernel skin with joint-centered
// soft-weighted limb rotations. Its proportions and skin are hypothetical.
func PurussaurusFresh4(phase, yaw, posture float64) ([]thing.Thing, error) {
	type lump struct {
		c, r maths.Vec
		s    float64
	}
	lumps := []lump{
		{maths.V(-.25, 0, 0), maths.V(.52, .28, .39), 1}, {maths.V(.20, 0, 0), maths.V(.58, .29, .41), 1}, {maths.V(.61, 0, 0), maths.V(.35, .23, .31), .9},
		{maths.V(.84, 0, 0), maths.V(.25, .16, .21), .9}, {maths.V(1.16, -.01, 0), maths.V(.31, .105, .18), 1.4},
	}
	for i := 0; i < 6; i++ {
		t := float64(i) / 5
		lumps = append(lumps, lump{maths.V(-.54-.20*float64(i), 0, 0), maths.V(.27, .235*(1-.82*t), .18*(1-.82*t)), 1})
	}
	for _, s := range []float64{-1, 1} {
		for _, x := range []float64{.43, -.40} {
			lumps = append(lumps, lump{maths.V(x, -.09, s*.28), maths.V(.17, .12, .16), 1}, lump{maths.V(x-.04, -.17, s*.42), maths.V(.13, .10, .13), .9}, lump{maths.V(x+.03, -.24, s*.53), maths.V(.13, .07, .14), 1})
			for toe := 0; toe < 4; toe++ {
				z := s * (.47 + .045*float64(toe))
				lumps = append(lumps, lump{maths.V(x+.14, -.25, z), maths.V(.08, .045, .027), .75})
			}
		}
	}
	field := func(p maths.Vec) float64 {
		density := 0.
		for _, m := range lumps {
			x := (p.X - m.c.X) / m.r.X
			y := (p.Y - m.c.Y) / m.r.Y
			z := (p.Z - m.c.Z) / m.r.Z
			density += m.s * math.Exp(-2.1*(x*x+y*y+z*z))
		}
		d := .34 - density
		if p.Y > .15 && p.X > -.35 && p.X < .70 {
			u := math.Cos(37 * p.X)
			v := math.Cos(23 * p.Z)
			d -= .012 * u * u * u * u * v * v * v * v * math.Min(1, (p.Y-.15)/.07)
		}
		return d
	}
	clamp := func(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
	warp := func(p maths.Vec) maths.Vec {
		w := clamp((-.40-p.X)/1.34, 0, 1)
		shift := .075*w*w*math.Sin(phase-4*w) + .16*yaw*w*w
		grad := (-1. / 1.34) * (.15*w*math.Sin(phase-4*w) - .30*w*w*math.Cos(phase-4*w) + .32*yaw*w)
		a := math.Atan(grad)
		p.X, p.Z = p.X-p.Z*math.Sin(a), shift+p.Z*math.Cos(a)
		s := 1.
		if p.Z < 0 {
			s = -1
		}
		z := math.Abs(p.Z)
		weight := clamp((z-.28)/.28, 0, 1) * math.Max(math.Exp(-math.Pow((p.X-.43)/.23, 2)), math.Exp(-math.Pow((p.X+.40)/.23, 2)))
		theta := 0.
		if posture > 0 {
			theta = .12
		}
		if posture < 0 {
			theta = -.12
		}
		dy, dz := p.Y+.09, z-.28
		c, q := math.Cos(theta), math.Sin(theta)
		p.Y += weight * (dy*c - dz*q - dy)
		p.Z += s * weight * (dy*q + dz*c - dz)
		return p
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 2.8)
	coat := &thing.Material{Name: "caiman skin", Texture: purussaurusSkin4{}, Gloss: .08, Satin: .35}
	dark := &thing.Material{Name: "eye and nostril", Colour: maths.RGB(.018, .017, .009), Gloss: .18}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, side := range []float64{-1, 1} {
		for _, a := range []struct{ x, z, r float64 }{{1.01, .115, .009}, {1.33, .045, .007}} {
			lo, hi := 0., .28
			for i := 0; i < 28; i++ {
				y := (lo + hi) / 2
				if field(maths.V(a.x, y, side*a.z)) < 0 {
					lo = y
				} else {
					hi = y
				}
			}
			at := warp(maths.V(a.x, (lo+hi)/2+.001, side*a.z))
			parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: a.r}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		}
	}
	return parts, nil
}

type purussaurusSkin4 struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and green elsewhere.
func (purussaurusSkin4) At(p maths.Vec) maths.Light {
	if p.X > .98 && p.X < 1.39 && p.Y > -.032 && p.Y < -.023 && math.Abs(p.Z) > .10 {
		return maths.RGB(.04, .045, .023)
	}
	return maths.RGB(.18, .21, .12)
}
