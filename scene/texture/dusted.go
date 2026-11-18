package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers Dusted is drawn with
const (
	// dustOctaves is how many scales of noise the patches have. picked:
	// patches, and the ragged edges of patches.
	dustOctaves = 3
	// dustSoft is how far, in the noise's own units, a patch's edge fades
	// from rock to dust. picked: soft enough not to show a pixel's stair.
	dustSoft = 0.15
	// dustRange is about how far either side of nought the noise runs.
	// picked: what three octaves of gradient noise mostly stay within.
	dustRange = 0.6
)

// Dusted is rock with fine dust lying on it in patches. A point is rock,
// dust, or between, by smooth noise in the thing's own space, so the
// patches lie across a stone's faces as dust settles, and their edges are
// ragged. Share is about how much of the surface the dust covers, nought
// to one; Patch is about how wide a patch is, in the thing's own units;
// Seed picks which patches.
type Dusted struct {
	Rock, Dust maths.Light
	Share      float64
	Patch      float64
	Seed       uint64
}

// At is the surface at p: the rock, the dust, or a blend of the two at a
// patch's edge.
func (dusted Dusted) At(p maths.Vec) maths.Light {
	cover := dusted.cover(p)
	bare := 1 - cover // defined: what the dust leaves
	return dusted.Rock.Scale(bare).Add(dusted.Dust.Scale(cover))
}

// cover is how much dust lies at p, from nought to one: none where the
// noise is below the level that leaves Share of the surface above it, all
// past that level by dustSoft, and a smooth blend between. No share is no
// dust; patches of no width are an even film of Share everywhere.
func (dusted Dusted) cover(p maths.Vec) float64 {
	if !(dusted.Share > 0) { // defined: no dust asked
		return 0 // defined: none
	}
	if !(dusted.Patch > 0) { // defined: no width
		return math.Min(1, math.Max(0, dusted.Share)) // defined: a share
	}
	at := p.Scale(1 / dusted.Patch) // defined: in patches
	wave := noise.Fractal(dusted.Seed, at.X, at.Y, at.Z, dustOctaves)
	level := dustRange * (1 - 2*dusted.Share)                  // defined: half at nought
	through := math.Min(1, math.Max(0, (wave-level)/dustSoft)) // defined
	return through * through * (3 - 2*through)                 // defined: smoothstep
}
