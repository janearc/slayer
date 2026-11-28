package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/bioforms/relief"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// SkinTone controls illustrative linear RGB skin reflectance. Melanin and
// Vascular are fractions. Each Undertone axis runs from -1 to 1.
type SkinTone struct {
	Melanin, Vascular float64
	Undertone         Chroma
}

// WithSkinTone sets the adult's overall pigment and undertone. NewHuman
// validates the fractions and chroma using the same bounds as the torso.
func (spec HumanSpec) WithSkinTone(tone SkinTone) HumanSpec {
	spec.Torso.Melanin = tone.Melanin
	spec.Torso.Vascular = tone.Vascular
	spec.Torso.Undertone = tone.Undertone
	return spec
}

// WithPaint places a validated pigment stack in the native human chart.
// It replaces the previously supplied stack. An empty stack clears it.
func (spec HumanSpec) WithPaint(paint pigment.Paint) HumanSpec {
	spec.Torso = spec.Torso.WithPaint(paint)
	return spec
}

// WithSkinMottle replaces the seeded mottling in the adult skin.
// A zero mottle keeps the broad anatomical tones and disables the mottling.
func (spec HumanSpec) WithSkinMottle(mottle pigment.Mottle) HumanSpec {
	spec.skinMottle = mottle
	return spec
}

// WithSkinMottleSeed sets the mottle's seed while keeping its scale and strength.
func (spec HumanSpec) WithSkinMottleSeed(seed uint64) HumanSpec {
	spec.skinMottle.Seed = seed
	return spec
}

// WithSkinDiffusion gives the default posed material a CPU diffusion profile.
// NewHuman validates it. A zero share keeps the usual local shading.
func (spec HumanSpec) WithSkinDiffusion(profile thing.SubsurfaceProfile) HumanSpec {
	spec.skinDiffusion = profile
	return spec
}

// WithSkinTransmission gives the default posed material a validated CPU
// thin-tissue profile independently of tone, paint and lateral diffusion.
// An empty profile disables transmission.
func (spec HumanSpec) WithSkinTransmission(profile thing.ThinTissue) HumanSpec {
	spec.skinTransmission = profile
	return spec
}

// AdultSkinDiffusion is the illustrative profile used in the studio studies.
// The widths are in mesh metres and the share is independent of skin tone.
func AdultSkinDiffusion() thing.SubsurfaceProfile {
	return thing.SubsurfaceProfile{Radius: skinDiffusionRadius, Share: .24}
}

type humanSkin struct {
	base                   Torso
	breasts                BreastForm
	marks                  *pigment.Paint
	melanin, fineMelanin   pigment.Mottle
	vascular, fineVascular pigment.Mottle
	pores                  relief.Pores
	furrows                relief.Furrows
}

func newHumanSkin(spec HumanSpec) humanSkin {
	torso := spec.Torso
	marks := torso.paint
	torso.Width, torso.Depth = 1, 1
	torso.paint = nil
	melanin, vascular := spec.skinMottle, spec.skinMottle
	vascular.Seed = noise.Mix(vascular.Seed ^ 0x76617363756c6172)
	vascular.Strength *= .48
	fineMelanin, fineVascular := melanin, vascular
	fineMelanin.Seed = noise.Mix(melanin.Seed ^ 0x6d656c616e696e)
	fineMelanin.Scale *= .25
	fineMelanin.Strength *= .81
	fineVascular.Seed = noise.Mix(vascular.Seed ^ 0x636170696c6c6172)
	fineVascular.Scale *= .18
	fineVascular.Strength *= 1.68
	return humanSkin{
		base: torso, breasts: spec.Breasts, marks: marks,
		melanin: melanin, fineMelanin: fineMelanin, vascular: vascular, fineVascular: fineVascular,
		pores: relief.Pores{
			Seed:    noise.Mix(melanin.Seed ^ 0x706f726573),
			Spacing: .00105, Radius: .00036, Depth: .000037,
		},
		furrows: relief.Furrows{
			Seed:    noise.Mix(melanin.Seed ^ 0x667572726f7773),
			Spacing: .0035, Width: .00105, Depth: .000029,
		},
	}
}

