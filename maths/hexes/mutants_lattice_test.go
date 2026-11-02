package hexes

import (
	"math"
	"sort"
	"testing"

	"github.com/janearc/slayer/maths"
)

// the hex lattice is worked here by hand, not by the code under test: the
// middle of column c, row r is s*(c + half an odd row) along and
// s*root3/2*r across, and the six neighbours of a middle lie s from it
// in the six directions that follow, sixty degrees apart.
var sixWays = [6][2]float64{
	{1, 0}, {-1, 0},
	{0.5, math.Sqrt(3) / 2}, {0.5, -math.Sqrt(3) / 2},
	{-0.5, math.Sqrt(3) / 2}, {-0.5, -math.Sqrt(3) / 2},
}

// middle is the lattice middle of a column and row, spacing s.
func middle(s float64, c, r int) (u, v float64) {
	return s * (float64(c) + 0.5*float64(r%2)), s * math.Sqrt(3) / 2 * float64(r)
}

// apothem is how far a point (du, dv) from a middle is from the edge of
// its hexagon: half the spacing less the most the point leans toward any
// of the six neighbours.
func apothem(s, du, dv float64) float64 {
	lean := math.Inf(-1)
	for _, w := range sixWays {
		lean = math.Max(lean, du*w[0]+dv*w[1])
	}
	return s/2 - lean
}

// byRowThenColumn puts tiles in the order of their ids: row first.
func byRowThenColumn(tiles []Tile) {
	sort.Slice(tiles, func(a, b int) bool {
		if tiles[a].J != tiles[b].J {
			return tiles[a].J < tiles[b].J
		}
		return tiles[a].I < tiles[b].I
	})
}

// sameTiles is whether two lists hold the same tiles in the same order.
func sameTiles(a, b []Tile) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

// the neighbours of a flat tile are the tiles one step from its middle,
// found by asking every tile, and come in order of id.
//
// worked by hand for step 2: the even row's tile (3, 2) has (2,2) and
// (4,2) beside it, (2,1) and (3,1) below, (2,3) and (3,3) above; the odd
// row's (3, 3) has (3,2) and (4,2) below and (3,4) and (4,4) above. the
// corner (0, 0) has two.
func TestFlatNeighboursAreTheTilesOneStepAway(t *testing.T) {
	f := Flat{Step: 2, Across: 7, Down: 5}
	for _, c := range []struct {
		tile Tile
		want []Tile
	}{
		{Tile{0, 3, 2}, []Tile{{0, 2, 1}, {0, 3, 1}, {0, 2, 2}, {0, 4, 2},
			{0, 2, 3}, {0, 3, 3}}},
		{Tile{0, 3, 3}, []Tile{{0, 3, 2}, {0, 4, 2}, {0, 2, 3}, {0, 4, 3},
			{0, 3, 4}, {0, 4, 4}}},
		{Tile{0, 0, 0}, []Tile{{0, 1, 0}, {0, 0, 1}}},
	} {
		if got := f.Neighbours(c.tile); !sameTiles(got, c.want) {
			t.Errorf("tile %v: neighbours %v, want %v", c.tile, got, c.want)
		}
	}
	for _, g := range []Flat{{Step: 1.5, Across: 6, Down: 5},
		{Step: 0.7, Across: 5, Down: 8}} {
		tiles := g.Tiles()
		for _, tile := range tiles {
			var want []Tile
			cu, cv := middle(g.Step, tile.I, tile.J)
			for _, other := range tiles {
				ou, ov := middle(g.Step, other.I, other.J)
				d := math.Hypot(ou-cu, ov-cv)
				if other != tile && math.Abs(d-g.Step) < 1e-9*g.Step {
					want = append(want, other)
				}
			}
			byRowThenColumn(want)
			if got := g.Neighbours(tile); !sameTiles(got, want) {
				t.Fatalf("%+v tile %v: neighbours %v, want %v", g, tile,
					got, want)
			}
		}
	}
}

