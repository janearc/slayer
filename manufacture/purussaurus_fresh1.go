package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PurussaurusFresh1 is a sourced, original first-stage caiman study. Its
// single implicit skin is posed by a traveling lateral tail displacement.
func PurussaurusFresh1(phase, yaw, ground float64) ([]thing.Thing, error) {
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
	field := func(p maths.Vec) float64 {
		d := ell(p, maths.V(.12, 0, 0), maths.V(.67, .24, .34))
		d = blend(d, ell(p, maths.V(.79, 0, 0), maths.V(.24, .14, .18)), .08)
		// The reference skull is broad and low; this oval is intentionally a
		// coarse first hypothesis rather than a traced or imported skull.
		d = blend(d, ell(p, maths.V(1.15, -.02, 0), maths.V(.25, .085, .1465)), .055)
		for i := 0; i < 5; i++ {
			t := float64(i) / 4
			x := -.61 - .23*float64(i)
			d = blend(d, ell(p, maths.V(x, 0, 0), maths.V(.28, .20*(1-.78*t), .16*(1-.82*t))), .08)
		}
		for _, s := range []float64{-1, 1} {
			for _, x := range []float64{.43, -.42} {
				d = blend(d, ell(p, maths.V(x, -.13, s*.37), maths.V(.14, .12, .22)), .08)
			}
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		w := math.Max(0, math.Min(1, (-.36-p.X)/1.35))
		p.Z += .16*w*w*math.Sin(phase-4*w) + yaw*w*w*.30
		leg := math.Max(0, math.Min(1, (math.Abs(p.Z)-.28)/.23))
		p.Y -= ground * .04 * leg
		return p
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 2.8)
	skinmat := &thing.Material{Name: "authored caiman skin", Colour: maths.RGB(.20, .225, .13), Gloss: .08, Satin: .35}
	eyes := &thing.Material{Name: "illustrative eye", Colour: maths.RGB(.025, .020, .009), Gloss: .25}
	parts := []thing.Thing{{Shape: skin, Material: skinmat, Motion: thing.Still(place)}}
	for _, s := range []float64{-1, 1} {
		at := warp(maths.V(1.06, .07, s*.12))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .018}, Material: eyes, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
	}
	return parts, nil
}
