package sky

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// coarse is the tiling most star tests walk: twelve splits is 1442 tiles,
// few enough to look at every one.
var coarse = hexes.Sphere{N: 12}

// dusk is a dark with a different part in each of red, green and blue, so
// that a part swapped for another shows.
var dusk = maths.RGB(0.01, 0.02, 0.03)

// placed is where a tile's star sits: the tile's middle moved by its three
// hashes less a half, over six tenths of the spacing, and put back on the
// sphere. there is no other way to know where a star is, so this is the
// one place the tests hold that rule.
func placed(tl hexes.Sphere, tile hexes.Tile, seed uint64) maths.Vec {
	id := tl.ID(tile)
	off := maths.V(noise.Hash(seed+1, id)-0.5, noise.Hash(seed+2, id)-0.5,
		noise.Hash(seed+3, id)-0.5)
	return tl.Centre(tile).Add(off.Scale(0.6 * tl.Spacing())).Unit()
}

// peak is the light of a tile's star at its middle, from the hashes as the
// doc for Stars gives them: bright times the cube of one hash, and the
// tint, the hash of the fifth less a half, taking red up and blue down by
// three tenths of it.
func peak(seed uint64, id int64, bright float64) maths.Light {
	b := noise.Hash(seed+4, id)
	tint := noise.Hash(seed+5, id) - 0.5
	l := bright * math.Pow(b, 3)
	return maths.RGB(l*(1+0.3*tint), l, l*(1-0.3*tint))
}

// within is whether two lights are within a trillionth in every part. the
// lights here are all below three.
func within(a, b maths.Light) bool { return near(a, b, 1e-12) }

// starOf is what a tile's star is worth at its own middle for a share and
// a bright: nothing when its hash is past the share.
func starOf(seed uint64, id int64, share, bright float64) (maths.Light, bool) {
	if noise.Hash(seed, id) > share {
		return maths.Light{}, false
	}
	return peak(seed, id, bright), true
}

// a star is in a tile when the tile's hash is no more than the share, and
// is bright by its own hashes. every tile of the coarse tiling is asked at
// the spot its star sits, for each way the share and the bright can be
// said: not at all, nonsense, or a number.
//
// the tiles are counted by hash and held to the share. a tile whose star
// sits past its own edge sees nothing of it at this size, and is not asked.
func TestEachTileHoldsAStarByItsHash(t *testing.T) {
	tl := coarse
	const seed = 5
	tiles := tl.Tiles()
	// a share of exactly one tile's hash: that tile is in, the next out
	exact := -1.0
	for _, tile := range tiles[600:] {
		if tl.Nearest(placed(tl, tile, seed)) == tile {
			exact = noise.Hash(seed, tl.ID(tile))
			break
		}
	}
	if exact < 0 {
		t.Fatal("no tile holds its own star")
	}
	cases := []struct {
		name                  string
		share, bright         float64
		wantShare, wantBright float64
	}{
		{"neither said", 0, 0, 0.2, 1},
		{"neither a number", -3, -1, 0.2, 1},
		{"a fifth and one", 0.2, 1, 0.2, 1},
		{"a half and a half", 0.5, 0.5, 0.5, 0.5},
		{"a twentieth and 2.5", 0.05, 2.5, 0.05, 2.5},
		{"the whole sky and 3", 1, 3, 1, 3},
		{"exactly a tile's hash", exact, 1, exact, 1},
	}
	for _, c := range cases {
		st := Stars{Tiling: tl, Seed: seed, Share: c.share, Bright: c.bright, Dark: dusk}
		lit, cut, wrong := 0, 0, 0
		for _, tile := range tiles {
			id := tl.ID(tile)
			star, holds := starOf(seed, id, c.wantShare, c.wantBright)
			if holds {
				lit++
			}
			at := placed(tl, tile, seed)
			if tl.Nearest(at) != tile {
				cut++
				continue
			}
			want := dusk.Add(star)
			if got := st.Light(at); !within(got, want) {
				wrong++
				if wrong < 4 {
					t.Errorf("%s: tile %v: at its star the sky is %v, want %v",
						c.name, tile, got, want)
				}
			}
		}
		n := float64(len(tiles))
		mean := n * c.wantShare
		sd := math.Sqrt(n * c.wantShare * (1 - c.wantShare))
		if math.Abs(float64(lit)-mean) > 5*sd {
			t.Errorf("%s: %d of %d tiles hold a star, five spreads from %v",
				c.name, lit, len(tiles), mean)
		}
		if c.wantShare == 1 && lit != len(tiles) {
			t.Errorf("%s: %d of %d tiles hold a star, want all", c.name, lit, len(tiles))
		}
		if cut > len(tiles)/4 {
			t.Errorf("%s: %d of %d stars spill past their tile", c.name, cut, len(tiles))
		}
	}
}

// with nothing said the stars are on a hundred and twenty splits, a fifth
// of the tiles, a bright of one and a dark of nothing. one tile in 29 of
// the 144002 is asked, and the same ones are held to the same hash.
func TestStarsWithNothingSaidAreOnAHundredAndTwentySplits(t *testing.T) {
	tl := hexes.Sphere{N: 120}
	const seed = 3
	st := Stars{Seed: seed}
	tiles := tl.Tiles()
	if len(tiles) != 144002 {
		t.Fatalf("%d tiles, want 144002", len(tiles))
	}
	asked, lit, wrong := 0, 0, 0
	for i := 0; i < len(tiles); i += 29 {
		tile := tiles[i]
		at := placed(tl, tile, seed)
		if tl.Nearest(at) != tile {
			continue
		}
		asked++
		star, holds := starOf(seed, tl.ID(tile), 0.2, 1)
		if holds {
			lit++
		}
		if got := st.Light(at); !within(got, star) {
			wrong++
			if wrong < 4 {
				t.Errorf("tile %v: at its star the sky is %v, want %v", tile, got, star)
			}
		}
	}
	mean := float64(asked) * 0.2
	if sd := math.Sqrt(float64(asked) * 0.2 * 0.8); math.Abs(float64(lit)-mean) > 5*sd {
		t.Errorf("%d of %d tiles hold a star, five spreads from %v", lit, asked, mean)
	}
}
