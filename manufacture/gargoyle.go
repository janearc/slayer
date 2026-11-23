package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// a gargoyle is a carved stone spout on a building, shaped like a
// creature. the creature is one of the kinds already made here, and it is
// used as a part: a gargoyle takes its shape and nothing else. the stone,
// the corbel it crouches on and the spout are the gargoyle's own.
//
// the front of the creature stands out of a corbel, and the rest of it
// goes into the wall behind, so that what is seen is one block with a
// creature coming out of it.
//
// the creature's own surface is set aside and the one stone put over
// every piece of it, so it and the corbel are the same material and the
// same grain.
//
// to add a creature, write its row in cities/gargoyle-creatures.yaml and
// name its study in gargoyle_studies.go. nothing else knows the list.

// the places a gargoyle's pose is drawn from in its seed, and how far a
// seed may stray from the row's pose. the numbers are picked by eye.
const (
	// carving is the part of a seed a gargoyle's pose is drawn in, after
	// the parts the other kinds draw from.
	carving = housing + 1
	// yawSwing is how far, in degrees, a seed may turn a gargoyle's head
	// to either side of straight out of the wall.
	yawSwing = 14.0
	// pitchSwing is how far, in degrees, a seed may tip a gargoyle's head
	// up or down from its row's.
	pitchSwing = 6.0
	// sizeSwing is how much larger or smaller, as a share, a seed may
	// carve a gargoyle than its row's.
	sizeSwing = 0.1
)

// the draws of the pose, in the carving part of a seed
const (
	sizeDraw = iota
	yawDraw
	pitchDraw
)

// spoutMaterial is the name of the material of the marker a gargoyle
// carries at its spout. It is the name package weathering looks for to
// find a place water runs from, and a test says the two are the same
// words.
const spoutMaterial = "gargoyle spout"

// spoutRadius is the radius, in metres, of the marker at a spout. The
// marker is hidden and only says where the spout is. picked: small
// enough to be nothing, if it were drawn.
const spoutRadius = 0.02

// gargoyleStone is a stone a gargoyle is carved from: a rock with flecks
// in it, in patches as wide as the grain, and how much of a mirror the cut
// face is. picked by eye; none is measured.
type gargoyleStone struct {
	name         string
	rock, flecks maths.Light
	// share is how much of the face the flecks cover, nought to one, and
	// grain how wide a fleck is, in metres.
	share, grain float64
	gloss        float64
	// board, if over nought, makes the stone board-marked concrete: how
	// tall a board is, in metres, and joint how wide the seam between two
	// boards is. The flecks are then the stain of the seams.
	board, joint float64
	// facet, if over nought, is how much a chisel's flat facets tip the
	// normal, and facetSize how wide a facet is, in metres.
	facet, facetSize float64
}

// look is the stone as a material of a name, for something at a scale: the
// texture is laid in the thing's own space, so a thing drawn at a tenth of
// its own size is given a grain ten times as wide to look the same.
//
// The diffuse colour is the average of the texture, so a surface that is
// too far off to show the grain is the stone's average, as the texture's
// own band limit fades it to: for a rock, the rock and its flecks by their
// share, and for concrete, what the texture says its average is. A stone
// with facets has them cut in it.
func (stone gargoyleStone) look(name string, number uint64,
	scale float64) *thing.Material {
	grained := stone.grained(number, scale)
	average := stone.rock.Scale(1 - stone.share).
		Add(stone.flecks.Scale(stone.share))
	if boarded, concrete := grained.(texture.Boarded); concrete {
		average = boarded.Average()
	}
	made := &thing.Material{Name: name, Colour: average, Gloss: stone.gloss,
		Texture: grained}
	if stone.facet > 0 {
		made.Crinkle = &thing.Crinkle{Size: stone.facetSize / scale,
			Facet: stone.facet, Seed: uint32(number)}
	}
	return made
}

