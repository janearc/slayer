package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// rungSpec is what one rung of a made-up tiling says: how far apart its
// middles are, how far from an edge every point is, which tile it names
// for any point, where each tile's middle is, and which tiles each one
// touches.
//
// the numbers are the test's own, so a city's share and its light can be
// worked by hand.
type rungSpec struct {
	spacing float64
	edge    float64
	nearest int
	centres map[int]maths.Vec
	around  map[int][]int
}

// rungs is a tiling made of rungSpecs, one for each rung down from the
// world's. Finer is the next spec; a rung past the last is an index out
// of range, and a method not defined here panics for the nil tiling it
// embeds, so a test learns at once when the texture asks for more.
type rungs struct {
	hexes.Tiling
	at    int
	specs []rungSpec
}

func (r rungs) spec() rungSpec { return r.specs[r.at] }

func (r rungs) Nearest(maths.Vec) hexes.Tile { return hexes.Tile{I: r.spec().nearest} }

func (r rungs) Centre(tile hexes.Tile) maths.Vec { return r.spec().centres[tile.I] }

func (r rungs) Edge(maths.Vec, hexes.Tile) float64 { return r.spec().edge }

func (r rungs) ID(tile hexes.Tile) int64 { return int64(tile.I) }

func (r rungs) Spacing() float64 { return r.spec().spacing }

func (r rungs) Finer() hexes.Tiling { return rungs{at: r.at + 1, specs: r.specs} }

func (r rungs) Neighbours(tile hexes.Tile) []hexes.Tile {
	var out []hexes.Tile
	for _, i := range r.spec().around[tile.I] {
		out = append(out, hexes.Tile{I: i})
	}
	return out
}

// the places the cities tests stand on, all unit vectors. heart is a metro's
// middle, 6/13 from the direction (3, 4, 12)/13 that a point of (0.3, 0.4, 1.2)
// looks toward: the difference of the two is (6/13, 0, 0).
//
// deep is a middle at sea, since a middle is land when its Z is not below zero.
// town is a fine tile's middle at 35 degrees of latitude, where the weight is
// one (it is the bell's own height).
var (
	heart = maths.V(-3.0/13, 4.0/13, 12.0/13)
	deep  = maths.V(0.6, 0, -0.8)
	town  = maths.V(math.Cos(35*maths.Degree), math.Sin(35*maths.Degree), 0)
)

// inland is land by the Z of a middle, as the stand-ins above say.
func inland(p maths.Vec) bool { return p.Z >= 0 }

// junk is a rung the test never means to be asked about: if it is, the
// numbers say so at once.
var junk = rungSpec{spacing: 99, edge: 99, nearest: 999}
