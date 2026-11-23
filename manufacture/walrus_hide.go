package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
)

// The colours of the hide. "Most walruses are yellowish brown"; out of the
// water "vaso-dilation causes blood and color to return to the skin", which
// turns pinkish to reddish; "Older males have many lumps or nodules on the
// neck and chest" and "most older males become nearly hairless" (all from
// the OBIS-SEAMAP species profile, as read). The reflectances themselves
// are chosen to match those words; none is sampled from a photograph.
var (
	walrusCinnamon    = maths.RGB(.25, .150, .085)
	walrusPale        = maths.RGB(.36, .25, .16)
	walrusBackSkin    = maths.RGB(.15, .09, .055)
	walrusFlipperSkin = maths.RGB(.16, .105, .07)
	walrusFlush       = maths.RGB(.38, .19, .15)
	walrusPadSkin     = maths.RGB(.33, .23, .17)
	walrusScarSkin    = maths.RGB(.48, .38, .32)
	walrusHairDark    = maths.RGB(.10, .06, .045)
	walrusHairPale    = maths.RGB(.62, .50, .36)
	walrusFollicleDot = maths.RGB(.07, .035, .03)
)

// walrusScar is a thin healed wound: the curve where the hide crosses a
// slab of the given width, as far as length from centre. The slab passes
// through centre, or offset from it along normal: two scars with one
// centre and different offsets lie side by side, as a pair of tusks
// leaves them.
type walrusScar struct {
	centre, normal        maths.Vec
	length, width, offset float64
}

// walrusScars are the scars: short ones, the first two a pair. Their
// places and sizes are chosen.
var walrusScars = []walrusScar{
	{maths.V(.72, .36, .10), maths.V(.5, .2, .84), .07, .0026, 0},
	{maths.V(.72, .36, .10), maths.V(.5, .2, .84), .07, .0026, .012},
	{maths.V(.55, .30, -.30), maths.V(.8, .1, -.6), .08, .0030, 0},
	{maths.V(.30, .05, .43), maths.V(.2, .9, .4), .09, .0026, 0},
	{maths.V(-.40, .20, -.40), maths.V(.3, .5, .8), .08, .0024, 0},
	{maths.V(.90, .30, .25), maths.V(.6, .6, -.5), .06, .0024, 0},
	{maths.V(-.90, .10, .43), maths.V(.9, .3, .3), .06, .0028, 0},
}

// walrusScarShare is how much of a scar's colour shows at p, from nought
// to one.
func walrusScarShare(p maths.Vec) float64 {
	share := 0.
	for _, scar := range walrusScars {
		offset := p.Sub(scar.centre)
		if offset.Len() > scar.length {
			continue
		}
		distance := offset.Dot(scar.normal.Unit()) - scar.offset
		share = math.Max(share, 1-maths.Smooth(0, scar.width, math.Abs(distance)))
	}
	return share
}

// walrusMix is the light between two, from the first at nought to the
// second at one.
func walrusMix(first, second maths.Light, share float64) maths.Light {
	return maths.RGB(first.R+(second.R-first.R)*share,
		first.G+(second.G-first.G)*share, first.B+(second.B-first.B)*share)
}

// walrusHair is the short sparse hair at p: nought where the hide is bare,
// minus one for a dark hair, one for a pale hair.
func walrusHair(p maths.Vec) float64 {
	// density is the share of cells with no hair: a bull is nearly
	// hairless.
	const (
		cell      = .011
		density   = .95
		tuftWidth = .0022
	)
	ix, iy, iz := int(math.Floor(p.X/cell)), int(math.Floor(p.Y/cell)),
		int(math.Floor(p.Z/cell))
	if walrusHash(ix, iy, iz, 6) < density {
		return 0
	}
	centre := maths.V((float64(ix)+walrusHash(ix, iy, iz, 7))*cell,
		(float64(iy)+walrusHash(ix, iy, iz, 8))*cell,
		(float64(iz)+walrusHash(ix, iy, iz, 9))*cell)
	if p.Sub(centre).Len() > tuftWidth {
		return 0
	}
	if walrusHash(ix, iy, iz, 10) < .5 {
		return -1
	}
	return 1
}

