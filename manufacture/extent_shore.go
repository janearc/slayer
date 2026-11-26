package manufacture

import (
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/manufacture/cities"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
)

// the kinds a shore asks for, said from their recipes. Each says its box
// without making the thing: from the numbers its recipe gives, or, for the
// kinds package cities draws, from the design drawn at its coarsest level,
// which has no mesh of a material, no lamp and nothing the manufactory
// adds, so each adds back what the finer levels and the fittings stand
// proud by.

// the allowances a box adds, in metres, so it is never too small.
const (
	// finerDetailMargin is how far a part drawn at a finer level than the
	// coarsest stands out of the coarsest's box: a window's frame, a cornice.
	finerDetailMargin = 1.0 // picked: a metre beyond any frame or cornice
	// towerFittings is how far what a tower is given beyond its design
	// stands out of its box: signs, gargoyles, shop fronts, and the trees
	// of a planted tower on its terraces.
	towerFittings = 12.0 // picked: a tree's crown and a sign's blade
	// boatGear is how far a boat's rigging, aerials, fenders and lifebuoys
	// stand out of its hull and fittings, every way.
	boatGear = 2.0 // picked: more than any hung or stood fitting
	// boatKeelMargin is how far below its draught a boat's keel and its
	// propeller reach.
	boatKeelMargin = 0.5 // picked: a propeller's diameter and a skeg
)

// grownBy is a box a margin larger every way.
func grownBy(box maths.Box, margin float64) maths.Box {
	by := maths.V(margin, margin, margin)
	return maths.Box{Min: box.Min.Sub(by), Max: box.Max.Add(by)}
}

// designExtent is the box of a design drawn at its coarsest level, with
// the margin finer levels need.
func designExtent(design cities.Design) (maths.Box, error) {
	return designExtentAt(design, "hull")
}

// designExtentAt is the box of a design drawn at a level, with the margin
// finer levels need. A design whose coarsest level leaves out parts the next
// one draws, such as the fittings of a house, says its box at the level
// that has them.
func designExtentAt(design cities.Design, level cities.Detail) (maths.Box,
	error) {
	reach, err := design.Reach(level)
	if err != nil {
		return maths.Box{}, err
	}
	return grownBy(reach, finerDetailMargin), nil
}

// extent is the box of the scale figure, from the ovals and limbs of its
// recipe: each oval its size about its place and each limb's ends and
// radius, and the mirror of each across the figure's middle.
func (row scaleHuman) extent(from seed) (maths.Box, error) {
	var recipe humanRecipe
	if err := yamlfile.Read("human-scale.yaml", humanScaleRaw,
		&recipe); err != nil {
		return maths.Box{}, err
	}
	box := maths.Box{Min: maths.V(math.Inf(1), math.Inf(1), math.Inf(1)),
		Max: maths.V(math.Inf(-1), math.Inf(-1), math.Inf(-1))}
	for _, oval := range recipe.Ovals {
		reach := maths.V(oval.Size.X/2, oval.Size.Y, oval.Size.Z/2)
		for _, across := range []float64{oval.At.X, -oval.At.X} {
			middle := maths.V(across, oval.At.Y, oval.At.Z)
			box = enclosing(box, maths.Box{Min: middle.Sub(reach),
				Max: middle.Add(reach)})
		}
	}
	for _, limb := range recipe.Limbs {
		for _, across := range []float64{1, -1} {
			from := maths.V(limb.From.X*across, limb.From.Y, limb.From.Z)
			to := maths.V(limb.To.X*across, limb.To.Y, limb.To.Z)
			box = enclosing(box, maths.SegmentBox(from, to, limb.Radius))
		}
	}
	return box, nil
}

// seen is how an eye sees the scale figure, from its extent.
func (row scaleHuman) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the plant: the box of its woody axes and roots,
// grown by the reach of the largest cell of foliage at any level and any
// variant, since a cell is posed at a tip and may turn any way.
func (plant seasonalPlant) extent(from seed) (maths.Box, error) {
	plan, err := plant.profile.Plan(from)
	if err != nil {
		return maths.Box{}, err
	}
	reach := 0.0
	for _, detail := range []string{"clumps", "quick", "demo"} {
		for variant := range int(plantValue("cell-variants")) {
			bounds, err := plant.profile.CellBounds(detail, variant)
			if err != nil {
				return maths.Box{}, err
			}
			reach = math.Max(reach, cornerReach(bounds))
		}
	}
	return grownBy(plan.Bounds(), reach), nil
}

// cornerReach is how far the furthest corner of a box is from its origin.
func cornerReach(box maths.Box) float64 {
	far := math.Max(box.Min.Len(), box.Max.Len())
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				far = math.Max(far, maths.V(x, y, z).Len())
			}
		}
	}
	return far
}

// seen is how an eye sees the plant, from its extent.
func (plant seasonalPlant) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(plant, from, at, by)
}

// extent is the box of the pier's design.
func (row pierRow) extent(from seed) (maths.Box, error) {
	design, err := cities.Pier(cities.Seed(from.number))
	if err != nil {
		return maths.Box{}, err
	}
	return designExtent(design)
}

