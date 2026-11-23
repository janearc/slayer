package cities

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// gargoyleCulture is the culture a gargoyle's corbel and cornice are drawn
// from.
const gargoyleCulture = "gargoyle"

// GargoyleKindPrefix begins the name of every gargoyle kind in the
// manufactory: the prefix, the creature it is carved as, and the stone it
// is carved from, joined by hyphens.
const GargoyleKindPrefix = "gargoyle"

// GargoyleKind is the name of the manufactory's kind for a gargoyle of a
// creature and a stone, such as gargoyle-orca-granite. The creature is the
// name of an entry of gargoyle-creatures.yaml, and the stone the name of
// a stone option of the culture's data.
func GargoyleKind(creature, stone string) string {
	return fmt.Sprintf("%s-%s-%s", GargoyleKindPrefix, creature, stone)
}

// GargoyleBlock is the stone a gargoyle is carved from: a panel set into
// the wall, a corbel, which is a stone bracket that sticks out of a wall
// in steps, under it, and the sizes the creature on it is posed by. Its
// frame has the wall's face in the plane z is nought, the wall on the
// side of negative z, and the top of the corbel's slab, which the
// creature crouches on, at y is nought.
//
// The panel is the block's back: it
// stands in the plane z is nought and goes into the wall behind, so the
// creature comes out of one stone.
type GargoyleBlock struct {
	Design Design
	// Width is the corbel's width along the wall, Proud how far its slab
	// stands out of the wall, and Slab how thick the slab is.
	Width, Proud, Slab float64
	// Panel is the width of the panel the creature comes out of, Above how
	// far the panel stands over the slab, and Back how far it goes into
	// the wall, behind the plane z is nought.
	Panel, Above, Back float64
}

// GargoyleStones are the names of the stones a gargoyle may be of, in the
// order the data writes them.
func GargoyleStones() ([]string, error) {
	drawing, err := newDrawer(gargoyleCulture, 0)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, one := range drawing.options("stone") {
		names = append(names, one.Name)
	}
	return names, drawing.trouble
}

// GargoyleCreatures are the names of the creatures a gargoyle may be
// carved as, in the order the data writes them. Each is the name of a
// creature the manufactory can carve.
func GargoyleCreatures() ([]string, error) {
	rows, err := GargoyleCreatureRows()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(rows))
	for index, each := range rows {
		names[index] = each.Name
	}
	return names, nil
}

// stoneOf is the stone option of a name: a gargoyle's stone is chosen by
// the kind that is asked for, not drawn, and a name that is not there is
// refused, naming the stones that are. An empty name is the district's
// draw, so a cornice that names no stone is cut from whichever the seed
// gives it.
func (drawing *drawer) stoneOf(name string) (option, error) {
	stones := drawing.options("stone")
	if name == "" {
		return drawing.pick("stone", stones, place{}), drawing.trouble
	}
	for _, one := range stones {
		if one.Name == name {
			return one, drawing.trouble
		}
	}
	return option{}, fmt.Errorf("cities: no stone named %q for a gargoyle, "+
		"only %v", name, gargoyleOptionNames(stones))
}

// gargoyleOptionNames are the names of some options.
func gargoyleOptionNames(from []option) []string {
	names := make([]string, len(from))
	for index, one := range from {
		names[index] = one.Name
	}
	return names
}

// chamfered is a rectangle's plan with its four corners cut off by cut,
// wound as a box's is, from the origin corner out to width and depth: the
// plan of a block whose upright edges a chisel has softened.
func chamfered(width, depth, cut float64) []outlineCorner {
	return []outlineCorner{{cut, 0}, {width - cut, 0}, {width, cut},
		{width, depth - cut}, {width - cut, depth}, {cut, depth},
		{0, depth - cut}, {0, cut}}
}

// carved is a block of stone of a size, its upright edges chamfered by
// cut, with its corner at a place of its holder's frame.
func carved(name string, size Box, cut float64, look *thing.Material,
	at maths.Vec) placedElement {
	plan := polygon{chamfered(size.Width, size.Depth, cut)}
	return put(name, prism{Bottom: plan, Top: plan, Height: size.Height},
		look, at)
}

