package primates

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/hair"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
	"github.com/janearc/slayer/scene/thing"
)

// BodyHair describes independently optional adult body hair. Density is
// the candidate shafts per square metre. Length and Radius are the
// maximum rendered shaft dimensions in metres. Zero selects the
// reference render values. All regional shares are opt-in.
type BodyHair struct {
	Seed                    uint64
	Density, Length, Radius float64
	Regions                 BodyHairRegions
	Material                *thing.Material
}

// BodyHairRegions gives each skin region a coverage share from zero to
// one. The same regional controls apply to either adult anatomy.
type BodyHairRegions struct {
	Arms, Legs, Torso, Axillae, Pubis float64
}

func (spec BodyHair) layer() (hair.Layer, error) {
	if spec.Density == 0 {
		spec.Density = 33_000
	}
	if spec.Length == 0 {
		spec.Length = .012
	}
	if spec.Radius == 0 {
		spec.Radius = .000049
	}
	regions := spec.Regions
	if spec.Material == nil || !between(spec.Density, 1, 125_000) ||
		!between(spec.Length, .002, .028) || !between(spec.Radius, .000009, .00011) {
		return hair.Layer{}, fmt.Errorf("adult human: body hair requires a material and controls in the supported range")
	}
	for _, share := range [...]float64{regions.Arms, regions.Legs, regions.Torso, regions.Axillae, regions.Pubis} {
		if !between(share, 0, 1) {
			return hair.Layer{}, fmt.Errorf("adult human: body hair region shares must be between zero and one")
		}
	}
	return hair.Layer{
		Seed: spec.Seed, Density: spec.Density, Segments: 3, Sides: 3,
		Material: spec.Material, FaceFilter: bodyHairFace, Profile: spec.strand,
	}, nil
}

func bodyHairFace(corners [3]maths.Vec) bool {
	return max(corners[0].Y, corners[1].Y, corners[2].Y) >= .094 &&
		min(corners[0].Y, corners[1].Y, corners[2].Y) <= 1.34
}

type bodyHairSite struct {
	coverage, length, radius, lift, wave float64
	flow                                 maths.Vec
}

func (spec BodyHair) strand(site hair.Site) hair.Strand {
	point := site.Chart
	local := spec.Regions.at(point)
	if local.coverage <= 0 {
		return hair.Strand{}
	}
	grain := noise.Gradient(spec.Seed, 188*point.X, 188*point.Y, 188*point.Z)
	return hair.Strand{
		Coverage: local.coverage * (.83 + .17*grain),
		Flow:     site.ChartFlow(local.flow),
		Length:   spec.Length * local.length * (.79 + .21*grain),
		Radius:   spec.Radius * local.radius * (.89 + .11*grain),
		RootLift: local.lift + .07*grain, TipLift: .06 + .07*grain,
		Sweep: .31 * grain, Wave: local.wave + .18*grain,
	}
}

// at blends authored beard-free coverage zones on the canonical body.
// Smooth bands keep ankles and wrists clear and local patches follow the
// lateral folds and the anterior pubic mound.
func (regions BodyHairRegions) at(point maths.Vec) bodyHairSite {
	x, y, z := math.Abs(point.X), point.Y, point.Z
	side := math.Copysign(1, point.X)
	down := maths.V(.10*side, -1, .07)
	arm := maths.Smooth(.180, .214, x) * hairBand(y, .824, 1.245, .032)
	forearm := 1 - maths.Smooth(1.030, 1.118, y)
	calf := 1 - maths.Smooth(.407, .480, y)
	leg := hairBand(y, .096, .735, .065) * maths.Smooth(.028, .057, x)
	sternum := (1 - maths.Smooth(.027, .087, x)) * maths.Smooth(.052, .091, z)
	back := .23 * maths.Smooth(.034, .085, -z)
	torso := (sternum + back) * hairBand(y, .913, 1.332, .060)
	axilla := hairBand(y, 1.251, 1.333, .023) * hairBand(x, .146, .204, .022) *
		(1 - maths.Smooth(.018, .041, math.Abs(z+.010)))
	pubis := maths.Smooth(.073, .095, z) *
		(1 - maths.Smooth(.63, 1, math.Hypot(x/.055, (y-.845)/.046)))

	// Region transitions are light enough that overlapping patches can
	// use the strongest local type without a seam in visible coverage.
	zones := [...]bodyHairSite{
		{regions.Arms * arm * (.62 + .24*forearm), .29 + .44*forearm, .40 + .27*forearm, .24, .12,
			maths.V(.22*side, -1, .11)},
		{regions.Legs * leg * (.76 - .18*calf), .49 + .27*calf, .48 + .24*calf, .29, .16, down},
		{regions.Torso * torso * .69, .49, .57, .31, .19, maths.V(-.24*side, -1, .10)},
		{regions.Axillae * axilla, .97, .90, .48, .51, maths.V(.62*side, -.70, .19)},
		{regions.Pubis * pubis, 1, 1, .52, .60, maths.V(.44*side, -.77, .14)},
	}
	var strongest bodyHairSite
	for _, zone := range zones {
		if zone.coverage > strongest.coverage {
			strongest = zone
		}
	}
	return strongest
}

func hairBand(point, low, high, feather float64) float64 {
	return maths.Smooth(low, low+feather, point) * (1 - maths.Smooth(high-feather, high, point))
}
