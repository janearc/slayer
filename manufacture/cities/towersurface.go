package cities

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// surfaced gives every material of the demo level's tower the surface its
// kind has up close, and never less than a row house has: stone cut in
// panels with joints, a grain of crystals and veins, and a slight relief;
// concrete in board marks, aggregate and patches; metal in panels with
// seams and a brushed finish.
//
// Only a true mirror is left smooth. It is
// the same weathering seam as plastered and timbered: the tones here are
// the state a pass that ages a tower would change.
func (tower *Tower) surfaced(made *towerLooks) {
	skin := &made.skin
	switch tower.Skin {
	case skinGranite:
		skin.stone = tower.stone(skin.stone, "granite")
	case skinGothic:
		skin.stone = tower.stone(skin.stone, "gothic")
		skin.figure = skin.stone
	case skinConcrete:
		skin.stone = tower.concreted(skin.stone)
	case skinCantilever:
		skin.plate = tower.concreted(skin.plate)
	case skinMirror:
		skin.cap = tower.metalled(skin.cap)
	case skinVaporwave:
		tower.vaporised(skin)
	}
	made.grid, made.plant = tower.metalled(made.grid),
		tower.metalled(made.plant)
	made.podium = tower.concreted(made.podium)
	fine := &made.fine
	fine.metal, fine.backpan = tower.metalled(fine.metal),
		tower.metalled(fine.backpan)
	fine.slab = tower.concreted(fine.slab)
	fine.head = tower.stone(fine.head, "granite")
}

// laid is a material with a texture and a name that says so, so that every
// ask for the same surface is one material.
func laid(base *thing.Material, how string,
	surface texture.Texture) *thing.Material {
	if base == nil {
		return nil
	}
	surfaced := *base
	surfaced.Name += " " + how
	surfaced.Texture = surface
	return &surfaced
}

// dusted is a grain of patches: white where the surface is clean and grey
// of a tone where it is dusty, about a share of it, patches of a size
// across, from the culture's data under a prefix.
func (tower *Tower) dusted(prefix string, salt uint64) texture.Dusted {
	drawing := tower.drawing
	return texture.Dusted{Rock: maths.Grey(1),
		Dust:  maths.Grey(drawing.fixed(prefix + "-tone")),
		Share: drawing.fixed(prefix + "-share"),
		Patch: drawing.fixed(prefix + "-size"),
		Seed:  uint64(drawing.seed) ^ salt}
}

// relief is the slight relief of a worked surface, its size and its slope
// from the culture's data under a prefix.
func (tower *Tower) relief(prefix string) *thing.Crinkle {
	drawing := tower.drawing
	return &thing.Crinkle{Size: drawing.fixed(prefix + "-relief-size"),
		Slope: drawing.fixed(prefix + "-relief-slope"),
		Seed:  uint32(drawing.seed) & crinkleSeedMask}
}

// stone is a material of stone cut in panels: each panel a little lighter
// or darker than the next, a joint between them, and over them a grain of
// crystals and the veins of the rock, with a slight relief. The prefix
// names the cut, granite or gothic.
func (tower *Tower) stone(base *thing.Material,
	prefix string) *thing.Material {
	if base == nil {
		return nil
	}
	drawing := tower.drawing
	panels := ashlar{brick{Brick: base.Colour,
		Mortar: base.Colour.Scale(drawing.fixed(prefix + "-joint-tone")),
		Length: drawing.fixed(prefix + "-panel-length"),
		Height: drawing.fixed(prefix + "-panel-height"),
		Joint:  drawing.fixed(prefix + "-joint"),
		Tone:   drawing.fixed(prefix + "-panel-tone")}}
	grain := product{Surface: tower.dusted("crystal", 0),
		Grain: tower.dusted("vein", veinSalt)}
	stone := laid(base, "panelled", product{Surface: panels, Grain: grain})
	stone.Crinkle = tower.relief(prefix)
	return stone
}

// veinSalt is mixed into the seed of the veins of stone, so that they are
// not the patches of its crystals. A choice: a few digits of pi.
const veinSalt = 0x3243f6a8885a308d

// concreted is a material of poured concrete: board marks in courses with
// a shadow at the foot of each board, the aggregate showing through in
// specks, the patches where it was mended, and a slight relief.
func (tower *Tower) concreted(base *thing.Material) *thing.Material {
	if base == nil {
		return nil
	}
	drawing := tower.drawing
	boards := coursed{lap{Board: base.Colour,
		Height: drawing.fixed("cc-board-height"),
		Shadow: drawing.fixed("cc-board-shadow"),
		Share:  drawing.fixed("cc-board-share"),
		Tone:   drawing.fixed("cc-board-tone")}}
	grain := product{Surface: tower.dusted("cc-patch", 0),
		Grain: tower.dusted("cc-aggregate", veinSalt)}
	poured := laid(base, "poured", product{Surface: boards, Grain: grain})
	poured.Crinkle = tower.relief("cc")
	return poured
}

// metalled is a material of metal in panels: a seam between every two
// panels, each a little lighter or darker than the next, and over them the
// lines of a brushed finish, running along.
func (tower *Tower) metalled(base *thing.Material) *thing.Material {
	if base == nil {
		return nil
	}
	drawing := tower.drawing
	panels := ashlar{brick{Brick: base.Colour,
		Mortar: base.Colour.Scale(drawing.fixed("metal-seam-tone")),
		Length: drawing.fixed("metal-panel-length"),
		Height: drawing.fixed("metal-panel-height"),
		Joint:  drawing.fixed("metal-seam"),
		Tone:   drawing.fixed("metal-panel-tone")}}
	brushed := texture.Wood{Early: maths.Grey(1),
		Late:      maths.Grey(drawing.fixed("metal-brush-tone")),
		Along:     maths.XAxis,
		Ring:      drawing.fixed("metal-brush-ring"),
		LateShare: drawing.fixed("metal-brush-late"),
		Wander:    drawing.fixed("metal-brush-wander")}
	return laid(base, "seamed", product{Surface: panels, Grain: brushed})
}

// grimed is a material under the cyberpunk layer: what it was, with soot
// settled on it in patches and streaks of grime running down it from every
// ledge and bar. The wet sheen is the gloss wet gives it.
func (tower *Tower) grimed(base *thing.Material) *thing.Material {
	if base == nil {
		return nil
	}
	drawing := tower.drawing
	var surface texture.Texture = uniform{Colour: base.Colour}
	if base.Texture != nil {
		surface = base.Texture
	}
	streaks := texture.Wood{Early: maths.Grey(1),
		Late:      maths.Grey(drawing.fixed("grime-streak-tone")),
		Along:     maths.YAxis,
		Ring:      drawing.fixed("grime-streak-ring"),
		LateShare: drawing.fixed("grime-streak-late"),
		Wander:    drawing.fixed("grime-streak-wander")}
	grain := product{Surface: tower.dusted("grime-soot", veinSalt),
		Grain: streaks}
	return laid(base, "grimed", product{Surface: surface, Grain: grain})
}
