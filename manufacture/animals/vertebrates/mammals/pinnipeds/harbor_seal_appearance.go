package pinnipeds

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/texture"
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

// harborSealPelt is the checked harbor-seal profile's picked coat.
var harborSealPelt = peltFromProfile(defaultPinniped("harbor-seal").Loft.Pelt)

func peltFromProfile(p loftPelt) sealPelt {
	light := func(c [3]float64) maths.Light { return maths.RGB(c[0], c[1], c[2]) }
	return sealPelt{back: light(p.Back), belly: light(p.Belly), mark: light(p.Mark),
		flipper: light(p.Flipper), pad: light(p.Pad), backLevel: p.BackLevel,
		bellyLevel: p.BellyLevel, depth: p.Depth, ringLight: p.RingLight}
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

// HarborSealCoat paints a light-phase spotted coat in rest coordinates.
// The colour pattern is chosen for the shot and repeatable by seed.
func HarborSealCoat(seed uint64) texture.Texture { return sealCoat{seed: seed} }

type sealCoat struct{ seed uint64 }

// At is the coat's colour at a point, with no footprint to average over.
func (coat sealCoat) At(point maths.Vec) maths.Light {
	return coat.Within(point, 0)
}

// ground is the coat's colour under its marks, dark on the back and
// light on the belly with a broad mottling, and how far the point lies
// on the light side, from none to one.
func (coat sealCoat) ground(point maths.Vec) (maths.Light, float64) {
	lightSide := 1 - maths.Smooth(-.16, .19, point.Y)
	mottling := noise.Gradient(coat.seed+1, point.X*5, point.Y*7, point.Z*7)
	back := maths.RGB(.24, .235, .205)
	belly := maths.RGB(.52, .485, .410)
	return back.Scale(1 - lightSide).Add(belly.Scale(lightSide)).
		Scale(1 + .26*mottling), lightSide
}

// sealMarkColour is the colour of the coat's marks at their darkest.
var sealMarkColour = maths.RGB(.032, .030, .023)

// sealBellyMarkFade is how much fainter a mark is on the light belly.
const sealBellyMarkFade = .29

// Within is the coat's colour at a point, averaged over a footprint in
// metres: the ground, with spots cut over it from a warped cell pattern.
func (coat sealCoat) Within(point maths.Vec, footprint float64) maths.Light {
	base, lightSide := coat.ground(point)

	// Work in centimetre-scale cells, with a slowly varying domain warp
	// to break up repeated spot outlines and mix blotches with speckles.
	pattern := coat.pattern(point)
	speckle := maths.Smooth(.24, .59,
		noise.Gradient(coat.seed+3, point.X*94, point.Y*94, point.Z*94))
	speckle *= 1 - maths.Smooth(.0025, .012, footprint)
	mark := math.Max(coat.spots(pattern, footprint, 1), .46*speckle)
	mark *= 1 - sealBellyMarkFade*lightSide
	return base.Scale(1 - mark).Add(sealMarkColour.Scale(mark))
}

// pattern is the point in spot-cell coordinates, warped by a slowly
// varying domain warp.
func (coat sealCoat) pattern(point maths.Vec) maths.Vec {
	warp := noise.Gradient(coat.seed+2, point.X*48, point.Y*48, point.Z*48)
	return maths.V(point.X*16+warp*.27, point.Y*21+warp*.21, point.Z*21-warp*.24)
}

// spots is how far a point lies in a spot, with every spot's radius
// multiplied by grow. A grow above one gives the spots' halos.
func (coat sealCoat) spots(at maths.Vec, footprint, grow float64) float64 {
	x, y, z := math.Floor(at.X), math.Floor(at.Y), math.Floor(at.Z)
	fade := .043 + math.Min(.32, footprint*11)
	var mark float64
	const span = 3
	for dx := range span {
		for dy := range span {
			for dz := range span {
				left, below, behind := x+float64(dx-1), y+float64(dy-1), z+float64(dz-1)
				// All spot centres stay inside the central 70% of the cell.
				if coatCellGap(at.X, left) > .76 || coatCellGap(at.Y, below) > .76 ||
					coatCellGap(at.Z, behind) > .76 {
					continue
				}
				mark = math.Max(mark, coat.cellSpot(at, left, below, behind, fade, grow))
			}
		}
	}
	return mark
}

// coatCellGap is how far a coordinate lies outside the middle of its cell,
// between .15 and .85 along from the corner: negative inside it.
func coatCellGap(coordinate, corner float64) float64 {
	return math.Max(corner+.15-coordinate, coordinate-corner-.85)
}

// cellSpot is the strength at a point of the spot its cell holds: the
// centre and radius come from the cell's hash, the radius grown by grow and
// the edge softened by fade.
func (coat sealCoat) cellSpot(at maths.Vec, x, y, z, fade, grow float64) float64 {
	i, j, k := int64(x), int64(y), int64(z)
	radius := grow * (.22 + .21*noise.Hash3(coat.seed+4, i, j, k))
	centre := maths.V(x+.15+.70*noise.Hash3(coat.seed+5, i, j, k),
		y+.15+.70*noise.Hash3(coat.seed+6, i, j, k),
		z+.15+.70*noise.Hash3(coat.seed+7, i, j, k))
	distance := at.Sub(centre).Len()
	return 1 - maths.Smooth(radius-fade, radius+fade, distance)
}

// The harbor seal's coat pattern by rule: "many fine to medium sized
// spots, usually with smaller numbers of ring-like markings and blotches"
// (National Museum of Nature and Science, Tokyo, Marine Mammals
// Information Database, harbor seal).
//
// Space is cut into cells; a cell holds at most one mark, a spot, a ring
// or a blotch, more often on the back than the belly. Every number below
// is chosen.
const (
	sealSpotCell      = .03  // a cell's edge, in metres
	sealRingShare     = .14  // of marked cells, the share holding a ring
	sealBlotchShare   = .08  // and the share holding a blotch
	sealFineRadius    = .004 // the smallest spot's radius, in metres
	sealMediumRadius  = .011 // the largest spot's
	sealRingRadius    = .012 // a ring's middle radius, in metres
	sealRingWidth     = .003 // a ring's half width
	sealBlotchRadius  = .015 // a blotch's lobes' radius, in metres
	sealBackMarked    = .85  // the share of cells marked on the back
	sealBellyMarked   = .35  // and on the belly
	sealBackFrom      = -.10 // the rest height where the back's density starts
	sealBackTo        = .12  // and where it is whole
	sealMarkEdge      = .0008
	sealCellMargin    = .25 // a mark's centre keeps this share from its cell's walls
	sealRingLightened = .6  // how much a ring's middle lightens the coat
)

// The kinds of mark a cell holds.
const (
	sealMarkSpot = iota
	sealMarkRing
	sealMarkBlotch
)

// sealMarks is how dark the coat's marks make a rest point, and how much
// a ring's light middle lifts it, each from none to one. Seen from far,
// where a pixel covers several cells, the marks blend to their mean.
func sealMarks(seed uint64, point maths.Vec, footprint float64) (float64,
	float64,
) {
	cell := point.Scale(1 / sealSpotCell)
	x, y, z := math.Floor(cell.X), math.Floor(cell.Y), math.Floor(cell.Z)
	edge := math.Max(sealMarkEdge, footprint)
	density := sealBellyMarked + (sealBackMarked-sealBellyMarked)*
		maths.Smooth(sealBackFrom, sealBackTo, point.Y)
	var dark, light float64
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				mark, lift := sealCellMark(seed, point, x+float64(dx),
					y+float64(dy), z+float64(dz), density, edge)
				dark, light = math.Max(dark, mark), math.Max(light, lift)
			}
		}
	}
	far := maths.Smooth(sealSpotCell/4, sealSpotCell, footprint)
	mean := density * sealMeanCover
	return dark*(1-far) + mean*far, light * (1 - far)
}

