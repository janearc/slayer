package hexes

import (
	"math"
	"testing"
)

// a band wider than thirty-two bits of columns, as an orbital is, still
// gives every tile its own ID: two tiles a 2^32 columns apart differ.
func TestAWideBandsTilesHaveTheirOwnIDs(t *testing.T) {
	b := Band{Radius: 1.5e9, Width: 1e6, Around: 15707963267}
	if a, c := b.ID(Tile{0, 0, 3}), b.ID(Tile{0, 1 << 32, 3}); a == c {
		t.Errorf("two tiles share the id %d", a)
	}
	ordinary := Band{Radius: 10, Width: 4, Around: 40}
	if ordinary.ID(Tile{0, 7, 2}) != 2<<32|7 {
		t.Errorf("an ordinary band's id moved: %d",
			ordinary.ID(Tile{0, 7, 2}))
	}
}

// a flat extent of a negative size has no tiles, rather than panicking.
func TestAFlatOfNoSizeHasNoTiles(t *testing.T) {
	for _, f := range []Flat{{Step: 1, Across: -2, Down: 3},
		{Step: 1, Across: 3, Down: -1}} {
		if tiles := f.Tiles(); len(tiles) != 0 {
			t.Errorf("%+v has %d tiles", f, len(tiles))
		}
	}
}

// somewhere on a sphere is always on it: the middle of the cube the three
// numbers span is a direction too.
func TestSomewhereIsOnTheSphere(t *testing.T) {
	for _, abc := range [][3]float64{{0.5, 0.5, 0.5}, {0, 0, 0},
		{0.9, 0.1, 0.5}} {
		p := (Sphere{N: 4}).Somewhere(abc[0], abc[1], abc[2])
		if math.Abs(p.Len()-1) > 1e-12 {
			t.Errorf("somewhere %v is %v, %v long", abc, p, p.Len())
		}
	}
}

// a sphere for a spacing that is no number is the coarsest, and for one of zero
// or less, or so fine its tiles would not count, the finest that counts in an
// int64: never a count less than zero, nor one that differs between machines.
func TestASphereForAnySpacingCounts(t *testing.T) {
	for _, radians := range []float64{math.NaN(), math.Inf(1), 0, -1, 1e-12,
		1e-300} {
		s := SphereFor(radians)
		if s.N < 1 || s.N > finest || s.Count() < 12 {
			t.Errorf("for %v: N %d, %d tiles", radians, s.N, s.Count())
		}
	}
	if SphereFor(math.NaN()).N != 1 || SphereFor(0).N != finest {
		t.Errorf("NaN is %d, nought is %d", SphereFor(math.NaN()).N,
			SphereFor(0).N)
	}
	if SphereFor(edge/40).N != 40 {
		t.Errorf("an ordinary spacing moved: %d", SphereFor(edge/40).N)
	}
}

// the rung below the finest is the finest, and every rung down to it counts its
// tiles: never a count less than zero.
func TestTheFinestHasNoFinerRung(t *testing.T) {
	s := Sphere{N: 1}
	for range 40 {
		s = s.Finer().(Sphere)
		if s.Count() < 12 {
			t.Fatalf("%+v counts %d tiles", s, s.Count())
		}
	}
	if s.N != finest {
		t.Errorf("forty rungs down is %+v, not the finest", s)
	}
}
