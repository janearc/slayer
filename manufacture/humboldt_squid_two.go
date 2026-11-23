package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"sync"
)

type squidTwoCurve struct {
	a, b, c maths.Vec
	radius  float64
	club    bool
}

// A new bounded field sculpture; coords are two per world metre.
var squidTwoRest = sync.OnceValues(func() (*shape.Mesh, error) {
	curves := []squidTwoCurve{}
	for k := 0; k < 10; k++ {
		a := 2 * math.Pi * float64(k%8) / 8
		if k >= 8 {
			a = math.Pi/4 + math.Pi*float64(k-8)
		}
		r := func(x, d float64) maths.Vec { return maths.V(x, d*math.Sin(a), d*math.Cos(a)) }
		length := .85
		club := k >= 8
		if club {
			length = 1.30
		}
		curves = append(curves, squidTwoCurve{r(.27, .10), r(.27+length*.55, .19), r(.27+length, .28), .055, club})
	}
	segment := func(p, a, b maths.Vec, ra, rb float64) float64 {
		v := b.Sub(a)
		u := math.Max(0, math.Min(1, p.Sub(a).Dot(v)/v.Dot(v)))
		return p.Sub(a.Add(v.Scale(u))).Len() - (ra + (rb-ra)*u)
	}
	soft := func(a, b, k float64) float64 {
		h := math.Max(k-math.Abs(a-b), 0) / k
		return math.Min(a, b) - k*h*h*.25
	}
	field := func(p maths.Vec) float64 {
		// Head and mantle are different masses; a shallow collar impression is
		// an artistic surface detail, not an open or anatomically complete cavity.
		x := p.X
		r := math.Hypot(p.Y, p.Z)
		u := math.Max(0, math.Min(1, -x/1.60))
		radius := .275 * math.Sqrt(math.Max(.003, 1-u*u))
		mantle := math.Max(r-radius, math.Max(-1.60-x, x-.02))
		head := math.Sqrt(math.Pow((x-.13)/.27, 2)+math.Pow(p.Y/.21, 2)+math.Pow(p.Z/.23, 2))*.21 - .21
		d := soft(mantle, head, .08)
		// Broad tapering fin masses, approximated by a diamond planform and thin
		// elliptical cross-section, join the mantle but do not model muscle fibres.
		span := math.Max(0, 1-math.Abs((x+1.24)/.36)) * .46
		fin := math.Max(math.Abs(p.Z)-span, math.Max(math.Abs(p.Y)-.022, math.Abs(x+1.24)-.36))
		d = soft(d, fin, .035)
		for _, c := range curves {
			first := segment(p, c.a, c.b, c.radius, c.radius*.65)
			second := segment(p, c.b, c.c, c.radius*.65, c.radius*.26)
			d = soft(d, math.Min(first, second), .025)
			if c.club {
				v := p.Sub(c.c)
				club := math.Sqrt(math.Pow(v.X/.12, 2)+math.Pow(v.Y/.045, 2)+math.Pow(v.Z/.045, 2))*.045 - .045
				d = soft(d, club, .02)
			}
		}
		// Shallow ventral funnel mass; no lumen or expelled jet is represented.
		funnel := segment(p, maths.V(.03, -.18, 0), maths.V(.21, -.28, 0), .065, .025)
		return soft(d, funnel, .025)
	}
	return bioformFieldMeshFine(field, func(p maths.Vec) maths.Vec { return p })
})

type squidTwoSkin struct{}

// At is the skin's colour at a point in rest coordinates: dark marks at the
// side of the head, and a mottled red elsewhere.
func (squidTwoSkin) At(p maths.Vec) maths.Light {
	// Posed mesh supplies rest coordinates; these marks are artistic, not data.
	n := noise.Fractal(823, p.X*23, p.Y*23, p.Z*23, 3)
	if p.X > .005 && p.X < .14 && math.Abs(p.Z) > .075 && p.Y > .015 && p.Y < .065 {
		return maths.RGB(.012, .015, .018)
	}
	return maths.RGB(.36+.035*n, .12+.018*n, .115+.016*n)
}

// HumboldtSquidTwo sculpts a single rest-space field and deforms indexed skin.
// The inferred swimming and pigment are visual hypotheses only.
func HumboldtSquidTwo(phase float64) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("squid 02: invalid phase")
	}
	raw, err := squidTwoRest()
	if err != nil {
		return nil, err
	}
	pose := func(p maths.Vec) maths.Vec {
		q := p
		// Radial mantle pulse; this does not solve water volume or jet thrust.
		if p.X < 0 {
			u := math.Max(0, math.Min(1, -p.X/1.6))
			f := 1 + .09*math.Sin(phase)*math.Sin(math.Pi*u)
			q.Y *= f
			q.Z *= f
		}
		// The fin and appendages bend with independent smooth weights.
		fin := math.Exp(-math.Pow((p.X+1.24)/.42, 4)) * math.Max(0, math.Min(1, (math.Abs(p.Z)-.12)/.30))
		q.Y += .15 * fin * math.Sin(phase+.4)
		arm := math.Max(0, math.Min(1, (p.X-.22)/1.2))
		q.Y += .12 * arm * math.Sin(phase+p.Z*5)
		q.Z += .07 * arm * math.Sin(phase+.6+p.Y*4)
		return q.Scale(.5)
	}
	posed, rest := make([]maths.Vec, len(raw.Vertices)), make([]maths.Vec, len(raw.Vertices))
	for i, p := range raw.Vertices {
		posed[i] = pose(p)
		rest[i] = p.Scale(.5)
	}
	shell, err := mesh.NewWithRestPoints(posed, rest, raw.Faces, sealSmoothNormals(posed, raw.Faces), true)
	if err != nil {
		return nil, err
	}
	hide := &thing.Material{Name: "squid 02 skin", Texture: squidTwoSkin{}, Colour: maths.RGB(.36, .12, .115), Satin: .47, Gloss: .14}
	ivory := &thing.Material{Name: "illustrative sucker beads", Colour: maths.RGB(.57, .38, .32), Satin: .4}
	out := []thing.Thing{{Shape: shell, Material: hide, Motion: thing.Still(maths.Local())}}
	for k := 0; k < 10; k++ {
		a := 2 * math.Pi * float64(k%8) / 8
		if k >= 8 {
			a = math.Pi/4 + math.Pi*float64(k-8)
		}
		length := .85
		if k >= 8 {
			length = 1.30
		}
		for j := 0; j < 10; j++ {
			u := .15 + .065*float64(j)
			if k >= 8 {
				u = .82 + .017*float64(j)
			}
			rad := .10 + .18*u - .038
			p := maths.V(.27+length*u, rad*math.Sin(a), rad*math.Cos(a))
			out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .004}, Material: ivory, Motion: thing.Still(maths.NewSubspace(pose(p), maths.Identity, 1))})
		}
	}
	return out, nil
}
