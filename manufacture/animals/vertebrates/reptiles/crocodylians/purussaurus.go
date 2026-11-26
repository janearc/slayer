package crocodylians

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// NewCrocodylian builds this profile on its authored family scaffold.
func NewCrocodylian(profile reptiles.Profile, phase, yaw, posture float64) ([]thing.Thing, error) {
	type axial struct{ a, b, ry0, ry1, rz0, rz1 float64 }
	bones := make([]axial, 0, len(profile.Axial))
	for _, p := range profile.Axial {
		bones = append(bones, axial{p[0], p[1], p[2], p[3], p[4], p[5]})
	}
	clamp := func(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
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
			d = geometry.SmoothMin(d, axialField(p, b), .028)
		}
		for _, s := range []float64{-1, 1} {
			for _, x := range profile.LimbRoots {
				joint := func(i int) maths.Vec { v := profile.LimbPoints[i]; return maths.V(x+v[0], v[1], s*v[2]) }
				a, b, c := joint(0), joint(1), joint(2)
				d = geometry.SmoothMin(d, segment(p, a, b, profile.LimbRadii[0], profile.LimbRadii[1]), .04)
				d = geometry.SmoothMin(d, segment(p, b, c, profile.LimbRadii[1], profile.LimbRadii[2]), .035)
				for toe := 0; toe < 4; toe++ {
					z := s * (.46 + .052*float64(toe))
					from := maths.V(x+.01, -.245, z)
					to := maths.V(x+.14, -.26, z+s*.02)
					d = geometry.SmoothMin(d, segment(p, from, to, .035, .023), .05)
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
		hipX := profile.LimbRoots[0]
		if math.Abs(p.X-profile.LimbRoots[1]) < math.Abs(p.X-profile.LimbRoots[0]) {
			hipX = profile.LimbRoots[1]
		}
		weight := clamp((z-.28)/.27, 0, 1) * math.Exp(-math.Pow((p.X-hipX)/.22, 2))
		theta := 0.
		if posture > 0 {
			theta = .12
		}
		if posture < 0 {
			theta = -.15
		}
		dy, dz := p.Y-profile.LimbPoints[0][1], z-profile.LimbPoints[0][2]
		c, s := math.Cos(theta), math.Sin(theta)
		p.Y += weight * (dy*c - dz*s - dy)
		p.Z += side * weight * (dy*s + dz*c - dz)
		return p
	}
	skin, e := geometry.FieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	place := maths.NewSubspace(maths.Vec{}, maths.Identity, profile.Scale)
	coat := &thing.Material{Name: "procedural caiman", Texture: purussaurusSkin{}, Gloss: .08, Satin: .35}
	dark := &thing.Material{Name: "eye and nostril", Colour: maths.RGB(.022, .02, .01), Gloss: .20}
	ivory := &thing.Material{Name: "hypothetical tooth", Colour: maths.RGB(.67, .63, .49), Gloss: .10}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(place)}}
	for _, s := range []float64{-1, 1} {
		for _, a := range []struct{ x, z, r float64 }{{1.00, .115, .010}, {1.31, .045, .006}} {
			lo, hi := 0., .25
			lo, hi = maths.Bisect(lo, hi, 28, func(y float64) bool { return field(maths.V(a.x, y, s*a.z)) < 0 })
			at := warp(maths.V(a.x, (lo+hi)/2+.001, s*a.z))
			parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: a.r}, Material: dark, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1).Within(place))})
		}
		for j := 0; j < 5; j++ {
			x := 1.06 + .065*float64(j)
			lo, hi := 0., .25
			lo, hi = maths.Bisect(lo, hi, 25, func(z float64) bool { return field(maths.V(x, -.035, s*z)) < 0 })
			root := warp(maths.V(x, -.045, s*((lo+hi)/2-.003)))
			tooth, e := reptiles.Tooth(root, .006, .035, 10, 0)
			if e != nil {
				return nil, e
			}
			parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(place)})
		}
	}
	return parts, nil
}

type purussaurusSkin struct{}

// At is the skin's colour at a point: dark marks on the head, a different
// tone along the back, and green elsewhere.
func (purussaurusSkin) At(p maths.Vec) maths.Light {
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
