package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// WalrusFresh4 builds new masses from a varying axial profile and tapered
// capsule appendages; attachment roots are solved against that field.
func WalrusFresh4(yaw, phase float64) ([]thing.Thing, error) {
	clamp := func(x, a, b float64) float64 { return math.Max(a, math.Min(b, x)) }
	smooth := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	capsule := func(p, a, b, scale maths.Vec, r float64) float64 {
		p = maths.V(p.X/scale.X, p.Y/scale.Y, p.Z/scale.Z)
		a = maths.V(a.X/scale.X, a.Y/scale.Y, a.Z/scale.Z)
		b = maths.V(b.X/scale.X, b.Y/scale.Y, b.Z/scale.Z)
		ba := b.Sub(a)
		pa := p.Sub(a)
		t := clamp(pa.Dot(ba)/ba.Dot(ba), 0, 1)
		return (pa.Sub(ba.Scale(t)).Len() - r) * math.Min(scale.X, math.Min(scale.Y, scale.Z))
	}
	field := func(p maths.Vec) float64 {
		t := clamp((p.X+1.50)/3, 0, 1)
		radius := .46 * math.Pow(math.Max(.012, math.Sin(math.Pi*t)), .34)
		if t > .80 {
			radius = math.Max(radius, .255)
		}
		// Slightly nonuniform paired folds across the broad neck.
		if t > .66 && t < .82 {
			radius += .012 * math.Sin((t-.66)*95) * math.Sin(math.Pi*(t-.66)/.16)
		}
		d := math.Max(math.Sqrt(p.Y*p.Y/.90+p.Z*p.Z)-radius, math.Max(-1.50-p.X, p.X-1.50))
		for _, s := range []float64{-1, 1} {
			// Broad bilobed upper lip merges into the axial skin.
			d = smooth(d, capsule(p, maths.V(1.25, -.05, s*.13), maths.V(1.47, -.07, s*.13), maths.V(1, .78, .82), .19), .055)
			// Short foreflippers at the shoulder, and two individually separated hind paddles.
			d = smooth(d, capsule(p, maths.V(.34, -.20, s*.35), maths.V(.14, -.24, s*.73), maths.V(1, .32, .72), .20), .075)
			d = smooth(d, capsule(p, maths.V(-1.30, -.11, s*.15), maths.V(-1.60, -.13, s*.29), maths.V(1, .42, .62), .16), .05)
		}
		return d
	}
	warp := func(p maths.Vec) maths.Vec {
		s := 1.5 - p.X
		if p.X < -1.12 {
			weight := clamp((-1.12-p.X)/.48, 0, 1)
			p.Y += .10 * math.Sin(phase) * weight
		}
		if math.Abs(yaw) < 1e-9 {
			return p
		}
		k := yaw / 3
		a := k * s
		return maths.V(1.5-math.Sin(a)/k+p.Z*math.Sin(a), p.Y, (1-math.Cos(a))/k+p.Z*math.Cos(a))
	}
	skin, e := bioformFieldMesh(field, warp)
	if e != nil {
		return nil, e
	}
	coat := &thing.Material{Name: "walrus skin", Colour: maths.RGB(.30, .185, .14), Gloss: .13, Satin: .59}
	ivory := &thing.Material{Name: "ivory", Colour: maths.RGB(.73, .65, .49), Gloss: .15, Satin: .42}
	eye := &thing.Material{Name: "eye", Colour: maths.RGB(.018, .013, .011), Gloss: .4}
	parts := []thing.Thing{{Shape: skin, Material: coat, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))}}
	for _, side := range []float64{-1, 1} {
		// Solve for the surface on a ray normal to the body side, then place
		// the eye slightly outside it. This avoids the buried-eye failure in v3.
		lo, hi := 0., .8
		for i := 0; i < 30; i++ {
			z := (lo + hi) / 2
			if field(maths.V(1.03, .13, side*z)) < 0 {
				lo = z
			} else {
				hi = z
			}
		}
		at := warp(maths.V(1.03, .13, side*((lo+hi)/2+.011)))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .028}, Material: eye, Motion: thing.Still(maths.NewSubspace(at, maths.Identity, 1))})
		nlo, nhi := 1.0, 1.7
		for i := 0; i < 30; i++ {
			x := (nlo + nhi) / 2
			if field(maths.V(x, .11, side*.055)) < 0 {
				nlo = x
			} else {
				nhi = x
			}
		}
		nostril := warp(maths.V((nlo+nhi)/2+.006, .11, side*.055))
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .014}, Material: eye, Motion: thing.Still(maths.NewSubspace(nostril, maths.Identity, 1))})
		tooth, e := walrusFresh4Tooth(side, warp)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: tooth, Material: ivory, Motion: thing.Still(maths.NewSubspace(maths.Vec{}, maths.Identity, 1))})
		// Deterministically scattered, short vibrissae start at the actual field.
		for i := 0; i < 86; i++ {
			u := math.Mod(float64(i)*.61803398875, 1)
			v := math.Mod(float64(i)*.7548776662, 1)
			y := .06 - .23*u
			z := side * (.025 + .20*v)
			a, b := 1.0, 1.7
			if field(maths.V(a, y, z)) >= 0 {
				continue
			}
			for j := 0; j < 24; j++ {
				mid := (a + b) / 2
				if field(maths.V(mid, y, z)) < 0 {
					a = mid
				} else {
					b = mid
				}
			}
			root := warp(maths.V((a+b)/2, y, z))
			tip := warp(maths.V((a+b)/2+.055+.025*v, y-.025-.055*u, z+side*(.025+.045*v)))
			axis := tip.Sub(root)
			parts = append(parts, thing.Thing{Shape: shape.Tube{Radius: .0008, Thick: .0016, Height: axis.Len()}, Material: ivory, Motion: thing.Still(maths.NewSubspace(root.Add(tip).Scale(.5), maths.Upright(axis), 1))})
		}
	}
	return parts, nil
}

// walrusFresh4Tooth is the mesh of one tusk, curved from its root to its
// tip and passed through warp; side is the sign of z.
func walrusFresh4Tooth(side float64, warp func(maths.Vec) maths.Vec) (*shape.Mesh, error) {
	const n, m = 18, 20
	v := []maths.Vec{}
	f := [][3]int{}
	point := func(t float64) maths.Vec { return maths.V(1.39+.07*t*t, -.14-.50*t, side*(.13+.025*t)) }
	for i := 0; i <= n; i++ {
		t := float64(i) / n
		r := .055 * math.Pow(1-.98*t, .7)
		c := point(t)
		for j := 0; j < m; j++ {
			a := 2 * math.Pi * float64(j) / m
			v = append(v, warp(c.Add(maths.V(r*math.Cos(a), 0, r*math.Sin(a)))))
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			a := i*m + j
			b := i*m + (j+1)%m
			c := (i+1)*m + (j+1)%m
			d := (i+1)*m + j
			f = append(f, [3]int{a, b, c}, [3]int{a, c, d})
		}
	}
	first := len(v)
	v = append(v, warp(point(0)))
	last := len(v)
	v = append(v, warp(point(1)))
	for j := 0; j < m; j++ {
		k := (j + 1) % m
		f = append(f, [3]int{first, k, j}, [3]int{last, n*m + j, n*m + k})
	}
	return shape.NewMesh(v, f, nil, false)
}