// the middle of a flat tile is on the plane, y zero, at the lattice place: for
// step 2, tile (3, 1) is at (7, 0, root3) and (0, 0) is the origin. a step of
// zero or less is read as one, which tiles the plane at spacing one; a step
// greater than zero is its own spacing.
func TestFlatMiddlesAndSpacing(t *testing.T) {
	f := Flat{Step: 2, Across: 7, Down: 5}
	want := maths.V(7, 0, math.Sqrt(3))
	if got := f.Centre(Tile{0, 3, 1}); !got.Near(want, 1e-12) || got.Y != 0 {
		t.Errorf("tile (3, 1) is at %v, want %v", got, want)
	}
	for _, step := range []float64{0, -3} {
		g := Flat{Step: step, Across: 4, Down: 4}
		if g.Spacing() != 1 {
			t.Errorf("step %v: spacing %v, want 1", step, g.Spacing())
		}
		if got := g.Centre(Tile{0, 2, 0}); !got.Near(maths.V(2, 0, 0), 1e-12) {
			t.Errorf("step %v: tile (2, 0) is at %v", step, got)
		}
		if got := g.Nearest(maths.V(2.1, 0, 0.1)); got != (Tile{0, 2, 0}) {
			t.Errorf("step %v: the point is in %v", step, got)
		}
	}
	for _, step := range []float64{0.5, 1, 7} {
		if got := (Flat{Step: step}).Spacing(); got != step {
			t.Errorf("step %v: spacing %v", step, got)
		}
	}
}

// the edge of a flat tile is the hexagon's: s/2 at the middle, less how
// far the point leans toward the nearest of the six neighbours. worked
// for several steps, in even and odd rows, at points all round.
func TestFlatEdgeIsTheHexagons(t *testing.T) {
	for _, s := range []float64{1, 2, 3.7} {
		f := Flat{Step: s, Across: 6, Down: 6}
		for _, tile := range []Tile{{0, 3, 2}, {0, 3, 3}, {0, 0, 0}, {0, 5, 5}} {
			cu, cv := middle(s, tile.I, tile.J)
			if e := f.Edge(f.Centre(tile), tile); math.Abs(e-s/2) > 1e-12 {
				t.Fatalf("step %v tile %v: the middle is %v from the edge",
					s, tile, e)
			}
			for k := range 40 {
				du := s * (random(int64(k)).X) * 0.9
				dv := s * (random(int64(k) + 99).Y) * 0.9
				p := maths.V(cu+du, 5, cv+dv)
				want := apothem(s, du, dv)
				if e := f.Edge(p, tile); math.Abs(e-want) > 1e-12*s {
					t.Fatalf("step %v tile %v offset (%v, %v): %v, want %v",
						s, tile, du, dv, e, want)
				}
			}
		}
	}
}

// the rung below a plane is the lattice at half the step, twice as many
// columns and rows.
func TestFlatFinerIsHalfTheStepAndTwiceTheExtent(t *testing.T) {
	if got := (Flat{Step: 3, Across: 5, Down: 4}).Finer(); got != (Flat{Step: 1.5, Across: 10, Down: 8}) {
		t.Errorf("the rung below is %+v", got)
	}
	if got := (Flat{Across: 2, Down: 1}).Finer(); got != (Flat{Step: 0.5, Across: 4, Down: 2}) {
		t.Errorf("the rung below a step of nought is %+v", got)
	}
}

