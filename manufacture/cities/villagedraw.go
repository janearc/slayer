package cities

import (
	"slices"

	"github.com/janearc/slayer/maths"
)

// A village house is a node: drawn whole at the level a pass gives the
// box it fills, so that a near house is drawn to its cage bars and a far
// one is its walls and its lit windows. It is made at the finest level it
// will be drawn at, and drawn coarser from that.

// roofed draws the house's flat roof, with its parapet, and what stands
// on it: the culture's tanks, stair head and masts along its middle, by
// the draws a shack at its front and planters along its back.
func (job *houseJob) roofed() {
	drawing, house := job.drawing, job.house
	lot := house.Lot
	shape := massShape{width: lot.width(), depth: lot.depth()}
	roof, err := drawing.roof(job.at, shape, house.wall).parts()
	drawing.fail(err)
	job.add(tiers(within(roof, standingAt(maths.V(lot.X0, house.Height,
		lot.Z0))), hullLevel))
	margin := drawing.fixed("parapet-wall") + drawing.fixed("roof-margin")
	area := roofArea{x0: lot.X0 + margin, z0: lot.Z0 + margin,
		x1: lot.X1 - margin, z1: lot.Z1 - margin,
		y: house.Height + drawing.fixed("slab-thickness")}
	job.add(tiers(drawing.crown(job.at, area), quickLevel))
	if drawing.chance("shack", job.at) {
		job.add(tiers(job.shack(area), quickLevel))
	}
	if drawing.chance("roof-garden", job.at) {
		job.add(tiers(job.planters(area), quickLevel))
	}
}

// shack is a room built on the roof at its front left corner, as deep as
// the draw gives and no deeper than the front third of the roof, clear of
// what stands along the roof's middle.
func (job *houseJob) shack(area roofArea) []placedElement {
	drawing := job.drawing
	deep := min(drawing.number("shack-depth", job.at),
		(area.z1-area.z0)/shackShare)
	size := Box{drawing.number("shack-width", job.at),
		drawing.number("shack-height", job.at), deep}
	if size.Width > area.x1-area.x0 {
		return nil
	}
	sheet := paint("shack", drawing.pick("shack-colour",
		drawing.options("shack-colour"), job.at))
	return []placedElement{put("shack", size, sheet,
		maths.V(area.x0, area.y, area.z1-deep))}
}

// shackShare is what share of the roof's depth a shack may take: a third,
// the front third, so that the middle is left to the tanks.
const shackShare = 3

// planters are boxes of earth along the back of the roof, the same gap
// apart, each with its leaves standing out of it.
func (job *houseJob) planters(area roofArea) []placedElement {
	drawing := job.drawing
	box := Box{drawing.fixed("planter-length"),
		drawing.fixed("planter-height"), drawing.fixed("planter-depth")}
	leaves := Box{box.Width - 2*drawing.fixed("leaf-inset"),
		drawing.fixed("leaf-height"), box.Depth -
			2*drawing.fixed("leaf-inset")}
	count := drawing.count("planters", job.at)
	widths := make([]float64, count)
	for index := range widths {
		widths[index] = box.Width
	}
	lefts, gap := spread(area.x0, area.x1, widths)
	if gap < 0 {
		return nil
	}
	earth := paint("planter", drawing.pick("planter-colour",
		drawing.options("planter-colour"), job.at))
	var parts []placedElement
	for _, left := range lefts {
		corner := maths.V(left, area.y, area.z0)
		parts = append(parts, put("planter", box, earth, corner),
			put("foliage", leaves, paint("foliage", foliage),
				corner.Add(maths.V(drawing.fixed("leaf-inset"), box.Height,
					drawing.fixed("leaf-inset")))))
	}
	return parts
}

// placed is the house as a node standing in the frame of what holds it.
func (house villageHouse) placed() placedElement {
	return deferred(&node{Bounds: house.bounds(), Draw: house.drawn})
}

// bounds is the box the house fills, from the lowest ground under it to
// the top of the most that may stand on its roof, in the frame of what
// holds it.
func (house villageHouse) bounds() maths.Box {
	lot, footing := house.Lot, house.Footing
	return maths.Box{Min: maths.V(lot.X0, footing.Low, lot.Z0),
		Max: maths.V(lot.X1, footing.Base+house.Height+roofAllowance,
			lot.Z1)}
}

