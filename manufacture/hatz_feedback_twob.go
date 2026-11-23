package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

type hatz2BPoint struct{ x, y float64 }

// Signed distance to an independently authored 2-D silhouette polygon.
func hatz2BPolygon(x, y float64, poly []hatz2BPoint) float64 {
	inside := false
	distance := math.Inf(1)
	for i, a := range poly {
		b := poly[(i+1)%len(poly)]
		if (a.y > y) != (b.y > y) && x < (b.x-a.x)*(y-a.y)/(b.y-a.y)+a.x {
			inside = !inside
		}
		dx, dy := b.x-a.x, b.y-a.y
		t := math.Max(0, math.Min(1, ((x-a.x)*dx+(y-a.y)*dy)/(dx*dx+dy*dy)))
		distance = math.Min(distance, math.Hypot(x-a.x-t*dx, y-a.y-t*dy))
	}
	if inside {
		return -distance
	}
	return distance
}

// hatz2BBody is the body of study 2B: the solid common to three outlines,
// one from the side, one from above and one from the front.
func hatz2BBody() (*shape.Mesh, error) {
	side := []hatz2BPoint{{-1.05, 0}, {-.80, .20}, {-.28, .24}, {.20, .20}, {.42, .13}, {.72, .12}, {.84, .18}, {1.08, .16}, {1.48, 0}, {1.08, -.09}, {.84, -.10}, {.72, -.08}, {.42, -.09}, {.20, -.15}, {-.28, -.20}, {-.85, -.15}}
	top := []hatz2BPoint{{-1.05, 0}, {-.80, .16}, {-.28, .20}, {.20, .16}, {.42, .09}, {.72, .075}, {.84, .12}, {1.08, .09}, {1.48, 0}, {1.08, -.09}, {.84, -.12}, {.72, -.075}, {.42, -.09}, {.20, -.16}, {-.28, -.20}, {-.80, -.16}}
	front := []hatz2BPoint{{-.24, 0}, {-.16, .16}, {0, .23}, {.16, .16}, {.24, 0}, {.16, -.16}, {0, -.20}, {-.16, -.16}}
	field := func(p maths.Vec) float64 {
		return math.Max(hatz2BPolygon(p.X, p.Y, side), math.Max(hatz2BPolygon(p.X, p.Z, top), hatz2BPolygon(p.Z, p.Y, front)))
	}
	return bioformFieldMesh(field, func(p maths.Vec) maths.Vec { return p.Scale(2) })
}

// hatz2BWing is one wing of study 2B at a phase in radians; side is the
// sign of z.
func hatz2BWing(phase, side float64) (*shape.Mesh, error) {
	// Local wing coordinates: x is chord, y thickness, z span. Independent
	// tapered analytic chord, extracted in rest space before posing.
	field := func(p maths.Vec) float64 {
		s := (p.Z + .68) / 1.36
		t := math.Max(0, math.Min(1, s))
		leading := .18 - .12*t
		trailing := -.55 + .40*t
		half := math.Max(.018, (leading-trailing)*.5)
		center := (leading + trailing) * .5
		return math.Max(math.Abs(p.Y)-.045, math.Max(math.Abs(p.X-center)-half, math.Max(-p.Z-.68, p.Z-.68)))
	}
	// Exact circular-arc centerline and transported normal; no material solver.
	warp := func(p maths.Vec) maths.Vec {
		s := (p.Z + .68) / 1.36 * 4.5
		k := .22 * math.Sin(phase)
		y, z := 0., s
		if math.Abs(k) > 1e-8 {
			y = (1 - math.Cos(k*s)) / k
			z = math.Sin(k*s) / k
		}
		a := k * s
		return maths.V(.10+2*p.X, y+2*p.Y*math.Cos(a), (.30 + z - 2*p.Y*math.Sin(a)))
	}
	m, err := bioformFieldMesh(field, warp)
	if err != nil || side > 0 {
		return m, err
	}
	v := append([]maths.Vec{}, m.Vertices...)
	f := append([][3]int{}, m.Faces...)
	for i := range v {
		v[i].Z = -v[i].Z
	}
	for i := range f {
		f[i][1], f[i][2] = f[i][2], f[i][1]
	}
	n := make([]maths.Vec, len(v))
	for _, t := range f {
		q := v[t[1]].Sub(v[t[0]]).Cross(v[t[2]].Sub(v[t[0]]))
		for _, i := range t {
			n[i] = n[i].Add(q)
		}
	}
	return shape.NewMesh(v, f, n, true)
}

// HatzFeedback2B is study 2B, a pterosaur, at a phase of its wing stroke
// in radians, placed at a place. It refuses a phase that is not finite.
func HatzFeedback2B(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(phase) || math.IsInf(phase, 0) {
		return nil, fmt.Errorf("hatz 02b: nonfinite phase")
	}
	body, err := hatz2BBody()
	if err != nil {
		return nil, err
	}
	hide := &thing.Material{Name: "hatz 02b visual hull", Colour: maths.RGB(.43, .34, .27), Satin: .42, Gloss: .06}
	membrane := &thing.Material{Name: "hatz 02b curved membrane", Colour: maths.RGB(.31, .24, .19), Satin: .4, Gloss: .05}
	eye := &thing.Material{Name: "hatz 02b eye", Colour: maths.RGB(.015, .014, .012), Gloss: .2}
	parts := []thing.Thing{{Shape: body, Material: hide, Motion: thing.Still(at)}}
	for _, side := range []float64{-1, 1} {
		wing, e := hatz2BWing(phase, side)
		if e != nil {
			return nil, e
		}
		parts = append(parts, thing.Thing{Shape: wing, Material: membrane, Motion: thing.Still(at)})
		parts = append(parts, thing.Thing{Shape: shape.Sphere{Radius: .043}, Material: eye, Motion: thing.Still(maths.NewSubspace(maths.V(1.87, .13, side*.18), maths.Identity, 1).Within(at))})
	}
	return parts, nil
}