// across a plane is the straight line, whichever side of the origin the
// points are on: (1, 2) and (4, 6) are 5 apart, and so are (-1, -2) and
// (2, 2).
func TestFlatDistanceIsTheStraightLine(t *testing.T) {
	f := Flat{Step: 1, Across: 3, Down: 3}
	for _, c := range []struct {
		a, b maths.Vec
		want float64
	}{
		{maths.V(1, 8, 2), maths.V(4, -3, 6), 5},
		{maths.V(-1, 0, -2), maths.V(2, 0, 2), 5},
		{maths.V(5, 0, -3), maths.V(5, 0, 1), 4},
	} {
		if d := f.Distance(c.a, c.b); math.Abs(d-c.want) > 1e-12 {
			t.Errorf("%v to %v is %v, want %v", c.a, c.b, d, c.want)
		}
	}
}

// the extent of a plane is the box from the origin to the far corner tile's
// middle: Across-1 columns of step along, a half step more when the last row is
// an odd one, and Down-1 rows of step*root3/2 across. an extent of zero or one
// holds one tile, which is a point.
//
// Somewhere puts a and c at that share of the box, and Latitude is how far a
// point is along north from the box's middle, as a share of the box's reach
// that way, held to one. all worked here from the box.
func TestFlatBoxSomewhereAndLatitude(t *testing.T) {
	norths := []maths.Vec{maths.V(0, 0, 1), maths.V(1, 0, 0), maths.V(1, 0, 1),
		maths.V(3, 9, 4), maths.V(-2, 1, 1), maths.V(0, 5, 0),
		maths.V(1e-13, 0, 0)}
	for _, s := range []float64{2, 0.7} {
		for across := 0; across <= 6; across++ {
			for down := 0; down <= 6; down++ {
				f := Flat{Step: s, Across: across, Down: down}
				a, d := max(1, across)-1, max(1, down)-1
				w := s * (float64(a) + 0.5*float64(d%2))
				h := float64(d) * s * math.Sqrt(3) / 2
				for _, ac := range [][2]float64{{0, 0}, {1, 1}, {0.5, 0.5},
					{0.25, 0.75}, {0.9, 0.1}} {
					want := maths.V(ac[0]*w, 0, ac[1]*h)
					got := f.Somewhere(ac[0], 0.3, ac[1])
					if !got.Near(want, 1e-12) || got.Y != 0 {
						t.Fatalf("%+v somewhere %v is %v, want %v", f, ac, got, want)
					}
				}
				mid := maths.V(w/2, 0, h/2)
				for _, p := range []maths.Vec{maths.V(0, 0, 0), maths.V(w, 3, h),
					maths.V(w/4, 0, 3*h/4), maths.V(w+4, 0, h+9),
					maths.V(-3, 0, 0.3), mid} {
					for _, n := range norths {
						flat := maths.V(n.X, 0, n.Z)
						want := 0.0
						if flat.Len() >= 1e-12 {
							flat = flat.Unit()
							reach := w/2*math.Abs(flat.X) + h/2*math.Abs(flat.Z)
							if reach > 0 {
								want = math.Min(1, math.Abs(p.Sub(mid).Dot(flat))/reach)
							}
						}
						if got := f.Latitude(p, n); math.Abs(got-want) > 1e-12 {
							t.Fatalf("%+v at %v under north %v: %v, want %v",
								f, p, n, got, want)
						}
					}
				}
			}
		}
	}
}

// a north that is only 1e-12 long in the plane is still a direction: the
// plane's far edge along it is at latitude one, and one shorter is none.
func TestFlatLatitudeStopsAtATinyNorth(t *testing.T) {
	f := Flat{Step: 1, Across: 5, Down: 5}
	edge := f.Somewhere(1, 0, 0.5)
	exact := maths.V(1e-12, 3, 0)
	if in := maths.V(exact.X, 0, exact.Z); in.Len() != 1e-12 {
		t.Fatalf("1e-12 is %v long in the plane here", in.Len())
	}
	if l := f.Latitude(edge, exact); math.Abs(l-1) > 1e-12 {
		t.Errorf("a north 1e-12 long gives latitude %v", l)
	}
	if l := f.Latitude(edge, maths.V(1e-13, 3, 0)); l != 0 {
		t.Errorf("a north 1e-13 long gives latitude %v", l)
	}
}