// sealMeanCover is the share of a marked cell's surface its mark darkens
// on average, for the far blend; worked from the radii above as a disc's
// area over a cell face, rounded. Chosen.
const sealMeanCover = .12

// sealCellMark is the mark of the cell with lowest corner x, y, z, at a
// point: its darkness and its light middle.
func sealCellMark(seed uint64, point maths.Vec, x, y, z, density,
	edge float64,
) (float64, float64) {
	i, j, k := int64(x), int64(y), int64(z)
	if noise.Hash3(seed+11, i, j, k) > density {
		return 0, 0
	}
	place := func(salt uint64) float64 {
		return sealCellMargin + (1-2*sealCellMargin)*noise.Hash3(seed+salt, i, j, k)
	}
	centre := maths.V(x+place(12), y+place(13), z+place(14)).Scale(sealSpotCell)
	distance := point.Sub(centre).Len()
	size := noise.Hash3(seed+15, i, j, k)
	switch sealMarkKind(noise.Hash3(seed+16, i, j, k)) {
	case sealMarkRing:
		radius := sealRingRadius * (.8 + .4*size)
		band := 1 - maths.Smooth(sealRingWidth-edge, sealRingWidth+edge,
			math.Abs(distance-radius))
		inside := 1 - maths.Smooth(radius-sealRingWidth-edge, radius-sealRingWidth,
			distance)
		return band, inside * sealRingLightened
	case sealMarkBlotch:
		return sealBlotchMark(seed, point, centre, i, j, k, edge), 0
	default:
		radius := sealFineRadius + (sealMediumRadius-sealFineRadius)*size*size
		return 1 - maths.Smooth(radius-edge, radius+edge, distance), 0
	}
}