// seen is how an eye sees the pier, from its extent.
func (row pierRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the village's design, houses and cables, drawn at
// the quick level: the hull leaves out the fittings of its houses.
func (row villageRow) extent(from seed) (maths.Box, error) {
	village, err := row.drawn(from, "quick")
	if err != nil {
		return maths.Box{}, err
	}
	return designExtentAt(village.Design(), "quick")
}

// seen is how an eye sees the village, from its extent.
func (row villageRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the village house's design, drawn at the quick
// level: the hull leaves out its fittings and its signs.
func (row villageHouseRow) extent(from seed) (maths.Box, error) {
	design, err := cities.VillageHouseAt(cities.Seed(from.number), row.ground,
		"quick")
	if err != nil {
		return maths.Box{}, err
	}
	return designExtentAt(design, "quick")
}

// seen is how an eye sees the village house, from its extent.
func (row villageHouseRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the bridge's design at time nought.
func (row bridgeRow) extent(from seed) (maths.Box, error) {
	bridge, err := row.drawn(from)
	if err != nil {
		return maths.Box{}, err
	}
	design, err := bridge.Design("hull", 0)
	if err != nil {
		return maths.Box{}, err
	}
	return designExtent(design)
}

// seen is how an eye sees the bridge, from its extent.
func (row bridgeRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the condominium's design and of each thing it
// mounts, each as that kind's own extent says, stood where it is mounted.
func (row condominiumRow) extent(from seed) (maths.Box, error) {
	design, err := cities.CondominiumAt(cities.Seed(from.number), row.ground,
		"hull")
	if err != nil {
		return maths.Box{}, err
	}
	box, err := designExtent(design)
	if err != nil {
		return maths.Box{}, err
	}
	return mountedExtent(box, design.Mounts, from)
}

// mountedExtent is a box with the extent of each mount's kind, from the
// seed, stood at the mount's place, enclosed in it.
func mountedExtent(box maths.Box, mounts []cities.Mount,
	from seed) (maths.Box, error) {
	for _, mount := range mounts {
		row, err := kindNamed(mount.Kind)
		if err != nil {
			return maths.Box{}, err
		}
		one, err := makerExtent(row.how, from)
		if err != nil {
			return maths.Box{}, err
		}
		box = enclosing(box, transformed(one, mount.Place))
	}
	return box, nil
}

// seen is how an eye sees the condominium, from its extent.
func (row condominiumRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the tower's design drawn at the hull, grown by what
// a tower is given beyond its design.
func (row towerRow) extent(from seed) (maths.Box, error) {
	tower, err := row.drawn(from)
	if err != nil {
		return maths.Box{}, err
	}
	design, err := tower.Design("hull", 0)
	if err != nil {
		return maths.Box{}, err
	}
	box, err := designExtent(design)
	if err != nil {
		return maths.Box{}, err
	}
	return grownBy(box, towerFittings), nil
}

// seen is how an eye sees the tower, from its extent.
func (row towerRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the superblock's design, and of each berth on its
// roofs, since a part is stood in a berth only if it fits inside it.
func (row superblockRow) extent(from seed) (maths.Box, error) {
	ask, err := row.asked("hull")
	if err != nil {
		return maths.Box{}, err
	}
	block, err := cities.NewSuperblock(ask, cities.Seed(from.number))
	if err != nil {
		return maths.Box{}, err
	}
	box, err := designExtent(block.Design())
	if err != nil {
		return maths.Box{}, err
	}
	for _, berth := range block.Berths {
		box = enclosing(box, maths.Box{
			Min: maths.V(berth.Low[0], berth.Low[1], berth.Low[2]),
			Max: maths.V(berth.High[0], berth.High[1], berth.High[2])})
	}
	return box, nil
}

// seen is how an eye sees the superblock, from its extent.
func (row superblockRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of the lashed street's blocks and cables, and of each
// slot a sign hangs in, which a sign is scaled to fit inside: a sign is no
// wider than the slot's width and no taller than its height, and is no
// thicker than it is wide.
func (row lashedStreetRow) extent(from seed) (maths.Box, error) {
	rows, lashings, err := row.drawn(from)
	if err != nil {
		return maths.Box{}, err
	}
	design := rows.Design()
	design.Elements = append(design.Elements, lashings.Design().Elements...)
	box, err := designExtent(design)
	if err != nil || row.spec.Sign == "" {
		return box, err
	}
	for _, slot := range lashings.Slots {
		reach := slot.Width + slot.Height
		box = enclosing(box, maths.Box{
			Min: slot.Middle.Sub(maths.V(reach, reach, reach)),
			Max: slot.Middle.Add(maths.V(reach, reach, reach))})
	}
	return box, nil
}

// seen is how an eye sees the lashed street, from its extent.
func (row lashedStreetRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}

// extent is the box of a boat from its hull's figures and its design's
// ranges: along its length, past either end by as much as a boom or an
// outboard reaches, across its beam and as much again, below its draught
// and keel, and up by every height its fittings add one on another.
func (row boatRow) extent(from seed) (maths.Box, error) {
	design := boatRead.Kinds[row.design]
	hull := newBoatHull(design, from, 0)
	along := hull.length/2 + design.pick("boom-length", from) +
		design.pick("outboard", from) + boatGear
	across := hull.beam + boatGear
	up := hull.freeboard + hull.spring + hull.rail + boatGear
	for _, name := range []string{"mast-height", "funnel-height",
		"house-height", "canopy-height", "upper-deck"} {
		up += design.pick(name, from)
	}
	return maths.Box{
		Min: maths.V(-across, -(hull.draught + boatKeelMargin), -along),
		Max: maths.V(across, up, along)}, nil
}

// seen is how an eye sees the boat, from its extent.
func (row boatRow) seen(from seed, at maths.Subspace,
	by strategies.Eye) (strategies.Seen, error) {
	return seenFrom(row, from, at, by)
}
