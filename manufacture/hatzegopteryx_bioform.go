package manufacture

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// HatzegopteryxStudy makes an original, inferred giant azhdarchid. The body,
// limbs and wings are separate overlapping pieces. The area-constrained wing
// is not a stress, tissue, aerodynamics, or contact-force simulation.
func HatzegopteryxStudy(fold, seconds float64, at maths.Subspace) ([]thing.Thing, error) {
	if math.IsNaN(fold) || math.IsInf(fold, 0) || fold < 0 || fold > 1 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return nil, fmt.Errorf("hatzegopteryx: invalid pose")
	}
	hide := &thing.Material{Name: "invented hide", Colour: maths.RGB(.31, .28, .20), Gloss: .08, Satin: .44}
	membrane := &thing.Material{Name: "inferred membrane", Colour: maths.RGB(.30, .20, .13), Gloss: .08, Satin: .45}
	bill := &thing.Material{Name: "invented beak", Colour: maths.RGB(.23, .17, .12), Gloss: .10, Satin: .38}
	eye := &thing.Material{Name: "eye", Colour: maths.RGB(.012, .009, .007), Gloss: .30}
	body, err := hatzBody(seconds)
	if err != nil {
		return nil, err
	}
	items := []thing.Thing{{Shape: body, Material: hide, Motion: thing.Still(at)}}
	loft := func(c []maths.Vec, r [][2]float64, mat *thing.Material, head bool) error {
		var rings [][]maths.Vec
		for i, p := range c {
			axis := c[min(i+1, len(c)-1)].Sub(c[max(i-1, 0)]).Unit()
			a := axis.Cross(maths.V(0, 0, 1))
			if a.Len() < 1e-9 {
				a = axis.Cross(maths.Up)
			}
			a = a.Unit()
			b := axis.Cross(a).Unit()
			rings = append(rings, sealRing(p, a.Scale(r[i][0]), b.Scale(r[i][1]), func(v maths.Vec) maths.Vec {
				if head {
					return hatzNeckPose(v, seconds, 1)
				}
				return v
			}))
		}
		piece, e := sealPiece(rings, mat, at)
		if e != nil {
			return e
		}
		items = append(items, piece)
		return nil
	}
	if err := loft([]maths.Vec{maths.V(1.35, 3.9, 0), maths.V(1.7, 3.98, 0), maths.V(2.15, 3.88, 0), maths.V(2.9, 3.70, 0), maths.V(3.85, 3.46, 0)}, [][2]float64{{.15, .14}, {.32, .27}, {.27, .22}, {.14, .12}, {.012, .01}}, bill, true); err != nil {
		return nil, err
	}
	for _, side := range []float64{-1, 1} {
		eyeAt := hatzNeckPose(maths.V(1.8, 4.03, side*.25), seconds, 1)
		items = append(items, thing.Thing{Shape: shape.Sphere{Radius: .045}, Material: eye, Motion: thing.Still(maths.NewSubspace(eyeAt, maths.Identity, 1).Within(at))})
		shoulder, elbow, wrist, tip, inner := hatzWingControls(fold, seconds, side)
		if err := loft([]maths.Vec{shoulder, elbow, wrist, tip}, [][2]float64{{.13, .13}, {.10, .09}, {.075, .07}, {.015, .015}}, hide, false); err != nil {
			return nil, err
		}
		sheet, e := hatzWingSurface(shoulder, elbow, wrist, tip, inner, fold, side)
		if e != nil {
			return nil, e
		}
		items = append(items, thing.Thing{Shape: sheet, Material: membrane, Motion: thing.Still(at)})
		hip := maths.V(-.88, 2.55, side*.25)
		knee := maths.V(-1, 1.45+.45*(1-fold), side*.40)
		foot := maths.V(-.78, .10+1.35*(1-fold), side*.42)
		if err := loft([]maths.Vec{hip, knee, foot}, [][2]float64{{.16, .14}, {.10, .09}, {.07, .07}}, hide, false); err != nil {
			return nil, err
		}
		for d := -1; d <= 1; d++ {
			p := foot.Add(maths.V(0, 0, float64(d)*.09))
			q := p.Add(maths.V(.25, -.035, float64(d)*.02))
			if err := loft([]maths.Vec{p, q}, [][2]float64{{.05, .05}, {.01, .01}}, bill, false); err != nil {
				return nil, err
			}
		}
		if fold > .99 {
			items = append(items, thing.Thing{Shape: shape.Sphere{Radius: .1}, Material: hide, Motion: thing.Still(maths.NewSubspace(wrist, maths.Identity, 1).Within(at))})
		}
	}
	return items, nil
}

