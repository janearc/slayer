package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// AdultProportions adjusts dimensions of the authored adult reference.
// Each field is a ratio. Zero uses 1, the reference dimension.
type AdultProportions struct {
	ShoulderBreadth, HipBreadth, CranialBreadth float64
	ArmDrop, ArmGirth, LegGirth                 float64
	NasalBreadth, NasalProjection               float64
	MandibularBreadth                           float64
}

func (proportions AdultProportions) resolved() (AdultProportions, error) {
	for _, measure := range [...]struct {
		name      string
		ratio     *float64
		low, high float64
	}{
		{"shoulder breadth", &proportions.ShoulderBreadth, .91, 1.10},
		{"hip breadth", &proportions.HipBreadth, .94, 1.12},
		{"cranial breadth", &proportions.CranialBreadth, .93, 1.10},
		{"arm drop", &proportions.ArmDrop, .88, 1.12},
		{"arm girth", &proportions.ArmGirth, .88, 1.16},
		{"leg girth", &proportions.LegGirth, .90, 1.13},
		{"nasal breadth", &proportions.NasalBreadth, .88, 1.15},
		{"nasal projection", &proportions.NasalProjection, .83, 1.16},
		{"mandibular breadth", &proportions.MandibularBreadth, .90, 1.12},
	} {
		if *measure.ratio == 0 {
			*measure.ratio = 1
		}
		if !between(*measure.ratio, measure.low, measure.high) {
			return AdultProportions{}, fmt.Errorf(
				"adult human: %s must be within %.2f–%.2f", measure.name, measure.low, measure.high)
		}
	}
	return proportions, nil
}

type humanPart uint8

const (
	corePart humanPart = iota
	armPart
	legPart
)

// proportionSite ties a vertex to the authored branch that grew it.
// At branch boundaries the existing trunk and pelvic skin stay shared.
type proportionSite struct {
	part       humanPart
	attachment float64
	side       float64
}

func (spec HumanSpec) baseHeight(y float64) float64 {
	pelvis := pelvisY * spec.LegLength
	// The two linear scales meet at the authored pelvis height. Ease
	// the different tangents across the adjacent soft tissue.
	if y <= .725 {
		return y * spec.LegLength
	}
	trunk := pelvis + (y-pelvisY)*(humanHeight-pelvis)/(humanHeight-pelvisY)
	if y >= .875 {
		return trunk
	}
	return mix(y*spec.LegLength, trunk, maths.Smooth(.725, .875, y))
}

func (spec HumanSpec) point(point maths.Vec) maths.Vec {
	return spec.measure(point, proportionSite{})
}

func (spec HumanSpec) measure(point maths.Vec, site proportionSite) maths.Vec {
	if point.Y >= 1.470 && point.Y < 1.573 {
		point = spec.Proportions.face(point)
	}
	shaped := maths.V(spec.coreX(point), spec.baseHeight(point.Y), point.Z)
	if site.attachment > 0 {
		side := humanSides[0]
		if site.side < 0 {
			side = humanSides[1]
		}
		switch site.part {
		case armPart:
			shaped = mixPoint(shaped, spec.armPoint(side, point, shaped), site.attachment)
		case legPart:
			shaped = mixPoint(shaped, spec.legPoint(side, point, shaped), site.attachment)
		}
	}
	return maths.V(shaped.X*spec.Torso.Width, shaped.Y, shaped.Z*spec.Torso.Depth).
		Scale(spec.Stature / humanHeight)
}

func (proportions AdultProportions) face(point maths.Vec) maths.Vec {
	x, y, z := math.Abs(point.X), point.Y, point.Z
	// Keep the nasal controls below the globes and give the nasal sidewalls
	// and vestibules the same field, independent of their mesh topology.
	nose := maths.Smooth(1.532, 1.547, y) * (1 - maths.Smooth(1.564, 1.573, y))
	nose *= maths.Smooth(.065, .081, z) * (1 - maths.Smooth(.021, .041, x))
	jaw := maths.Smooth(1.470, 1.511, y) * (1 - maths.Smooth(1.542, 1.561, y))
	jaw *= maths.Smooth(-.046, -.008, z) * maths.Smooth(.022, .050, x)
	point.X *= 1 + (proportions.NasalBreadth-1)*nose + (proportions.MandibularBreadth-1)*jaw
	point.Z += (z - .080) * (proportions.NasalProjection - 1) * nose
	return point
}

func (spec HumanSpec) coreX(point maths.Vec) float64 {
	proportions := spec.Proportions
	shoulderY := humanSides[0].shoulder().Y
	hipY := humanSides[0].hip().Y
	shoulder := (proportions.ShoulderBreadth - 1) * bell((point.Y-shoulderY)/.057) *
		(1 - maths.Smooth(1.414, 1.455, point.Y))
	pelvis := (proportions.HipBreadth - 1) * bell((point.Y-hipY)/.079)
	vault := (proportions.CranialBreadth - 1) * maths.Smooth(1.586, 1.643, point.Y)
	return point.X * (1 + shoulder + pelvis + vault)
}

func (spec HumanSpec) armPoint(side humanSide, rest, core maths.Vec) maths.Vec {
	shoulder, elbow, wrist := side.shoulder(), side.elbow(), side.wrist()
	share := 1 - maths.Smooth(1.122, 1.253, rest.Y)
	offset := shoulder.X * (spec.Proportions.ShoulderBreadth - 1)
	point := maths.V(mix(core.X, rest.X+offset, share), core.Y, core.Z)
	if spec.LegLength != 1 || spec.Proportions.ArmDrop != 1 {
		drop := (shoulder.Y - math.Max(wrist.Y, rest.Y)) * spec.Proportions.ArmDrop
		hand := math.Min(0, rest.Y-wrist.Y)
		height := spec.baseHeight(shoulder.Y) - drop + hand
		point.Y = mix(core.Y, height, share)
	}
	centre := limbAxisAt(rest.Y, shoulder, elbow, wrist)
	girth := (spec.Proportions.ArmGirth - 1) *
		maths.Smooth(.819, .884, rest.Y) * (1 - maths.Smooth(1.196, 1.253, rest.Y))
	point.X += (rest.X - centre.X) * girth
	point.Z += (rest.Z - centre.Z) * girth
	return point
}

func (spec HumanSpec) legPoint(side humanSide, rest, core maths.Vec) maths.Vec {
	hip, knee, ankle := side.hip(), side.knee(), side.ankle()
	departure := 1 - maths.Smooth(.664, pelvisY, rest.Y)
	travel := maths.Smooth(.138, .710, rest.Y)
	lateral := hip.X * (spec.Proportions.HipBreadth - 1) * travel
	point := maths.V(mix(core.X, rest.X+lateral, departure), core.Y, core.Z)
	girth := (spec.Proportions.LegGirth - 1) *
		maths.Smooth(.088, .163, rest.Y) * (1 - maths.Smooth(.685, .789, rest.Y))
	centre := limbAxisAt(rest.Y, hip, knee, ankle)
	// Medial tissue gains less transverse width near the perineal bridge.
	medial := maths.Smooth(.005, .034, math.Abs(rest.X))
	point.X += (rest.X - centre.X) * girth * medial
	point.Z += (rest.Z - centre.Z) * girth
	return point
}

func limbAxisAt(y float64, high, hinge, low maths.Vec) maths.Vec {
	first, last := high, hinge
	if y < hinge.Y {
		first, last = hinge, low
	}
	share := (y - first.Y) / (last.Y - first.Y)
	return mixPoint(first, last, math.Max(0, math.Min(1, share)))
}