// Corbel is the corbel a gargoyle crouches on, in the stone of a name: a
// panel set into the wall to hold the creature, a slab standing out of
// the wall, and under it steps that each stand out less than the one
// above and are each a little narrower, down to the wall. The upright
// edges are chamfered, as a chisel leaves them. All of it is one stone,
// so a creature carved on it reads as cut from one block.
func Corbel(seed Seed, stone string) (GargoyleBlock, error) {
	drawing, err := newDrawer(gargoyleCulture, seed)
	if err != nil {
		return GargoyleBlock{}, err
	}
	chosen, err := drawing.stoneOf(stone)
	if err != nil {
		return GargoyleBlock{}, err
	}
	look := paint("corbel", chosen)
	width, proud := drawing.fixed("corbel-width"),
		drawing.fixed("corbel-proud")
	slab, rise := drawing.fixed("corbel-slab"),
		drawing.fixed("corbel-step-height")
	steps := drawing.count("corbel-steps", place{})
	cut, narrow := drawing.fixed("corbel-chamfer"),
		drawing.fixed("corbel-step-narrowing")
	panel := width + 2*drawing.fixed("panel-margin")
	above, back := drawing.fixed("panel-above"), drawing.fixed("panel-back")
	if drawing.trouble != nil {
		return GargoyleBlock{}, drawing.trouble
	}
	low := -slab - float64(steps)*rise
	parts := []placedElement{
		carved("corbel panel", Box{panel, above - low, back}, cut, look,
			maths.V(-panel/half, low, -back)),
		carved("corbel slab", Box{width, slab, proud}, cut, look,
			maths.V(-width/half, -slab, 0))}
	for step := 1; step <= steps; step++ {
		out := proud * (1 - float64(step)/float64(steps+1))
		across := width - float64(step)*narrow
		parts = append(parts, carved("corbel step", Box{across, rise, out},
			cut, look, maths.V(-across/half, -slab-float64(step)*rise, 0)))
	}
	return GargoyleBlock{Design: Design{Name: "corbel", Elements: parts},
		Width: width, Proud: proud, Slab: slab, Panel: panel, Above: above,
		Back: back}, nil
}

// GargoyleCornice is a wall with a row of gargoyles under its eave, in the
// stone of a name, or in the stone the seed draws if the name is empty.
// The wall stands with its face in the plane z is nought, on the side of
// negative z, its foot at y nought and its middle at x nought.
//
// The gargoyles are the design's mounts: each a kind of the manufactory, a
// creature of gargoyle-creatures.yaml in the order it is written, standing
// on a corbel it brings, its panel set into the wall a little out of its
// face. A cornice of more gargoyles than there are creatures goes round
// again.
func GargoyleCornice(seed Seed, stone string) (Design, error) {
	drawing, err := newDrawer(gargoyleCulture, seed)
	if err != nil {
		return Design{}, err
	}
	chosen, err := drawing.stoneOf(stone)
	if err != nil {
		return Design{}, err
	}
	creatures, err := GargoyleCreatures()
	if err != nil {
		return Design{}, err
	}
	count := drawing.count("gargoyle-count", place{})
	pitch, margin := drawing.fixed("gargoyle-pitch"),
		drawing.fixed("end-margin")
	length := 2*margin + float64(count)*pitch
	height, thick := drawing.fixed("wall-height"),
		drawing.fixed("wall-thickness")
	course, mould := drawing.fixed("course-height"),
		drawing.fixed("mould-height")
	look := paint("wall", chosen)
	left := -length / half
	parts := []placedElement{
		put("wall", Box{length, height, thick}, look, maths.V(left, 0, -thick)),
		put("cornice course", Box{length, course,
			drawing.fixed("course-proud")}, look,
			maths.V(left, height-course, 0)),
		put("cornice moulding", Box{length, mould,
			drawing.fixed("mould-proud")}, look,
			maths.V(left, height-course-mould, 0)),
	}
	standing := height - course - drawing.fixed("gargoyle-drop")
	boss := drawing.fixed("boss-proud")
	var mounts []Mount
	for index := range count {
		along := left + margin + (float64(index)+1/float64(half))*pitch
		mounts = append(mounts, Mount{
			Kind: GargoyleKind(creatures[index%len(creatures)],
				chosen.Name),
			Place: standingAt(maths.V(along, standing, boss))})
	}
	if drawing.trouble != nil {
		return Design{}, drawing.trouble
	}
	return Design{Name: "gargoyle cornice", Elements: parts,
		Mounts: mounts}, nil
}
