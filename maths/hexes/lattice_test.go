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
	flat := Flat{Step: 2, Across: 7, Down: 5}
	tiles := flat.Tiles()
	if len(tiles) != 35 {
		t.Fatalf("%d tiles", len(tiles))
	}
	ids := map[int64]bool{}
	for _, tile := range tiles {
		if ids[flat.ID(tile)] {
			t.Fatalf("two tiles share id %d", flat.ID(tile))
		}
		ids[flat.ID(tile)] = true
		ns := flat.Neighbours(tile)
		inside := tile.Column > 0 && tile.Row > 0 && tile.Column < 6 && tile.Row < 4
		if inside && len(ns) != 6 || !inside && len(ns) >= 6 {
			t.Fatalf("tile %v has %d neighbours", tile, len(ns))
		}
		for _, n := range ns {
			distance := flat.Centre(tile).Sub(flat.Centre(n)).Len()
			if !(math.Abs(distance-2) <= 1e-12) {
				t.Fatalf("tile %v and %v are %v apart", tile, n, distance)
			}
		}
	}
}

// Nearest on a plane finds the same tile as asking every tile, for points
// all over the extent, and the middle of a tile is in that tile.
func TestFlatNearestIsTheTrueNearest(t *testing.T) {
	flat := Flat{Step: 1.5, Across: 12, Down: 9}
	tiles := flat.Tiles()
	for k := range int64(5000) {
		point := maths.V(noise.Hash(1, k)*16-1, 0, noise.Hash(2, k)*12-1)
		got := flat.Nearest(point)
		best, want := math.Inf(1), Tile{}
		for _, tile := range tiles {
			if distance := point.Sub(flat.Centre(tile)).Len(); distance < best {
				best, want = distance, tile
			}
		}
		// outside the extent the true nearest may be a tile Tiles
		// does not list, so only points inside are held to it
		if flat.holds(got.Column, got.Row) && got != want &&
			math.Abs(point.Sub(flat.Centre(got)).Len()-best) > 1e-12 {
			t.Fatalf("point %d at %v: tile %v, want %v", k, point, got, want)
		}
	}
	for _, tile := range tiles {
		if flat.Nearest(flat.Centre(tile)) != tile {
			t.Fatalf("tile %v's middle is not in it", tile)
		}
	}
}

// On a plane a point is zero from the edge at the midpoint between two
// neighbours, inside its tile by Spacing/2 at the middle, and outside a
// neighbour's tile.
func TestFlatEdge(t *testing.T) {
	flat := Flat{Step: 1, Across: 6, Down: 6}
	tile := Tile{0, 3, 3}
	for _, n := range flat.Neighbours(tile) {
		mid := flat.Centre(tile).Add(flat.Centre(n)).Scale(0.5)
		if e := flat.Edge(mid, tile); !(math.Abs(e) <= 1e-12) {
			t.Fatalf("midpoint with %v is %v from the edge", n, e)
		}
		if e := flat.Edge(flat.Centre(tile), n); e >= 0 {
			t.Fatalf("the middle of %v is inside %v", tile, n)
		}
	}
	if e := flat.Edge(flat.Centre(tile), tile); !(math.Abs(e-0.5) <= 1e-12) {
		t.Fatalf("the middle is %v from the edge", e)
	}
}

// A band wraps: the last column's neighbours include the first, a point just
// past the seam is in column zero, and the middles are Spacing apart across the
// seam too.
func TestBandWraps(t *testing.T) {
	band := Band{Radius: 10, Width: 4, Around: 40}
	last := Tile{0, 39, 2}
	wrapped := false
	for _, n := range band.Neighbours(last) {
		wrapped = wrapped || n.Column == 0
		distance := band.Centre(last).Sub(band.Centre(n)).Len()
		if distance > 1.01*band.Spacing() {
			t.Fatalf("%v and %v are %v apart, spacing %v", last, n, distance,
				band.Spacing())
		}
	}
	if !wrapped {
		t.Fatal("the last column does not reach the first")
	}
	just := maths.V(1e-6, 0, 10)
	if got := band.Nearest(just); got.Column != 0 {
		t.Fatalf("just past the seam is column %d", got.Column)
	}
	seam := band.Centre(last).Add(band.Centre(Tile{0, 0, 2})).Scale(0.5)
	if e := band.Edge(seam, last); !(math.Abs(e) <= 1e-9) {
		t.Fatalf("the seam midpoint is %v from the edge", e)
	}
}

