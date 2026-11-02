package hexes

import (
	"math"
	"sort"

	"github.com/janearc/slayer/maths"
)

// A hex lattice on a plane, pointy side up, rows along one axis and
// columns along the other, with odd rows pushed half a tile along. Flat
// lays it on the x z plane and Band wraps it round a cylinder.
//
// Both do their arithmetic here, in the lattice's own two coordinates, u
// along a row and v across the rows.

// lattice is a hex lattice with a spacing between neighbouring middles.
type lattice struct{ s float64 }

// root3 is the square root of three, which is the shape of a hexagon.
var root3 = math.Sqrt(3)

// centre is the middle of column c, row r.
func (l lattice) centre(c, r int) (u, v float64) {
	u = l.s * (float64(c) + 0.5*float64(r&1))
	v = l.s * root3 / 2 * float64(r)
	return u, v
}

// nearest is the column and row a point is in: the lattice's axial
// coordinates, rounded as cube coordinates, then made offset.
func (l lattice) nearest(u, v float64) (c, r int) {
	size := l.s / root3
	fr := v * 2 / (3 * size)
	fq := u/(size*root3) - fr/2
	q, r := roundHex(fq, fr)
	return q + (r-(r&1))/2, r
}

// roundHex rounds axial coordinates to the nearest hex, through the three
// cube coordinates, fixing whichever one rounding moved the most.
func roundHex(fq, fr float64) (q, r int) {
	fs := -fq - fr
	rq, rr, rs := math.Round(fq), math.Round(fr), math.Round(fs)
	dq, dr, ds := math.Abs(rq-fq), math.Abs(rr-fr), math.Abs(rs-fs)
	switch {
	case dq > dr && dq > ds:
		rq = -rr - rs
	case dr > ds:
		rr = -rq - rs
	}
	return int(rq), int(rr)
}

// steps6 is the six neighbours of a tile in offset coordinates, for even
// rows and for odd.
var steps6 = [2][6][2]int{
	{{-1, 0}, {1, 0}, {-1, -1}, {0, -1}, {-1, 1}, {0, 1}},
	{{-1, 0}, {1, 0}, {0, -1}, {1, -1}, {0, 1}, {1, 1}},
}

// around is the six neighbours of column c, row r, before any wrap or
// edge is applied.
func around(c, r int) [6][2]int {
	var out [6][2]int
	for i, s := range steps6[r&1] {
		out[i] = [2]int{c + s[0], r + s[1]}
	}
	return out
}

// fromEdge is how far a point u, v is from the edge of a tile, given the
// tile's middle cu, cv and its neighbours' middles: the nearest of the
// half-planes between the middle and each neighbour.
func fromEdge(u, v, cu, cv float64, near [][2]float64) float64 {
	best := math.Inf(1)
	for _, n := range near {
		du, dv := cu-n[0], cv-n[1]
		l := math.Hypot(du, dv)
		mu, mv := (cu+n[0])/2, (cv+n[1])/2
		best = math.Min(best, ((u-mu)*du+(v-mv)*dv)/l)
	}
	return best
}

// Flat is a hex lattice on the x z plane, y up, with Step between
// neighbouring middles: a landscape. Across and Down are how many columns
// and rows it holds, from the origin toward plus x and plus z; a point
// beyond them is still in a tile, but Tiles does not list it.
type Flat struct {
	Step         float64
	Across, Down int
}

// spacing is the step, with zero read as one.
func (f Flat) spacing() float64 {
	if f.Step <= 0 {
		return 1
	}
	return f.Step
}

// lat is the lattice.
func (f Flat) lat() lattice { return lattice{f.spacing()} }

// Nearest is the tile under a point, ignoring its height.
func (f Flat) Nearest(p maths.Vec) Tile {
	c, r := f.lat().nearest(p.X, p.Z)
	return Tile{0, c, r}
}

// Centre is the middle of a tile, on the plane.
func (f Flat) Centre(tile Tile) maths.Vec {
	u, v := f.lat().centre(tile.I, tile.J)
	return maths.V(u, 0, v)
}

// Neighbours is the six tiles round this one, fewer at the edge of the
// extent, in order of ID.
func (f Flat) Neighbours(tile Tile) []Tile {
	var out []Tile
	for _, n := range around(tile.I, tile.J) {
		if f.holds(n[0], n[1]) {
			out = append(out, Tile{0, n[0], n[1]})
		}
	}
	sort.Slice(out, func(a, b int) bool {
		return f.ID(out[a]) < f.ID(out[b])
	})
	return out
}

// holds is whether a column and row are inside the extent.
func (f Flat) holds(c, r int) bool {
	return c >= 0 && r >= 0 && c < f.Across && r < f.Down
}