// sealMarkKind is which kind of mark a share from nought to one picks.
func sealMarkKind(share float64) int {
	switch {
	case share < sealRingShare:
		return sealMarkRing
	case share < sealRingShare+sealBlotchShare:
		return sealMarkBlotch
	default:
		return sealMarkSpot
	}
}

// sealBlotchMark is a blotch: two overlapping lobes about the centre, at
// a turn and spread the cell's hashes pick.
func sealBlotchMark(seed uint64, point, centre maths.Vec, i, j, k int64,
	edge float64,
) float64 {
	turn := 2 * math.Pi * noise.Hash3(seed+17, i, j, k)
	spread := sealBlotchRadius * (.4 + .4*noise.Hash3(seed+18, i, j, k))
	offset := maths.V(math.Cos(turn), math.Sin(turn), math.Cos(2*turn)).
		Unit().Scale(spread / 2)
	var mark float64
	for _, lobe := range [...]maths.Vec{centre.Add(offset), centre.Sub(offset)} {
		distance := point.Sub(lobe).Len()
		mark = math.Max(mark, 1-maths.Smooth(sealBlotchRadius*.7-edge,
			sealBlotchRadius*.7+edge, distance))
	}
	return mark
}

// The pelage is the coat as one material: sealMarks's spots, rings and
// blotches on a dark back and light belly, hair streaks along the flow,
// a bare dark nose pad, and a wet coat's darkening and sheen. Every
// number here is chosen.
const (
	// sealHaloLight is how strongly a ring's light middle lightens the
	// coat.
	sealHaloLight = .55
	// sealStreakAlong and sealStreakAcross are how many streak cells lie
	// to the metre along and across the flow, and sealStreakStrength how
	// far streaks darken and lighten the coat.
	sealStreakAlong, sealStreakAcross, sealStreakStrength = 60., 700., .22
	// sealStreakNear and sealStreakFar are the pixel footprints, in
	// metres, between which streaks fade out so they never glitter.
	sealStreakNear, sealStreakFar = .0006, .003
	// sealWetDarken is how much of its colour a soaked coat loses.
	sealWetDarken = .77
	// sealUnderskin is the share of the coat's colour the skin under the
	// hair keeps.
	sealUnderskin = .5
)

var (
	// sealNoseColour is the bare nose pad.
	sealNoseColour = maths.RGB(.014, .012, .011)
	// sealHaloColour is the light the halo ring round a spot adds.
	sealHaloColour = maths.RGB(.30, .28, .24)
	// sealWetFloor is the colour a soaked coat keeps from the water on
	// its hair.
	sealWetFloor = maths.RGB(.021, .019, .016)
)

// sealPelage is the texture of a seal's coat, read at rest coordinates.
type sealPelage struct {
	coat sealCoat
	flow sealFlow
	wet  float64
}

