package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The numbers Wood falls back on, and the two waves its rings wander by.
const (
	woodRing     = 0.1  // picked: a year's growth, a tenth of a unit, when none is given
	woodLate     = 0.25 // picked: a quarter of a year's ring is late wood, when no share is given
	woodLong     = 0.7  // picked: the long wave, under a turn in a unit along the grain
	woodShort    = 2.9  // picked: the short wave, about three turns in a unit, not a multiple of the long one
	woodShortBit = 0.35 // picked: the short wave is a third the size of the long one
	woodSoft     = 0.08 // picked: the share of a ring over which early wood turns to late, and back
)

// Wood is sawn timber: the rings a tree grew, cut through by whatever
// faces the board has. It is a function of position in the thing's own
// space, as Checker is, so a board shows rings on its end, long grain on
// its faces, and the two meet at every edge as they do in a real board.
//
// The tree's pith is a line through Pith along Along. A point's distance
// from that line, counted in Rings, says which year it grew in and how
// far through the year: pale Early wood for most of it, dark Late wood at
// the end. The rings wander a little along the grain, by Wander ring
// widths, as no trunk is a true cylinder; nought is rings as true as a
// lathe's.
type Wood struct {
	// Early and Late are the pale wood a tree grows fast in spring and
	// the dark wood it grows slowly at the end of summer.
	Early, Late maths.Light
	// Pith is a point the tree's pith passes through.
	Pith maths.Vec
	// Along is which way the grain runs. Zero means along x.
	Along maths.Vec
	// Ring is a year's width. Zero means a tenth of a unit.
	Ring float64
	// LateShare is how much of a ring is late wood, from nought to one.
	// Zero means a quarter.
	LateShare float64
	// Wander is how far the rings wander along the grain, in ring
	// widths.
	Wander float64
}

// At is the wood at p: early or late by how far through its year the
// point's ring is, blended across the turn from one to the other.
func (wood Wood) At(p maths.Vec) maths.Light {
	late := wood.late(p)
	return wood.Early.Scale(1 - late).Add(wood.Late.Scale(late))
}

// late is how much of the wood at p is late wood, from nought to one.
func (wood Wood) late(p maths.Vec) float64 {
	along := wood.Along
	if along.Len() == 0 {
		along = maths.XAxis
	}
	along = along.Unit()
	ring := wood.Ring
	if ring <= 0 {
		ring = woodRing
	}
	share := wood.LateShare
	if share <= 0 {
		share = woodLate
	}
	from := p.Sub(wood.Pith)
	run := from.Dot(along)
	out := from.Sub(along.Scale(run)).Len()
	wave := math.Sin(woodLong*run) + woodShortBit*math.Sin(woodShort*run)
	years := out/ring + wood.Wander*wave
	through := years - math.Floor(years)
	// early wood from the ring's start, turning to late wood at the last
	// share of it, and back to early at the year's end, each over a soft
	// edge so a ring's line is not a pixel's stair
	start := 1 - share
	rise := smoothed((through - start) / woodSoft)
	fall := smoothed((1 - through) / woodSoft)
	return rise * fall
}

// smoothed is a step from nought to one over nought to one, eased at
// both ends.
func smoothed(x float64) float64 {
	t := math.Max(0, math.Min(1, x))
	return t * t * (3 - 2*t)
}