// hatzNeckPose bends a point of the neck about its base, swaying with the
// time in seconds, more as weight rises toward the head.
func hatzNeckPose(p maths.Vec, seconds, weight float64) maths.Vec {
	angle := .12 * weight * math.Sin(2*math.Pi*seconds/3)
	x, y := p.X-.65, p.Y-2.8
	return maths.V(.65+x*math.Cos(angle)-y*math.Sin(angle), 2.8+x*math.Sin(angle)+y*math.Cos(angle), p.Z)
}

// One continuous closed body/neck skin. Cross sections remain circular or
// elliptical approximations rather than measured biological tissue.
func hatzBody(seconds float64) (*mesh.Mesh, error) {
	c := []maths.Vec{maths.V(-1.25, 2.55, 0), maths.V(-.9, 2.62, 0), maths.V(-.35, 2.68, 0), maths.V(.3, 2.7, 0), maths.V(.65, 2.85, 0), maths.V(.9, 3.2, 0), maths.V(1.18, 3.6, 0), maths.V(1.45, 3.9, 0)}
	r := [][2]float64{{.04, .05}, {.31, .27}, {.45, .36}, {.38, .31}, {.25, .22}, {.19, .18}, {.16, .14}, {.14, .13}}
	const around = 24
	var rest, posed []maths.Vec
	var faces [][3]int
	for i, p := range c {
		axis := c[min(i+1, len(c)-1)].Sub(c[max(i-1, 0)]).Unit()
		u := axis.Cross(maths.V(0, 0, 1)).Unit()
		v := axis.Cross(u).Unit()
		for j := 0; j < around; j++ {
			a := 2 * math.Pi * float64(j) / around
			q := p.Add(u.Scale(r[i][0] * math.Cos(a))).Add(v.Scale(r[i][1] * math.Sin(a)))
			weight := math.Max(0, math.Min(1, (q.Y-2.8)/.9))
			rest = append(rest, q)
			posed = append(posed, hatzNeckPose(q, seconds, weight))
		}
	}
	for i := 0; i < len(c)-1; i++ {
		for j := 0; j < around; j++ {
			a := i*around + j
			b := i*around + (j+1)%around
			d := a + around
			z := b + around
			faces = append(faces, [3]int{a, b, z}, [3]int{a, z, d})
		}
	}
	for _, i := range []int{0, len(c) - 1} {
		p := c[i]
		n := len(rest)
		rest = append(rest, p)
		weight := math.Max(0, math.Min(1, (p.Y-2.8)/.9))
		posed = append(posed, hatzNeckPose(p, seconds, weight))
		for j := 0; j < around; j++ {
			k := (j + 1) % around
			if i == 0 {
				faces = append(faces, [3]int{n, k, j})
			} else {
				faces = append(faces, [3]int{n, i*around + j, i*around + k})
			}
		}
	}
	vol := 0.0
	for _, f := range faces {
		vol += posed[f[0]].Dot(posed[f[1]].Cross(posed[f[2]]))
	}
	if vol < 0 {
		for i := range faces {
			faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
		}
	}
	return mesh.NewWithRestPoints(posed, rest, faces, sealSmoothNormals(posed, faces), true)
}

// hatzMix is the point a share t of the way from a to b.
func hatzMix(a, b maths.Vec, t float64) maths.Vec { return a.Scale(1 - t).Add(b.Scale(t)) }

// hatzWingControls is the five control points of a wing, shoulder to tip,
// for a fold from nought to one at the time in seconds; side is the sign
// of z.
func hatzWingControls(fold, seconds, side float64) (maths.Vec, maths.Vec, maths.Vec, maths.Vec, maths.Vec) {
	s := maths.V(.32, 2.70, side*.36)
	w := hatzMix(maths.V(.15, 2.80, side*2.70), maths.V(1.08, .10, side*.76), fold)
	v := w.Sub(s)
	d := v.Len()
	u := v.Unit()
	l1, l2 := 1.35, 1.45
	a := (l1*l1 - l2*l2 + d*d) / (2 * d)
	h := math.Sqrt(math.Max(0, l1*l1-a*a))
	forward := maths.V(1, 0, 0).Sub(u.Scale(u.Dot(maths.V(1, 0, 0)))).Unit()
	e := s.Add(u.Scale(a)).Add(forward.Scale(h))
	flap := .1 * math.Sin(2*math.Pi*seconds/3) * (1 - fold)
	flight := maths.V(-.25, flap, side).Unit()
	ground := maths.V(-1.88, 2.15, side*.12).Unit()
	tip := w.Add(hatzMix(flight, ground, fold).Unit().Scale(2.85))
	inner := hatzMix(maths.V(-1.3, 2.38, side*.46), maths.V(-1.18, 2.36, side*.40), fold)
	return s, e, w, tip, inner
}

const hatzSpan, hatzChord = 18, 18

