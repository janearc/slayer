package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// VellusHair grows fine fibres independently of the coarser body groom.
// Density is the number of candidate shafts per square metre. Zero
// disables the layer. Zero Length and Radius select the reference
// render scales in metres. FlowTurn rotates the local lay in radians.
type VellusHair struct {
	Seed                    uint64
	Density, Length, Radius float64
	FlowTurn                float64
	Regions                 VellusRegions
	Material                *thing.Material
}

// VellusRegions gives each region a separate coverage share from zero to
// one. Finger and toe coverage is confined to the backs of proximal digits.
type VellusRegions struct {
	Cheeks, Forehead, Torso, Arms, Legs, Fingers, Toes float64
}

func (spec VellusHair) layer(digits []digitHairPatch) (*hair.Layer, error) {
	regions := spec.Regions
	if spec.Density == 0 || regions == (VellusRegions{}) {
		return nil, nil
	}
	if spec.Length == 0 {
		spec.Length = .0022
	}
	if spec.Radius == 0 {
		spec.Radius = .000012
	}
	if spec.Material == nil || !between(spec.Density, 1, 240_000) ||
		!between(spec.Length, .0006, .006) || !between(spec.Radius, .000003, .000028) ||
		!between(spec.FlowTurn, -math.Pi, math.Pi) {
		return nil, fmt.Errorf("adult human: vellus hair requires a material and controls in the supported range")
	}
	for _, share := range [...]float64{regions.Cheeks, regions.Forehead, regions.Torso, regions.Arms,
		regions.Legs, regions.Fingers, regions.Toes} {
		if !between(share, 0, 1) {
			return nil, fmt.Errorf("adult human: vellus region shares must be between zero and one")
		}
	}
	return &hair.Layer{
		Seed: spec.Seed, Density: spec.Density, Segments: 3, Sides: 3,
		Material: spec.Material, FaceFilter: regions.overlaps,
		Profile: func(site hair.Site) hair.Strand { return spec.strand(site, digits) },
	}, nil
}

func (regions VellusRegions) overlaps(corners [3]maths.Vec) bool {
	low := min(corners[0].Y, corners[1].Y, corners[2].Y)
	high := max(corners[0].Y, corners[1].Y, corners[2].Y)
	front := max(corners[0].Z, corners[1].Z, corners[2].Z)
	outer := max(math.Abs(corners[0].X), math.Abs(corners[1].X), math.Abs(corners[2].X))
	return (regions.Cheeks > 0 && high >= 1.520 && low <= 1.559 && front >= .027) ||
		(regions.Forehead > 0 && high >= 1.592 && low <= 1.628 && front >= .057) ||
		(regions.Torso > 0 && high >= .910 && low <= 1.337) ||
		(regions.Arms > 0 && high >= .811 && low <= 1.289 && outer >= .169) ||
		(regions.Legs > 0 && high >= .084 && low <= .759) ||
		(regions.Fingers > 0 && high >= .663 && low <= .787 && outer >= .200) ||
		(regions.Toes > 0 && low <= .040 && front >= .113)
}

type vellusSite struct {
	coverage, length float64
	heading          maths.Vec
}

func (spec VellusHair) strand(site hair.Site, digits []digitHairPatch) hair.Strand {
	region := spec.Regions
	local := region.skin(site.Chart)
	if region.Fingers > 0 || region.Toes > 0 {
		if digit := region.digit(site, digits); digit.coverage > local.coverage {
			local = digit
		}
	}
	if local.coverage <= 0 {
		return hair.Strand{}
	}
	point := site.Chart
	grain := max(-1, min(1, noise.Gradient(spec.Seed, 120*point.X, 120*point.Y, 120*point.Z)))
	wander := hairRootDraw(spec.Seed+1, site)
	turn := maths.Rotation(site.ChartNormal, spec.FlowTurn+.30*(wander-.5)+.17*grain)
	length := hairRootDraw(spec.Seed+2, site)
	width := hairRootDraw(spec.Seed+3, site)
	return hair.Strand{
		Coverage: local.coverage * (.88 + .10*grain),
		Flow:     site.ChartFlow(turn.Out(local.heading)),
		Length:   spec.Length * local.length * mix(.63, 1.15, length),
		Radius:   spec.Radius * mix(.62, 1, width),
		RootLift: mix(.25, .42, wander),
		TipLift:  mix(.04, .16, wander),
		Sweep:    .16 * grain,
		Taper:    hair.ShaftTaper{Start: .19, TipShare: .03},
	}
}

func (regions VellusRegions) skin(point maths.Vec) vellusSite {
	x, y, z := math.Abs(point.X), point.Y, point.Z
	side := math.Copysign(1, point.X)
	cheek := 1 - maths.Smooth(.47, 1, math.Hypot((x-.047)/.024, (y-1.5395)/.0195))
	// Leave the orbit bare as the cheek band approaches the lower lid.
	orbit := maths.Smooth(.013, .023, math.Hypot(x-eyeHalfSpacing, y-eyeHeight))
	cheek *= orbit * maths.Smooth(.027, .057, z)
	forehead := hairBand(y, 1.592, 1.628, .009) * maths.Smooth(.057, .073, z)
	arm := hairBand(y, .811, 1.289, .032) * maths.Smooth(.169, .212, x)
	leg := hairBand(y, .084, .759, .055) * (1 - maths.Smooth(.166, .198, x))
	torso := hairBand(y, .910, 1.337, .052) * (1 - maths.Smooth(.14, .180, x))
	zones := [...]vellusSite{
		{regions.Cheeks * cheek, .86, maths.V(.50*side, -.87, .08)},
		{regions.Forehead * forehead, .70, maths.V(.45*side, .89, .08)},
		{regions.Arms * arm * .66, 1, maths.V(.20*side, -1, .08)},
		{regions.Legs * leg * .51, .88, maths.V(-.14*side, -1, .08)},
		{regions.Torso * torso * .54, .78, maths.V(.24*side, -1, .08)},
	}
	var strongest vellusSite
	for _, zone := range zones {
		if zone.coverage > strongest.coverage {
			strongest = zone
		}
	}
	return strongest
}

func (regions VellusRegions) digit(site hair.Site, digits []digitHairPatch) vellusSite {
	for _, patch := range digits {
		if site.Face < patch.first {
			break
		}
		if site.Face >= patch.end {
			continue
		}
		coverage := regions.Toes
		if patch.hand {
			coverage = regions.Fingers
		}
		dorsal := maths.Smooth(.36, .82, site.ChartNormal.Dot(patch.dorsal))
		along := site.Chart.Sub(patch.origin).Dot(patch.along)
		return vellusSite{coverage * dorsal * hairBand(along, .09, 1, .25), .92, patch.along}
	}
	return vellusSite{}
}
