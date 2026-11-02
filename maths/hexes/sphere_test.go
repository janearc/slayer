package hexes

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// random is a direction all over the sphere, the same for the same k.
func random(k int64) maths.Vec {
	return maths.V(noise.Hash(1, k)*2-1, noise.Hash(2, k)*2-1,
		noise.Hash(3, k)*2-1).Unit()
}

// Splitting each face n times gives 10n^2+2 tiles, every one with its own
// ID: exactly twelve pentagons and the rest hexagons. That count is the
// Goldberg polyhedron's, and a tiling off by one tile has a seam somewhere.
func TestTwelvePentagonsAndTheRestHexagons(t *testing.T) {
	for _, n := range []int{1, 2, 5, 16} {
		tl := Sphere{N: n}
		tiles := tl.Tiles()
		if len(tiles) != tl.Count() || tl.Count() != 10*n*n+2 {
			t.Fatalf("n=%d: %d tiles, want %d", n, len(tiles),
				10*n*n+2)
		}
		five, six := 0, 0
		ids := map[int64]bool{}
		for _, tile := range tiles {
			if ids[tl.ID(tile)] {
				t.Fatalf("n=%d: two tiles share id %d", n,
					tl.ID(tile))
			}
			ids[tl.ID(tile)] = true
			switch len(tl.Neighbours(tile)) {
			case 5:
				five++
			case 6:
				six++
			default:
				t.Fatalf("n=%d: tile %v has %d neighbours", n,
					tile, len(tl.Neighbours(tile)))
			}
		}
		if five != 12 {
			t.Fatalf("n=%d: %d pentagons", n, five)
		}
	}
}

// Nearest finds the same tile as asking every tile, for directions all
// over the sphere, pentagons and face edges included.
func TestNearestIsTheTrueNearest(t *testing.T) {
	for _, n := range []int{1, 3, 12} {
		tl := Sphere{N: n}
		tiles := tl.Tiles()
		for k := range int64(20000) {
			p := random(k)
			got := tl.Nearest(p)
			best, want := math.Inf(1), Tile{}
			for _, tile := range tiles {
				if d := p.Sub(tl.Centre(tile)).Len(); d < best {
					best, want = d, tile
				}
			}
			if got != want &&
				math.Abs(p.Sub(tl.Centre(got)).Len()-best) > 1e-12 {
				t.Fatalf("n=%d point %d: tile %v, want %v", n,
					k, got, want)
			}
		}
	}
}

// Neighbours are each other's neighbours, and a tile is never its own.
func TestNeighboursAreMutual(t *testing.T) {
	tl := Sphere{N: 4}
	for _, tile := range tl.Tiles() {
		for _, nb := range tl.Neighbours(tile) {
			if nb == tile {
				t.Fatalf("%v is its own neighbour", tile)
			}
			back := false
			for _, b := range tl.Neighbours(nb) {
				back = back || b == tile
			}
			if !back {
				t.Fatalf("%v has %v, which does not have it",
					tile, nb)
			}
		}
	}
}

// A direction is further from its tile's edge the nearer it is to the
// middle, and at the midpoint between two neighbours it is on the edge.
func TestEdgeIsNoughtBetweenNeighbours(t *testing.T) {
	tl := Sphere{N: 8}
	tiles := tl.Tiles()
	for i := 0; i < len(tiles); i += 37 {
		tile := tiles[i]
		c := tl.Centre(tile)
		n := tl.Centre(tl.Neighbours(tile)[0])
		mid := c.Add(n).Unit()
		if e := tl.Edge(mid, tile); math.Abs(e) > 1e-9 {
			t.Fatalf("tile %v: midpoint is %v from the edge", tile,
				e)
		}
		nearer := c.Scale(3).Add(n).Unit()
		if tl.Edge(c, tile) <= tl.Edge(nearer, tile) {
			t.Fatalf("tile %v: the middle is no further from the "+
				"edge", tile)
		}
	}
}

// Every middle of a tiling is a middle of the one a rung finer, so a
// thing placed on a tile stays put when the rung changes.
func TestEveryMiddleIsAMiddleOneRungFiner(t *testing.T) {
	tl := Sphere{N: 5}
	fine := tl.Finer().(Sphere)
	for _, tile := range tl.Tiles() {
		c := tl.Centre(tile)
		if got := fine.Centre(fine.Nearest(c)); !got.Near(c, 1e-12) {
			t.Fatalf("tile %v's middle %v is %v on the finer rung",
				tile, c, got)
		}
	}
}

// Spacing is the icosahedron's edge over n, and SphereFor undoes it; the
// zero Tiling is the icosahedron.
func TestSpacing(t *testing.T) {
	if (Sphere{}).Count() != 12 {
		t.Fatalf("the zero tiling has %d tiles", (Sphere{}).Count())
	}
	if got := (Sphere{N: 10}).Spacing(); math.Abs(got-math.Atan(2)/10) > 1e-12 {
		t.Fatalf("spacing at 10 is %v", got)
	}
	if got := SphereFor((Sphere{N: 300}).Spacing()); got.N != 300 {
		t.Fatalf("for its own spacing, %v", got)
	}
}

// A manhole on a state: the earth tiled 0.6 metres apart is twelve
// million splits and 10^15 tiles, and Nearest still answers, without a
// table, with a tile whose middle is within a spacing of the direction
// and whose neighbours are all about a spacing away. Distances are
// chords, since an angle this small is lost in the arc cosine.
func TestAManholeOnAState(t *testing.T) {
	const earth, manhole = 6.371e6, 0.6
	tl := SphereFor(manhole / earth)
	if tl.N < 1e7 || tl.N > 2e7 {
		t.Fatalf("the manhole rung is %d splits", tl.N)
	}
	for k := range int64(200) {
		p := random(k)
		tile := tl.Nearest(p)
		c := tl.Centre(tile)
		if off := p.Sub(c).Len(); off > tl.Spacing() {
			t.Fatalf("point %d: middle %v away, spacing %v", k, off,
				tl.Spacing())
		}
		if e := tl.Edge(p, tile); e < -1e-12 {
			t.Fatalf("point %d: %v outside its own tile", k, e)
		}
		if tl.Nearest(p) != tile {
			t.Fatalf("point %d: a different tile the second time", k)
		}
		for _, nb := range tl.Neighbours(tile) {
			d := c.Sub(tl.Centre(nb)).Len()
			if d > 1.3*tl.Spacing() || d < 0.7*tl.Spacing() {
				t.Fatalf("point %d: a neighbour %v away", k, d)
			}
		}
	}
}

func BenchmarkNearestCoarse(b *testing.B) {
	tl := Sphere{N: 12}
	for i := range b.N {
		tl.Nearest(random(int64(i % 1000)))
	}
}

func BenchmarkNearestFine(b *testing.B) {
	tl := Sphere{N: 1 << 23}
	for i := range b.N {
		tl.Nearest(random(int64(i % 1000)))
	}
}

func BenchmarkEdgeCoarse(b *testing.B) {
	tl := Sphere{N: 12}
	p := random(7)
	tile := tl.Nearest(p)
	for range b.N {
		tl.Edge(p, tile)
	}
}
