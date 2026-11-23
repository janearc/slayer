package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// The harbor seal's coat pattern by rule: "many fine to medium sized
// spots, usually with smaller numbers of ring-like markings and blotches"
// (National Museum of Nature and Science, Tokyo, Marine Mammals
// Information Database, harbor seal).
//
// Space is cut into cells; a cell holds at most one mark, a spot, a ring
// or a blotch, more often on the back than the belly. Every number below
// is chosen.
const (
	sealSpotCell      = .03  // a cell's edge, in metres
	sealRingShare     = .14  // of marked cells, the share holding a ring
	sealBlotchShare   = .08  // and the share holding a blotch
	sealFineRadius    = .004 // the smallest spot's radius, in metres
	sealMediumRadius  = .011 // the largest spot's
	sealRingRadius    = .012 // a ring's middle radius, in metres
	sealRingWidth     = .003 // a ring's half width
	sealBlotchRadius  = .015 // a blotch's lobes' radius, in metres
	sealBackMarked    = .85  // the share of cells marked on the back
	sealBellyMarked   = .35  // and on the belly
	sealBackFrom      = -.10 // the rest height where the back's density starts
	sealBackTo        = .12  // and where it is whole
	sealMarkEdge      = .0008
	sealCellMargin    = .25 // a mark's centre keeps this share from its cell's walls
	sealRingLightened = .6  // how much a ring's middle lightens the coat
)

// The kinds of mark a cell holds.
const (
	sealMarkSpot = iota
	sealMarkRing
	sealMarkBlotch
)

// sealMarks is how dark the coat's marks make a rest point, and how much
// a ring's light middle lifts it, each from none to one. Seen from far,
// where a pixel covers several cells, the marks blend to their mean.
func sealMarks(seed uint64, point maths.Vec, footprint float64) (float64,
	float64,
) {
	cell := point.Scale(1 / sealSpotCell)
	x, y, z := math.Floor(cell.X), math.Floor(cell.Y), math.Floor(cell.Z)
	edge := math.Max(sealMarkEdge, footprint)
	density := sealBellyMarked + (sealBackMarked-sealBellyMarked)*
		maths.Smooth(sealBackFrom, sealBackTo, point.Y)
	var dark, light float64
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				mark, lift := sealCellMark(seed, point, x+float64(dx),
					y+float64(dy), z+float64(dz), density, edge)
				dark, light = math.Max(dark, mark), math.Max(light, lift)
			}
		}
	}
	far := maths.Smooth(sealSpotCell/4, sealSpotCell, footprint)
	mean := density * sealMeanCover
	return dark*(1-far) + mean*far, light * (1 - far)
}

// sealMeanCover is the share of a marked cell's surface its mark darkens
// on average, for the far blend; worked from the radii above as a disc's
// area over a cell face, rounded. Chosen.
const sealMeanCover = .12

// sealCellMark is the mark of the cell with lowest corner x, y, z, at a
// point: its darkness and its light middle.
func sealCellMark(seed uint64, point maths.Vec, x, y, z, density,
	edge float64,
) (float64, float64) {
	i, j, k := int64(x), int64(y), int64(z)
	if noise.Hash3(seed+11, i, j, k) > density {
		return 0, 0
	}
	place := func(salt uint64) float64 {
		return sealCellMargin + (1-2*sealCellMargin)*noise.Hash3(seed+salt, i, j, k)
	}
	centre := maths.V(x+place(12), y+place(13), z+place(14)).Scale(sealSpotCell)
	distance := point.Sub(centre).Len()
	size := noise.Hash3(seed+15, i, j, k)
	switch sealMarkKind(noise.Hash3(seed+16, i, j, k)) {
	case sealMarkRing:
		radius := sealRingRadius * (.8 + .4*size)
		band := 1 - maths.Smooth(sealRingWidth-edge, sealRingWidth+edge,
			math.Abs(distance-radius))
		inside := 1 - maths.Smooth(radius-sealRingWidth-edge, radius-sealRingWidth,
			distance)
		return band, inside * sealRingLightened
	case sealMarkBlotch:
		return sealBlotchMark(seed, point, centre, i, j, k, edge), 0
	default:
		radius := sealFineRadius + (sealMediumRadius-sealFineRadius)*size*size
		return 1 - maths.Smooth(radius-edge, radius+edge, distance), 0
	}
}

// sealMarkKind is which kind of mark a share from nought to one picks.
func sealMarkKind(share float64) int {
	switch {
	case share < sealRingShare:
		return sealMarkRing
	case share < sealRingShare+sealBlotchShare:
		return sealMarkBlotch
	default:
		return sealMarkSpot
	}
}

// sealBlotchMark is a blotch: two overlapping lobes about the centre, at
// a turn and spread the cell's hashes pick.
func sealBlotchMark(seed uint64, point, centre maths.Vec, i, j, k int64,
	edge float64,
) float64 {
	turn := 2 * math.Pi * noise.Hash3(seed+17, i, j, k)
	spread := sealBlotchRadius * (.4 + .4*noise.Hash3(seed+18, i, j, k))
	offset := maths.V(math.Cos(turn), math.Sin(turn), math.Cos(2*turn)).
		Unit().Scale(spread / 2)
	var mark float64
	for _, lobe := range [...]maths.Vec{centre.Add(offset), centre.Sub(offset)} {
		distance := point.Sub(lobe).Len()
		mark = math.Max(mark, 1-maths.Smooth(sealBlotchRadius*.7-edge,
			sealBlotchRadius*.7+edge, distance))
	}
	return mark
}
