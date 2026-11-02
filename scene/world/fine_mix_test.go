package world

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
)

// a fine tile takes its class's paint, lightened or darkened by 0.4 of its own
// hash minus a half: the paint times 1 + vary * (h - 0.5), h the hash of the
// tile's name under the world's seed plus 91. a vary of zero leaves the paint
// as it is. six tiles on a chart, all grass.
func TestAFineTileVariesByItsOwnHash(t *testing.T) {
	const seed = 5
	c := chained(6)
	w := on(c, seed)
	w.biome = []Biome{Grass, Grass, Grass, Grass, Grass, Grass}
	look := paints()
	base := look.Paints[Grass]
	for _, vary := range []float64{0, 0.4} {
		tx := w.Fine(look, 0, vary, 0)
		for i := range c.at {
			h := noise.Hash(seed+91, int64(i)) - 0.5
			want := base.Scale(1 + vary*h)
			got := tx.At(c.at[i])
			if !close12(got.R, want.R) || !close12(got.G, want.G) ||
				!close12(got.B, want.B) {
				t.Errorf("tile %d, vary %v: %v, want %v", i, vary, got, want)
			}
		}
	}
}

// the mix on a chart of one tile, a middle at zero, where the noise is zero for
// any seed or scale: a tile takes its kin's class where the noise is over 0.55
// - 0.5 mix.
//
// so grass stays grass for a mix of zero, 0.5 (a line at 0.3) and 1.1 (a line
// at zero, which zero is not over), and is forest, grass's kin, at 2 (a line at
// -0.45).
func TestAMixTakesTheKinOnlyOverItsLine(t *testing.T) {
	w := on(chained(1), 5)
	w.biome = []Biome{Grass}
	look := paints()
	for _, c := range []struct {
		mix  float64
		want Biome
	}{{0, Grass}, {0.5, Grass}, {1.1, Grass}, {2, Forest}} {
		got := w.Fine(look, 0, 0, c.mix).At(maths.Vec{})
		if got != look.Paints[c.want] {
			t.Errorf("a mix of %v is %v, want the paint of %d", c.mix, got, c.want)
		}
	}
}

// the mix over a sphere, worked from the noise itself: a fine tile of a world
// on the rung 3 below its own takes its coarse tile's class, or its kin's if
// the noise at its middle, scaled by 0.15 over the fine spacing, is over 0.55 -
// 0.5 mix. 3000 points at four mixes.
//
// a mix of zero never takes the kin, though some noise there is over 0.55,
// which is counted, and the others do for some tiles and not others.
func TestAMixIsTheNoiseOverALine(t *testing.T) {
	w := New(Terrain{Tiling: hexes.Sphere{N: 3}, Seed: 5, Ocean: 0.5, Plates: 5})
	look := paints()
	fine := w.Tiling.Finer().Finer().Finer()
	state := uint64(3)
	for _, mix := range []float64{0, 0.3, 0.8, 1} {
		tx := w.Fine(look, 3, 0, mix)
		over, took, kept := 0, 0, 0
		for n := 0; n < 3000; n++ {
			var v [3]float64
			for k := range v {
				state = state*6364136223846793005 + 1442695040888963407
				v[k] = float64(state>>11)/(1<<53)*2 - 1
			}
			p := maths.V(v[0], v[1], v[2]).Unit()
			c := fine.Centre(fine.Nearest(p))
			class := w.Biome(w.Tiling.Nearest(c))
			k := 0.15 / fine.Spacing()
			noisy := noise.Fractal(w.Seed+90, c.X*k, c.Y*k, c.Z*k, 3)
			if noisy > 0.55 {
				over++
			}
			if mix > 0 && noisy > 0.55-0.5*mix {
				class = kin[class]
				took++
			} else {
				kept++
			}
			if got := tx.At(p); got != look.Paints[class] {
				t.Fatalf("mix %v, %v: %v, want the paint of %d", mix, p, got, class)
			}
		}
		if mix == 0 && over == 0 || mix > 0 && (took == 0 || kept == 0) {
			t.Errorf("mix %v: %d over 0.55, %d took the kin, %d kept the class",
				mix, over, took, kept)
		}
	}
}
