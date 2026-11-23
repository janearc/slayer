package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The numbers Wood falls back on, and the two waves its rings wander by,
// each picked and held in picked-numbers.yaml.
var (
	// woodRing is a year's growth when none is given.
	woodRing = pickedNumber("wood-ring")
	// woodLate is the share of a year's ring that is late wood, when no
	// share is given.
	woodLate = pickedNumber("wood-late")
	// woodLong is the long wave the rings wander by, in turns along the
	// grain.
	woodLong = pickedNumber("wood-long")
	// woodShort is the short wave the rings wander by, in turns along the
	// grain, not a multiple of the long one.
	woodShort = pickedNumber("wood-short")
	// woodShortBit is how big the short wave is, against the long one.
	woodShortBit = pickedNumber("wood-short-bit")
	// woodSoft is the share of a ring over which early wood turns to late,
	// and back.
	woodSoft = pickedNumber("wood-soft")
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

// ring is a year's width, with none read as the fallback.
func (wood Wood) ring() float64 {
	if wood.Ring <= nought {
		return woodRing
	}
	return wood.Ring
}

// lateShare is how much of a ring is late wood, with none read as the
// fallback.
func (wood Wood) lateShare() float64 {
	if wood.LateShare <= nought {
		return woodLate
	}
	return wood.LateShare
}

// At is the wood at the point: early or late by how far through its year
// the point's ring is, blended across the turn from one to the other.
func (wood Wood) At(point maths.Vec) maths.Light {
	late := wood.late(point)
	return wood.Early.Scale(unity - late).Add(wood.Late.Scale(late))
}

// late is how much of the wood at the point is late wood, from nought to
// one.
func (wood Wood) late(point maths.Vec) float64 {
	along := wood.Along
	if along.Len() == nought {
		along = maths.XAxis
	}
	along = along.Unit()
	ring := wood.ring()
	share := wood.lateShare()
	from := point.Sub(wood.Pith)
	run := from.Dot(along)
	out := from.Sub(along.Scale(run)).Len()
	wave := math.Sin(woodLong*run) + woodShortBit*math.Sin(woodShort*run)
	years := out/ring + wood.Wander*wave
	through := years - math.Floor(years)
	// early wood from the ring's start, turning to late wood at the last
	// share of it, and back to early at the year's end, each over a soft
	// edge so a ring's line is not a pixel's stair
	start := unity - share
	rise := smoothed((through - start) / woodSoft)
	fall := smoothed((unity - through) / woodSoft)
	return rise * fall
}

// smoothed is a step from nought to one over nought to one, eased at
// both ends.
func smoothed(x float64) float64 {
	held := math.Max(nought, math.Min(unity, x))
	return held * held * (smoothCubic - pair*held)
}
