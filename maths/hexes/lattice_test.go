package hexes

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Every tiling is a Tiling.
var (
	_ Tiling = Sphere{}
	_ Tiling = Flat{}
	_ Tiling = Band{}
)

// A flat lattice holds Across by Down tiles, each with six neighbours
// inside and fewer at the edge, every id its own, and the middles are
// Spacing apart.
func TestFlatHoldsItsExtent(t *testing.T) {
	f := Flat{Step: 2, Across: 7, Down: 5}
	tiles := f.Tiles()
	if len(tiles) != 35 {
		t.Fatalf("%d tiles", len(tiles))
	}
	ids := map[int64]bool{}
	for _, tile := range tiles {
		if ids[f.ID(tile)] {
			t.Fatalf("two tiles share id %d", f.ID(tile))
		}
		ids[f.ID(tile)] = true
		ns := f.Neighbours(tile)
		inside := tile.I > 0 && tile.J > 0 && tile.I < 6 && tile.J < 4
		if inside && len(ns) != 6 || !inside && len(ns) >= 6 {
			t.Fatalf("tile %v has %d neighbours", tile, len(ns))
		}
		for _, n := range ns {
			d := f.Centre(tile).Sub(f.Centre(n)).Len()
			if math.Abs(d-2) > 1e-12 {
				t.Fatalf("tile %v and %v are %v apart", tile, n, d)
			}
		}
	}
}

// Nearest on a plane finds the same tile as asking every tile, for points
// all over the extent, and the middle of a tile is in that tile.
func TestFlatNearestIsTheTrueNearest(t *testing.T) {
	f := Flat{Step: 1.5, Across: 12, Down: 9}
	tiles := f.Tiles()
	for k := range int64(5000) {
		p := maths.V(noise.Hash(1, k)*16-1, 0, noise.Hash(2, k)*12-1)
		got := f.Nearest(p)
		best, want := math.Inf(1), Tile{}
		for _, tile := range tiles {
			if d := p.Sub(f.Centre(tile)).Len(); d < best {
				best, want = d, tile
			}
		}
		// outside the extent the true nearest may be a tile Tiles
		// does not list, so only points inside are held to it
		if f.holds(got.I, got.J) && got != want &&
			math.Abs(p.Sub(f.Centre(got)).Len()-best) > 1e-12 {
			t.Fatalf("point %d at %v: tile %v, want %v", k, p, got, want)
		}
	}
	for _, tile := range tiles {
		if f.Nearest(f.Centre(tile)) != tile {
			t.Fatalf("tile %v's middle is not in it", tile)
		}
	}
}

// On a plane a point is zero from the edge at the midpoint between two
// neighbours, inside its tile by Spacing/2 at the middle, and outside a
// neighbour's tile.
func TestFlatEdge(t *testing.T) {
	f := Flat{Step: 1, Across: 6, Down: 6}
	tile := Tile{0, 3, 3}
	for _, n := range f.Neighbours(tile) {
		mid := f.Centre(tile).Add(f.Centre(n)).Scale(0.5)
		if e := f.Edge(mid, tile); math.Abs(e) > 1e-12 {
			t.Fatalf("midpoint with %v is %v from the edge", n, e)
		}
		if e := f.Edge(f.Centre(tile), n); e >= 0 {
			t.Fatalf("the middle of %v is inside %v", tile, n)
		}
	}
	if e := f.Edge(f.Centre(tile), tile); math.Abs(e-0.5) > 1e-12 {
		t.Fatalf("the middle is %v from the edge", e)
	}
}

// A band wraps: the last column's neighbours include the first, a point just
// past the seam is in column zero, and the middles are Spacing apart across the
// seam too.
func TestBandWraps(t *testing.T) {
	b := Band{Radius: 10, Width: 4, Around: 40}
	last := Tile{0, 39, 2}
	wrapped := false
	for _, n := range b.Neighbours(last) {
		wrapped = wrapped || n.I == 0
		d := b.Centre(last).Sub(b.Centre(n)).Len()
		if d > 1.01*b.Spacing() {
			t.Fatalf("%v and %v are %v apart, spacing %v", last, n, d,
				b.Spacing())
		}
	}
	if !wrapped {
		t.Fatal("the last column does not reach the first")
	}
	just := maths.V(1e-6, 0, 10)
	if got := b.Nearest(just); got.I != 0 {
		t.Fatalf("just past the seam is column %d", got.I)
	}
	seam := b.Centre(last).Add(b.Centre(Tile{0, 0, 2})).Scale(0.5)
	if e := b.Edge(seam, last); math.Abs(e) > 1e-9 {
		t.Fatalf("the seam midpoint is %v from the edge", e)
	}
}

