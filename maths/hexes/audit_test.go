package hexes

import (
	"math"
	"testing"
)

// a band wider than thirty-two bits of columns, as an orbital is, still
// gives every tile its own ID: two tiles a 2^32 columns apart differ.
func TestAWideBandsTilesHaveTheirOwnIDs(t *testing.T) {
	band := Band{Radius: 1.5e9, Width: 1e6, Around: 15707963267}
	if a, c := band.ID(Tile{0, 0, 3}), band.ID(Tile{0, 1 << 32, 3}); a == c {
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
	for _, flat := range []Flat{{Step: 1, Across: -2, Down: 3},
		{Step: 1, Across: 3, Down: -1}} {
		if tiles := flat.Tiles(); len(tiles) != 0 {
			t.Errorf("%+v has %d tiles", flat, len(tiles))
		}
	}
}

// somewhere on a sphere is always on it: the middle of the cube the three
// numbers span is a direction too.
func TestSomewhereIsOnTheSphere(t *testing.T) {
	for _, abc := range [][3]float64{{0.5, 0.5, 0.5}, {0, 0, 0},
		{0.9, 0.1, 0.5}} {
		point := (Sphere{Splits: 4}).Somewhere(abc[0], abc[1], abc[2])
		if !(math.Abs(point.Len()-1) <= 1e-12) {
			t.Errorf("somewhere %v is %v, %v long", abc, point, point.Len())
		}
	}
}

// a sphere for a spacing that is no number is the coarsest, and for one of zero
// or less, or so fine its tiles would not count, the finest that counts in an
// int64: never a count less than zero, nor one that differs between machines.
func TestASphereForAnySpacingCounts(t *testing.T) {
	for _, radians := range []float64{math.NaN(), math.Inf(1), 0, -1, 1e-12,
		1e-300} {
		sphere := SphereFor(radians)
		if sphere.Splits < 1 || sphere.Splits > finest || sphere.count() < 12 {
			t.Errorf("for %v: N %d, %d tiles", radians, sphere.Splits, sphere.count())
		}
	}
	if SphereFor(math.NaN()).Splits != 1 || SphereFor(0).Splits != finest {
		nanSplits := SphereFor(math.NaN()).Splits
		noughtSplits := SphereFor(0).Splits
		t.Errorf("NaN is %d, nought is %d", nanSplits,
			noughtSplits)
	}
	if SphereFor(edge/40).Splits != 40 {
		t.Errorf("an ordinary spacing moved: %d", SphereFor(edge/40).Splits)
	}
}

// the rung below the finest is the finest, and every rung down to it counts its
// tiles: never a count less than zero.
func TestTheFinestHasNoFinerRung(t *testing.T) {
	sphere := Sphere{Splits: 1}
	for range 40 {
		sphere = sphere.Finer().(Sphere)
		if sphere.count() < 12 {
			t.Fatalf("%+v counts %d tiles", sphere, sphere.count())
		}
	}
	if sphere.Splits != finest {
		t.Errorf("forty rungs down is %+v, not the finest", sphere)
	}
}
