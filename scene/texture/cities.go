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
	Land   func(p maths.Vec) bool
	Coast  func(p maths.Vec) bool
	Rungs  int
	Share  float64
	Metro  float64
	Glow   maths.Light
	Seed   uint64
}

// rung is the tiling n rungs below the world's.
func (c Cities) rung(n int) hexes.Tiling {
	t := c.Tiling
	for range max(0, n) {
		t = t.Finer()
	}
	return t
}

// weight is how much a place wants a city: the coast's half again
// and the latitude's bell.
func (c Cities) weight(p maths.Vec) float64 {
	sine := math.Max(-1, math.Min(1, p.Unit().Y))
	lat := math.Abs(math.Asin(sine)) / maths.Degree
	bell := math.Exp(-(lat - 35) * (lat - 35) / (2 * 14 * 14))
	w := 0.15 + 0.85*bell
	if c.Coast != nil && c.Coast(p) {
		w *= 1.6
	}
	return w
}

// metro is how near a point is to a metro's heart, from one at the heart to
// zero a coarse tile's width out, and whether it is in or next to a metro's
// coarse tile at all, for the towns round it.
func (c Cities) metro(p maths.Vec) (core float64, near bool) {
	if c.Metro <= 0 {
		return 0, false
	}
	coarse := c.rung(c.Rungs - 2)
	tile := coarse.Nearest(p)
	is := func(t hexes.Tile) bool {
		centre := coarse.Centre(t)
		chance := noise.Hash(c.Seed+2, coarse.ID(t))
		return c.Land(centre) && chance < c.Metro*c.weight(centre)
	}
	if is(tile) {
		apart := p.Unit().Add(coarse.Centre(tile).Scale(-1)).Len()
		return math.Max(0, 1-apart/coarse.Spacing()), true
	}
	for _, n := range coarse.Neighbours(tile) {
		if is(n) {
			return 0, true
		}
	}
	return 0, false
}

// At is the light at p: a city's glow or nothing.
func (c Cities) At(p maths.Vec) maths.Light {
	if c.Tiling == nil || c.Land == nil || c.Share <= 0 {
		return maths.Light{}
	}
	fine := c.rung(c.Rungs)
	tile := fine.Nearest(p)
	id := fine.ID(tile)
	centre := fine.Centre(tile)
	if !c.Land(centre) {
		return maths.Light{}
	}
	core, near := c.metro(p)
	share := c.Share * c.weight(centre)
	glow := 1.0
	switch {
	case core > 0:
		// a metro: every tile lit at its heart, sprawl thinning out
		// from it, the heart four times a town's light
		share, glow = math.Max(share*3, core*core), 1+3*core
	case near:
		share *= 3
	}
	if noise.Hash(c.Seed, id) > share {
		return maths.Light{}
	}
	// a city's heart is bright and its edge is not, and within it the
	// light is clumped by a fine noise, so a tile reads as a cluster of
	// lights and not as a hexagon
	edge := fine.Edge(p, tile) / (0.5 * fine.Spacing())
	size := 0.4 + 0.6*noise.Hash(c.Seed+1, id)
	k := 3 / fine.Spacing()
	clump := 0.5 + noise.Fractal(c.Seed+3, p.X*k, p.Y*k, p.Z*k, 2)
	lit := glow * size * maths.Smooth(0, 0.6, edge)
	return c.Glow.Scale(lit * math.Max(0.2, clump))
}
