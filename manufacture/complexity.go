package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// Complexities names the representations, from a silhouette to foliage.
func Complexities() []string {
	return []string{"hull", "clumps", "quick", "demo"}
}

// MakeWithComplexity makes the same specimen at a chosen representation.
// Make retains its original quick geometry. Unsupported levels are refused.
func MakeWithComplexity(row kind, from seed, detail string) (good, error) {
	valid := false
	for _, name := range Complexities() {
		valid = valid || name == detail
	}
	if !valid {
		return good{}, fmt.Errorf("manufacture: complexity %q; choose %v",
			detail, Complexities())
	}
	if maker, ok := row.how.(interface {
		madeAt(seed, string) (good, error)
	}); ok {
		return maker.madeAt(from, detail)
	}
	if detail != "quick" {
		return good{}, fmt.Errorf("manufacture: %s has no %s representation",
			row.name, detail)
	}
	return makeOf(row, from)
}

// Bounds is the specimen's local extent, independent of representation.
func (made good) Bounds() maths.Box {
	if made.extent != nil {
		return *made.extent
	}
	box := partBounds(made.parts)
	for _, group := range made.groups {
		box = enclosing(box, group.Bounds())
	}
	return box
}

// partBounds encloses parts in the specimen's own space.
func partBounds(parts []part) maths.Box {
	box := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, piece := range parts {
		for _, triangle := range piece.skin() {
			for _, point := range triangle {
				box.Min = maths.V(math.Min(box.Min.X, point.X),
					math.Min(box.Min.Y, point.Y), math.Min(box.Min.Z, point.Z))
				box.Max = maths.V(math.Max(box.Max.X, point.X),
					math.Max(box.Max.Y, point.Y), math.Max(box.Max.Z, point.Z))
			}
		}
	}
	return box
}

// enclosing retains both physical extents in one box.
func enclosing(a, b maths.Box) maths.Box {
	return maths.Box{Min: maths.V(math.Min(a.Min.X, b.Min.X),
		math.Min(a.Min.Y, b.Min.Y), math.Min(a.Min.Z, b.Min.Z)),
		Max: maths.V(math.Max(a.Max.X, b.Max.X),
			math.Max(a.Max.Y, b.Max.Y), math.Max(a.Max.Z, b.Max.Z))}
}

// transformed encloses all eight corners in the parent space.
func transformed(box maths.Box, space maths.Subspace) maths.Box {
	first := space.Out(box.Min)
	moved := maths.Box{Min: first, Max: first}
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				point := space.Out(maths.V(x, y, z))
				moved = enclosing(moved, maths.Box{Min: point, Max: point})
			}
		}
	}
	return moved
}

// madeAt first grows the unchanged specimen, then changes its skin.
func (row conifer) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	if row.foliageStyle != 0 {
		made, err := row.scaffold(from)
		if err != nil {
			return good{}, err
		}
		return recursiveFoliage(made, from, row.foliageStyle, detail)
	}
	made, err := row.made(from)
	if err != nil || detail == "quick" {
		return made, err
	}
	box := made.Bounds()
	made.extent = &box
	switch detail {
	case "hull":
		var leaves []part
		for _, piece := range made.parts {
			if piece.of == leaf {
				leaves = append(leaves, piece)
			}
		}
		crown := partBounds(leaves)
		made.parts = append([]part{made.parts[0]}, part{shape: crownShape{crown},
			of: leaf, space: maths.Subspace{Scale: 1}})
	case "clumps":
		for i := range made.parts {
			made.parts[i].fine.around = leastAround
		}
	case "demo":
		made = individualFoliage(made, from, row.foliageStyle)
	}
	return made, nil
}

// crownShape is an eight-face crown. Its extrema retain the grown crown.
type crownShape struct{ box maths.Box }

// skin closes a coarse crown around its four widest points.
func (shape crownShape) skin(cut) surface {
	box := shape.box
	x, z := (box.Min.X+box.Max.X)/2, (box.Min.Z+box.Max.Z)/2
	y := (box.Min.Y + box.Max.Y) / 2
	bottom, top := maths.V(x, box.Min.Y, z), maths.V(x, box.Max.Y, z)
	ring := []maths.Vec{maths.V(box.Max.X, y, z), maths.V(x, y, box.Min.Z),
		maths.V(box.Min.X, y, z), maths.V(x, y, box.Max.Z)}
	var skin surface
	for i, point := range ring {
		next := ring[(i+1)%len(ring)]
		skin = append(skin, face{top, point, next}, face{bottom, next, point})
	}
	return skin
}

// foliageBudget is an authored preview budget, not a botanical density.
// Blades and their supporting twigs share the existing face cap.
const foliageBudget = 12000