// roofAllowance is how far over a house's top what stands on its roof may
// reach, in metres: past the culture's tallest mast, so a node's box
// holds all of it.
const roofAllowance = 8

// volume is the box of air the house's walls fill, for what is hung
// between houses: from its floor to its top, over its lot.
func (house villageHouse) volume(plot int) Volume {
	lot, base := house.Lot, house.Footing.Base
	return Volume{Kind: "village house", Plot: plot,
		Low:  [3]float64{lot.X0, base, lot.Z0},
		High: [3]float64{lot.X1, base + house.Height, lot.Z1}}
}

// drawn is the house at a level: its walls, every part kept for the level
// or a coarser one, and the signs on its lanes above the hull, all stood
// on its footing.
func (house villageHouse) drawn(level Detail) ([]placedElement, error) {
	parts, err := house.walled(level)
	if err != nil {
		return nil, err
	}
	for _, one := range house.tiers {
		if rank(level) >= rank(one.least) {
			parts = append(parts, one.placedElement)
		}
	}
	signs, err := house.signed(level)
	if err != nil {
		return nil, err
	}
	return planted(append(parts, signs...), house.Lot, house.Footing)
}

// walled is the house's walls at a level: at the hull one box over its
// lot with its lit windows on it, and at any other its body behind the
// cuts of its faces and each face as it is drawn at the level.
func (house villageHouse) walled(level Detail) ([]placedElement, error) {
	lot := house.Lot
	if level == hullLevel {
		parts := []placedElement{put("wall", Box{lot.width(), house.Height,
			lot.depth()}, house.wall, maths.V(lot.X0, 0, lot.Z0))}
		for side, face := range house.faces {
			frame, _ := wallFrame(lot, side, house.shell)
			parts = append(parts, within(face.lit(), frame)...)
		}
		return parts, nil
	}
	inset := house.shell
	parts := []placedElement{put("wall", Box{lot.width() - 2*inset, house.Height,
		lot.depth() - 2*inset}, house.wall, maths.V(lot.X0+inset, 0,
		lot.Z0+inset))}
	for side, face := range house.faces {
		frame, _ := wallFrame(lot, side, inset)
		drawn, err := face.whole(level)
		if err != nil {
			return nil, err
		}
		parts = append(parts, within(drawn, frame)...)
	}
	return parts, nil
}

// lit is the lit windows of a face on the plane of its wall, each a pane
// that glows with the light of its lamp, if it has one; a dark window is
// none.
func (facade facade) lit() []placedElement {
	var parts []placedElement
	for index, openings := range facade.Storeys {
		floor := facade.floorAt(index)
		for _, opening := range openings {
			parts = append(parts, facade.litPane(opening,
				maths.V(opening.Left, floor+opening.Sill, 0))...)
		}
	}
	return parts
}

// rank is how fine a level is: its place among the levels a face is drawn
// at, coarsest first, and the quick level's for one that is not there.
func rank(level Detail) int {
	if index := slices.Index(facadeLevels, level); index >= 0 {
		return index
	}
	return slices.Index(facadeLevels, quickLevel)
}

// signed is the signs the house hangs on its lanes at a level: none at the
// hull, where a sign is not seen.
func (house villageHouse) signed(level Detail) ([]placedElement, error) {
	if level == hullLevel || !slices.Contains(house.Lanes[:], true) {
		return nil, nil
	}
	return signs(house, house.signs, ConstantLevel(level))
}

// SignMounts is the places a house offers a sign: on each wall that faces
// a lane, a blade sign at its start and an awning over its foot.
func (house villageHouse) SignMounts() ([]signMount, error) {
	drawing, err := newDrawer(signCulture, 0)
	if err != nil {
		return nil, err
	}
	var mounts []signMount
	for side, lane := range house.Lanes {
		if !lane {
			continue
		}
		frame, width := wallFrame(house.Lot, side, 0)
		wall := wallEdge{mid: frame.Out(maths.V(width/half, 0, 0)),
			out:    frame.Orientation.Out(maths.ZAxis),
			length: width, height: house.Height}
		mounts = append(mounts, drawing.bladeMount(wall))
		mounts = append(mounts, drawing.awningMounts(wall)...)
	}
	return mounts, drawing.trouble
}
