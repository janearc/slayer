package manufacture

import (
	"errors"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// a home is the modern square house on a cliff over the sea: flat
// roofs, boxes stacked and cantilevered past the one below, whole walls
// of glass lit from inside at dusk, made 2026-10-17 for the scene saturn
// in opposition.
//
// this is a first cut, drawn so one can be seen. what it stands on is
// hers to rule (asks/2026-10-17-homes-and-trees.md in veronica's
// record): the box as a third form, glass that glows rather than glass
// you see through, rooms with nothing in them, and the seed's ranges.

// box is a form of six flat sides: so wide along x, tall up y and deep
// along z, its middle over its space's start and its floor on the
// ground. a cut is nothing to it; it has no round side to cut.
type box struct{ wide, tall, deep measure.Length }

var errBox = errors.New("manufacture: a box needs a width, a height " +
	"and a depth, each above nought")

// newBox is a box of a width, a height and a depth, or why there is
// none.
func newBox(wide, tall, deep measure.Length) (box, error) {
	for _, side := range []measure.Length{wide, tall, deep} {
		if !positive(inScene(side)) {
			return box{}, errBox
		}
	}
	return box{wide: wide, tall: tall, deep: deep}, nil
}

// skin is the box's six sides, two faces each, every one turning
// counterclockwise seen from outside.
func (made box) skin(cut) surface {
	x, y := inScene(made.wide)/2, inScene(made.tall)
	z := inScene(made.deep) / 2
	corner := func(east, up, south bool) maths.Vec {
		at := maths.V(-x, 0, -z)
		if east {
			at.X = x
		}
		if up {
			at.Y = y
		}
		if south {
			at.Z = z
		}
		return at
	}
	sides := [6][4]maths.Vec{
		{corner(true, false, true), corner(true, false, false),
			corner(true, true, false), corner(true, true, true)},
		{corner(false, false, false), corner(false, false, true),
			corner(false, true, true), corner(false, true, false)},
		{corner(false, true, false), corner(false, true, true),
			corner(true, true, true), corner(true, true, false)},
		{corner(false, false, false), corner(true, false, false),
			corner(true, false, true), corner(false, false, true)},
		{corner(false, false, true), corner(true, false, true),
			corner(true, true, true), corner(false, true, true)},
		{corner(true, false, false), corner(false, false, false),
			corner(false, true, false), corner(true, true, false)},
	}
	skin := make(surface, 0, 12)
	for _, side := range sides {
		skin = append(skin, face{side[0], side[1], side[2]},
			face{side[0], side[2], side[3]})
	}
	return skin
}

// block is a box where a home plans it: its middle on the ground at,
// how high its floor stands, and its width, height and depth, all in
// metres.
type block struct {
	at                    maths.Vec
	low, wide, tall, deep float64
}

// top is how high the block's roof is.
func (of block) top() float64 { return of.low + of.tall }

// front is how far out along z the block's front stands.
func (of block) front() float64 { return of.at.Z + of.deep/2 }

// back is how far out along z the block's back stands.
func (of block) back() float64 { return of.at.Z - of.deep/2 }

// the walls a pane can stand on: the front, which faces down z, and the
// side that faces down x.
const (
	frontWall = iota
	sideWall
)

// pane is one sheet of glass: a block, the storey it is on and which of
// that storey's walls, and whether the room behind it is lit.
type pane struct {
	block
	on, wall int
	lit      bool
}

// across is how much of its wall a pane covers.
func (glass pane) across() float64 {
	if glass.wall == sideWall {
		return glass.deep
	}
	return glass.wide
}

// plan is a home before it is made: its storeys, bottom up, a roof slab
// over each, and its panes.
type plan struct {
	storeys, slabs []block
	panes          []pane
}

// home is the row of a home: how many storeys, how wide and deep each
// is and how tall a storey, in metres; how far each storey reaches
// past the front of the one below, as a share of its own depth, and
// strays to either side, as a share of the width below.
//
// then how much of a wall is glass, how wide a pane is, in metres, and
// how many of its rooms are lit.
type home struct {
	storeys, wide, deep, storey, past, stray, glazed, pane span
	lit                                                    float64
}

// homeRow is the home. every number is made up to sound right, by me,
// 2026-10-17, and none is measured: two or three storeys, each 6 to
// 14 m a side and 3 to 3.5 m tall, reaching up to 40% of its depth
// past the floor below.
//
// glass is on 30 to 90% of a wall, in panes 1.2 to 2.4 m wide, and four
// rooms in five are lit.
func homeRow() home {
	return home{storeys: span{2, 3}, wide: span{6, 14}, deep: span{6, 12},
		storey: span{3, 3.5}, past: span{0, 0.4}, stray: span{0, 0.25},
		glazed: span{0.3, 0.9}, pane: span{1.2, 2.4}, lit: 0.8}
}

// the numbers of a home's make that are not the seed's, picked, mine:
// a roof slab's thickness, how far it overhangs, and how far down into
// it the storey below and up into it the storey above reach, so no two
// surfaces lie in one plane.
//
// then how deep a pane is and how far off its wall, its margins below
// and above, the gap between two panes, and how far a home's floor is
// sunk into the ground.
const (
	slabThick, slabOver, slabIn       = 0.25, 0.3, 0.05
	paneThick, paneOff                = 0.02, 0.005
	paneBelow, paneAbove, paneBetween = 0.15, 0.3, 0.12
	sunk                              = 0.05
)

// planned is a home's plan from a seed.
func (row home) planned(from seed) plan {
	var made plan
	count := row.storeys.count(from.draw(housing, 0, 0))
	paneWide := row.pane.pick(from.draw(housing, 0, 1))
	low, front, middle, below := -sunk, 0.0, 0.0, 0.0
	for level := range count {
		wide := row.wide.pick(from.draw(housing, 1+level, 0))
		deep := row.deep.pick(from.draw(housing, 1+level, 1))
		tall := row.storey.pick(from.draw(housing, 1+level, 2))
		if level == 0 {
			front = deep / 2
		} else {
			past := row.past.pick(from.draw(housing, 1+level, 3))
			front += past * deep
			middle += from.swing(housing, 1+level, 4) *
				row.stray.most * below
		}
		storey := block{at: maths.V(middle, 0, front-deep/2), low: low,
			wide: wide, tall: tall, deep: deep}
		made.storeys = append(made.storeys, storey)
		made.slabs = append(made.slabs, block{at: storey.at,
			low: storey.top() - slabIn, wide: wide + 2*slabOver,
			tall: slabThick, deep: deep + 2*slabOver})
		lit := from.draw(housing, 1+level, 5) < row.lit
		made.panes = append(made.panes,
			row.glazing(storey, level, paneWide, lit, from)...)
		low = storey.top() - slabIn + slabThick - slabIn
		below = wide
	}
	return made
}

// glazing is a storey's panes: on its front, and on its side when the
// seed glazes it, evenly along the wall and centred on it.
func (row home) glazing(storey block, level int, wide float64, lit bool,
	from seed) []pane {
	var panes []pane
	tall := storey.tall - paneBelow - paneAbove
	low := storey.low + paneBelow
	for wall, long := range []float64{storey.wide, storey.deep} {
		if wall == sideWall && from.draw(housing, 1+level, 6) < 0.3 {
			continue
		}
		share := row.glazed.pick(from.draw(housing, 1+level, 7+wall))
		count := int(share * long / wide)
		for index := range count {
			along := (float64(index) - float64(count-1)/2) * wide
			glass := onWall(storey, wall, along, wide-paneBetween)
			glass.low, glass.tall = low, tall
			panes = append(panes, pane{block: glass, on: level,
				wall: wall, lit: lit})
		}
	}
	return panes
}

// onWall is a pane so wide, standing just off one of a storey's walls,
// so far along it from its middle; its height is the caller's.
func onWall(storey block, wall int, along, wide float64) block {
	off := paneOff + paneThick/2
	if wall == frontWall {
		return block{wide: wide, deep: paneThick,
			at: maths.V(storey.at.X+along, 0, storey.front()+off)}
	}
	return block{wide: paneThick, deep: wide,
		at: maths.V(storey.at.X+storey.wide/2+off, 0,
			storey.at.Z+along)}
}

// the looks a home is picked from: its walls in white render,
// board-marked concrete or charred timber, its slabs in a dark trim
// or, over charred walls, a pale one; glass that mirrors the sky and,
// where a room is lit, glows warm. colours picked, mine, none measured.
var (
	homeWalls = []*thing.Material{plain("white render", 0.82, 0.8, 0.76),
		plain("concrete", 0.52, 0.51, 0.49),
		plain("charred timber", 0.16, 0.13, 0.11)}
	darkTrim, paleTrim = plain("dark trim", 0.12, 0.12, 0.13),
		plain("pale trim", 0.78, 0.77, 0.74)
	litGlass = &thing.Material{Name: "lit glass", Colour: maths.Grey(0.04),
		Gloss: 0.5, Glow: maths.RGB(1, 0.72, 0.42).Scale(0.9)}
	darkGlass = &thing.Material{Name: "dark glass",
		Colour: maths.Grey(0.03), Gloss: 0.8}
)

// made is a home from a seed: its walls, its slabs and its glass, each
// a thing of boxes in its own look, gathered.
func (row home) made(from seed) (good, error) {
	planned := row.planned(from)
	wall := int(from.draw(housing, 0, 2) * float64(len(homeWalls)))
	trim := darkTrim
	if wall == len(homeWalls)-1 {
		trim = paleTrim
	}
	walls, err := boxes(planned.storeys)
	if err != nil {
		return good{}, err
	}
	slabs, err := boxes(planned.slabs)
	if err != nil {
		return good{}, err
	}
	var lit, dark []block
	for _, glass := range planned.panes {
		if glass.lit {
			lit = append(lit, glass.block)
		} else {
			dark = append(dark, glass.block)
		}
	}
	things := []good{walls.in(looks{body: homeWalls[wall]}),
		slabs.in(looks{body: trim})}
	for _, glass := range []struct {
		blocks []block
		look   *thing.Material
	}{{lit, litGlass}, {dark, darkGlass}} {
		if len(glass.blocks) == 0 {
			continue
		}
		panes, err := boxes(glass.blocks)
		if err != nil {
			return good{}, err
		}
		things = append(things, panes.in(looks{body: glass.look}))
	}
	return gather(things...)
}

// boxes is one thing of a box at every block, all of one matter.
func boxes(blocks []block) (good, error) {
	fine, err := cutInto(leastAround)
	if err != nil {
		return good{}, err
	}
	var made good
	for _, at := range blocks {
		shape, err := newBox(measure.Metres(at.wide),
			measure.Metres(at.tall), measure.Metres(at.deep))
		if err != nil {
			return good{}, err
		}
		standing := maths.Subspace{Origin: maths.V(at.at.X, at.low,
			at.at.Z)}
		made.parts = append(made.parts, part{shape: shape, fine: fine,
			of: body, space: standing})
	}
	return made, nil
}