// Edge is how far a point is from the edge of a tile, on the plane.
func (f Flat) Edge(p maths.Vec, tile Tile) float64 {
	l := f.lat()
	cu, cv := l.centre(tile.I, tile.J)
	var near [][2]float64
	for _, n := range around(tile.I, tile.J) {
		u, v := l.centre(n[0], n[1])
		near = append(near, [2]float64{u, v})
	}
	return fromEdge(p.X, p.Z, cu, cv, near)
}

// ID is the row then the column, each in its own thirty-two bits, so two
// tiles 2^32 columns apart share one; a flat extent is never so wide.
func (f Flat) ID(tile Tile) int64 { return pair(tile.I, tile.J) }

// pair is a column and a row as one number.
func pair(c, r int) int64 { return int64(r)<<32 | int64(uint32(c)) }

// Spacing is the distance between neighbouring middles.
func (f Flat) Spacing() float64 { return f.spacing() }

// Tiles is every tile inside the extent, in order of ID.
func (f Flat) Tiles() []Tile {
	out := make([]Tile, 0, max(0, f.Across)*max(0, f.Down))
	for r := 0; r < f.Down; r++ {
		for c := 0; c < f.Across; c++ {
			out = append(out, Tile{0, c, r})
		}
	}
	return out
}

// Finer is the lattice at half the spacing, twice as many columns and
// rows, with every middle of this one.
func (f Flat) Finer() Tiling {
	return Flat{Step: f.spacing() / 2, Across: 2 * f.Across,
		Down: 2 * f.Down}
}

// Band is a hex lattice wrapped once round a cylinder of Radius about the y
// axis, Around tiles round, so a row is a ring: the inside of a ring world.
// Rows run along y from minus Width/2, as many as fit.
//
// Angle is measured from plus z toward plus x, so a point on the cylinder at z
// = Radius is in column zero.
type Band struct {
	Radius float64
	Width  float64
	Around int
}

// around is how many tiles go round, with fewer than six read as six.
func (b Band) around() int { return max(6, b.Around) }

// spacing is the distance between neighbouring middles: the
// circumference over Around.
func (b Band) spacing() float64 {
	return 2 * math.Pi * b.Radius / float64(b.around())
}

// lat is the lattice.
func (b Band) lat() lattice { return lattice{b.spacing()} }

// rows is how many rows fit across the width.
func (b Band) rows() int {
	return max(1, int(math.Floor(b.Width/(b.spacing()*root3/2)))+1)
}

// unwrap is a point as the lattice sees it: u along the circumference, from
// zero up to it, and v along y from the band's bottom.
func (b Band) unwrap(p maths.Vec) (u, v float64) {
	a := math.Atan2(p.X, p.Z)
	if a < 0 {
		a += 2 * math.Pi
	}
	return b.Radius * a, p.Y + b.Width/2
}

// wrap keeps a column inside the band.
func (b Band) wrap(c int) int {
	n := b.around()
	return ((c % n) + n) % n
}

// Nearest is the tile a point is in, by its angle round the axis and its
// height, whatever its distance from the axis.
func (b Band) Nearest(p maths.Vec) Tile {
	u, v := b.unwrap(p)
	c, r := b.lat().nearest(u, v)
	return Tile{0, b.wrap(c), r}
}

// Centre is the middle of a tile, on the cylinder.
func (b Band) Centre(tile Tile) maths.Vec {
	u, v := b.lat().centre(tile.I, tile.J)
	a := u / b.Radius
	return maths.V(b.Radius*math.Sin(a), v-b.Width/2, b.Radius*math.Cos(a))
}