// Topologically directed grid directions carry through the fold. Pleat
// amplitude is selected by area, not by any physical constitutive model.
func hatzWingPoints(s, e, w, tip, inner maths.Vec, fold, side, amplitude float64) []maths.Vec {
	var out []maths.Vec
	normal := maths.V(0, 1-fold, side*fold).Unit()
	for i := 0; i <= hatzSpan; i++ {
		t := float64(i) / hatzSpan
		var lead maths.Vec
		if t < 1.0/3 {
			lead = hatzMix(s, e, t*3)
		} else if t < 2.0/3 {
			lead = hatzMix(e, w, (t-1.0/3)*3)
		} else {
			lead = hatzMix(w, tip, (t-2.0/3)*3)
		}
		trail := hatzMix(inner, tip, t).Add(maths.V(-.04*t, 0, 0))
		trail = hatzMix(trail, lead, .06*math.Sin(math.Pi*t)*(1-fold))
		trail = hatzMix(trail, lead, .8*fold)
		for j := 0; j <= hatzChord; j++ {
			q := float64(j) / hatzChord
			p := hatzMix(lead, trail, q)
			p.Y += .04 * (1 - fold) * math.Sin(math.Pi*q) * math.Sin(math.Pi*t)
			p = p.Add(normal.Scale(amplitude * math.Sin(6*math.Pi*q) * math.Sin(math.Pi*t)))
			out = append(out, p)
		}
	}
	return out
}

// hatzSurfaceArea is the area of a grid of points, hatzSpan by hatzChord
// quads, summed over the quads.
func hatzSurfaceArea(p []maths.Vec) float64 {
	a := 0.0
	stride := hatzChord + 1
	for i := 0; i < hatzSpan; i++ {
		for j := 0; j < hatzChord; j++ {
			v, w, x, y := p[i*stride+j], p[(i+1)*stride+j], p[(i+1)*stride+j+1], p[i*stride+j+1]
			a += .5*w.Sub(v).Cross(x.Sub(v)).Len() + .5*x.Sub(v).Cross(y.Sub(v)).Len()
		}
	}
	return a
}

// hatzWingSurface is the wing's mesh through its control points, with a
// width found by bisection so that it keeps the area of the rest wing.
func hatzWingSurface(s, e, w, tip, inner maths.Vec, fold, side float64) (*mesh.Mesh, error) {
	rs, re, rw, rt, ri := hatzWingControls(0, 0, side)
	reference := hatzWingPoints(rs, re, rw, rt, ri, 0, side, 0)
	target := hatzSurfaceArea(reference)
	low, high := 0.0, .8
	for i := 0; i < 24; i++ {
		mid := (low + high) / 2
		if hatzSurfaceArea(hatzWingPoints(s, e, w, tip, inner, fold, side, mid)) < target {
			low = mid
		} else {
			high = mid
		}
	}
	amplitude := (low + high) / 2
	if fold == 0 {
		amplitude = 0
	}
	surface := hatzWingPoints(s, e, w, tip, inner, fold, side, amplitude)
	normal := maths.V(0, 1-fold, side*fold).Unit().Scale(.0075)
	restNormal := maths.Up.Scale(.0075)
	n := len(surface)
	var points, rest []maths.Vec
	for _, p := range surface {
		points = append(points, p.Add(normal))
	}
	for _, p := range surface {
		points = append(points, p.Sub(normal))
	}
	for _, p := range reference {
		rest = append(rest, p.Add(restNormal))
	}
	for _, p := range reference {
		rest = append(rest, p.Sub(restNormal))
	}
	idx := func(i, j int) int { return i*(hatzChord+1) + j }
	var faces [][3]int
	for i := 0; i < hatzSpan; i++ {
		for j := 0; j < hatzChord; j++ {
			a, b, c, d := idx(i, j), idx(i+1, j), idx(i+1, j+1), idx(i, j+1)
			faces = append(faces, [3]int{a, b, c}, [3]int{a, c, d}, [3]int{a + n, c + n, b + n}, [3]int{a + n, d + n, c + n})
		}
	}
	edge := func(a, b int) { faces = append(faces, [3]int{b, a, a + n}, [3]int{b, a + n, b + n}) }
	for j := 0; j < hatzChord; j++ {
		edge(idx(0, j+1), idx(0, j))
		edge(idx(hatzSpan, j), idx(hatzSpan, j+1))
	}
	for i := 0; i < hatzSpan; i++ {
		edge(idx(i, 0), idx(i+1, 0))
		edge(idx(i+1, hatzChord), idx(i, hatzChord))
	}
	vol := 0.0
	for _, f := range faces {
		vol += points[f[0]].Dot(points[f[1]].Cross(points[f[2]]))
	}
	if vol < 0 {
		for i := range faces {
			faces[i][1], faces[i][2] = faces[i][2], faces[i][1]
		}
	}
	return mesh.NewWithRestPoints(points, rest, faces, sealSmoothNormals(points, faces), true)
}