// a band of Radius 6/pi with 6 round has spacing 2 and rows root3 apart: a
// width of 7 holds rows at 0, 1.73, 3.46, 5.2 and 6.93, five; a width of zero
// or a negative one holds the one row at the bottom, and fewer than six round
// is read as six.
func TestBandRowsAndColumnsFit(t *testing.T) {
	r := 6 / math.Pi
	for _, c := range []struct {
		width      float64
		around     int
		spacing    float64
		rows, want int
	}{
		{7, 6, 2, 5, 30},
		{7, 3, 2, 5, 30},
		{7, 0, 2, 5, 30},
		{3.5, 6, 2, 3, 18},
		{1.7, 6, 2, 1, 6},
		{0, 6, 2, 1, 6},
		{-5, 6, 2, 1, 6},
		{7, 7, 12.0 / 7, 5, 35},
	} {
		b := Band{Radius: r, Width: c.width, Around: c.around}
		if math.Abs(b.Spacing()-c.spacing) > 1e-12 {
			t.Errorf("%+v: spacing %v, want %v", b, b.Spacing(), c.spacing)
		}
		if b.rows() != c.rows || len(b.Tiles()) != c.want {
			t.Errorf("%+v: %d rows, %d tiles, want %d and %d", b, b.rows(),
				len(b.Tiles()), c.rows, c.want)
		}
	}
}

// a point is in the column its angle says, whichever way round it is: with 40
// round, column c is c/40 of a turn from plus z toward plus x, so plus x is
// column 10, minus z is column 20 and minus x is column 30.
//
// the rows are 1.36 apart from the bottom, minus two, so a height of 0.72 is in
// row two and one of minus two is in row zero.
func TestBandNearestByAngle(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	for _, c := range []struct {
		p    maths.Vec
		want Tile
	}{
		{maths.V(10, -2, 0), Tile{0, 10, 0}},
		{maths.V(0, 0.7207, -10), Tile{0, 20, 2}},
		{maths.V(-10, 0.7207, 0), Tile{0, 30, 2}},
		{maths.V(-1e-6, -2, 10), Tile{0, 0, 0}},
		{maths.V(0, -2, 10), Tile{0, 0, 0}},
		{maths.V(-10, -2, 0), Tile{0, 30, 0}},
		{maths.V(-5, -2, 5), Tile{0, 35, 0}},
	} {
		if got := b.Nearest(c.p); got != c.want {
			t.Errorf("%v is in %v, want %v", c.p, got, c.want)
		}
	}
}

// the neighbours of a band tile are the tiles one step away round the
// cylinder, found by asking every tile: the arc between the middles, the
// shorter way round, with the height between them, is the spacing. they
// come in order of id, wrap at the seam, and stop at the rims.
//
// worked by hand for radius 10 with 40 round: the odd row's last column
// (39, 1) has (38, 1), (0, 1), (39, 0), (0, 0), (39, 2) and (0, 2); (5, 0)
// at the bottom rim and (5, 2) at the top have four.
func TestBandNeighboursAreTheTilesOneStepAway(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	for _, c := range []struct {
		tile Tile
		want []Tile
	}{
		{Tile{0, 39, 1}, []Tile{{0, 0, 0}, {0, 39, 0}, {0, 0, 1}, {0, 38, 1},
			{0, 0, 2}, {0, 39, 2}}},
		{Tile{0, 5, 0}, []Tile{{0, 4, 0}, {0, 6, 0}, {0, 4, 1}, {0, 5, 1}}},
		{Tile{0, 5, 2}, []Tile{{0, 4, 1}, {0, 5, 1}, {0, 4, 2}, {0, 6, 2}}},
	} {
		if got := b.Neighbours(c.tile); !sameTiles(got, c.want) {
			t.Errorf("tile %v: neighbours %v, want %v", c.tile, got, c.want)
		}
	}
	for _, g := range []Band{{Radius: 10, Width: 4, Around: 40},
		{Radius: 2, Width: 6, Around: 6}, {Radius: 3, Width: 0, Around: 12},
		{Radius: 1, Width: 5, Around: 7}, {Radius: 1, Width: 3, Around: 3}} {
		s := 2 * math.Pi * g.Radius / float64(g.around())
		tiles := g.Tiles()
		for _, tile := range tiles {
			var want []Tile
			au, av := middle(s, tile.I, tile.J)
			for _, other := range tiles {
				bu, bv := middle(s, other.I, other.J)
				turn := math.Abs(math.Remainder((bu-au)/g.Radius, 2*math.Pi))
				d := math.Hypot(turn*g.Radius, bv-av)
				if other != tile && math.Abs(d-s) < 1e-9*s {
					want = append(want, other)
				}
			}
			byRowThenColumn(want)
			if got := g.Neighbours(tile); !sameTiles(got, want) {
				t.Fatalf("%+v tile %v: neighbours %v, want %v", g, tile,
					got, want)
			}
		}
	}
}