// A band's tiles are Around by the rows that fit, the middles lie on the
// cylinder, and a point at any distance from the axis is in the tile its
// angle and height say.
func TestBandTilesLieOnTheCylinder(t *testing.T) {
	band := Band{Radius: 3, Width: 2, Around: 24}
	tiles := band.Tiles()
	if len(tiles) != 24*band.rows() || band.rows() < 2 {
		t.Fatalf("%d tiles in %d rows", len(tiles), band.rows())
	}
	for _, tile := range tiles {
		centre := band.Centre(tile)
		if radius := math.Hypot(centre.X, centre.Z); !(math.Abs(radius-3) <= 1e-12) {
			t.Fatalf("tile %v's middle is %v from the axis", tile, radius)
		}
		far := maths.V(centre.X*5, centre.Y, centre.Z*5) // further from the axis
		if band.Nearest(centre) != tile || band.Nearest(far) != tile {
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
			centre := tl.Centre(tile)
			if got := fine.Centre(fine.Nearest(centre)); !got.Near(centre, 1e-9) {
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
	band := Band{Radius: 1.5e9, Width: 1e6, Around: int(circumference / 0.6)}
	for k := range int64(100) {
		angle := noise.Hash(4, k) * 2 * math.Pi
		point := maths.V(1.5e9*math.Sin(angle), noise.Hash(5, k)*1e6-5e5,
			1.5e9*math.Cos(angle))
		tile := band.Nearest(point)
		if distance := point.Sub(band.Centre(tile)).Len(); distance > band.Spacing() {
			t.Fatalf("point %d: middle %v away, spacing %v", k, distance,
				band.Spacing())
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
	sphere := Sphere{Splits: 4}
	if distance := sphere.Distance(maths.V(1, 0, 0), maths.V(0, 1, 0)); !near(distance, math.Pi/2) {
		t.Errorf("a quarter turn on the sphere is %v", distance)
	}
	if latitude := sphere.Latitude(maths.V(0, 1, 0), maths.Up); !near(latitude, 1) {
		t.Errorf("the pole's latitude is %v", latitude)
	}
	if point := sphere.Somewhere(0.3, 0.6, 0.9); !near(point.Len(), 1) {
		t.Errorf("somewhere on the sphere is %v long", point.Len())
	}
	band := Band{Radius: 10, Width: 4, Around: 40}
	point, q := band.Somewhere(0.05, 0.5, 0), band.Somewhere(0.95, 0.5, 0)
	if distance := band.Distance(point, q); !near(distance, 0.1*2*math.Pi*10) {
		t.Errorf("round the band the short way is %v", distance)
	}
	if latitude := band.Latitude(band.Somewhere(0, 0.5, 0), maths.Up); !near(latitude, 0) {
		t.Errorf("the band's middle has latitude %v", latitude)
	}
	if latitude := band.Latitude(band.Somewhere(0, 1, 0), maths.Up); !near(latitude, 1) {
		t.Errorf("the band's rim has latitude %v", latitude)
	}
	if radius := math.Hypot(point.X, point.Z); !near(radius, 10) {
		t.Errorf("somewhere on the band is %v from the axis", radius)
	}
	flat := Flat{Step: 1, Across: 20, Down: 20}
	if distance := flat.Distance(maths.V(0, 0, 0), maths.V(3, 7, 4)); !near(distance, 5) {
		t.Errorf("across the plane is %v", distance)
	}
	if latitude := flat.Latitude(flat.Somewhere(0.5, 0, 1), maths.Up); latitude != 0 {
		t.Errorf("a plane under a north straight up has latitude %v", latitude)
	}
	if latitude := flat.Latitude(flat.Somewhere(0.5, 0, 1), maths.V(0, 0, 1)); !near(latitude, 1) {
		t.Errorf("the plane's far edge along its north has latitude %v", latitude)
	}
	if latitude := flat.Latitude(flat.Somewhere(0.5, 0, 0.5), maths.V(0, 0, 1)); !near(latitude, 0) {
		t.Errorf("the plane's middle has latitude %v", latitude)
	}
	if flat.Nearest(flat.Somewhere(0.2, 0, 0.8)) == flat.Nearest(flat.Somewhere(0.8, 0, 0.2)) {
		t.Error("two corners of the plane land on one tile")
	}
}
