package pliosaurs

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/reptiles"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"math"
)

// NewComposite constructs the original composite pliosaur study.
func NewComposite(profile reptiles.Profile, phase float64, at maths.Subspace) ([]thing.Thing, error) {
	clay := &thing.Material{Name: "pliosaur neutral study clay", Colour: maths.RGB(.31, .35, .32), Gloss: .16, Satin: .52}
	sections := make([]struct{ x, y, w, h float64 }, 0, len(profile.Sections))
	for _, s := range profile.Sections {
		sections = append(sections, struct{ x, y, w, h float64 }{s[0], s[1], s[2], s[3]})
	}
	var rings [][]maths.Vec
	identity := func(p maths.Vec) maths.Vec { return p }
	for i := 0; i < len(sections)-1; i++ {
		a, b := sections[i], sections[i+1]
		for j := 0; j < 4; j++ {
			t := float64(j) / 4
			t = t * t * (3 - 2*t)
			s := struct{ x, y, w, h float64 }{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t, a.w + (b.w-a.w)*t, a.h + (b.h-a.h)*t}
			rings = append(rings, geometry.Ring(maths.V(s.x, s.y, 0), maths.V(0, s.h, 0), maths.V(0, 0, s.w), identity))
		}
	}
	last := sections[len(sections)-1]
	rings = append(rings, geometry.Ring(maths.V(last.x, last.y, 0), maths.V(0, last.h, 0), maths.V(0, 0, last.w), identity))
	body, err := geometry.LoftPiece(rings, clay, at)
	if err != nil {
		return nil, err
	}
	out := []thing.Thing{body}
	for _, root := range profile.FinRoots {
		for _, side := range []float64{-1, 1} {
			var fin [][]maths.Vec
			stroke := .42 * math.Sin(phase)
			if root < 0 {
				stroke = .38 * math.Sin(phase+.55)
			}
			for _, s := range profile.FinSections {
				span, chord, thick, sweep := s[0], s[1], s[2], s[3]
				y := profile.FinOrigin[0] + math.Sin(stroke)*(span-profile.FinOrigin[1])
				z := side * (profile.FinOrigin[1] + math.Cos(stroke)*(span-profile.FinOrigin[1]))
				fin = append(fin, geometry.Ring(maths.V(root+sweep, y, z), maths.V(chord, 0, 0), maths.V(0, thick*math.Cos(stroke), -side*thick*math.Sin(stroke)), identity))
			}
			p, e := geometry.LoftPiece(fin, clay, at)
			if e != nil {
				return nil, e
			}
			out = append(out, p)
		}
	}
	dark := &thing.Material{Name: "pliosaur eye and mouth", Colour: maths.RGB(.025, .027, .022), Gloss: .25}
	tooth := &thing.Material{Name: "illustrative pliosaur tooth", Colour: maths.RGB(.68, .65, .55), Gloss: .1}
	for _, side := range []float64{-1, 1} {
		eye := maths.V(profile.Eye[0], profile.Eye[1], side*profile.Eye[2])
		out = append(out, thing.Thing{Shape: shape.Sphere{Radius: profile.Eye[3]}, Material: dark, Motion: thing.Still(maths.NewSubspace(eye, maths.Identity, 1).Within(at))})
		// An approximate line and individual cones make the baseline
		// reviewable, not a measured reconstruction of the soft jaw.
		for i := 0; i < 12; i++ {
			x := 2.48 + .11*float64(i)
			z := side * (.405 - .14*(x-2.48))
			y := -.11
			pos := maths.V(x, y, z)
			dir := maths.V(.10, 0, -side*.014)
			out = append(out, thing.Thing{Shape: shape.Tube{Radius: .008, Thick: .016, Height: dir.Len()}, Material: dark, Motion: thing.Still(maths.NewSubspace(pos.Add(dir.Scale(.5)), maths.Upright(dir), 1).Within(at))})
			t, e := reptiles.Tooth(pos, .025, .105, 10, 0)
			if e != nil {
				return nil, e
			}
			out = append(out, thing.Thing{Shape: t, Material: tooth, Motion: thing.Still(at)})
		}
	}
	return out, nil
}