// the edge of a band tile is the hexagon's, measured round the cylinder
// the short way: a point just over the seam from a tile in the last
// column is a little past its edge, not a whole turn away, and a point
// on the far side of the band is as far as half a turn.
func TestBandEdgeIsTheHexagonsTheShortWayRound(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	s := b.Spacing()
	for _, tile := range []Tile{{0, 0, 1}, {0, 39, 1}, {0, 1, 0}, {0, 38, 2},
		{0, 20, 1}, {0, 19, 1}, {0, 21, 0}, {0, 10, 2}, {0, 30, 1}} {
		cu, cv := middle(s, tile.I, tile.J)
		for k := range 96 {
			ang := (float64(k) + 0.37) * 2 * math.Pi / 96
			if k%4 == 0 {
				ang = float64(k) * 2 * math.Pi / 96 // on the seam and the quarters
			}
			y := float64(k%7)*0.4 - 1.2
			p := maths.V(10*math.Sin(ang), y, 10*math.Cos(ang))
			turn := math.Remainder(ang-cu/10, 2*math.Pi)
			want := apothem(s, 10*turn, y-(cv-2))
			if e := b.Edge(p, tile); math.Abs(e-want) > 1e-9 {
				t.Fatalf("tile %v at angle %v: %v from the edge, want %v",
					tile, ang, e, want)
			}
		}
	}
}

// ids of a band that is more than 2^32 columns round are row by row: 6 thousand
// million columns, row 2, column 7 is 2*6e9+7, and column 2^32 is not column
// zero. at exactly 2^32 columns the number is the same as the pair of
// thirty-two bits, since every column is below it.
func TestBandIDsPastThirtyTwoBits(t *testing.T) {
	b := Band{Radius: 1.5e9, Width: 1e6, Around: 6_000_000_000}
	if got := b.ID(Tile{0, 7, 2}); got != 12_000_000_007 {
		t.Errorf("id of (7, 2) is %d", got)
	}
	if b.ID(Tile{0, 1 << 32, 3}) == b.ID(Tile{0, 0, 3}) {
		t.Error("column 2^32 shares an id with column nought")
	}
	exact := Band{Radius: 1.5e9, Width: 1e6, Around: 1 << 32}
	if got := exact.ID(Tile{0, 7, 2}); got != 2<<32|7 {
		t.Errorf("id of (7, 2) at exactly 2^32 columns is %d", got)
	}
}

// the rung below a band has twice as many columns, round the same
// radius and width; fewer than six are read as six first.
func TestBandFinerHasTwiceTheColumns(t *testing.T) {
	if got := (Band{Radius: 2, Width: 1.5, Around: 12}).Finer(); got != (Band{Radius: 2, Width: 1.5, Around: 24}) {
		t.Errorf("the rung below is %+v", got)
	}
	if got := (Band{Radius: 2, Width: 1.5, Around: 3}).Finer(); got != (Band{Radius: 2, Width: 1.5, Around: 12}) {
		t.Errorf("the rung below three round is %+v", got)
	}
}