// grained is the texture of the stone: board-marked for concrete, dusted
// with flecks for the rest.
func (stone gargoyleStone) grained(number uint64,
	scale float64) texture.Texture {
	if stone.board > 0 {
		return texture.Boarded{Concrete: stone.rock, Stain: stone.flecks,
			Board: stone.board / scale, Joint: stone.joint / scale,
			Seed: number}
	}
	return texture.Dusted{Rock: stone.rock, Dust: stone.flecks,
		Share: stone.share, Patch: stone.grain / scale, Seed: number}
}

// gargoyleStones are the stones, by name: grey granite, with pale flecks
// of feldspar a few centimetres across, and board-marked concrete, poured
// against timber, with a dark seam between boards. Both are cut or cast
// in facets. picked by eye; none is measured.
func gargoyleStones() map[string]gargoyleStone {
	return map[string]gargoyleStone{
		"granite": {name: "granite", rock: maths.RGB(0.3, 0.3, 0.31),
			flecks: maths.RGB(0.5, 0.47, 0.45), share: 0.4, grain: 0.025,
			gloss: 0.12, facet: 0.3, facetSize: 0.12},
		"concrete": {name: "concrete", rock: maths.RGB(0.46, 0.45, 0.43),
			flecks: maths.RGB(0.28, 0.27, 0.26), share: 0.3, grain: 0.45,
			gloss: 0.04, board: 0.2, joint: 0.012, facet: 0.15,
			facetSize: 0.2},
	}
}

// gargoyleCreature is how a creature of the catalogue is carved as a
// gargoyle: the data's row, whose numbers say where it is held and how it
// is turned. Its own frame is the creature's: it faces along x for the
// animals that swim or walk, and the pose says how that is turned to face
// out of a wall.
type gargoyleCreature struct{ cities.Creature }

// pieces is the creature at rest, in its own frame and its own
// materials, which the gargoyle sets aside.
func (creature gargoyleCreature) pieces() ([]thing.Thing, error) {
	study, err := gargoyleStudy(creature.Study)
	if err != nil {
		return nil, err
	}
	return study()
}

// gargoyleCreatures are the creatures, by the name the data gives them.
// If the data is refused there are none, and no gargoyle kind is made;
// the data's own test says it is not refused.
func gargoyleCreatures() map[string]gargoyleCreature {
	rows, err := cities.GargoyleCreatureRows()
	creatures := map[string]gargoyleCreature{}
	if err != nil {
		return creatures
	}
	for _, each := range rows {
		creatures[each.Name] = gargoyleCreature{each}
	}
	return creatures
}

// pose is where the creature stands in the block's frame: scaled, turned,
// and set so that its held point is where its row holds it. A seed strays
// from the row's size, yaw and pitch, so no two gargoyles of a row are the
// same.
func (creature gargoyleCreature) pose(from seed,
	block cities.GargoyleBlock) maths.Subspace {
	size := creature.Scale * (1 + sizeSwing*from.swing(carving, sizeDraw, 0))
	yaw := creature.Yaw + yawSwing*from.swing(carving, yawDraw, 0)
	pitch := creature.Pitch + pitchSwing*from.swing(carving, pitchDraw, 0)
	turn := maths.Rotation(maths.XAxis, creature.Lean*maths.Degree).Then(
		maths.Rotation(maths.YAxis, yaw*maths.Degree).Then(
			maths.Rotation(maths.ZAxis, pitch*maths.Degree).Then(
				maths.Rotation(maths.XAxis, creature.Roll*maths.Degree))))
	hold := maths.V(0, creature.Up, creature.Out*block.Proud)
	return maths.NewSubspace(
		hold.Sub(turn.Out(creature.HeldAt().Scale(size))), turn, size)
}

