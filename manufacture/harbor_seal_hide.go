package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// sealPelt is a seal's coat as data: the colours of its back, its
// belly, its marks, its flippers and its bare nose pad; the level the
// marks' field must pass on the back and on the belly for a point to be
// marked, lower so more of it is; how dark a mark is, nought for a
// plain coat; and how far a pale ring round a mark lightens the coat.
type sealPelt struct {
	back, belly, mark, flipper, pad maths.Light
	backLevel, bellyLevel           float64
	depth, ringLight                float64
}

// harborSealPelt is the harbor seal's coat: a dark back shading to a
// pale belly, and over it dark marks cut from a wandering field, so no
// two are one size or shape and near ones run together, "many fine to
// medium sized spots, usually with smaller numbers of ring-like
// markings and blotches" (National Museum of Nature and Science, Tokyo,
// Marine Mammals Information Database, harbor seal).
//
// Colours and levels
// picked.
var harborSealPelt = sealPelt{
	back:      maths.RGB(.115, .108, .098),
	belly:     maths.RGB(.420, .400, .360),
	mark:      maths.RGB(.030, .028, .025),
	flipper:   maths.RGB(.085, .080, .074),
	pad:       maths.RGB(.022, .020, .019),
	backLevel: .17, bellyLevel: .37, depth: .92, ringLight: .40,
}

// How any pelt is laid over the rest skin. Every number is picked.
const (
	// The back's shade runs from the belly's at sealHideLow to the
	// back's at sealHideHigh, heights in half heights of the section
	// from its centre, wandering by sealHideWander at sealHideWanders
	// waves to the metre.
	sealHideLow, sealHideHigh       = -.60, .45
	sealHideWander, sealHideWanders = .22, 6.
	// The marks' field has sealHideMarks waves to the metre and a finer
	// one of sealHideFineShare its strength at sealHideFine, both
	// warped; a mark's edge is sealHideEdge of the field wide. A ring
	// lies between two levels under the mark's, where a slow field
	// allows it.
	sealHideMarks, sealHideFine         = 58., 150.
	sealHideFineShare                   = .22
	sealHideEdge                        = .085
	sealHideWarp, sealHideWarps         = .012, 9.
	sealHideRingFrom, sealHideRingTo    = .17, .07
	sealHideRings                       = 5.
	sealHideRingStart, sealHideRingFull = .05, .25
	// A flipper is where a rest point lies this far outside the
	// outline's section, in the section's own radii, or behind the
	// ankles; it is darker by sealHideFlipperShare, and a limb's own
	// skin by sealHideLimbShare at the least.
	sealHideOutFrom, sealHideOutTo = 1.2, 1.6
	sealHideAftFrom, sealHideAftTo = -.90, -.82
	sealHideFlipperShare           = .65
	sealHideLimbShare              = .45
	// The bare nose pad begins between these X.
	sealHidePadFrom, sealHidePadTo = .826, .832
	// sealHideWetDark is the share of its colour a soaked coat loses.
	sealHideWetDark = .45
)

// sealHide is a pelt as a texture over the rest points of an outline's
// skin, for a seed, dry at wet nought and soaked at one; limb is
// whether it is a flipper's skin, which is darker than the trunk's.
type sealHide struct {
	pelt    sealPelt
	outline sealOutline
	seed    uint64
	wet     float64
	limb    bool
}

// At is the coat's colour at a rest point.
func (hide sealHide) At(point maths.Vec) maths.Light {
	return hide.Within(point, 0)
}

// Within is the coat's colour at a rest point seen through a pixel
// footprint metres wide.
func (hide sealHide) Within(point maths.Vec, footprint float64,
) maths.Light {
	pelt, section := hide.pelt, hide.outline.at(point.X)
	back := hide.back(point, section)
	coat := pelt.belly.Scale(1 - back).Add(pelt.back.Scale(back))
	mark, ring := hide.marks(point, back, footprint)
	coat = coat.Scale(1 + pelt.ringLight*ring)
	coat = coat.Scale(1 - mark).Add(pelt.mark.Scale(mark))
	flipper := sealHideFlipperShare * sealHideOutside(point, section)
	if hide.limb {
		flipper = math.Max(flipper, sealHideLimbShare)
	}
	coat = coat.Scale(1 - flipper).Add(pelt.flipper.Scale(flipper))
	pad := maths.Smooth(sealHidePadFrom, sealHidePadTo, point.X)
	coat = coat.Scale(1 - pad).Add(pelt.pad.Scale(pad))
	return coat.Scale((1 - sealHideWetDark*hide.wet) *
		(1 - sealFaceInk(hide.outline, point, section)))
}

// back is how far a rest point's coat is the back's, from none on the
// belly to one on the back, the border between them wandering.
func (hide sealHide) back(point maths.Vec, section sealSection) float64 {
	height := (point.Y - section.y) / section.height
	wander := sealHideWander * noise.Gradient(hide.seed+1,
		point.X*sealHideWanders, point.Y*sealHideWanders,
		point.Z*sealHideWanders)
	return maths.Smooth(sealHideLow, sealHideHigh, height+wander)
}

// marks is how dark the marks make a rest point, and how far a pale
// ring round a mark lightens it, each from none to one; both none on a
// plain coat.
func (hide sealHide) marks(point maths.Vec, back, footprint float64,
) (mark, ring float64) {
	pelt := hide.pelt
	if pelt.depth == 0 {
		return 0, 0
	}
	field := func(salt uint64, at maths.Vec, waves float64) float64 {
		return noise.Gradient(hide.seed+salt, at.X*waves, at.Y*waves,
			at.Z*waves)
	}
	warped := point.Add(maths.V(field(2, point, sealHideWarps),
		field(3, point, sealHideWarps), field(4, point, sealHideWarps)).
		Scale(sealHideWarp))
	value := field(5, warped, sealHideMarks) +
		sealHideFineShare*field(6, warped, sealHideFine)
	level := pelt.bellyLevel + (pelt.backLevel-pelt.bellyLevel)*back
	edge := math.Max(sealHideEdge, footprint*sealHideMarks)
	mark = maths.Smooth(level-edge, level+edge, value)
	ringed := maths.Smooth(sealHideRingStart, sealHideRingFull,
		field(7, point, sealHideRings))
	ring = ringed * (1 - mark) * maths.Smooth(level-sealHideRingFrom,
		level-sealHideRingTo, value)
	return pelt.depth * mark, ring
}

// sealHideOutside is how far a rest point belongs to a flipper and not
// to the trunk, from none to one.
func sealHideOutside(point maths.Vec, section sealSection) float64 {
	out := math.Hypot((point.Y-section.y)/section.height,
		point.Z/section.width)
	aft := 1 - maths.Smooth(sealHideAftFrom, sealHideAftTo, point.X)
	return math.Max(aft, maths.Smooth(sealHideOutFrom, sealHideOutTo, out))
}

// look is the hide as a material: a damp sheen, broad and soft, wider
// and stronger the wetter the coat. Picked.
func (hide sealHide) look() *thing.Material {
	return &thing.Material{Name: "seal hide", Texture: hide,
		Gloss: .04 + .12*hide.wet, Satin: .62 - .14*hide.wet, Index: 1.4}
}
