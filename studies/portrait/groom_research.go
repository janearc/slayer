package main

import (
	"github.com/janearc/slayer/bioforms/assembly"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
	"math"
	"math/rand"
)

// layeredGuide has a narrow side part, lifted roots and alternating bands
// of movement. The long side falls in front of the ear and kicks outward.
func layeredGuide(u, t, side float64) maths.Vec {
	p := comb(u, t, side)
	p.X += (.0018*math.Sin(8*u) + .00055*math.Sin(33*u)) * math.Exp(-11*t)
	atEar := math.Sin(math.Pi * u)
	fall := maths.Smooth(.30, .97, t)
	p.X += side * (.00042*math.Exp(-35*t) + .009*fall*atEar*math.Sin(6*t+7*u+.7*side))
	p.Z += .014 * fall * atEar * math.Sin(7*t-5*u+.6*side)
	p.Y += .0033*math.Sin(math.Pi*t)*math.Sin(math.Pi*u) - .012*fall*atEar*(.6+.4*side)
	return p
}

// Broad shafts retain fine tangent-aligned pigment lines. Varying shaft
// colour is separate from the native dielectric surface response.
func shaftColour(p maths.Vec) maths.Light {
	fine := noise.Gradient(572, p.X*27+.65*math.Sin(7*p.Y+p.Z), p.Y*2, p.Z)
	bands := noise.Gradient(882, p.X*8, p.Y*2, p.Z)
	root := .74 + .26*maths.Smooth(0, .46, p.Y)
	return rgb(.112, .045, .024).Scale(root * (.47 + .57*p.Z) * (.94 + .24*fine + .22*bands))
}
func partAndFlyaways() []assembly.Part {
	var seam, loose geom
	const steps = 42
	for i := 0; i <= steps; i++ {
		u := .03 + .74*float64(i)/steps
		middle := layeredGuide(u, 0, 1)
		n := v(middle.X/.093, (middle.Y-1.551)/.121, (middle.Z+.021)/.117).Unit()
		width := .00009 + .00029*math.Pow(math.Sin(math.Pi*float64(i)/steps), 2)*(.64+.36*math.Cos(30*u))
		p := middle.Add(n.Scale(.00035))
		k := len(seam.pts)
		seam.pts = append(seam.pts, p.Sub(v(width, 0, 0)), p.Add(v(width, 0, 0)))
		if i > 0 {
			seam.faces = append(seam.faces, [3]int{k - 2, k, k - 1}, [3]int{k - 1, k, k + 1})
		}
	}
	rng := rand.New(rand.NewSource(956))
	for i := 0; i < 930; i++ {
		side := 1.
		if i%2 == 0 {
			side = -1
		}
		u := rng.Float64()
		start := .21 + .61*rng.Float64()
		end := .92 + .16*rng.Float64()
		lift := .0002 + .0021*math.Pow(rng.Float64(), 2)
		curl := (rng.Float64() - .5) * .056
		w := .000095 + .00019*rng.Float64()
		if i >= 510 {
			start = .001
			end = .05 + .24*rng.Float64()
			lift *= .22
			curl *= .25
			w *= .63
		}
		for j := 0; j <= 18; j++ {
			s := float64(j) / 18
			t := start + s*(end-start)
			track := u + curl*math.Sin(math.Pi*s)
			p := layeredGuide(track, t, side)
			n := v(p.X/.093, (p.Y-1.551)/.121, (p.Z+.021)/.117).Unit()
			p = p.Add(n.Scale(lift*math.Sin(math.Pi*s) + .0015))
			tangent := layeredGuide(track, t+.001, side).Sub(p.Sub(n.Scale(lift*math.Sin(math.Pi*s) + .0015))).Unit()
			across := tangent.Cross(n).Unit().Scale(w * (1.01 - s))
			k := len(loose.pts)
			loose.pts = append(loose.pts, p.Sub(across), p.Add(across))
			if j > 0 {
				loose.faces = append(loose.faces, [3]int{k - 2, k, k - 1}, [3]int{k - 1, k, k + 1})
			}
		}
	}
	return []assembly.Part{{Rest: seam.pts, Faces: seam.faces, Material: &thing.Material{Name: "scalp line", Colour: rgb(.101, .048, .031), Gloss: .025}}, {Rest: loose.pts, Faces: loose.faces, Material: &thing.Material{Name: "loose auburn fibres", Colour: rgb(.095, .035, .016), Gloss: .09, Satin: .48}}}
}
