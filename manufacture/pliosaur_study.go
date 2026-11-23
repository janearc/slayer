package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// PliosaurStudy is an original composite informed by the skull of P. kevani.
// Its postcranial dimensions, tissue and swimming phase are hypotheses.
// This baseline has detached fins and is not a single connected whole-animal skin.
func PliosaurStudy(phase float64, at maths.Subspace) ([]thing.Thing, error) {
	clay := &thing.Material{Name: "pliosaur neutral study clay", Colour: maths.RGB(.31, .35, .32), Gloss: .16, Satin: .52}
	sections := []struct{ x, y, w, h float64 }{
		{-5.00, 0, .015, .015}, {-4.5, 0, .09, .08}, {-4.0, 0, .19, .17}, {-3.3, 0, .32, .28},
		{-2.6, 0, .52, .43}, {-1.8, 0, .78, .64}, {-.8, 0, 1.00, .76}, {.25, 0, 1.02, .77},
		{1.10, .02, .84, .66}, {1.75, .04, .61, .50}, {2.05, .05, .51, .45}, {2.45, .06, .43, .37},
		{3.00, .06, .37, .30}, {3.50, .04, .29, .24}, {3.95, .02, .19, .16}, {4.00, .02, .015, .02},
	}
	var rings [][]maths.Vec
	identity := func(p maths.Vec) maths.Vec { return p }
	for i := 0; i < len(sections)-1; i++ {
		a, b := sections[i], sections[i+1]
		for j := 0; j < 4; j++ {
			t := float64(j) / 4
			t = t * t * (3 - 2*t)
			s := struct{ x, y, w, h float64 }{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t, a.w + (b.w-a.w)*t, a.h + (b.h-a.h)*t}
			rings = append(rings, sealRing(maths.V(s.x, s.y, 0), maths.V(0, s.h, 0), maths.V(0, 0, s.w), identity))
		}
	}
	last := sections[len(sections)-1]
	rings = append(rings, sealRing(maths.V(last.x, last.y, 0), maths.V(0, last.h, 0), maths.V(0, 0, last.w), identity))
	body, err := sealPiece(rings, clay, at)
	if err != nil {
		return nil, err
	}
	out := []thing.Thing{body}
	for _, root := range []float64{.65, -2.1} {
		for _, side := range []float64{-1, 1} {
			var fin [][]maths.Vec
			stroke := .42 * math.Sin(phase)
			if root < 0 {
				stroke = .38 * math.Sin(phase+.55)
			}
			for _, s := range []struct{ span, chord, thick, sweep float64 }{{.65, .45, .14, 0}, {.95, .59, .12, -.06}, {1.5, .53, .085, -.25}, {2.05, .37, .05, -.48}, {2.45, .12, .025, -.72}, {2.52, .01, .006, -.77}} {
				y := -.24 + math.Sin(stroke)*(s.span-.65)
				z := side * (.65 + math.Cos(stroke)*(s.span-.65))
				fin = append(fin, sealRing(maths.V(root+s.sweep, y, z), maths.V(s.chord, 0, 0), maths.V(0, s.thick*math.Cos(stroke), -side*s.thick*math.Sin(stroke)), identity))
			}
			p, e := sealPiece(fin, clay, at)
			if e != nil {
				return nil, e
			}
			out = append(out, p)
		}
	}
	dark := &thing.Material{Name: "pliosaur eye and mouth", Colour: maths.RGB(.025, .027, .022), Gloss: .25}
	tooth := &thing.Material{Name: "illustrative pliosaur tooth", Colour: maths.RGB(.68, .65, .55), Gloss: .1}
	for _, side := range []float64{-1, 1} {
		eye := maths.V(2.67, .14, side*.37)
		out = append(out, thing.Thing{Shape: shape.Sphere{Radius: .045}, Material: dark, Motion: thing.Still(maths.NewSubspace(eye, maths.Identity, 1).Within(at))})
		// An approximate line and individual cones make the baseline
		// reviewable, not a measured reconstruction of the soft jaw.
		for i := 0; i < 12; i++ {
			x := 2.48 + .11*float64(i)
			z := side * (.405 - .14*(x-2.48))
			y := -.11
			pos := maths.V(x, y, z)
			dir := maths.V(.10, 0, -side*.014)
			out = append(out, thing.Thing{Shape: shape.Tube{Radius: .008, Thick: .016, Height: dir.Len()}, Material: dark, Motion: thing.Still(maths.NewSubspace(pos.Add(dir.Scale(.5)), maths.Upright(dir), 1).Within(at))})
			t, e := pliosaurBaselineTooth(pos, .025, .105)
			if e != nil {
				return nil, e
			}
			out = append(out, thing.Thing{Shape: t, Material: tooth, Motion: thing.Still(at)})
		}
	}
	return out, nil
}

// pliosaurBaselineTooth is the mesh of one pointed tooth of a radius and
// length, set at a root point, as the study's baseline draws it.
func pliosaurBaselineTooth(root maths.Vec, radius, length float64) (*shape.Mesh, error) {
	const n = 10
	v := make([]maths.Vec, 0, n+2)
	f := make([][3]int, 0, 2*n)
	for j := 0; j < n; j++ {
		a := 2 * math.Pi * float64(j) / n
		v = append(v, root.Add(maths.V(radius*math.Cos(a), 0, radius*math.Sin(a))))
	}
	base := len(v)
	v = append(v, root)
	tip := len(v)
	v = append(v, root.Add(maths.V(0, -length, 0)))
	for j := 0; j < n; j++ {
		k := (j + 1) % n
		f = append(f, [3]int{j, k, tip}, [3]int{base, k, j})
	}
	return shape.NewMesh(v, f, nil, false)
}
