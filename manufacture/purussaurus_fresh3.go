package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PurussaurusFresh3 is an original station-surface skin with smoothly weighted
// limb controls. Extinct soft-tissue shape and locomotion remain conjectural.
func PurussaurusFresh3(phase, yaw, posture float64) ([]thing.Thing, error) {
	type section struct{ x, ry, rz float64 }
	stations := []section{{-1.72, .012, .010}, {-1.46, .065, .045}, {-1.0, .14, .10}, {-.45, .21, .17}, {-.24, .235, .27}, {.10, .24, .33}, {.50, .22, .29}, {.78, .15, .20}, {.90, .105, .155}, {1.03, .095, .147}, {1.25, .086, .142}, {1.39, .070, .12}, {1.42, .03, .07}}
	clamp := func(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
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
	radii := func(x float64) (float64, float64) {
		for i := 1; i < len(stations); i++ {
			if x <= stations[i].x {
				a, b := stations[i-1], stations[i]
				t := clamp((x-a.x)/(b.x-a.x), 0, 1)
				return a.ry*(1-t) + b.ry*t, a.rz*(1-t) + b.rz*t
			}
		}
		a := stations[len(stations)-1]
		return a.ry, a.rz
	}
	field := func(p maths.Vec) float64 {
		ry, rz := radii(p.X)
		rad := (math.Sqrt((p.Y/ry)*(p.Y/ry)+(p.Z/rz)*(p.Z/rz)) - 1) * math.Min(ry, rz)
		d := math.Max(rad, math.Max(-1.72-p.X, p.X-1.42))
		for _, s := range []float64{-1, 1} {
			// Eye mound and snout skin are not claims about fossil soft tissue.
			d = blend(d, mass(p, maths.V(1.00, .075, s*.115), maths.V(.065, .045, .052)), .018)
			for _, x := range []float64{.43, -.40} {
				d = blend(d, mass(p, maths.V(x, -.10, s*.30), maths.V(.16, .10, .14)), .065)
				d = blend(d, mass(p, maths.V(x-.04, -.17, s*.43), maths.V(.12, .085, .12)), .045)
				d = blend(d, mass(p, maths.V(x+.02, -.24, s*.54), maths.V(.11, .065, .13)), .04)
				for toe := 0; toe < 4; toe++ {
					z := s * (.47 + .047*float64(toe))
					d = blend(d, mass(p, maths.V(x+.12, -.255, z), maths.V(.095, .038, .026)), .018)
				}
			}
		}
		// Small low dorsal ridges are an analogue rather than fossil plate maps.
		if p.X > -.35 && p.X < .70 && p.Y > .13 {
			a := math.Sin((p.X + .35) * 38)
			b := math.Sin(p.Z * 20)
			d -= .007 * a * a * b * b * clamp((p.Y-.13)/.08, 0, 1)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		w := clamp((-.40-p.X)/1.32, 0, 1)
		bend := .11*w*w*math.Sin(phase-4*w) + .20*yaw*w*w
		d := (-1. / 1.32) * (.22*w*math.Sin(phase-4*w) - .44*w*w*math.Cos(phase-4*w) + .40*yaw*w)
		ang := math.Atan(d)
		p.X, p.Z = p.X-p.Z*math.Sin(ang), bend+p.Z*math.Cos(ang)
		// Proximal limb stays near body; distal control shifts ground and swim poses.
		z := math.Abs(p.Z)
		distal := clamp((z-.29)/.28, 0, 1)
		wX := math.Max(math.Exp(-math.Pow((p.X-.43)/.22, 2)), math.Exp(-math.Pow((p.X+.40)/.22, 2)))
		weight := distal * wX
		if posture > 0 {
			p.Y -= .035 * weight
			p.X += .035 * weight
		}
		if posture < 0 {
			side := 1.
			if p.Z < 0 {
				side = -1
			}
			p.Z -= side * .055 * weight
			p.X -= .03 * weight
		}
		return p
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, 2.8)
	coat := &thing.Material{Name: "procedural caiman skin", Texture: purussaurusSkin3{}, Gloss: .08, Satin: .35}
	dark := &thing.Material{Name: "eye and nostril", Colour: maths.RGB(.02, .018, .009), Gloss: .18}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, s := range []float64{-1, 1} {
		for _, a := range []struct{ x, z, r float64 }{{1.00, .115, .010}, {1.33, .047, .007}} {
			lo, hi := 0., .25
			for i := 0; i < 28; i++ {
				y := (lo + hi) / 2
				if field(maths.V(a.x, y, s*a.z)) < 0 {
					lo = y
				} else {
					hi = y
				}
			}
			at := warp(maths.V(a.x, (lo+hi)/2+.002, s*a.z))
			parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: a.r}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		}
	}
	return parts, nil
}

type purussaurusSkin3 struct{}

// At is the skin's colour at a point: a dark mark on each side of the
// head, and green elsewhere.
func (purussaurusSkin3) At(p maths.Vec) maths.Light {
	if p.X > .97 && p.X < 1.4 && p.Y < -.020 && p.Y > -.029 && math.Abs(p.Z) > .10 {
		return maths.RGB(.045, .045, .025)
	}
	return maths.RGB(.19, .215, .125)
}
