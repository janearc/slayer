package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// Cities is a world's lights at night, as a shine over its ground, laid
// as the black marble looks, by eye: metros on a coarse rung, two below
// Rungs; towns on the fine rung, thrice as many round a metro. Metro and
// Share are the shares lit where the weight below is one.
//
// the weight is more near a coast, six tenths again, and a bell round
// thirty-five degrees: 1.6 at its height, 0.19 at the equator, 0.63 at
// fifty, 0.15 at the poles, measured. its numbers are chosen, not from
// a source.
//
// Land says whether a point is land and Coast whether it is near
// water; without Coast every point is inland. Each metro and town
// Glows, a metro four times a town. By day the sun drowns them, so
// nothing is added for day. Seed makes a world's own cities.
type Cities struct {
	Tiling hexes.Tiling
	Land   func(point maths.Vec) bool
	Coast  func(point maths.Vec) bool
	Rungs  int
	Share  float64
	Metro  float64
	Glow   maths.Light
	Seed   uint64
}

// The streams of a city's hash, each defined as a distinct offset from
// the seed, so that one tile's metro, size and clump are drawn alone.
const (
	// sizeStream is defined: the offset of the stream a tile's size is
	// drawn from.
	sizeStream = 1
	// metroStream is defined: the offset of the stream a metro is drawn
	// from.
	metroStream = 2
	// clumpStream is defined: the offset of the stream a tile's clump is
	// drawn from.
	clumpStream = 3
)

// the numbers Cities is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// cityBellCentre is the latitude, in degrees, a city's weight peaks at.
	cityBellCentre = pickedNumber("city-bell-centre")
	// cityBellWidth is the spread of the weight's bell, in degrees.
	cityBellWidth = pickedNumber("city-bell-width")
	// cityFloor is the weight at the poles, where the bell is nothing.
	cityFloor = pickedNumber("city-floor")
	// cityBellHeight is the bell's height over the floor.
	cityBellHeight = pickedNumber("city-bell-height")
	// cityCoastBoost is how many times a coast's weight is an inland one's.
	cityCoastBoost = pickedNumber("city-coast-boost")
	// metroRungs is how many rungs below the fine one a metro's are.
	metroRungs = int(pickedNumber("city-metro-rungs"))
	// townMultiple is how many times likelier a town is next to a metro,
	// and at its heart.
	townMultiple = pickedNumber("city-town-multiple")
	// heartBoost is how much brighter than a town a metro's heart is, less
	// the town's own light.
	heartBoost = pickedNumber("city-heart-boost")
	// sizeFloor is the least a city's size is, as a share of the most.
	sizeFloor = pickedNumber("city-size-floor")
	// sizeSpread is how far a city's size is drawn above its floor.
	sizeSpread = pickedNumber("city-size-spread")
	// clumpPerSpacing is how many clumps of lights fit in a tile's width.
	clumpPerSpacing = pickedNumber("city-clump-per-spacing")
	// clumpOctaves is how many scales of noise a clump of lights has.
	clumpOctaves = int(pickedNumber("city-clump-octaves"))
	// clumpFloor is the least a clump dims a city's light to.
	clumpFloor = pickedNumber("city-clump-floor")
	// edgeFade is how far in from its edge, as a share of the way to its
	// middle, a tile's light has come to its full.
	edgeFade = pickedNumber("city-edge-fade")
)

// rung is the tiling n rungs below the world's.
func (cities Cities) rung(rungs int) hexes.Tiling {
	tiling := cities.Tiling
	for range max(nought, rungs) {
		tiling = tiling.Finer()
	}
	return tiling
}

// weight is how much a place wants a city: the coast's six tenths again
// and the latitude's bell.
func (cities Cities) weight(point maths.Vec) float64 {
	sine := clamped(point.Unit().Y)
	latitude := math.Abs(math.Asin(sine)) / maths.Degree
	offset := latitude - cityBellCentre
	bell := math.Exp(-offset * offset / (pair * cityBellWidth * cityBellWidth))
	weight := cityFloor + cityBellHeight*bell
	if cities.Coast != nil && cities.Coast(point) {
		weight *= cityCoastBoost
	}
	return weight
}

// metroTile is whether a coarse tile holds a metro.
func (cities Cities) metroTile(coarse hexes.Tiling, tile hexes.Tile) bool {
	centre := coarse.Centre(tile)
	chance := noise.Hash(cities.Seed+metroStream, coarse.ID(tile))
	return cities.Land(centre) && chance < cities.Metro*cities.weight(centre)
}

// metro is how near a point is to a metro's heart, from one at the heart to
// zero a coarse tile's width out, and whether it is in or next to a metro's
// coarse tile at all, for the towns round it.
func (cities Cities) metro(point maths.Vec) (core float64, near bool) {
	if cities.Metro <= nought {
		return nought, false
	}
	coarse := cities.rung(cities.Rungs - metroRungs)
	tile := coarse.Nearest(point)
	if cities.metroTile(coarse, tile) {
		heart := coarse.Centre(tile).Scale(-unity)
		apart := point.Unit().Add(heart).Len()
		return math.Max(nought, unity-apart/coarse.Spacing()), true
	}
	for _, neighbour := range coarse.Neighbours(tile) {
		if cities.metroTile(coarse, neighbour) {
			return nought, true
		}
	}
	return nought, false
}

// litShare is the share of tiles lit at a place, and how bright they are:
// in a metro every tile is lit at its heart, sprawl thinning out from it,
// the heart four times a town's light, and next to one the towns are
// likelier.
func (cities Cities) litShare(centre maths.Vec, core float64,
	near bool) (share, glow float64) {
	share = cities.Share * cities.weight(centre)
	glow = unity
	switch {
	case core > nought:
		return math.Max(share*townMultiple, core*core), unity + heartBoost*core
	case near:
		share *= townMultiple
	}
	return share, glow
}

// At is the light at the point: a city's glow or nothing.
func (cities Cities) At(point maths.Vec) maths.Light {
	if cities.Tiling == nil || cities.Land == nil || cities.Share <= nought {
		return noPaint
	}
	fine := cities.rung(cities.Rungs)
	tile := fine.Nearest(point)
	id := fine.ID(tile)
	centre := fine.Centre(tile)
	if !cities.Land(centre) {
		return noPaint
	}
	core, near := cities.metro(point)
	share, glow := cities.litShare(centre, core, near)
	if noise.Hash(cities.Seed, id) > share {
		return noPaint
	}
	return cities.shine(fine, tile, id, point, glow)
}

// shine is the light of a lit tile at the point: a city's heart is bright
// and its edge is not, and within it the light is clumped by a fine
// noise, so a tile reads as a cluster of lights and not as a hexagon.
func (cities Cities) shine(fine hexes.Tiling, tile hexes.Tile, id int64,
	point maths.Vec, glow float64) maths.Light {
	edge := fine.Edge(point, tile) / (half * fine.Spacing())
	size := sizeFloor + sizeSpread*noise.Hash(cities.Seed+sizeStream, id)
	scale := clumpPerSpacing / fine.Spacing()
	clump := half + noise.Fractal(cities.Seed+clumpStream, point.X*scale,
		point.Y*scale, point.Z*scale, clumpOctaves)
	lit := glow * size * maths.Smooth(nought, edgeFade, edge)
	return cities.Glow.Scale(lit * math.Max(clumpFloor, clump))
}
