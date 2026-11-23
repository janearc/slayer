package texture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// the numbers Dusted is drawn with, each picked and held in
// picked-numbers.yaml
var (
	// dustOctaves is how many scales of noise the patches have.
	dustOctaves = int(pickedNumber("dust-octaves"))
	// dustSoft is how far, in the noise's own units, a patch's edge fades
	// from rock to dust.
	dustSoft = pickedNumber("dust-soft")
	// dustRange is about how far either side of nought the noise runs.
	dustRange = pickedNumber("dust-range")
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

// At is the surface at the point: the rock, the dust, or a blend of the
// two at a patch's edge.
func (dusted Dusted) At(point maths.Vec) maths.Light {
	cover := dusted.cover(point)
	bare := unity - cover
	return dusted.Rock.Scale(bare).Add(dusted.Dust.Scale(cover))
}

// cover is how much dust lies at the point, from nought to one: none where
// the noise is below the level that leaves Share of the surface above it,
// all past that level by dustSoft, and a smooth blend between. No share is
// no dust; patches of no width are an even film of Share everywhere.
func (dusted Dusted) cover(point maths.Vec) float64 {
	if !(dusted.Share > nought) {
		return nought
	}
	if !(dusted.Patch > nought) {
		return math.Min(unity, math.Max(nought, dusted.Share))
	}
	at := point.Scale(unity / dusted.Patch)
	wave := noise.Fractal(dusted.Seed, at.X, at.Y, at.Z, dustOctaves)
	level := dustRange * (unity - pair*dusted.Share)
	return smoothed((wave - level) / dustSoft)
}