// individualFoliage replaces foliage sleeves with closed individual blades.
// Their supporting shoots follow the same seed and segment placements.
func individualFoliage(made good, from seed, style int) good {
	leaves, wood := 0, 0
	for _, piece := range made.parts {
		if piece.of == leaf {
			leaves++
		} else {
			wood += len(piece.skin())
		}
	}
	available := min(foliageBudget, (mostFaces-wood-24*leaves)/6)
	per := max(1, available/max(1, leaves))
	parts := make([]part, 0, len(made.parts)+leaves)
	for i, piece := range made.parts {
		if piece.of != leaf {
			parts = append(parts, piece)
			continue
		}
		shape, ok := piece.shape.(tube)
		if !ok {
			parts = append(parts, piece)
			continue
		}
		spray := foliageSpray{envelope: shape, count: per,
			seed: from.sprout(i, 0), style: style}
		piece.shape = spray
		parts = append(parts, piece)
		piece.shape, piece.of = foliageTwigs{spray}, body
		parts = append(parts, piece)
		// A fine woody spine remains visible between the blades.
		shape.foot = measure.Metres(math.Min(shape.foot.InMetres()/8, 0.006))
		shape.tip = shape.foot.Times(0.3)
		piece.shape, piece.of, piece.fine = shape, body, cut{around: 3}
		parts = append(parts, piece)
	}
	made.parts = parts
	return made
}

// foliageSpray carries blades on twigs within a shoot's envelope.
// It is a bounded visual representation, not a measured leaf-area model.
type foliageSpray struct {
	envelope tube
	count    int
	seed     seed
	style    int
}

// skin builds individual blades rooted along supporting twigs.
func (spray foliageSpray) skin(cut) surface {
	shape := spray.envelope
	length := shape.length.InMetres()
	var skin surface
	for i := range spray.count {
		up, turn, out := spray.twig(i / 8)
		radius := shape.foot.InMetres()*(1-up) + shape.tip.InMetres()*up
		along := (float64(i%8) + 0.5) / 8
		root := maths.V(0, up*length, 0).Add(out.Scale(radius * 0.8 * along))
		// Dimensions are picked for a readable leaf-sized detail.
		long := math.Min(0.025, radius*0.15)
		wide := long * 0.12
		if spray.style == 1 {
			wide = long * 0.2
		}
		if spray.style == 2 {
			long *= 0.4
			wide = long * 0.4
		}
		way := maths.V(math.Sin(turn), 0, math.Cos(turn))
		if i%2 == 0 {
			way = way.Neg()
		}
		if spray.style != 1 {
			way = way.Scale(math.Cos(float64(i) * 2.4)).Add(maths.Up.Scale(math.Sin(float64(i) * 2.4)))
		}
		side := out.Scale(wide)
		tip := root.Add(way.Scale(long))
		a, b := root.Add(side), root.Sub(side)
		back := root.Add(maths.V(0, wide, 0))
		blade := surface{{a, b, tip}, {b, back, tip},
			{back, a, tip}, {a, back, b}}
		if b.Sub(a).Dot(back.Sub(a).Cross(tip.Sub(a))) < 0 {
			for i := range blade {
				blade[i][1], blade[i][2] = blade[i][2], blade[i][1]
			}
		}
		skin = append(skin, blade...)
	}
	return skin
}

// twig gives a supporting shoot its seeded place and direction.
func (spray foliageSpray) twig(i int) (up, turn float64, out maths.Vec) {
	count := (spray.count + 7) / 8
	up = (float64(i) + 0.5) / float64(count)
	turn = fullTurn * spray.seed.draw(0, i, 0)
	if spray.style == 1 {
		turn = math.Pi * float64(i%2)
	}
	out = maths.V(math.Cos(turn), 0, -math.Sin(turn))
	return
}

// foliageTwigs joins every blade's root to the shoot's woody spine.
type foliageTwigs struct{ spray foliageSpray }

// skin joins each foliage spray to its woody spine.
func (twigs foliageTwigs) skin(cut) surface {
	spray := twigs.spray
	var skin surface
	for i := range (spray.count + 7) / 8 {
		up, _, out := spray.twig(i)
		radius := spray.envelope.foot.InMetres()*(1-up) + spray.envelope.tip.InMetres()*up
		tube, _ := newTube(measure.Metres(radius*0.8), measure.Metres(0.001), measure.Metres(0.0003))
		space, _ := spaceAlong(maths.V(0, up*spray.envelope.length.InMetres(), 0), out)
		piece := part{shape: tube, space: space, fine: cut{around: 3}}
		skin = append(skin, piece.skin()...)
	}
	return skin
}
