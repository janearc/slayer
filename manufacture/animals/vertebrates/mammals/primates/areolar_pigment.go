package primates

import "github.com/janearc/slayer/maths"

// PigmentShift adjusts regional epidermal absorption, vascular colour and
// chroma around the reference response. Each control runs from -1 to 1.
type PigmentShift struct {
	Melanin, Vascular float64
	Chroma            Chroma
}

// AreolarSideTone controls the areola and the papillary centre on one side.
// Papilla adjustments are relative to the areola. A positive EdgeFeather
// softens the transition into the surrounding chest skin.
type AreolarSideTone struct {
	Areola, Papilla PigmentShift
	EdgeFeather     float64
}

// AreolarTone varies pigmentation independently on the specimen's left
// and right sides. The zero value uses the tone-dependent reference.
type AreolarTone struct {
	Left, Right AreolarSideTone
}

// WithAreolarTone sets the local pigment controls without changing the
// torso's overall skin tone. Check validates each control from -1 to 1.
func (s Torso) WithAreolarTone(tone AreolarTone) Torso {
	s.areolarTone = tone
	return s
}

// WithAreolarTone sets the local pigment controls used by the native
// human's default posed skin. NewHuman validates the control ranges.
func (spec HumanSpec) WithAreolarTone(tone AreolarTone) HumanSpec {
	spec.Torso = spec.Torso.WithAreolarTone(tone)
	return spec
}

func (tone AreolarTone) valid() bool {
	for _, side := range [...]AreolarSideTone{tone.Left, tone.Right} {
		if !between(side.EdgeFeather, -1, 1) {
			return false
		}
		for _, shift := range [...]PigmentShift{side.Areola, side.Papilla} {
			if !between(shift.Melanin, -1, 1) || !between(shift.Vascular, -1, 1) ||
				!shift.Chroma.valid() {
				return false
			}
		}
	}
	return true
}

type areolarMasks struct {
	areola, papilla, tubercles float64
}

// areolarAbsorption adds authored relative pigment to the adjacent chest
// skin. The default regional melanin contrast grows with background
// melanin. Less vascular tint comes through as that absorption increases.
func (s Torso) areolarAbsorption(tone AreolarSideTone, mask areolarMasks,
	sign, fine, grain float64) (melanin, blood float64) {
	pigment := s.Melanin
	base := (.135 + .175*pigment + .025*pigment*pigment) * (1 + .035*sign)
	mottle := max(0, 1+.20*fine+.24*grain-.065*mask.tubercles)
	areola := base * (1 + tone.Areola.Melanin) * mottle
	// The areola already carries the darker epidermal load. A separate
	// papillary addition contributes less absorption on darker skin.
	centre := .021 + .045*(1-pigment) + (.10+.055*pigment)*tone.Papilla.Melanin
	melanin = mask.areola*areola + mask.papilla*centre

	vascular := s.Vascular - .50
	areolaVascular := (.26 + .048*vascular) * (1 - .38*pigment)
	centreVascular := .143 - .033*pigment + .045*vascular + .19*tone.Papilla.Vascular
	blood = mask.areola*areolaVascular*(1+tone.Areola.Vascular) + mask.papilla*centreVascular
	return melanin, blood
}

func (tone AreolarSideTone) chroma(mask areolarMasks) Chroma {
	areola := .72 * mask.areola
	papilla := .56 * mask.papilla
	a, p := tone.Areola.Chroma, tone.Papilla.Chroma
	return Chroma{
		Red:   areola*a.Red + papilla*(.14+p.Red),
		Gold:  areola*a.Gold + papilla*(-.075+p.Gold),
		Olive: areola*a.Olive + papilla*p.Olive,
	}
}

// applyRegionalChroma compresses saturation if a shift would push skin
// reflectance past one. Blending toward its own grey preserves luminance.
func applyRegionalChroma(base maths.Light, shift Chroma) maths.Light {
	colour := shift.apply(base)
	peak := max(colour.R, colour.G, colour.B)
	if peak <= 1 {
		return colour
	}
	luma := base.Luminance()
	share := (1 - luma) / (peak - luma)
	return maths.Grey(luma).Scale(1 - share).Add(colour.Scale(share))
}
