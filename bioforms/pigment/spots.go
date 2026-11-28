// Package pigment samples species-independent markings in rest-space metres.
package pigment

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// Spots scatters softened spherical pigment deposits in a jittered lattice.
// Spacing is the cell width. Radius is the maximum spot radius, up to half a cell.
type Spots struct {
	Seed            uint64
	Spacing, Radius float64
}

func scramble(n uint64) uint64 {
	n ^= n >> 30
	n *= 0xbf58476d1ce4e5b9
	n ^= n >> 27
	n *= 0x94d049bb133111eb
	return n ^ (n >> 31)
}
func fraction(n uint64) float64 { return float64(n>>11) / 9007199254740992 }

// At gives pigment coverage at a rest point, with a soft edge.
func (s Spots) At(p maths.Vec) float64 {
	if !positive(s.Spacing) || !positive(s.Radius) {
		return 0
	}
	p = p.Scale(1 / s.Spacing)
	// Leave room for consecutive integral cell coordinates and their offsets.
	const gridLimit = 1 << 50
	if !finite(p) || max(math.Abs(p.X), math.Abs(p.Y), math.Abs(p.Z)) > gridLimit {
		return 0
	}
	limit := math.Min(.5, s.Radius/s.Spacing)
	// Neighbouring cells own marks that reach across the cell edges.
	// Sampling all potentially overlapping centres keeps planar fins marked.
	strength := 0.
	for x := math.Floor(p.X - limit); x <= math.Floor(p.X+limit); x++ {
		for y := math.Floor(p.Y - limit); y <= math.Floor(p.Y+limit); y++ {
			for z := math.Floor(p.Z - limit); z <= math.Floor(p.Z+limit); z++ {
				h := scramble(s.Seed ^ uint64(int64(x))*73856093 ^ uint64(int64(y))*19349663 ^ uint64(int64(z))*83492791)
				r := limit * (.45 + .55*fraction(h))
				centre := func() float64 { h = scramble(h + 1); return fraction(h) }
				d := maths.V(p.X-x-centre(), p.Y-y-centre(), p.Z-z-centre()).Len()
				strength = math.Max(strength, 1-maths.Smooth(.56*r, r, d))
			}
		}
	}
	return strength
}

// Coverage is the unfiltered spot field. A region mask can restrict it to
// the anatomical locations where this species develops spots.
func (s Spots) Coverage(point maths.Vec, _ float64) float64 { return s.At(point) }