// walrusBody is the colour of the body by region before any detail: the
// yellowish brown back and flanks, the paler belly, the flush of blood in
// broad patches over the neck, chest and shoulders where the skin is
// thin and the blood comes back out of the water, and the darker bare
// flippers, all broken by slow mottling, a back darker than the flanks,
// and broad darker blotches. Scales and shares chosen.
func walrusBody(p maths.Vec) maths.Light {
	const (
		mottleScale, mottleShare = 2.5, .85
		patchScale, patchShare   = 9., .22
		flushScale, flushShare   = 3.5, .30
		bellyHigh, bellyLow      = -.12, -.40
		backLow, backHigh        = .15, .50
		neckStart, neckEnd       = .50, 1.00
		neckFade, neckGone       = 1.30, 1.45
		bellyShare, backShare    = .70, .45
		blotchScale, blotchShare = 1.6, .50
		blotchLow, blotchHigh    = .45, .75
	)
	pale := walrusMix(walrusCinnamon, walrusPale, mottleShare*walrusNoise(p.Scale(mottleScale)))
	neck := maths.Smooth(neckStart, neckEnd, p.X) * (1 - maths.Smooth(neckFade, neckGone, p.X))
	belly := 1 - maths.Smooth(bellyLow, bellyHigh, p.Y)
	colour := walrusMix(walrusCinnamon, pale, .8)
	colour = walrusMix(colour, walrusPale, bellyShare*belly)
	colour = walrusMix(colour, walrusBackSkin,
		backShare*maths.Smooth(backLow, backHigh, p.Y))
	blotch := maths.Smooth(blotchLow, blotchHigh,
		walrusNoise(p.Scale(blotchScale).Add(maths.V(3, 11, 5))))
	colour = walrusMix(colour, walrusBackSkin, blotchShare*blotch*(1-belly))
	flush := maths.Smooth(.35, .75, walrusNoise(p.Scale(flushScale).Add(maths.V(7, 3, 5))))
	colour = walrusMix(colour, walrusFlush, flushShare*flush*(.35+.65*neck))
	colour = walrusMix(colour, walrusFlipperSkin, walrusLimbShare(p))
	tone := 1 - patchShare/2 + patchShare*walrusNoise(p.Scale(patchScale))
	return maths.RGB(colour.R*tone, colour.G*tone, colour.B*tone)
}

// walrusPadShare is how much of the muzzle pad's skin there is at rest
// point p, from nought to one: in front of the skull's snout, between
// the nostrils and the mouth, as far out as the pad reaches. Chosen.
func walrusPadShare(p maths.Vec) float64 {
	const (
		start, top, bottom, width = .96, -.11, .18, .165
		edge                      = .03
	)
	t, across, drop := walrusFrame.local(p)
	return maths.Smooth(start, start+edge, t) *
		maths.Smooth(top-edge, top, drop) * (1 - maths.Smooth(bottom, bottom+edge, drop)) *
		(1 - maths.Smooth(width, width+edge, math.Abs(across)))
}

// walrusHideTexture is the walrus hide as a texture: a colour for every
// point in the rest pose, by region, read at the rest place the hide's
// mesh carries. It adds scars, the follicle dots on the muzzle pad, and
// sparse short hair. Creases and bosses are not coloured: the hide has no
// fold cut in it.
type walrusHideTexture struct{}

// At is the hide's reflectance at p.
func (walrusHideTexture) At(p maths.Vec) maths.Light {
	const (
		dotWidth, padShare   = .0030, .75
		scarShare, hairShare = .12, .22
	)
	colour := walrusBody(p)
	pad := walrusPadShare(p)
	colour = walrusMix(colour, walrusPadSkin, padShare*pad)
	if pad > .5 && walrusFollicleDistance(p) < dotWidth {
		colour = walrusMix(colour, walrusFollicleDot, 1)
	}
	colour = walrusMix(colour, walrusScarSkin, scarShare*walrusScarShare(p))
	switch hair := walrusHair(p); {
	case hair < 0:
		colour = walrusMix(colour, walrusHairDark, hairShare)
	case hair > 0:
		colour = walrusMix(colour, walrusHairPale, hairShare)
	}
	return colour
}

// walrusIvoryTexture is the colour of a tusk by how far down it a point
// lies: stained and darker where it comes out of the jaw, paler at the
// worn end, with fine lines round it at the growth rings.
type walrusIvoryTexture struct{}

// At is the ivory's reflectance at p.
func (walrusIvoryTexture) At(p maths.Vec) maths.Light {
	const (
		stainTop, stainBottom = .14, -.25
		ringsPerMetre         = 90.
		ringShare             = .06
	)
	stained, worn := maths.RGB(.52, .40, .24), maths.RGB(.80, .74, .60)
	down := maths.Smooth(stainBottom, stainTop, p.Y)
	colour := walrusMix(worn, stained, down)
	ring := ringShare * math.Sin(ringsPerMetre*p.Y)
	return maths.RGB(colour.R*(1+ring), colour.G*(1+ring), colour.B*(1+ring))
}