// Neighbours is the six tiles round this one, wrapping round the band,
// fewer at the band's rim, in order of ID.
func (b Band) Neighbours(tile Tile) []Tile {
	var out []Tile
	for _, n := range around(tile.I, tile.J) {
		if n[1] < 0 || n[1] >= b.rows() {
			continue
		}
		t := Tile{0, b.wrap(n[0]), n[1]}
		dup := false
		for _, have := range out {
			dup = dup || have == t
		}
		if !dup {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(x, y int) bool {
		return b.ID(out[x]) < b.ID(out[y])
	})
	return out
}

// Edge is how far a point is from the edge of a tile, measured on the
// unwrapped cylinder, with the seam crossed where a neighbour is over it.
func (b Band) Edge(p maths.Vec, tile Tile) float64 {
	l := b.lat()
	u, v := b.unwrap(p)
	cu, cv := l.centre(tile.I, tile.J)
	circ := 2 * math.Pi * b.Radius
	// bring the point to the same side of the seam as the middle
	if u-cu > circ/2 {
		u -= circ
	} else if cu-u > circ/2 {
		u += circ
	}
	var near [][2]float64
	for _, n := range around(tile.I, tile.J) {
		nu, nv := l.centre(n[0], n[1])
		near = append(near, [2]float64{nu, nv})
	}
	return fromEdge(u, v, cu, cv, near)
}

// ID is the row then the column, each in its own thirty-two bits; a band
// of more columns than that, as an orbital is, numbers its tiles row by
// row instead, so no two share one.
func (b Band) ID(tile Tile) int64 {
	if n := b.around(); n > 1<<32 {
		return int64(tile.J)*int64(n) + int64(tile.I)
	}
	return pair(tile.I, tile.J)
}

// Spacing is the distance between neighbouring middles.
func (b Band) Spacing() float64 { return b.spacing() }

// Tiles is every tile of the band, in order of ID.
func (b Band) Tiles() []Tile {
	rows, n := b.rows(), b.around()
	out := make([]Tile, 0, rows*n)
	for r := 0; r < rows; r++ {
		for c := 0; c < n; c++ {
			out = append(out, Tile{0, c, r})
		}
	}
	return out
}

// Finer is the band with twice as many tiles round, and so half the
// spacing and twice the rows, with every middle of this one.
func (b Band) Finer() Tiling {
	return Band{Radius: b.Radius, Width: b.Width, Around: 2 * b.around()}
}

// Distance is the straight line on the plane, ignoring height.
func (f Flat) Distance(a, b maths.Vec) float64 {
	return math.Hypot(a.X-b.X, a.Z-b.Z)
}

// bounds is the extent's middle and its half-sizes across and down, from
// the four corner tiles' middles.
func (f Flat) bounds() (mid maths.Vec, hx, hz float64) {
	a, d := max(1, f.Across), max(1, f.Down)
	lo, hi := math.Inf(1), math.Inf(-1)
	lz, hz2 := math.Inf(1), math.Inf(-1)
	for _, t := range []Tile{{0, 0, 0}, {0, a - 1, 0}, {0, 0, d - 1},
		{0, a - 1, d - 1}} {
		c := f.Centre(t)
		lo, hi = math.Min(lo, c.X), math.Max(hi, c.X)
		lz, hz2 = math.Min(lz, c.Z), math.Max(hz2, c.Z)
	}
	return maths.V((lo+hi)/2, 0, (lz+hz2)/2), (hi - lo) / 2, (hz2 - lz) / 2
}

// Latitude is how far a point lies along north from the middle of the extent,
// as a share of the extent's reach that way: north is taken in the plane, and a
// north that points straight up gives zero.
func (f Flat) Latitude(p, north maths.Vec) float64 {
	n := maths.V(north.X, 0, north.Z)
	if n.Len() < 1e-12 {
		return 0
	}
	n = n.Unit()
	mid, hx, hz := f.bounds()
	reach := hx*math.Abs(n.X) + hz*math.Abs(n.Z)
	if reach <= 0 {
		return 0
	}
	return math.Min(1, math.Abs(p.Sub(mid).Dot(n))/reach)
}

// Somewhere is a point of the extent: a across it and c down it.
func (f Flat) Somewhere(a, _, c float64) maths.Vec {
	mid, hx, hz := f.bounds()
	return mid.Add(maths.V((a*2-1)*hx, 0, (c*2-1)*hz))
}

// azimuth is a point's angle round the band, as Centre lays it out.
func azimuth(p maths.Vec) float64 { return math.Atan2(p.X, p.Z) }

// Distance is along the cylinder: the shorter way round, times the
// radius, and the difference along the axis, together.
func (b Band) Distance(p, q maths.Vec) float64 {
	round := math.Abs(math.Remainder(azimuth(p)-azimuth(q), 2*math.Pi))
	return math.Hypot(round*b.Radius, p.Y-q.Y)
}

// Latitude is how far a point lies from the band's mid-line toward its rims,
// which are a band's poles: zero in the middle, one at a rim. North is the
// band's own axis and the given one is not used.
func (b Band) Latitude(p, _ maths.Vec) float64 {
	if b.Width <= 0 {
		return 0
	}
	return math.Min(1, math.Abs(p.Y)/(b.Width/2))
}

// Somewhere is a point of the band: a round it and b along it.
func (b Band) Somewhere(a, b2, _ float64) maths.Vec {
	ang := a * 2 * math.Pi
	return maths.V(b.Radius*math.Sin(ang), (b2*2-1)*b.Width/2,
		b.Radius*math.Cos(ang))
}
