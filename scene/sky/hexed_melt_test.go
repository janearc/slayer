package sky

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// spy is a sky that is all one grey and writes down every direction it is
// asked about.
type spy struct{ asked []maths.Vec }

func (s *spy) Light(toward maths.Vec) maths.Light {
	s.asked = append(s.asked, toward)
	return maths.Grey(1)
}

// from a tile's middle to the seam with each neighbour, a hexed sky is the
// flat colour while the seam is farther than the melt, and from there in
// to the seam a blend of the true sky and the flat one.
//
// the blend is by the smooth step of how far the point is from the seam,
// over how wide the melt is: 3x^2 - 2x^3, worked by hand. the seam's
// distance is the tiling's own.
//
// a melt said as nothing is a quarter of the spacing. the walks are
// counted, so the test fails if none of them is in a melt.
func TestHexedMeltsByTheSmoothStepOfTheSeam(t *testing.T) {
	tl := hexes.Sphere{N: 6}
	cases := []struct {
		name       string
		soft, melt float64
		someFlat   bool
	}{
		{"nothing said", 0, tl.Spacing() / 4, true},
		{"a twentieth", 0.05, 0.05, true},
		{"wider than any tile", 0.3, 0.3, false},
	}
	for _, c := range cases {
		h := Hexed{Of: g, Tiling: tl, Soft: c.soft}
		melting, flat := 0, 0
		for _, tile := range tl.Tiles() {
			middle := tl.Centre(tile)
			colour := g.Light(middle)
			for _, nb := range tl.Neighbours(tile) {
				seam := middle.Add(tl.Centre(nb)).Unit()
				for k := 0; k <= 40; k++ {
					d := middle.Add(seam.Sub(middle).Scale(float64(k) / 40)).Unit()
					if tl.Nearest(d) != tile {
						continue
					}
					in := 1.0
					if e := tl.Edge(d, tile); e <= 0 {
						in = 0
					} else if e < c.melt {
						x := e / c.melt
						in = x * x * (3 - 2*x)
					}
					if in > 0 && in < 1 {
						melting++
					}
					if in == 1 {
						flat++
					}
					want := g.Light(d).Scale(1 - in).Add(colour.Scale(in))
					if got := h.Light(d); !within(got, want) {
						t.Fatalf("%s: tile %v toward %v: %v, want %v (in %v)",
							c.name, tile, nb, got, want, in)
					}
				}
			}
		}
		if melting < 100 || (c.someFlat && flat < 100) {
			t.Errorf("%s: %d points melting and %d flat, too few to say", c.name,
				melting, flat)
		}
	}
}

// inside a tile the true sky is not asked: the flat colour is the sky at
// the tile's middle and that is the one question. at the seam the flat
// colour and the true sky are both asked, the middle first.
func TestHexedAsksTheSkyOnceInsideATile(t *testing.T) {
	tl := hexes.Sphere{N: 6}
	for _, tile := range tl.Tiles() {
		inside := &spy{}
		(Hexed{Of: inside, Tiling: tl}).Light(tl.Centre(tile))
		if len(inside.asked) != 1 || inside.asked[0] != tl.Centre(tile) {
			t.Fatalf("tile %v: the middle asked %v, want the middle once", tile, inside.asked)
		}
		nb := tl.Neighbours(tile)[0]
		seam := tl.Centre(tile).Add(tl.Centre(nb)).Unit()
		at := &spy{}
		(Hexed{Of: at, Tiling: tl}).Light(seam)
		// the seam is in either tile that meets there
		if len(at.asked) != 2 || (at.asked[0] != tl.Centre(tile) &&
			at.asked[0] != tl.Centre(nb)) || !at.asked[1].Near(seam, 1e-15) {
			t.Fatalf("tile %v: the seam asked %v, want the middle then the seam", tile, at.asked)
		}
	}
}

// with no tiling it is one tile everywhere, which is no tiles: the sky
// itself, asked the way it was asked, whatever the melt.
func TestHexedWithNoTilingIsTheSkyItself(t *testing.T) {
	ways := []maths.Vec{maths.V(3, 4, 0), maths.V(0, -1, 2), maths.V(1e200, 1, 0),
		maths.V(1e-200, 0, 1e-200)}
	for _, soft := range []float64{0, 0.3} {
		h := Hexed{Of: g, Tiling: nil, Soft: soft}
		for _, v := range ways {
			if got, want := h.Light(v), g.Light(v); got != want {
				t.Errorf("soft %v toward %v: %v, the sky gives %v", soft, v, got, want)
			}
			s := &spy{}
			(Hexed{Of: s, Soft: soft}).Light(v)
			if len(s.asked) != 1 || s.asked[0] != v {
				t.Errorf("soft %v toward %v: the sky was asked %v", soft, v, s.asked)
			}
		}
	}
}
