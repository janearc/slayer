package primates

import (
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

type humanSide struct {
	prefix string
	sign   float64
}

var humanSides = [...]humanSide{{"left", 1}, {"right", -1}}

const shoulderSupport = "shoulder-support"

func (side humanSide) at(x, y, z float64) maths.Vec { return maths.V(side.sign*x, y, z) }
func (side humanSide) name(bone string) string      { return side.prefix + "-" + bone }

func (side humanSide) shoulder() maths.Vec { return side.at(.181, 1.327, -.005) }
func (side humanSide) elbow() maths.Vec    { return side.at(.229, 1.061, -.025) }
func (side humanSide) wrist() maths.Vec    { return side.at(.253, .820, -.004) }
func (side humanSide) hip() maths.Vec      { return side.at(.087, .820, -.013) }
func (side humanSide) knee() maths.Vec     { return side.at(.089, .447, .004) }
func (side humanSide) ankle() maths.Vec    { return side.at(.097, .074, -.005) }

type jointWeights struct {
	base, joint, middle string
	share, middleShare  float64
}

func fixedWeight(name string) jointWeights { return jointWeights{base: name} }

func blendWeight(base, joint string, share float64) jointWeights {
	return jointWeights{base: base, joint: joint, share: share}
}

// withMiddle assigns part of the base share to an intermediate bone.
func (weights jointWeights) withMiddle(name string, share float64) jointWeights {
	weights.middle, weights.middleShare = name, (1-weights.share)*share
	return weights
}

// towardsJoint transfers both of the proximal shares to the distal bone.
func (weights jointWeights) towardsJoint(share float64) jointWeights {
	weights.share = mix(weights.share, 1, share)
	weights.middleShare *= 1 - share
	return weights
}

func (weights jointWeights) influences(bones *skeleton.Skeleton) []skeleton.Influence {
	base, _ := bones.Find(weights.base)
	if weights.share == 0 && weights.middleShare == 0 {
		return []skeleton.Influence{skeleton.NewInfluence(base, 1)}
	}
	joint, _ := bones.Find(weights.joint)
	hold := []skeleton.Influence{
		skeleton.NewInfluence(base, 1-weights.share-weights.middleShare),
		skeleton.NewInfluence(joint, weights.share),
	}
	if weights.middleShare != 0 {
		middle, _ := bones.Find(weights.middle)
		hold = append(hold, skeleton.NewInfluence(middle, weights.middleShare))
	}
	return hold
}

func trunkWeights(y float64, point maths.Vec) jointWeights {
	switch {
	case y < .97:
		// Lateral hip skin moves into the proximal thigh. The perineum and
		// the medial gluteal cleft remain keyed to the pelvis on both sides.
		side := humanSides[0]
		if point.X < 0 {
			side = humanSides[1]
		}
		thigh := .30 * (1 - maths.Smooth(.80, .91, y)) *
			maths.Smooth(.018, .133, math.Abs(point.X))
		if thigh > 0 {
			return blendWeight("pelvis", side.name("thigh"), thigh)
		}
		return blendWeight("pelvis", "lumbar", maths.Smooth(.89, .97, y))
	case y < 1.16:
		column := maths.Smooth(1.00, 1.16, y)
		// The anterior costal wall carries the breast through thorax
		// rotation. The more mobile abdomen takes up the blend below it.
		costal := maths.Smooth(1.024, 1.137, y) * maths.Smooth(.047, .091, point.Z)
		return blendWeight("lumbar", "thorax", mix(column, 1, costal))
	default:
		return blendWeight("thorax", "neck", maths.Smooth(1.405, 1.455, y))
	}
}

func shoulderWeights(weights jointWeights, angle, y float64, regions [2]shoulderRegion) jointWeights {
	for i, region := range regions {
		across, height := (angle-region.angle)/region.angleHalf, (y-region.y)/region.yHalf
		radius := math.Hypot(across, height)
		const widest = 1.92
		if radius >= widest {
			continue
		}
		direction := 1 / math.Max(radius, 1e-9)
		vertical := height * direction
		superior := maths.Smooth(-.20, .94, vertical)
		axillary := 1 - maths.Smooth(-.98, -.22, vertical)
		// The lower arc hands motion over to a broad patch of sidewall.
		// Both sides of the shared rim begin with the same weights and an
		// eased slope into the torso or the proximal arm, respectively.
		width := mix(1.61, widest, axillary)
		fade := (1 - maths.Smooth(1, width, radius)) * (1 - maths.Smooth(1.371, 1.405, y))
		if fade > 0 {
			side := humanSides[i]
			anterior := maths.Smooth(.10, .92, -side.sign*across*direction)
			arm := (.34 + .10*superior + .18*axillary) * (1 - .13*anterior) * fade
			// The cap follows clavicle lift through its support bone.
			// A turn on the support affects only the skin in this blend.
			girdle := (.23 + .32*superior + .05*anterior) * fade
			return blendWeight("thorax", side.name("upper-arm"), arm).
				withMiddle(side.name(shoulderSupport), girdle)
		}
	}
	return weights
}

type humanBone struct {
	name, parent string
	head, tail   maths.Vec
	half         float64
	site         proportionSite
}

func (anatomy humanAnatomy) skeleton() (*skeleton.Skeleton, error) {
	bones := skeleton.New()
	root := []humanBone{
		{"pelvis", "", maths.V(0, .804, -.018), maths.V(0, .896, -.031), .083, proportionSite{}},
		{"lumbar", "pelvis", maths.V(0, .896, -.031), maths.V(0, 1.08, -.039), .064, proportionSite{}},
		{"thorax", "lumbar", maths.V(0, 1.08, -.039), maths.V(0, 1.36, -.018), .080, proportionSite{}},
		{"neck", "thorax", maths.V(0, 1.36, -.018), maths.V(0, 1.465, -.011), .034, proportionSite{}},
		{"head", "neck", maths.V(0, 1.465, -.011), maths.V(0, 1.675, -.011), .052, proportionSite{}},
	}
	for _, side := range humanSides {
		root = append(root, side.bones()...)
	}
	// Append the skin controls so all existing bone indices keep their
	// order. A support inherits the clavicle without carrying the arm.
	for _, side := range humanSides {
		root = append(root, humanBone{side.name(shoulderSupport), side.name("clavicle"),
			side.at(.070, 1.293, -.053), side.shoulder(), .008, proportionSite{}})
	}
	for _, bone := range root {
		half := bone.half * anatomy.spec.Stature / humanHeight
		err := bones.Add(bone.name, bone.parent, anatomy.spec.measure(bone.head, bone.site),
			anatomy.spec.measure(bone.tail, bone.site), half, half)
		if err != nil {
			return nil, err
		}
	}
	return bones, nil
}

func (side humanSide) bones() []humanBone {
	hand, foot := side.handDigits(), side.footDigits()
	bones := []humanBone{
		{side.name("clavicle"), "thorax", side.at(.017, 1.331, .046), side.shoulder(), .010, proportionSite{}},
		{side.name("upper-arm"), side.name("clavicle"), side.shoulder(), side.elbow(), .025, hand.site},
		{side.name("forearm"), side.name("upper-arm"), side.elbow(), side.wrist(), .018, hand.site},
		{side.name("hand"), side.name("forearm"), side.wrist(), side.at(.260, .745, -.001), .024, hand.site},
		{side.name("thigh"), "pelvis", side.hip(), side.knee(), .043, foot.site},
		{side.name("shin"), side.name("thigh"), side.knee(), side.ankle(), .027, foot.site},
		{side.name("foot"), side.name("shin"), side.ankle(), side.at(.100, .020, .120), .027, foot.site},
	}
	for _, group := range []digitGroup{hand, foot} {
		for _, digit := range group.digits {
			bones = append(bones, digit.bones(group)...)
		}
	}
	return bones
}
