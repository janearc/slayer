package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PurussaurusFresh2 is a fresh varying-section axial surface with capsule
// limb chains; skull, soft tissue and kinematics are illustrative.
func PurussaurusFresh2(phase, yaw, ground float64) ([]thing.Thing, error) {
	clamp := func(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
	mix := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	capsule := func(p, a, b maths.Vec, r float64) float64 {
		ab := b.Sub(a)
		t := clamp(p.Sub(a).Dot(ab)/ab.Dot(ab), 0, 1)
		return p.Sub(a.Add(ab.Scale(t))).Len() - r
	}
	super := func(p, c, r maths.Vec) float64 {
		x := (p.X - c.X) / r.X
		y := (p.Y - c.Y) / r.Y
		z := (p.Z - c.Z) / r.Z
		return (math.Pow(math.Pow(x, 4)+math.Pow(y, 4)+math.Pow(z, 4), .25) - 1) * math.Min(r.X, math.Min(r.Y, r.Z))
	}
	field := func(p maths.Vec) float64 {
		d := super(p, maths.V(.12, 0, 0), maths.V(.67, .23, .34))
		t := clamp((p.X+1.68)/1.28, 0, 1)
		ry := .012 + .20*math.Pow(t, .75)
		rz := .009 + .15*math.Pow(t, .8)
		rad := (math.Sqrt((p.Y/ry)*(p.Y/ry)+(p.Z/rz)*(p.Z/rz)) - 1) * math.Min(ry, rz)
		tail := math.Max(rad, math.Max(-1.68-p.X, p.X+.40))
		d = mix(d, tail, .09)
		d = mix(d, super(p, maths.V(.79, 0, 0), maths.V(.25, .13, .19)), .08)
		d = mix(d, super(p, maths.V(1.15, -.015, 0), maths.V(.25, .085, .1465)), .045)
		for _, s := range []float64{-1, 1} {
			for _, x := range []float64{.43, -.42} {
				hip := maths.V(x, -.08, s*.26)
				elbow := maths.V(x-.05, -.17, s*.43)
				wrist := maths.V(x+.02, -.25, s*.56)
				d = mix(d, capsule(p, hip, elbow, .095), .05)
				d = mix(d, capsule(p, elbow, wrist, .072), .045)
				for toe := 0; toe < 4; toe++ {
					z := s * (.48 + .052*float64(toe))
					start := maths.V(x+.01, -.255, z)
					tip := maths.V(x+.16, -.267, z+s*.025)
					d = mix(d, capsule(p, start, tip, .030), .025)
				}
			}
		}
		// Low-amplitude patterned dorsal relief is an invented analogue.
		if p.Y > .13 && p.X > -.4 && p.X < .72 {
			u := math.Sin(42 * (p.X + .4))
			v := math.Sin(22 * p.Z)
			d -= .009 * u * u * v * v * clamp((p.Y-.13)/.07, 0, 1)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		w := clamp((-.38-p.X)/1.3, 0, 1)
		offset := .12*w*w*math.Sin(phase-4*w) + .22*yaw*w*w
		// A local tangent turns cross sections with the bent centerline.
		slope := -1. / 1.3 * (.24*w*math.Sin(phase-4*w) - .48*w*w*math.Cos(phase-4*w) + .44*yaw*w)
		ang := math.Atan(slope)
		x, z := p.X-p.Z*math.Sin(ang), offset+p.Z*math.Cos(ang)
		p.X, p.Z = x, z
		limb := clamp((math.Abs(p.Z)-.31)/.30, 0, 1)
		p.Y -= ground * .025 * limb
		return p
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 2.8)
	coat := &thing.Material{Name: "caiman skin study", Colour: maths.RGB(.19, .215, .125), Gloss: .08, Satin: .36}
	dark := &thing.Material{Name: "eyes and mouth", Colour: maths.RGB(.018, .018, .008), Gloss: .24}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, s := range []float64{-1, 1} {
		lo, hi := 0., .3
		for j := 0; j < 26; j++ {
			z := (lo + hi) / 2
			if field(maths.V(1.03, .045, s*z)) < 0 {
				lo = z
			} else {
				hi = z
			}
		}
		at := warp(maths.V(1.03, .045, s*((lo+hi)/2+.004)))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .013}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		lo, hi = 0, .2
		for j := 0; j < 26; j++ {
			y := (lo + hi) / 2
			if field(maths.V(1.32, y, s*.05)) < 0 {
				lo = y
			} else {
				hi = y
			}
		}
		at = warp(maths.V(1.32, (lo+hi)/2+.003, s*.05))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .010}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		a, b := warp(maths.V(1.17, -.055, s*.143)), warp(maths.V(1.38, -.055, s*.143))
		axis := b.Sub(a)
		parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .0025, Thick: .005, Height: axis.Len()}, Material: dark, Motion: thing.Still(maths.NewSubspace(a.Add(b).Scale(.5), maths.Upright(axis), 1).Within(place))})
	}
	return parts, nil
}
