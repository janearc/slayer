package primates

import (
	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

func (style ScalpWispShape) defaults() ScalpWispShape {
	if style.Spacing == 0 {
		style.Spacing = .036
	}
	if style.TipSpread == 0 {
		style.TipSpread = .027
	}
	if style.PartReach == 0 {
		style.PartReach = .027
	}
	if style.Volume == 0 {
		style.Volume = 1
	}
	return style
}

func (style ScalpWispShape) valid() bool {
	return between(style.Spacing, .018, .068) && between(style.TipSpread, .004, .062) &&
		between(style.PartReach, .012, .055) && between(style.Volume, .15, 1.65)
}

func (spec ScalpHair) mediumWispPart(from, to maths.Vec) bool {
	end := -spec.FieldWisps.PartReach
	if max(from.Z, to.Z) <= end || max(from.Y, to.Y) <= 1.608 {
		return false
	}
	first, second := spec.mediumPart(from), spec.mediumPart(to)
	if (first < 0) == (second < 0) {
		return false
	}
	crossing := mixPoint(from, to, first/(first-second))
	return crossing.Z > end && crossing.Y > 1.608
}

func (spec ScalpHair) mediumWisps(surface hair.Surface) (*hair.WispField, error) {
	style := spec.FieldWisps
	guides, err := hair.NewSurfaceGuides(hair.GuideFieldSpec{
		Surface: surface, Region: spec.mediumScalpRegion, Spacing: style.Spacing,
		Seed: noise.Mix(spec.Seed ^ 0x7769737073686170),
	})
	if err != nil {
		return nil, err
	}
	return hair.NewWispField(guides, func(anchor hair.GuideAnchor) hair.WispGuide {
		point, normal := anchor.Point, anchor.Normal
		heading, _ := spec.mediumTangent(point, normal)
		across := normal.Cross(heading).Unit()
		body := noise.Gradient(spec.Seed+106, 28*point.X, 31*point.Y, 29*point.Z)
		sweep := noise.Gradient(spec.Seed+114, 19*point.X, 23*point.Y, 22*point.Z)
		cut := noise.Gradient(spec.Seed+122, 32*point.X, 34*point.Y, 35*point.Z)
		rear := maths.Smooth(.018, .077, -point.Z)
		shoulder := normal.Scale(style.Volume * (-.004 - .009*rear + .013*body)).
			Add(across.Scale(.012 * style.Volume * sweep)).
			Add(heading.Scale(.010 * body))
		tip := maths.V(0, -style.TipSpread*cut, .012*sweep).Add(across.Scale(.008 * body))
		return hair.WispGuide{
			Heading: heading, Shoulder: shoulder, Tip: tip,
			Lift: style.Volume * (-.0013 + .0028*body),
		}
	})
}