// A band's tiles are Around by the rows that fit, the middles lie on the
// cylinder, and a point at any distance from the axis is in the tile its
// angle and height say.
func TestBandTilesLieOnTheCylinder(t *testing.T) {
	b := Band{Radius: 3, Width: 2, Around: 24}
	tiles := b.Tiles()
	if len(tiles) != 24*b.rows() || b.rows() < 2 {
		t.Fatalf("%d tiles in %d rows", len(tiles), b.rows())
	}
	for _, tile := range tiles {
		c := b.Centre(tile)
		if r := math.Hypot(c.X, c.Z); math.Abs(r-3) > 1e-12 {
			t.Fatalf("tile %v's middle is %v from the axis", tile, r)
		}
		far := maths.V(c.X*5, c.Y, c.Z*5) // further from the axis
		if b.Nearest(c) != tile || b.Nearest(far) != tile {
			t.Fatalf("tile %v's middle is not in it", tile)
		}
	}
}

// The rungs nest on a plane and on a band: every middle is a middle one
// rung finer.
func TestLatticeRungsNest(t *testing.T) {
	for _, tl := range []Tiling{Flat{Step: 1, Across: 5, Down: 4},
		Band{Radius: 2, Width: 1.5, Around: 12}} {
		fine := tl.Finer()
		for _, tile := range tl.Tiles() {
			c := tl.Centre(tile)
			if got := fine.Centre(fine.Nearest(c)); !got.Near(c, 1e-9) {
				t.Fatalf("%T tile %v's middle moved to %v one rung down",
					tl, tile, got)
			}
		}
	}
}

// A manhole on an orbital: a ring three million kilometres across tiled
// at six tenths of a metre is a lattice of sixteen billion columns, and
// Nearest still answers with a tile whose middle is within a spacing.
func TestAManholeOnAnOrbital(t *testing.T) {
	circumference := 2 * math.Pi * 1.5e9
	b := Band{Radius: 1.5e9, Width: 1e6, Around: int(circumference / 0.6)}
	for k := range int64(100) {
		a := noise.Hash(4, k) * 2 * math.Pi
		p := maths.V(1.5e9*math.Sin(a), noise.Hash(5, k)*1e6-5e5,
			1.5e9*math.Cos(a))
		tile := b.Nearest(p)
		if d := p.Sub(b.Centre(tile)).Len(); d > b.Spacing() {
			t.Fatalf("point %d: middle %v away, spacing %v", k, d,
				b.Spacing())
		}
	}
}

// Every surface measures along itself, knows its poles, and scatters
// points onto itself: a sphere's distance is the angle and its poles lie
// along north; a band's distance goes the short way round and its poles
// are its rims.
//
// A plane's distance is the straight line and its poles are the edge of
// its extent along north, none at all for a north straight up.
func TestEverySurfaceHasDistanceLatitudeAndSomewhere(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	s := Sphere{N: 4}
	if d := s.Distance(maths.V(1, 0, 0), maths.V(0, 1, 0)); !near(d, math.Pi/2) {
		t.Errorf("a quarter turn on the sphere is %v", d)
	}
	if l := s.Latitude(maths.V(0, 1, 0), maths.Up); !near(l, 1) {
		t.Errorf("the pole's latitude is %v", l)
	}
	if p := s.Somewhere(0.3, 0.6, 0.9); !near(p.Len(), 1) {
		t.Errorf("somewhere on the sphere is %v long", p.Len())
	}
	b := Band{Radius: 10, Width: 4, Around: 40}
	p, q := b.Somewhere(0.05, 0.5, 0), b.Somewhere(0.95, 0.5, 0)
	if d := b.Distance(p, q); !near(d, 0.1*2*math.Pi*10) {
		t.Errorf("round the band the short way is %v", d)
	}
	if l := b.Latitude(b.Somewhere(0, 0.5, 0), maths.Up); !near(l, 0) {
		t.Errorf("the band's middle has latitude %v", l)
	}
	if l := b.Latitude(b.Somewhere(0, 1, 0), maths.Up); !near(l, 1) {
		t.Errorf("the band's rim has latitude %v", l)
	}
	if r := math.Hypot(p.X, p.Z); !near(r, 10) {
		t.Errorf("somewhere on the band is %v from the axis", r)
	}
	f := Flat{Step: 1, Across: 20, Down: 20}
	if d := f.Distance(maths.V(0, 0, 0), maths.V(3, 7, 4)); !near(d, 5) {
		t.Errorf("across the plane is %v", d)
	}
	if l := f.Latitude(f.Somewhere(0.5, 0, 1), maths.Up); l != 0 {
		t.Errorf("a plane under a north straight up has latitude %v", l)
	}
	if l := f.Latitude(f.Somewhere(0.5, 0, 1), maths.V(0, 0, 1)); !near(l, 1) {
		t.Errorf("the plane's far edge along its north has latitude %v", l)
	}
	if l := f.Latitude(f.Somewhere(0.5, 0, 0.5), maths.V(0, 0, 1)); !near(l, 0) {
		t.Errorf("the plane's middle has latitude %v", l)
	}
	if f.Nearest(f.Somewhere(0.2, 0, 0.8)) == f.Nearest(f.Somewhere(0.8, 0, 0.2)) {
		t.Error("two corners of the plane land on one tile")
	}
}
