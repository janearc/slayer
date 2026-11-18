package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// weather applies one continuous orientation-preserving warp to all connected
// parts. Height and the requested envelope remain bounded. The warp and growth
// proportions are authored fantasy, not ageing or biological measurements.
func (row macrostructureMaker) weather(made good, from seed, structural int) good {
	for i, piece := range made.parts {
		skin := piece.skin()
		if row.spec.Wildness > 0 && i < structural {
			levels := 2
			if piece.of == macroGlass {
				bounds := partBounds([]part{piece})
				size := bounds.Max.Sub(bounds.Min)
				least := math.Min(size.X, math.Min(size.Y, size.Z))
				most := math.Max(size.X, math.Max(size.Y, size.Z))
				// Thin glazing needs more stations before a nonlinear warp: a coarse
				// long solid strip can cross its opposite skin between its corners.
				if least > 0 && most/least > 100 {
					levels = 4
				}
			}
			skin = macroRefinedSurface(skin, levels)
		}
		for f := range skin {
			for v, p := range skin[f] {
				skin[f][v] = row.gnarled(p, from)
			}
		}
		made.parts[i] = partOfSurface(skin, piece.of)
	}
	for key, look := range made.dressed {
		copied := *look
		copied.Name = "ancient living " + look.Name
		copied.Texture = macroPatina{base: look.Colour, seed: from.number + uint64(key), width: row.spec.Width.InMetres(), height: row.spec.Height.InMetres(), depth: row.spec.Depth.InMetres(), green: key == macroPark}
		made.dressed[key] = &copied
	}
	return made
}

// partOfSurface retains generated closed faces with no extra placements.
func partOfSurface(skin surface, material slot) part {
	return part{shape: macroSurface{skin}, space: maths.Local(), of: material}
}

// gnarled moves p part of the way toward a twisted, swaying version of the
// row's footprint, by the row's wildness. A row with no wildness returns p
// unchanged.
func (row macrostructureMaker) gnarled(p maths.Vec, from seed) maths.Vec {
	if row.spec.Wildness == 0 {
		return p
	}
	w, d, h := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	t := p.Y / h
	phase := from.draw(8, 0, 0) * 2 * math.Pi
	turn := row.spec.Wildness * (.65*math.Sin(2*math.Pi*t+phase) + 1.1*t)
	x, z := 2*p.X/w, 2*p.Z/d
	a, b := x*math.Cos(turn)-z*math.Sin(turn), x*math.Sin(turn)+z*math.Cos(turn)
	sway := row.spec.Wildness * .22 * math.Sin(math.Pi*t) * math.Sin(4*math.Pi*t+phase)
	warped := maths.V(w*.5*math.Tanh(1.25*a+sway), p.Y, d*.5*math.Tanh(1.25*b-sway*.7))
	return p.Scale(1 - row.spec.Wildness).Add(warped.Scale(row.spec.Wildness))
}

// living anchors giant native branching thickets on the actual generated skin.
// The resulting roots and canopies share the architectural warp, so they stay
// attached. A tree-like silhouette here is kilometres high on a planetary city.
func (row macrostructureMaker) living(a *instrumentAssembly, from seed) {
	var faces surface
	for _, part := range a.object.parts {
		faces = append(faces, part.skin()...)
	}
	if len(faces) == 0 {
		return
	}
	w, d, h := row.spec.Width.InMetres(), row.spec.Depth.InMetres(), row.spec.Height.InMetres()
	for i := range 96 {
		f := faces[int(from.draw(9, i, 0)*float64(len(faces)))%len(faces)]
		normal := f[1].Sub(f[0]).Cross(f[2].Sub(f[0])).Unit()
		if normal.Y < -.1 {
			continue
		}
		u, v := from.draw(9, i, 1), from.draw(9, i, 2)
		if u+v > 1 {
			u = 1 - u
			v = 1 - v
		}
		root := f[0].Scale(1 - u - v).Add(f[1].Scale(u)).Add(f[2].Scale(v))
		if math.Abs(root.X) > w*.43 || math.Abs(root.Z) > d*.43 || root.Y > h*.95 || root.Y < h*.02 {
			continue
		}
		height := h * (.008 + .014*from.draw(9, i, 3))
		tip := root.Add(maths.V(from.swing(9, i, 4)*height*.35, height, from.swing(9, i, 5)*height*.35))
		a.rod(root, tip, height*.09, macroMetal)
		for branch := range 3 {
			angle := float64(branch)*2*math.Pi/3 + from.draw(10, i, branch)
			end := tip.Add(maths.V(math.Cos(angle)*height*.45, height*.2, math.Sin(angle)*height*.45))
			a.rod(tip.Sub(maths.V(0, height*.3, 0)), end, height*.045, macroMetal)
			skin := ball(cut{around: 5})
			for n := range skin {
				for c, p := range skin[n] {
					skin[n][c] = end.Add(maths.V(p.X*height*.45, p.Y*height*.26, p.Z*height*.4))
				}
			}
			a.add(macroSurface{skin}, maths.Local(), macroPark)
		}
	}
}

// macroPatina mixes seeded mineral staining and living cover over multiple
// physical scales; it has no bitmap-resolution limit or render-time mutation.
type macroPatina struct {
	base                 maths.Light
	seed                 uint64
	width, height, depth float64
	green                bool
}

// At returns the patina colour at p. It mixes the base colour with a mineral
// colour by a seeded share, and uses a green mineral where the patina is
// green or the coarse noise is above .12.
func (paint macroPatina) At(p maths.Vec) maths.Light {
	x, y, z := p.X/paint.width, p.Y/paint.height, p.Z/paint.depth
	coarse := noise.Gradient(paint.seed, x*9, y*18, z*9)
	fine := noise.Gradient(paint.seed+1, x*83, y*150, z*83)
	share := maths.Smooth(-.2, .38, coarse+.3*fine)
	mineral := maths.RGB(.34, .25, .16)
	if paint.green || coarse > .12 {
		mineral = maths.RGB(.08, .27, .115)
	}
	return paint.base.Scale(.35 + .3*(fine+1)).Scale(1 - share*.8).Add(mineral.Scale(share * .8))
}

// macroRefinedSurface gives a continuous warp enough stations on every shared
// edge. Refining all faces equally keeps each original closed component closed.
func macroRefinedSurface(skin surface, levels int) surface {
	for range levels {
		next := make(surface, 0, len(skin)*4)
		for _, f := range skin {
			ab, bc, ca := f[0].Add(f[1]).Scale(.5), f[1].Add(f[2]).Scale(.5), f[2].Add(f[0]).Scale(.5)
			next = append(next, face{f[0], ab, ca}, face{ab, f[1], bc}, face{ca, bc, f[2]}, face{ab, bc, ca})
		}
		skin = next
	}
	return skin
}
