package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// crinkleSeedMask keeps a stucco seed below 2^24, as thing.Crinkle needs
// for the GPU's floats to hold it whole.
const crinkleSeedMask = 1<<24 - 1

// plastered is a wall's material with the surface of plaster at the demo
// level: the wall's own colour, with patches of a darker tone where it is
// stained, laid across it by noise in the part's own space. At any other
// level the material is returned as it is.
//
// This is the weathering seam. How stained a wall is, and how wide and
// how dark the patches are, are drawn here from the culture's data as one
// fixed state. A pass that ages a building changes them by age, in this
// function and in timbered, and touches no geometry.
func (drawing *drawer) plastered(base *thing.Material) *thing.Material {
	if drawing.level != demoLevel {
		return base
	}
	if clad := drawing.clad(base); clad != nil {
		return clad
	}
	surfaced := *base
	surfaced.Name += " plaster"
	surfaced.Texture = texture.Dusted{Rock: base.Colour,
		Dust:  base.Colour.Scale(drawing.fixed("plaster-stain")),
		Share: drawing.fixed("plaster-share"),
		Patch: drawing.fixed("plaster-patch"), Seed: uint64(drawing.seed)}
	surfaced.Crinkle = &thing.Crinkle{Size: drawing.fixed("stucco-size"),
		Slope: drawing.fixed("stucco-slope"),
		Seed:  uint32(drawing.seed) & crinkleSeedMask}
	return &surfaced
}

// timbered is a painted timber's material with the grain of sawn wood at
// the demo level: the painted colour with darker rings, the grain running
// up the part. At any other level the material is returned as it is. It is
// the other half of the weathering seam named on plastered: the rings and
// their tone are the fixed state a weathering pass would change.
func (drawing *drawer) timbered(base *thing.Material) *thing.Material {
	if drawing.level != demoLevel {
		return base
	}
	surfaced := *base
	surfaced.Name += " timber"
	surfaced.Texture = texture.Wood{Early: base.Colour,
		Late:      base.Colour.Scale(drawing.fixed("timber-late-tone")),
		Along:     maths.YAxis,
		Ring:      drawing.fixed("timber-ring"),
		LateShare: drawing.fixed("timber-late-share"),
		Wander:    drawing.fixed("timber-wander")}
	return &surfaced
}