// round a band is the shorter way times the radius, with the height
// between: radius 10, angles of minus and plus 0.9 pi are 0.2 pi apart
// across the seam, and two heights apart with no turn between them are
// the difference, not the sum.
func TestBandDistanceIsAlongTheCylinder(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	at := func(az, y float64) maths.Vec {
		return maths.V(10*math.Sin(az), y, 10*math.Cos(az))
	}
	if d := b.Distance(at(-0.9*math.Pi, 0), at(0.9*math.Pi, 0)); math.Abs(d-0.2*math.Pi*10) > 1e-9 {
		t.Errorf("across the seam: %v", d)
	}
	if d := b.Distance(at(1, 3), at(1, 1)); math.Abs(d-2) > 1e-12 {
		t.Errorf("two apart in height: %v", d)
	}
	if d := b.Distance(at(0.5, 1), at(0.9, -1)); math.Abs(d-math.Hypot(0.4*10, 2)) > 1e-12 {
		t.Errorf("slantwise: %v", d)
	}
}

// the poles of a band are its rims: a point a quarter of the width up
// is at latitude a half, one at the rim is one, and past it is one; a
// band of no width has no latitude. Somewhere puts b along the width.
func TestBandLatitudeAndSomewhere(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	for _, c := range []struct{ y, want float64 }{{1, 0.5}, {-1.5, 0.75},
		{2, 1}, {3, 1}, {0, 0}} {
		if l := b.Latitude(maths.V(10, c.y, 0), maths.Up); math.Abs(l-c.want) > 1e-12 {
			t.Errorf("height %v: latitude %v, want %v", c.y, l, c.want)
		}
	}
	for _, w := range []float64{0, -2} {
		if l := (Band{Radius: 10, Width: w, Around: 40}).Latitude(maths.V(10, 3, 0), maths.Up); l != 0 {
			t.Errorf("width %v: latitude %v", w, l)
		}
	}
	if l := (Band{Radius: 10, Width: 0.5, Around: 40}).Latitude(maths.V(10, 0.125, 0), maths.Up); math.Abs(l-0.5) > 1e-12 {
		t.Errorf("a narrow band: latitude %v", l)
	}
	if p := b.Somewhere(0.25, 0.75, 0); !p.Near(maths.V(10, 1, 0), 1e-9) {
		t.Errorf("a quarter round, three quarters up: %v", p)
	}
	if p := b.Somewhere(0.5, 0, 0); !p.Near(maths.V(0, -2, -10), 1e-9) {
		t.Errorf("half round, at the bottom: %v", p)
	}
}

// unwrap gives u from zero up to the circumference, going round from plus z
// toward plus x, and v from the band's bottom.
//
// a point a quarter turn toward plus x is a quarter of the circumference along,
// one toward minus x is three quarters, and one on the seam, at plus z, is
// zero, not a whole turn.
func TestBandUnwrapGoesRoundFromNought(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	for _, c := range []struct{ turns float64 }{{0}, {0.1}, {0.25}, {0.5},
		{0.75}, {1.0 - 1e-9}, {1.0 / 3}, {0.04}} {
		a := c.turns * 2 * math.Pi
		u, v := b.unwrap(maths.V(10*math.Sin(a), 1.5, 10*math.Cos(a)))
		if want := 10 * a; math.Abs(u-want) > 1e-9 || v != 3.5 {
			t.Errorf("%v of a turn round: u %v v %v, want %v and 3.5", c.turns, u, v, want)
		}
	}
	u, _ := b.unwrap(maths.V(0, 0, 10))
	if u != 0 {
		t.Errorf("on the seam u is %v", u)
	}
}