// Pelage is the seal's coat as one texture over its rest skin, dry at
// look.Wet of nought and soaked at one.
func (rig *SealRig) Pelage(look SealLook) texture.Texture {
	return sealPelage{coat: sealCoat{seed: look.Seed}, flow: rig.flowField(), wet: look.Wet}
}

// At is the pelage's colour at a point, with no footprint.
func (pelage sealPelage) At(point maths.Vec) maths.Light {
	return pelage.Within(point, 0)
}

// Within is the pelage's colour at a point over a footprint in metres: the
// ground, a halo and the marks cut over it, the marks fading on the belly.
func (pelage sealPelage) Within(point maths.Vec, footprint float64) maths.Light {
	base, lightSide := pelage.coat.ground(point)
	mark, lift := sealMarks(pelage.coat.seed, point, footprint)
	mark *= 1 - sealBellyMarkFade*lightSide
	dry := base.Add(sealHaloColour.Scale(sealHaloLight * lift))
	dry = dry.Scale(1 - mark).Add(sealMarkColour.Scale(mark))
	dry = dry.Scale(1 + pelage.streak(point, footprint))
	coat := dry.Scale(1 - sealWetDarken*pelage.wet).
		Add(sealWetFloor.Scale(pelage.wet))
	nose := sealNose(point)
	return coat.Scale(1 - nose).Add(sealNoseColour.Scale(nose))
}

// streak is the lightening or darkening of hair streaks at a point:
// noise stretched along the coat's flow and thin across it.
func (pelage sealPelage) streak(point maths.Vec, footprint float64) float64 {
	flow := sealFrame(pelage.flow.at(point), maths.Up)
	value := noise.Gradient(pelage.coat.seed+9, point.Dot(flow.Z)*sealStreakAlong,
		point.Dot(flow.X)*sealStreakAcross, point.Dot(flow.Y)*sealStreakAcross)
	return sealStreakStrength * value *
		(1 - maths.Smooth(sealStreakNear, sealStreakFar, footprint))
}

// sealNose is how far a rest point lies in the bare nose pad, from none
// to one. The pad is an oval at the front of the muzzle.
func sealNose(point maths.Vec) float64 {
	width := .028 + .024*maths.Smooth(.023, .077, point.Y)
	pad := 1 - maths.Smooth(.76, 1.02, math.Hypot((point.Y-.059)/.038, point.Z/width))
	return pad * maths.Smooth(.779, .804, point.X)
}

// sealDimmed is a texture at a share of its colour.
type sealDimmed struct {
	inner texture.Texture
	share float64
}

// At is the dimmed texture's colour at a point, with no footprint.
func (dim sealDimmed) At(point maths.Vec) maths.Light { return dim.Within(point, 0) }

// Within is the inner texture sampled over a footprint and scaled by the
// dimmer's share.
func (dim sealDimmed) Within(point maths.Vec, footprint float64) maths.Light {
	return texture.Sample(dim.inner, point, footprint).Scale(dim.share)
}

// Materials are the seal's skin and hair materials for a look: the skin
// under the hair is dimmer, a wet coat glosses and a dry one does not.
func (rig *SealRig) Materials(look SealLook) (skin, hair *thing.Material) {
	pelage := rig.Pelage(look)
	gloss := .05 + .37*look.Wet
	satin := .18 + .15*look.Wet
	skin = &thing.Material{Name: "harbor seal skin", Texture: sealDimmed{pelage, sealUnderskin},
		Gloss: gloss, Satin: satin, Index: 1.43}
	hair = &thing.Material{Name: "harbor seal hair", Texture: pelage,
		Gloss: gloss * .7, Satin: satin, Index: 1.43}
	return skin, hair
}

