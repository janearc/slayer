package cities

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// What a vaporwave tower is made of up close, said once. Its marble is
// veined: thin dark waves that wander through the stone, in a pastel of
// the stone's own colour. Its glass block is square blocks in pale joints,
// each brighter in its middle where the glass bulges toward the eye and
// darker toward its rim, in the pastel of the band it stands over. Both
// fade to their average once a block or a vein is smaller than a pixel.

// the numbers marble and glass block are drawn with
const (
	// veinLean is how far a vein runs up for each of its length along,
	// so that veins cross a panel slantwise. picked: a slab cut on a
	// slant.
	veinLean = 0.6
	// veinOctaves is how many scales of noise a vein wanders by. picked:
	// a vein that wanders, and wanders again on a smaller scale.
	veinOctaves = 4
	// bulgeMiddle is where in a block, as a share of it, its middle is.
	bulgeMiddle = 0.5
)

// marbled is veined marble: stone of a colour with veins of another, each
// a wave through the stone so many metres apart, as wide as a share of
// that, wandered by noise as far as the turbulence says.
type marbled struct {
	Stone, Vein              maths.Light
	Apart, Width, Turbulence float64
	Seed                     uint64
}

// At is the marble at p: the vein's colour where a vein runs, softening
// into the stone either side of it.
func (marble marbled) At(p maths.Vec) maths.Light {
	return mixed(marble.Stone, marble.Vein, marble.vein(p))
}

// Within is the marble at p for a pixel footprint wide: veins narrower
// than the pixel fade to the share of the stone they cover.
func (marble marbled) Within(p maths.Vec, footprint float64) maths.Light {
	shown := texture.Detail(footprint, marble.Apart*marble.Width)
	average := mixed(marble.Stone, marble.Vein, marble.Width/2)
	return marble.At(p).Scale(shown).Add(average.Scale(1 - shown))
}

// vein is how much of a vein lies at p, from nought in clear stone to one
// on the vein's middle line.
func (marble marbled) vein(p maths.Vec) float64 {
	if !(marble.Apart > 0) || !(marble.Width > 0) {
		return 0
	}
	at := p.Scale(1 / marble.Apart)
	wander := noise.Fractal(marble.Seed, at.X, at.Y, at.Z, veinOctaves)
	wave := math.Abs(math.Sin(math.Pi * (at.X + veinLean*at.Y +
		marble.Turbulence*wander)))
	near := math.Max(0, 1-wave/marble.Width)
	return near * near
}

// mixed is from, and to as far as share of the way.
func mixed(from, to maths.Light, share float64) maths.Light {
	return from.Scale(1 - share).Add(to.Scale(share))
}

// glassBlock is a wall of glass blocks seen from outside: square blocks a
// side across in the surface's x and y, each set in a joint of a width,
// brighter at its middle and darker toward its rim by the rim's share.
type glassBlock struct {
	Glass, Joint     maths.Light
	Side, Width, Rim float64
}

// At is the glass block at p: the joint's colour in a joint, and the
// glass, dimmed toward each block's rim, elsewhere.
func (block glassBlock) At(p maths.Vec) maths.Light {
	if !(block.Side > 0) {
		return block.Glass
	}
	x := p.X/block.Side - math.Floor(p.X/block.Side)
	y := p.Y/block.Side - math.Floor(p.Y/block.Side)
	if min(x, 1-x, y, 1-y)*block.Side < block.Width/2 {
		return block.Joint
	}
	away := math.Hypot(x-bulgeMiddle, y-bulgeMiddle) / math.Hypot(
		bulgeMiddle, bulgeMiddle)
	return block.Glass.Scale(1 - block.Rim*away*away)
}

// Within is the glass block at p for a pixel footprint wide: once a block
// is smaller than the pixel it fades to the average of glass and joint.
func (block glassBlock) Within(p maths.Vec, footprint float64) maths.Light {
	shown := texture.Detail(footprint, block.Side)
	joint := math.Min(1, block.Width/math.Max(block.Side, tinyBlock))
	average := mixed(block.Glass.Scale(1-block.Rim/2), block.Joint, joint)
	return block.At(p).Scale(shown).Add(average.Scale(1 - shown))
}

// tinyBlock keeps a block's side from being nought where it is divided by.
const tinyBlock = 1e-9

// vaporised is a vaporwave tower's skin up close: its marble and its
// pastel bands veined, and each of its steps of glass block laid in
// blocks in joints of its marble.
func (tower *Tower) vaporised(skin *skinLooks) {
	if skin.stone == nil {
		return
	}
	stone := skin.stone
	skin.stone = tower.veined(stone)
	for index, step := range skin.steps {
		skin.steps[index] = tower.veined(step)
	}
	for index, block := range skin.blocks {
		skin.blocks[index] = blocked(block, stone, tower.drawing)
	}
}

// veined is a material of marble of its own colour, with veins in a
// darker tone of it.
func (tower *Tower) veined(base *thing.Material) *thing.Material {
	drawing := tower.drawing
	return laid(base, "veined", marbled{Stone: base.Colour,
		Vein:       base.Colour.Scale(drawing.fixed("marble-vein-tone")),
		Apart:      drawing.fixed("marble-vein-apart"),
		Width:      drawing.fixed("marble-vein-width"),
		Turbulence: drawing.fixed("marble-vein-turbulence"),
		Seed:       uint64(drawing.seed) ^ veinSalt})
}

// blocked is a step of glass block laid in blocks, its joints of the
// marble it stands in.
func blocked(block, marble *thing.Material, drawing *drawer) *thing.Material {
	return laid(block, "blocks", glassBlock{Glass: block.Colour,
		Joint: marble.Colour.Scale(drawing.fixed("glass-block-joint-tone")),
		Side:  drawing.fixed("glass-block-side"),
		Width: drawing.fixed("glass-block-joint"),
		Rim:   drawing.fixed("glass-block-rim")})
}
