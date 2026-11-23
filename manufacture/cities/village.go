package cities

import (
	"cmp"
	"fmt"

	"github.com/janearc/slayer/maths"
)

// A village is a parcel packed with village houses, as an urban village
// is built: plot after plot across the parcel and down it, each house to
// the edges of its own plot, the plots an alley apart, so close that
// neighbours could shake hands across, and every so many plots a lane.
//
// The parcel's edges are lanes too. A house's walls that face a lane have
// its shops and its signs; the alleys are dark slots between walls with a
// strip of sky over them, lit by the stair doors and the windows.
//
// Cables are hung across the lanes by the lashings, and across the alleys
// by the alley's lashings, a variant of them for gaps a few metres wide:
// mostly bundles of wire. Every size is the culture's, by the seed; the
// parcel and the seed are the set's.

// alleyCulture is the variant of the lashings an alley is hung by.
const alleyCulture = "alley"

// alleySalt is mixed into the seed of the alleys' lashings, so that they
// are not drawn as the lanes' are.
const alleySalt = 0x616c6c6579

// VillageAsk is what a set asks of a village: the width of its parcel
// along x and its depth along z in metres, nought for the culture's own,
// and the finest level its houses are drawn at, the quick level if none.
type VillageAsk struct {
	Width, Depth float64
	Level        Detail
}

// Village is a parcel built: its houses, in the order of their plots, and
// how many cables are hung across its lanes and its alleys.
type Village struct {
	Ask    VillageAsk
	Houses []villageHouse
	Cables int
	parts  []placedElement
}

// strip is one plot's run along an axis of the parcel, and whether a lane
// and not an alley lies before it and after it.
type strip struct {
	low, high             float64
	laneBefore, laneAfter bool
}

// NewVillage packs a parcel with houses by a seed, on the ground given or
// on flat ground if none, which is paved. It refuses a parcel below
// nought, one that holds not one plot, a level a building is not drawn at,
// and anything the culture cannot say.
func NewVillage(ask VillageAsk, seed Seed, ground Terrain) (Village, error) {
	drawing, err := villageDrawer(seed, ask.Level)
	if err != nil {
		return Village{}, err
	}
	if ask.Width < 0 || ask.Depth < 0 {
		return Village{}, fmt.Errorf("cities: a village's parcel is not "+
			"below nought, not %v by %v m", ask.Width, ask.Depth)
	}
	ask.Width = cmp.Or(ask.Width, drawing.fixed("default-width"))
	ask.Depth = cmp.Or(ask.Depth, drawing.fixed("default-depth"))
	across := drawing.strips(ask.Width, acrossX)
	along := drawing.strips(ask.Depth, acrossZ)
	if len(across) == 0 || len(along) == 0 {
		return Village{}, fmt.Errorf("cities: a parcel of %v by %v m holds "+
			"no plot", ask.Width, ask.Depth)
	}
	village := Village{Ask: ask}
	if err := village.build(drawing, across, along, ground, seed); err != nil {
		return Village{}, err
	}
	return village, village.hang(seed)
}

// strips are the plots along one axis of a parcel of a length, centred on
// its middle: each as wide as its draw, an alley after each, and a lane
// after so many of them. The parcel's edges are lanes.
func (drawing *drawer) strips(length float64, axis int) []strip {
	var runs []strip
	next, since, before := 0.0, 0, true
	every := drawing.count("lane-every", place{Row: axis})
	for index := 0; drawing.trouble == nil; index++ {
		at := place{Row: axis, Bay: index}
		size := drawing.number("plot-size", at)
		if next+size > length {
			break
		}
		since++
		lane := since >= every
		gap := drawing.number("alley-width", at)
		if lane {
			gap, since = drawing.number("lane-width", at), 0
		}
		runs = append(runs, strip{low: next, high: next + size,
			laneBefore: before, laneAfter: lane})
		next, before = next+size+gap, lane
	}
	return centred(runs)
}

// centred is the runs moved so that their middle is the parcel's, the
// last with a lane after it, for it is the parcel's edge.
func centred(runs []strip) []strip {
	if len(runs) == 0 {
		return nil
	}
	runs[len(runs)-1].laneAfter = true
	shift := -runs[len(runs)-1].high / half
	for index := range runs {
		runs[index].low += shift
		runs[index].high += shift
	}
	return runs
}

// build draws a house on every plot the seed does not leave open, and the
// paving under them where the ground is flat.
func (village *Village) build(drawing *drawer, across, along []strip,
	ground Terrain, seed Seed) error {
	if ground == nil {
		village.parts = append(village.parts, village.paving(drawing))
	}
	for column, x := range across {
		for row, z := range along {
			index := column*len(along) + row
			if drawing.chance("open-plot", place{Row: index}) {
				continue
			}
			lanes := [sidesAround]bool{frontSide: z.laneAfter,
				eastSide: x.laneAfter, backSide: z.laneBefore,
				westSide: x.laneBefore}
			lot := drawing.setBack(index, rect{x.low, z.low, x.high, z.high})
			house, err := drawing.villageHouse(index, lot, lanes, ground,
				seed)
			if err != nil {
				return err
			}
			village.Houses = append(village.Houses, house)
			village.parts = append(village.parts, house.placed())
		}
	}
	return drawing.trouble
}

// setBack is a plot's lot drawn in from each of its edges by the draw for
// that side, so that no two alleys are quite the same width.
func (drawing *drawer) setBack(index int, plot rect) rect {
	back := func(side int) float64 {
		return drawing.number("setback", place{Row: index, Bay: side})
	}
	return rect{X0: plot.X0 + back(westSide), Z0: plot.Z0 + back(backSide),
		X1: plot.X1 - back(eastSide), Z1: plot.Z1 - back(frontSide)}
}

// paving is the parcel's floor on flat ground: a thin slab under the
// lanes, the alleys and the houses, wet.
func (village *Village) paving(drawing *drawer) placedElement {
	ask := village.Ask
	look := paint("paving", drawing.pick("paving-colour",
		drawing.options("paving-colour"), place{}))
	return put("paving", Box{ask.Width, drawing.fixed("paving-thickness"),
		ask.Depth}, look, maths.V(-ask.Width/half, 0, -ask.Depth/half))
}

// hang strings the lanes and the alleys between the houses' walls, by
// the lashings and by the alley's variant of them, each at the density
// the culture gives it.
func (village *Village) hang(seed Seed) error {
	drawing, err := newDrawer(villageCulture, seed)
	if err != nil {
		return err
	}
	volumes := make([]Volume, len(village.Houses))
	for index, house := range village.Houses {
		volumes[index] = house.volume(index)
	}
	asks := []struct {
		ask  LashingsAsk
		seed Seed
	}{{LashingsAsk{Density: drawing.fixed("lane-density")}, seed},
		{LashingsAsk{Density: drawing.fixed("alley-density"),
			Culture: alleyCulture}, seed ^ alleySalt}}
	for _, one := range asks {
		hung, err := NewLashings(one.ask, volumes, one.seed)
		if err != nil {
			return err
		}
		village.Cables += hung.Cables
		village.parts = append(village.parts, hung.Design().Elements...)
	}
	return drawing.trouble
}

// Design is the village as the parts it is drawn from: its paving, each
// house a node a pass levels, and the cables over its lanes and alleys.
func (village Village) Design() Design {
	return Design{Name: "village", Elements: village.parts}
}
