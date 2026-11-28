package manufacture

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/texture"
	"github.com/janearc/slayer/scene/thing"
)

// The pterosaur's colours are authored. No pigment of any pterosaur is
// known for the Hatz holotype; melanosomes are reported in other
// pterosaurs, and what is drawn here is a plausible brown animal, with a
// pale bare beak and darker, patterned skin. The paint only tints the
// geometry: the form and the detail are real surface.

// Numbers the paint uses.
const (
	// hideFine and hideCoarse are the sizes, in metres, of the two
	// mottling patterns on the hide. Authored.
	hideFine   = .045
	hideCoarse = .22
	// hideCoarseShare is how much of the mottle is the coarse pattern.
	hideCoarseShare = .6
	// membraneDistal is the distance along the leading edge, in metres,
	// over which the membrane's colour passes from its root to its tip
	// colour. Authored.
	membraneDistal = 4.4
	// membraneStreak is the size of the streaks in the membrane's colour,
	// in metres. Authored.
	membraneStreak = .09
	// hashMultiplierA to hashMultiplierC mix a lattice point's three
	// integers into one; they are large odd primes.
	hashMultiplierA = 73856093
	hashMultiplierB = 19349663
	hashMultiplierC = 83492791
	// hashFold is the multiplier of the second mixing round.
	hashFold = 0x5bd1e995
	// hashBits is how many bits of the mixed value are kept.
	hashBits = 24
)

// pterosaurHash is a repeatable value from 0 up to but not including 1
// for a lattice point.
func pterosaurHash(a, b, c int) float64 {
	mixed := uint32(a)*hashMultiplierA ^ uint32(b)*hashMultiplierB ^
		uint32(c)*hashMultiplierC
	mixed ^= mixed >> 13
	mixed *= hashFold
	mixed ^= mixed >> 15
	return float64(mixed&(1<<hashBits-1)) / float64(uint32(1)<<hashBits)
}

// pterosaurNoise is smooth value noise from 0 to 1 for a point, with
// features about size metres across.
func pterosaurNoise(point maths.Vec, size float64) float64 {
	x, y, z := point.X/size, point.Y/size, point.Z/size
	ix, iy, iz := int(math.Floor(x)), int(math.Floor(y)), int(math.Floor(z))
	fx, fy, fz := smoothShare(x-float64(ix)), smoothShare(y-float64(iy)),
		smoothShare(z-float64(iz))
	mix := func(a, b, share float64) float64 { return a + (b-a)*share }
	corner := func(dx, dy, dz int) float64 {
		return pterosaurHash(ix+dx, iy+dy, iz+dz)
	}
	low := mix(mix(corner(0, 0, 0), corner(1, 0, 0), fx),
		mix(corner(0, 1, 0), corner(1, 1, 0), fx), fy)
	high := mix(mix(corner(0, 0, 1), corner(1, 0, 1), fx),
		mix(corner(0, 1, 1), corner(1, 1, 1), fx), fy)
	return mix(low, high, fz)
}

// lightMix is the light a share of the way from a to b.
func lightMix(a, b maths.Light, share float64) maths.Light {
	return a.Scale(1 - share).Add(b.Scale(share))
}

// pterosaurHide is the paint of the skin: a mottle between two colours.
type pterosaurHide struct{ light, dark maths.Light }

// At is the hide's colour at a point of the animal's own space.
func (hide pterosaurHide) At(point maths.Vec) maths.Light {
	share := hideCoarseShare*pterosaurNoise(point, hideCoarse) +
		(1-hideCoarseShare)*pterosaurNoise(point, hideFine)
	return lightMix(hide.light, hide.dark, share)
}

// pterosaurMembranePaint is the paint of the membrane, by the flat rest
// layout: the first coordinate is the distance along the leading edge,
// the second the distance across the chord. The skin is darker at the
// root, paler and warmer toward the tip, and streaked along the fibres.
type pterosaurMembranePaint struct{ root, tip maths.Light }

// At is the membrane's colour at a rest point.
func (paint pterosaurMembranePaint) At(point maths.Vec) maths.Light {
	along := clamp(point.X/membraneDistal, 0, 1)
	stretched := maths.V(point.X*.2, point.Y, 0)
	streak := pterosaurNoise(stretched, membraneStreak)
	return lightMix(paint.root, paint.tip, along).Scale(.85 + .3*streak)
}

// pterosaurLooks are the materials of every part, made once.
type pterosaurLooks struct {
	hide, beak, fibre, membrane, ridge, eye, iris, pupil, claw,
	mouth *thing.Material
}

// newPterosaurLooks makes the materials.
func newPterosaurLooks() pterosaurLooks {
	hide := pterosaurHide{light: maths.RGB(.34, .26, .20),
		dark: maths.RGB(.17, .125, .10)}
	membrane := pterosaurMembranePaint{root: maths.RGB(.30, .19, .14),
		tip: maths.RGB(.52, .38, .26)}
	skin := paintOf("hide", maths.RGB(.27, .20, .15), .5, 0)
	skin.Texture = texture.Texture(hide)
	sheet := paintOf("membrane", maths.RGB(.40, .28, .20), .5, 0)
	sheet.Texture = texture.Texture(membrane)
	return pterosaurLooks{
		hide:     skin,
		beak:     paintOf("beak", maths.RGB(.62, .53, .38), .35, .08),
		fibre:    paintOf("fibre", maths.RGB(.50, .39, .28), .7, 0),
		membrane: sheet,
		ridge: paintOf("fibre-ridge", maths.RGB(.34, .22, .15),
			.4, 0),
		eye:   paintOf("eye", maths.RGB(.025, .018, .012), .1, .5),
		iris:  paintOf("iris", maths.RGB(.50, .28, .05), .2, .3),
		pupil: paintOf("pupil", maths.RGB(.004, .004, .004), .1, .4),
		claw:  paintOf("claw", maths.RGB(.09, .075, .065), .3, .1),
		mouth: paintOf("mouth", maths.RGB(.50, .20, .21), .3, .15),
	}
}

// silverPterosaurLooks is an artistic cool, pale material choice. It keeps
// ordinary reflection and a mottled hide; none of the materials emit light.
func silverPterosaurLooks() pterosaurLooks {
	looks := newPterosaurLooks()
	looks.hide.Colour = maths.RGB(.65, .73, .79)
	looks.hide.Texture = pterosaurHide{light: maths.RGB(.78, .84, .89), dark: maths.RGB(.38, .47, .56)}
	looks.membrane.Colour = maths.RGB(.74, .81, .87)
	looks.fibre.Colour = maths.RGB(.72, .80, .87)
	looks.ridge.Colour = maths.RGB(.47, .57, .67)
	looks.beak.Colour = maths.RGB(.70, .72, .73)
	return looks
}

// paintOf is a plain material of the pterosaur, named for the part.
func paintOf(part string, colour maths.Light, satin,
	gloss float64) *thing.Material {
	return &thing.Material{Name: "pterosaur-" + part, Colour: colour,
		Satin: satin, Gloss: gloss}
}