// carve is the creature's shape in the stone, in the pose: every piece of
// it with its own material set aside and the stone's put on, the part of
// it that is behind the wall cut away, and a hidden marker at its spout.
func (creature gargoyleCreature) carve(pose maths.Subspace,
	stone *thing.Material) (good, error) {
	pieces, err := creature.pieces()
	if err != nil {
		return good{}, err
	}
	carved := make([]thing.Thing, 0, len(pieces)+1)
	for _, piece := range pieces {
		place := piece.At(0).Within(pose)
		made, left, err := carvedPiece(piece, place, stone)
		if err != nil {
			return good{}, err
		}
		if left {
			carved = append(carved, made)
		}
	}
	carved = append(carved, creature.marker(pose))
	return good{dressed: looks{}, native: carved}, nil
}

// marker is the hidden thing at a creature's spout, where the weathering
// finds the place water runs from. It is drawn nowhere and casts no
// shadow.
func (creature gargoyleCreature) marker(pose maths.Subspace) thing.Thing {
	return thing.Thing{Shape: shape.Sphere{Radius: spoutRadius},
		Material: &thing.Material{Name: spoutMaterial},
		Motion: thing.Still(maths.NewSubspace(pose.Out(creature.MouthAt()),
			maths.Identity, 1)),
		Attrs: thing.Attrs{Hidden: true}}
}

// gargoyleRow is the kind of one creature in one stone: a gargoyle, on its
// corbel.
type gargoyleRow struct {
	creature gargoyleCreature
	stone    gargoyleStone
}

// made is the gargoyle at the quick detail level.
func (row gargoyleRow) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt is the gargoyle at the named detail level: the corbel, drawn by
// package cities, and the creature carved on it, in one stone. It returns
// errGood when the seed is not made.
func (row gargoyleRow) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	block, drawn, err := row.corbel(from, detail)
	if err != nil {
		return good{}, err
	}
	base, err := goodOfRendered(drawn)
	if err != nil {
		return good{}, err
	}
	for of, look := range base.dressed {
		base.dressed[of] = row.stone.look(look.Name, from.number, 1)
	}
	pose := row.creature.pose(from, block)
	carved, err := row.creature.carve(pose, row.stone.look(
		"gargoyle "+row.stone.name, from.number, 1))
	if err != nil {
		return good{}, err
	}
	return gather(base, carved)
}

// corbel is the block the gargoyle is carved from the wall out of, and
// what it draws at a detail level.
func (row gargoyleRow) corbel(from seed, detail string) (
	cities.GargoyleBlock, []cities.Rendered, error) {
	block, err := cities.Corbel(cities.Seed(from.number), row.stone.name)
	if err != nil {
		return cities.GargoyleBlock{}, nil, err
	}
	drawn, err := block.Design.Rendered(cities.Detail(detail))
	return block, drawn, err
}

// spouts is where the gargoyle lets water go, in its own frame: the
// creature's mouth, or its funnel, where the pose has put it. A gargoyle
// has one.
func (row gargoyleRow) spouts(from seed, detail string) ([]maths.Vec, error) {
	block, _, err := row.corbel(from, detail)
	if err != nil {
		return nil, err
	}
	pose := row.creature.pose(from, block)
	return []maths.Vec{pose.Out(row.creature.MouthAt())}, nil
}

// gargoyleRows is every gargoyle kind: each creature in each stone, named
// by cities.GargoyleKind, such as gargoyle-orca-granite, and a cornice of
// each stone, such as gargoyle-cornice-concrete.
func gargoyleRows() map[string]maker {
	rows := map[string]maker{}
	for stoneName, stone := range gargoyleStones() {
		for creatureName, creature := range gargoyleCreatures() {
			rows[cities.GargoyleKind(creatureName, stoneName)] = gargoyleRow{
				creature: creature, stone: stone}
		}
		rows[gargoyleCorniceName(stoneName)] = gargoyleCorniceRow{stone: stone}
	}
	return rows
}

// gargoyleCorniceName is the name of the kind that is a cornice wall of a
// stone with its row of gargoyles.
func gargoyleCorniceName(stone string) string {
	return fmt.Sprintf("%s-cornice-%s", cities.GargoyleKindPrefix, stone)
}