func (paint humanSkin) At(rest maths.Vec) maths.Light {
	return paint.Within(rest, 0)
}

// Within forwards the footprint to regional pigment and custom marks.
func (paint humanSkin) Within(rest maths.Vec, footprint float64) maths.Light {
	chroma := maths.RGB(1, 1, 1)
	lips, lids, seam := faceFeaturePigment(rest)
	if max(lips, lids) > 0 {
		chroma = maths.RGB(1-.10*lips-.045*lids-seam,
			1-.25*lips-.081*lids-seam, 1-.21*lips-.065*lids-seam)
	}
	plate := nailMask(rest)
	face := facialSkinAt(rest)
	absorption, glabrous := adultSkinRegions(rest, face, paint.base.Melanin)
	coverage := (1 - .80*glabrous) * (1 - max(lips, lids, plate))
	absorption.melanin += coverage * (paint.melanin.At(rest, footprint) +
		paint.fineMelanin.At(rest, footprint))
	bloodDetail := .45 + .55*max(face.cheek, face.nose, face.chin)
	absorption.blood += coverage * (paint.vascular.At(rest, footprint) +
		bloodDetail*paint.fineVascular.At(rest, footprint))
	base := paint.base.opticsWithAbsorption(rest, 0, footprint, absorption, paint.breasts).Colour
	keratin := maths.RGB(.61, .41, .35)
	colour := base.Times(chroma).Scale(1 - .13*plate).Add(keratin.Scale(.13 * plate))
	if paint.marks != nil {
		colour = paint.marks.At(colour, rest, footprint)
	}
	return colour
}

// FinishAt keeps the surface's pores, fine grooves and sheen independent
// of its pigmentation. The T-zone carries a slightly tighter highlight.
func (paint humanSkin) FinishAt(rest maths.Vec, footprint float64) (maths.Vec, float64) {
	face := facialSkinAt(rest)
	tZone := max(face.nose, .76*face.forehead, .50*face.chin)
	satin := .69 - .23*tZone + .025*face.cheek
	nails := nailMask(rest)
	if nails > 0 {
		satin += nails * (.48 - satin)
	}
	lips, lids, _ := faceFeaturePigment(rest)
	cover := 1 - max(lips, lids, nails)
	pores := paint.pores.Slope(rest, footprint).
		Scale(cover * (.32 + .68*max(tZone, .82*face.cheek)))
	furrows := paint.furrows.Slope(rest, footprint).
		Scale((1 - nails) * (.61 + .39*face.cheek))
	return pores.Add(furrows), satin
}

func faceFeaturePigment(rest maths.Vec) (lips, lids, seam float64) {
	x, y, z := math.Abs(rest.X), rest.Y, rest.Z
	if y <= 1.48 || z <= .035 {
		return 0, 0, 0
	}
	forward := maths.Smooth(.035, .069, z)
	lip := lipsAt(x)
	lips = forward * lip.pigment(y)
	lids = forward * lidPigmentMask(x, y)
	seam = .09 * lips * bell((y-lip.line)/.0012)
	return lips, lids, seam
}

func lidPigmentMask(x, y float64) float64 {
	across := (x - eyeHalfSpacing) / lidHalfWidth
	if math.Abs(across) > 1.10 {
		return 0
	}
	contour := lidContour(across)
	upper := bell((y - contour.upper) / .00165)
	lower := .63 * bell((y-contour.lower)/.0012)
	return math.Max(upper, lower) * (1 - maths.Smooth(.92, 1.10, math.Abs(across)))
}

func nailMask(point maths.Vec) float64 {
	side := humanSides[0]
	if point.X < 0 {
		side = humanSides[1]
	}
	var group digitGroup
	switch {
	case point.Y > .640 && point.Y < .749 && math.Abs(point.X) > .202:
		group = side.handDigits()
	case point.Y < .038 && point.Z > .130:
		group = side.footDigits()
	default:
		return 0
	}
	mask := 0.0
	for _, digit := range group.digits {
		mask = math.Max(mask, digit.nailAt(point, group))
	}
	return mask
}
