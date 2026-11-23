package thing

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers a chisel's facets are drawn with, each read from picks.yaml
var (
	// facetWander is how far, in widths of a facet, the lattice the facets
	// lie on is pushed about by noise.
	facetWander = pick("facet-wander")
	// facetOctaves is how many sizes of noise push the lattice about.
	facetOctaves = int(pick("facet-octaves"))
	// facetSalt keeps one axis's noise far from another's in the noise's
	// seeds, which share fields a small step apart.
	facetSalt = uint64(pick("facet-salt"))
)

// the large primes that keep the three counts of a cell apart in one
// number, so that two cells differ in the key. Each is read from picks.yaml.
var (
	facetKeyX = int64(pick("facet-key-x"))
	facetKeyY = int64(pick("facet-key-y"))
	facetKeyZ = int64(pick("facet-key-z"))
)

// the counts a chisel's arithmetic is written with
const (
	// facetNone is defined: no amount.
	facetNone = 0
	// facetWhole is defined: one amount, the whole of a share.
	facetWhole = 1
	// facetSecondSalt is defined: the salts the third axis's seed lies
	// from the first's, one more than the second axis's.
	facetSecondSalt = 2
	// facetSpan is how wide a tip is drawn, from minus half of it to plus
	// half: a draw of nought to one, made from minus one to one. It is
	// defined by that range.
	facetSpan = 2
	// facetHalf is defined: half of a draw's range of nought to one.
	facetHalf = 0.5
)

// chisel is the tilt a stone carver's tool leaves in the normal at a
// point: the surface is cut in flat facets about Size across, each tipped
// a little this way or that, so that a face is lit in patches and not
// evenly. The facets lie on a lattice pushed about by noise, so they are
// not in rows.
//
// Facet is the most a facet tips the normal, as the share of
// a unit; shown is how much of the detail the pixel can see, from nought
// to one, so a far wall is not shimmering.
func (crinkle *Crinkle) chisel(point, normal maths.Vec,
	shown float64) maths.Vec {
	if !crinklePositive(crinkle.Facet) {
		return maths.V(facetNone, facetNone, facetNone)
	}
	at := point.Scale(facetWhole / crinkle.Size)
	seed := uint64(crinkle.Seed)
	pushed := maths.V(
		noise.Fractal(seed, at.X, at.Y, at.Z, facetOctaves),
		noise.Fractal(seed+facetSalt, at.X, at.Y, at.Z, facetOctaves),
		noise.Fractal(seed+facetSecondSalt*facetSalt,
			at.X, at.Y, at.Z, facetOctaves),
	).Scale(facetWander).Add(at)
	cell := facetCell(pushed)
	tip := maths.V(noise.Hash(seed, cell)-facetHalf,
		noise.Hash(seed+facetSalt, cell)-facetHalf,
		noise.Hash(seed+facetSecondSalt*facetSalt, cell)-facetHalf).
		Scale(facetSpan)
	along := tip.Dot(normal)
	across := tip.Sub(normal.Scale(along))
	return across.Scale(crinkle.Facet * shown)
}

// facetCell is the key of the lattice cell a point lies in: the whole
// counts of the cell along each axis, each taken by its own large prime.
func facetCell(pushed maths.Vec) int64 {
	return int64(math.Floor(pushed.X))*facetKeyX ^
		int64(math.Floor(pushed.Y))*facetKeyY ^
		int64(math.Floor(pushed.Z))*facetKeyZ
}
