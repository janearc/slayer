package manufacture

import (
	"math"
	"sync"

	"github.com/janearc/slayer/maths"
)

// orcaStation is one cross-section of the trunk in rest space: where it
// lies along the body, the half height and half width of its ellipse,
// and the height of the ellipse's centre.
type orcaStation struct{ x, high, wide, centre float64 }

// orcaStations lists the trunk's cross-sections from the tail stock to
// the snout. The numbers were chosen for a cow of about six metres: the
// greatest girth lies a third of the way back, the head is blunt, its
// forehead (the melon) steep over a low short rostrum, and the tail
// stock is deeper than it is wide.
var orcaStations = []orcaStation{
	{x: -2.60, high: .060, wide: .030, centre: 0},
	{x: -2.45, high: .190, wide: .120, centre: -.010},
	{x: -2.20, high: .260, wide: .175, centre: -.010},
	{x: -1.80, high: .350, wide: .280, centre: -.005},
	{x: -1.40, high: .430, wide: .370, centre: 0},
	{x: -0.70, high: .530, wide: .480, centre: .020},
	{x: 0.00, high: .600, wide: .550, centre: .030},
	{x: 0.70, high: .610, wide: .560, centre: .030},
	{x: 1.40, high: .570, wide: .520, centre: .020},
	{x: 2.00, high: .500, wide: .470, centre: 0},
	{x: 2.40, high: .400, wide: .380, centre: 0},
	{x: 2.70, high: .290, wide: .270, centre: -.010},
	{x: 2.90, high: .150, wide: .160, centre: -.055},
	{x: 3.02, high: .060, wide: .075, centre: -.060},
	{x: 3.06, high: .010, wide: .012, centre: -.050},
}

// orcaTrunkSamples is how many entries the interpolated profile holds.
const orcaTrunkSamples = 4096

// orcaTrunkTable is the stations interpolated onto an even spacing in x,
// so that a lookup is one step and one blend.
type orcaTrunkTable struct {
	first, last, step float64
	entries           []orcaStation
}

var (
	orcaTrunkOnce   sync.Once
	orcaTrunkShared *orcaTrunkTable
)

// orcaTrunkProfile returns the interpolated profile, built on first use.
func orcaTrunkProfile() *orcaTrunkTable {
	orcaTrunkOnce.Do(func() {
		first := orcaStations[0].x
		last := orcaStations[len(orcaStations)-1].x
		table := &orcaTrunkTable{first: first, last: last,
			step: (last - first) / (orcaTrunkSamples - 1)}
		for i := 0; i < orcaTrunkSamples; i++ {
			table.entries = append(table.entries,
				orcaStationAt(first+float64(i)*table.step))
		}
		orcaTrunkShared = table
	})
	return orcaTrunkShared
}

// orcaHermite is the cubic through (x0, y0) and (x1, y1) with slopes m0
// and m1 there, evaluated at x between them.
func orcaHermite(x0, x1, y0, y1, m0, m1, x float64) float64 {
	span := x1 - x0
	t := (x - x0) / span
	square, cube := t*t, t*t*t
	return (2*cube-3*square+1)*y0 + (cube-2*square+t)*span*m0 +
		(3*square-2*cube)*y1 + (cube-square)*span*m1
}

// orcaSlope is the finite-difference slope of one station quantity at
// station index i, taken across its neighbours.
func orcaSlope(i int, pick func(orcaStation) float64) float64 {
	low, high := max(i-1, 0), min(i+1, len(orcaStations)-1)
	return (pick(orcaStations[high]) - pick(orcaStations[low])) /
		(orcaStations[high].x - orcaStations[low].x)
}

// orcaBlend is one station quantity interpolated at x.
func orcaBlend(x float64, i int, pick func(orcaStation) float64) float64 {
	a, b := orcaStations[i], orcaStations[i+1]
	return orcaHermite(a.x, b.x, pick(a), pick(b),
		orcaSlope(i, pick), orcaSlope(i+1, pick), x)
}

// orcaStationAt is the smooth cross-section at x, from the cubic through
// the listed stations.
func orcaStationAt(x float64) orcaStation {
	i := 0
	for i < len(orcaStations)-2 && x > orcaStations[i+1].x {
		i++
	}
	return orcaStation{
		x:      x,
		high:   orcaBlend(x, i, func(s orcaStation) float64 { return s.high }),
		wide:   orcaBlend(x, i, func(s orcaStation) float64 { return s.wide }),
		centre: orcaBlend(x, i, func(s orcaStation) float64 { return s.centre }),
	}
}

// at returns the cross-section at x, clamped to the ends of the trunk.
func (table *orcaTrunkTable) at(x float64) orcaStation {
	f := (x - table.first) / table.step
	i := int(math.Floor(f))
	i = max(0, min(i, len(table.entries)-2))
	t := math.Min(math.Max(f-float64(i), 0), 1)
	a, b := table.entries[i], table.entries[i+1]
	return orcaStation{
		x:      x,
		high:   a.high + t*(b.high-a.high),
		wide:   a.wide + t*(b.wide-a.wide),
		centre: a.centre + t*(b.centre-a.centre),
	}
}

// orcaTrunk is the signed distance, near the surface, from p to the
// trunk: negative inside. Each cross-section is an ellipse and the
// distance to it is the first-order estimate, the ellipse's own
// function over the length of its gradient.
func orcaTrunk(p maths.Vec) float64 {
	table := orcaTrunkProfile()
	s := table.at(p.X)
	down := p.Y - s.centre
	ring := math.Hypot(down/s.high, p.Z/s.wide)
	gap := -math.Min(s.high, s.wide)
	if ring > 0 {
		slope := math.Hypot(down/(s.high*s.high), p.Z/(s.wide*s.wide)) / ring
		gap = (ring - 1) / slope
	}
	return math.Max(gap, math.Max(p.X-table.last, table.first-p.X))
}
