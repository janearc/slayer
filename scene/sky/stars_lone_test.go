package sky

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// loneStar is a sky of one star: a share of exactly the smallest hash of
// any tile of the tiling, which is held to one tile by counting every
// tile's hash against it.
//
// it asks for the seed, from 1 up, whose star sits inside its tile and
// within a twenty-fifth of a spacing of that tile's edge, so that a star
// of any size reaches over the edge.
type loneStar struct {
	tl    hexes.Sphere
	seed  uint64
	tile  hexes.Tile
	share float64
	at    maths.Vec
}

func findLoneStar(t *testing.T) loneStar {
	t.Helper()
	tl := coarse
	tiles := tl.Tiles()
	for seed := uint64(1); seed < 400; seed++ {
		best, share := tiles[0], 2.0
		for _, tile := range tiles {
			if h := noise.Hash(seed, tl.ID(tile)); h < share {
				best, share = tile, h
			}
		}
		at := placed(tl, best, seed)
		if tl.Nearest(at) != best {
			continue
		}
		if e := tl.Edge(at, best); e <= 0 || e > tl.Spacing()/25 {
			continue
		}
		held := 0
		for _, tile := range tiles {
			if noise.Hash(seed, tl.ID(tile)) <= share {
				held++
			}
		}
		if held != 1 {
			t.Fatalf("seed %d: %d tiles hold a star at the smallest hash", seed, held)
		}
		return loneStar{tl, seed, best, share, at}
	}
	t.Fatal("no seed to 399 puts the smallest hash's star near its tile's edge")
	return loneStar{}
}

// around is the direction a chord away from at, which is a unit vector,
// turned phi from some fixed way round it. a chord c is the straight line
// between two points of the unit sphere, 2 sin(theta/2) for the angle
// theta between them, so theta is twice the arc sine of c over two.
func around(at maths.Vec, phi, chord float64) maths.Vec {
	k := maths.YAxis
	if math.Abs(at.Y) > 0.9 {
		k = maths.XAxis
	}
	u := at.Cross(k).Unit()
	w := at.Cross(u)
	e := u.Scale(math.Cos(phi)).Add(w.Scale(math.Sin(phi)))
	theta := 2 * math.Asin(chord/2)
	return at.Scale(math.Cos(theta)).Add(e.Scale(math.Sin(theta)))
}

// fall is how much of a star's light is left at a share t of its radius
// from it: one less the smooth step 3t^2 - 2t^3, worked by hand. a quarter
// leaves 1 - 0.15625, a half leaves a half, three quarters 1 - 0.84375.
func fall(t float64) float64 {
	if t >= 1 {
		return 0
	}
	return 1 - t*t*(3-2*t)
}

// a lone star is its peak at its middle, falling smoothly to nothing at
// its radius, which is its size in spacings, and everywhere else is the
// dark.
//
// but only the tile a direction is in, and with a size past 0.3 the tiles
// beside that, are asked for stars: so from a star near its edge, a size
// of 0.3 or under is cut at the edge and a bigger one reaches over.
func TestALoneStarFallsOffAndIsCutByItsTiles(t *testing.T) {
	lone := findLoneStar(t)
	tl := lone.tl
	next := math.Nextafter(0.3, 1)
	cases := []struct {
		name   string
		size   float64
		bright float64
		ring   bool
		radius float64
	}{
		{"no size", 0, 0, false, 0.1},
		{"a tenth, a bright of a half", 0.1, 0.5, false, 0.1},
		{"a fifth", 0.2, 2.5, false, 0.2},
		{"exactly 0.3", 0.3, 1, false, 0.3},
		{"just over 0.3", next, 1, true, next},
		{"six tenths", 0.6, 2.5, true, 0.6},
		{"one", 1, 1, true, 1},
		{"one and a fifth", 1.2, 0.5, true, 1.2},
		{"a negative size", -2, 1, false, 0.1},
	}
	allowed := map[hexes.Tile]bool{lone.tile: true}
	for _, nb := range tl.Neighbours(lone.tile) {
		allowed[nb] = true
	}
	for _, c := range cases {
		st := Stars{Tiling: tl, Seed: lone.seed, Share: lone.share, Size: c.size,
			Bright: c.bright, Dark: dusk}
		bright := c.bright
		if bright <= 0 {
			bright = 1
		}
		top := peak(lone.seed, tl.ID(lone.tile), bright)
		if got, want := st.Light(lone.at), dusk.Add(top); !within(got, want) {
			t.Errorf("%s: at the star the sky is %v, want %v", c.name, got, want)
		}
		radius := c.radius * tl.Spacing()
		reached, cutOff, over := 0, 0, 0
		for _, out := range []float64{0.25, 0.5, 0.75, 0.95, 1.05, 1.5} {
			for k := range 24 {
				d := around(lone.at, float64(k)*math.Pi/12, out*radius)
				// the tile asked is the one the direction is in, and with the
				// ring on the ones beside it, so the star shows from its own
				// tile and its neighbours if the ring is on
				in := tl.Nearest(d)
				shown := in == lone.tile || (c.ring && allowed[in])
				want := dusk
				if shown {
					want = dusk.Add(top.Scale(fall(out)))
				}
				if got := st.Light(d); !within(got, want) {
					t.Errorf("%s: %.2f of the radius out at %d/24: %v, want %v",
						c.name, out, k, got, want)
				}
				if out < 1 && in == lone.tile {
					reached++
				}
				if out < 1 && in != lone.tile && !c.ring {
					cutOff++
				}
				if out < 1 && in != lone.tile && c.ring && allowed[in] {
					over++
				}
			}
		}
		if reached == 0 {
			t.Errorf("%s: no direction within the star's radius is in its tile", c.name)
		}
		if !c.ring && cutOff == 0 {
			t.Errorf("%s: no direction within the star's radius is in another tile", c.name)
		}
		if c.ring && over == 0 {
			t.Errorf("%s: no direction within the star's radius is in a neighbour", c.name)
		}
	}
}

// a star of endless light is endless at and near its middle and the dark past
// its radius: endless light times the zero of the edge is no number, so a star
// past its radius has to add nothing, not zero times its light.
func TestAnEndlessStarIsDarkPastItsRadius(t *testing.T) {
	lone := findLoneStar(t)
	tl := lone.tl
	st := Stars{Tiling: tl, Seed: lone.seed, Share: lone.share, Size: 0.2,
		Bright: math.Inf(1), Dark: dusk}
	radius := 0.2 * tl.Spacing()
	for k := range 24 {
		phi := float64(k) * math.Pi / 12
		for _, out := range []float64{1.05, 1.5, 3} {
			if got := st.Light(around(lone.at, phi, out*radius)); got != dusk {
				t.Errorf("%.2f of the radius out at %d/24: %v, want the dark %v", out, k, got, dusk)
			}
		}
		d := around(lone.at, phi, 0.5*radius)
		if got := st.Light(d); tl.Nearest(d) == lone.tile && !math.IsInf(got.G, 1) {
			t.Errorf("half the radius out at %d/24 in the star's tile: %v, want endless", k, got)
		}
	}
	if got := st.Light(lone.at); !math.IsInf(got.G, 1) {
		t.Errorf("at the star: %v, want endless", got)
	}
}