// A lofted seal's face. The eyes are large, dark and set forward; the
// nostrils are "small and terminal, forming a "V" that converges at the
// bottom" (Duke OBIS-SEAMAP, harbor seal species page). Places and sizes
// in metres at rest, picked.
const (
	// An eye's ball: the station it sits at, how far round from the top
	// of the head, its radius, and how deep its centre lies in the skin.
	sealFaceEyeX, sealFaceEyeTurn     = .756, .88
	sealFaceEyeRadius, sealFaceEyeSet = .021, .011
	// The dark skin round an eye: whole within the first distance of the
	// eye's place on the skin and gone by the second; its strength.
	sealFaceRimFrom, sealFaceRimTo, sealFaceRim = .021, .036, .6
	// An ear is a small dark opening, with no flap, "set slightly behind
	// and below the eyes" (the same page): its station, its turn from
	// the top, its radius and its strength.
	sealFaceEarX, sealFaceEarTurn    = .716, 1.12
	sealFaceEarWide, sealFaceEarDark = .006, .75
	// A nostril is a slit on the nose's front from its upper, outer end
	// to its lower, inner end, as heights over the nose's centre and
	// distances from the middle; its half width and strength.
	sealFaceSlitFrom                   = .800
	sealFaceSlitHigh, sealFaceSlitOut  = .012, .016
	sealFaceSlitLow, sealFaceSlitIn    = -.005, .005
	sealFaceSlitWide, sealFaceSlitDark = .0032, .9
	// The mouth's line runs back along the muzzle from under the nose,
	// this share of the section's half height under its centre.
	sealFaceMouthFrom, sealFaceMouthTo  = .765, .826
	sealFaceMouthLow, sealFaceMouthWide = .55, .0022
	sealFaceMouthDark                   = .6
)

// eye is where an eye's ball meets the outline's skin at rest, and the
// way out of the head there; side is one for the right eye.
func (outline sealOutline) eye(side float64) (onSkin, out maths.Vec) {
	return outline.radial(sealFaceEyeX, side*sealFaceEyeTurn)
}

// sealFaceInk is how far the face's dark lines blacken the coat at a
// rest point of the outline's skin, from none to one: the skin round
// the eyes, the ears' openings, the nostrils' slits and the line of
// the mouth.
func sealFaceInk(outline sealOutline, point maths.Vec,
	section sealSection,
) float64 {
	if point.X < sealFaceEarX-sealFaceRimTo {
		return 0
	}
	side := 1.
	if point.Z < 0 {
		side = -1
	}
	eye, _ := outline.eye(side)
	ink := sealFaceRim * (1 - maths.Smooth(sealFaceRimFrom, sealFaceRimTo,
		point.Sub(eye).Len()))
	ear, _ := outline.radial(sealFaceEarX, side*sealFaceEarTurn)
	ink = math.Max(ink, sealFaceEarDark*(1-maths.Smooth(sealFaceEarWide/2,
		sealFaceEarWide, point.Sub(ear).Len())))
	return math.Max(ink, math.Max(sealFaceSlit(point, section),
		sealFaceMouth(point, section)))
}

// sealFaceSlit is how far a nostril's slit blackens a rest point.
func sealFaceSlit(point maths.Vec, section sealSection) float64 {
	if point.X < sealFaceSlitFrom {
		return 0
	}
	height, out := point.Y-section.y, math.Abs(point.Z)
	alongY := sealFaceSlitLow - sealFaceSlitHigh
	alongZ := sealFaceSlitIn - sealFaceSlitOut
	share := ((height-sealFaceSlitHigh)*alongY +
		(out-sealFaceSlitOut)*alongZ) / (alongY*alongY + alongZ*alongZ)
	share = math.Min(math.Max(share, 0), 1)
	gap := math.Hypot(height-sealFaceSlitHigh-share*alongY,
		out-sealFaceSlitOut-share*alongZ)
	return sealFaceSlitDark *
		(1 - maths.Smooth(sealFaceSlitWide/2, sealFaceSlitWide, gap))
}

// sealFaceMouth is how far the mouth's line blackens a rest point.
func sealFaceMouth(point maths.Vec, section sealSection) float64 {
	if point.X < sealFaceMouthFrom || point.X > sealFaceMouthTo {
		return 0
	}
	line := section.y - sealFaceMouthLow*section.height
	return sealFaceMouthDark * (1 - maths.Smooth(sealFaceMouthWide/2,
		sealFaceMouthWide, math.Abs(point.Y-line)))
}

// The whiskers of the muzzle: "~44 whiskers in seven rows" (Graff and
// others 2024, Journal of Experimental Biology 227(22), Europe PMC
// PMC11634035), read as one side's. Each row's count, the places,
// lengths, radii and the droop are picked.
var sealBristleRows = [...]int{7, 8, 7, 7, 6, 5, 4}

