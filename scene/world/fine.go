package world

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
)

// Fine is the world's surface painted on a rung below its own: every fine tile
// takes the class of the coarse tile it sits in and varies it, by hash in its
// light and by noise in its class, so from orbit a landscape has texture at any
// scale without the world holding a tile.
//
// This is the ladder in use: the coarse world decides, the fine rung
// elaborates. Rungs is how many rungs down; zero lets the picture choose, a
// rung where a tile is about Across pixels wide, by the footprint of the pixel
// asking.
//
// Vary is how much a fine tile's light may differ from its class's paint, from
// zero to one; Mix how readily a fine tile takes a neighbouring class, forest
// in grass, rock in desert, deep in sea.
func (w *World) Fine(look Look, rungs int, vary, mix float64) texture.Texture {
	return &fineness{w: w, look: look, rungs: rungs, vary: vary, mix: mix}
}

// Across is how many pixels wide a tile is, about, when a fine texture
// picks its own rung.
const Across = 6

// fineness is a world's surface on a fine rung, fixed or chosen by the
// pixel asking.
type fineness struct {
	w         *World
	look      Look
	rungs     int
	vary, mix float64
}

// rung is the tiling rungs below the world's own.
func (f *fineness) rung(rungs int) hexes.Tiling {
	fine := f.w.Tiling
	for range max(0, rungs) {
		fine = fine.Finer()
	}
	return fine
}

// tiled is the surface on a tiling, painted and lined.
func (f *fineness) tiled(fine hexes.Tiling) texture.Tiled {
	return texture.Tiled{Tiling: fine, Lines: f.look.Lines,
		Paints: finer{f.w, f.look, fine, f.vary, f.mix}}
}

// At is the paint at p on the fixed rung, or the world's own rung when
// the picture was to choose and there is no picture.
func (f *fineness) At(p maths.Vec) maths.Light {
	return f.tiled(f.rung(f.rungs)).At(p)
}

// Within is the paint at p for a pixel footprint wide: on the fixed
// rung if there is one, else on the rung where a tile is about Across
// footprints across, and never above the world's own.
func (f *fineness) Within(p maths.Vec, footprint float64) maths.Light {
	if f.rungs > 0 || footprint <= 0 {
		return f.At(p)
	}
	want := Across * footprint
	rungs := 0
	for s := f.w.Tiling.Spacing(); s > want && rungs < 40; s /= 2 {
		rungs++
	}
	return f.tiled(f.rung(rungs)).At(p)
}

// finer paints a fine tile from its coarse tile's class.
type finer struct {
	w         *World
	look      Look
	fine      hexes.Tiling
	vary, mix float64
}

// kin is the class a fine tile may take instead of its coarse tile's,
// where the noise says so: the pair a landscape mixes at that scale.
var kin = [Biomes]Biome{
	Deep: Sea, Sea: Deep, Ice: Snow, Beach: Sea, Desert: Rock,
	Grass: Forest, Forest: Grass, Jungle: Forest, Tundra: Rock,
	Rock: Tundra, Snow: Rock, Lava: Ash, Ash: Rock,
}

// Paint is the fine tile's paint: its class's, or its kin's where the
// noise says, lightened or darkened by its own hash.
func (f finer) Paint(tile hexes.Tile) maths.Light {
	c := f.fine.Centre(tile)
	class := f.w.Biome(f.w.Tiling.Nearest(c))
	if f.mix > 0 {
		// the mix's features are a few fine tiles across, whatever the
		// rung, so a strip of a few tiles still has some of both
		k := 0.15 / f.fine.Spacing()
		n := noise.Fractal(f.w.Seed+90, c.X*k, c.Y*k, c.Z*k, 3)
		if n > 0.55-0.5*f.mix {
			class = kin[class]
		}
	}
	paint := f.look.Paints[class]
	if f.vary > 0 {
		h := noise.Hash(f.w.Seed+91, f.fine.ID(tile)) - 0.5
		paint = paint.Scale(1 + f.vary*h)
	}
	return paint
}