const (
	// The pad: the rearmost column's station and the gap to the next,
	// the lowest row's turn from the top of the muzzle and the turn up
	// to the next row.
	sealBristleX, sealBristleStep    = .768, .0052
	sealBristleTurn, sealBristleRise = 1.98, .115
	// The longest strand, at the rear of the lowest row, and how much
	// shorter each column forward and each row up is.
	sealBristleLong                = .072
	sealBristleLess, sealBristleUp = .0050, .0040
	sealBristleLeast               = .018
	// A strand leaves the skin along its normal, leaning along the body
	// by sealBristleAft at the rearmost column and by sealBristleFore
	// eight columns on, so the pad's strands fan; it droops this share
	// of its length, and is drawn as this many rods of this radius.
	sealBristleAft, sealBristleFore = -.50, .25
	sealBristleDroop                = .12
	sealBristleRods                 = 4
	sealBristleRadius               = .00025
	sealBristleColumns              = 8
)

// bristles is one side's whiskers on the outline's muzzle at rest, each
// a line of points from its root on the skin to its tip.
func (outline sealOutline) bristles(side float64) [][]maths.Vec {
	var strands [][]maths.Vec
	for row, count := range sealBristleRows {
		turn := side * (sealBristleTurn - sealBristleRise*float64(row))
		for column := range count {
			root, out := outline.radial(sealBristleX+
				sealBristleStep*float64(column), turn)
			long := math.Max(sealBristleLeast, sealBristleLong-
				sealBristleLess*float64(column)-sealBristleUp*float64(row))
			lean := sealBristleAft + (sealBristleFore-sealBristleAft)*
				float64(column)/sealBristleColumns
			way := out.Add(maths.V(lean, 0, 0)).Unit()
			strand := make([]maths.Vec, sealBristleRods+1)
			for at := range strand {
				share := float64(at) / sealBristleRods
				strand[at] = root.Add(way.Scale(long * share)).Add(
					maths.V(0, -sealBristleDroop*long*share*share, 0))
			}
			strands = append(strands, strand)
		}
	}
	return strands
}

var (
	// sealFaceEyeLook is the eye: near black and wet. Picked.
	sealFaceEyeLook = &thing.Material{Name: "seal eye",
		Colour: maths.RGB(.010, .008, .007), Gloss: .6, Satin: .12}
	// sealBristleLook is a pale whisker. Picked.
	sealBristleLook = &thing.Material{Name: "seal whisker",
		Colour: maths.RGB(.46, .43, .37), Gloss: .15, Satin: .4}
)

// face is the eyes and the whiskers of both sides of the outline's
// head. place takes a rest point of the head to where it ends up, and
// scale is the animal's size over the rest space's.
func (outline sealOutline) face(place func(maths.Vec) maths.Vec,
	scale float64,
) []thing.Thing {
	var parts []thing.Thing
	for _, side := range [...]float64{1, -1} {
		onSkin, out := outline.eye(side)
		centre := place(onSkin.Sub(out.Scale(sealFaceEyeSet)))
		parts = append(parts, thing.Thing{
			Shape:    shape.Sphere{Radius: scale * sealFaceEyeRadius},
			Material: sealFaceEyeLook,
			Motion: thing.Still(maths.NewSubspace(centre,
				maths.Identity, 1))})
		for _, strand := range outline.bristles(side) {
			parts = append(parts, sealBristleRodsOf(strand, place,
				scale*sealBristleRadius)...)
		}
	}
	return parts
}

// sealBristleRodsOf is one whisker as rods of a radius, end to end
// through its placed points.
func sealBristleRodsOf(strand []maths.Vec,
	place func(maths.Vec) maths.Vec, radius float64,
) []thing.Thing {
	var rods []thing.Thing
	previous := place(strand[0])
	for _, rest := range strand[1:] {
		at := place(rest)
		run := at.Sub(previous)
		rods = append(rods, thing.Thing{
			Shape: shape.Tube{Radius: radius, Thick: 2 * radius,
				Height: run.Len()},
			Material: sealBristleLook,
			Motion: thing.Still(maths.NewSubspace(
				previous.Add(at).Scale(.5), maths.Upright(run), 1))})
		previous = at
	}
	return rods
}
