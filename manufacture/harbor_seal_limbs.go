package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// Forelimb bones of the Burke Museum's harbor seal UWBM 81073, an adult
// female of 83.3 lb, from its record at paleo.iri.isu.edu specimen 633.
// The record's "Length" column, units not stated, read as millimetres.
const (
	sealScapulaLength = .1406 // 140.554
	sealHumerusLength = .1076 // 107.575
	sealRadiusLength  = .1027 // 102.745
	// sealScapulaBreadth is the record's "Thickness" for the scapula,
	// 134.929, read as the breadth of the blade.
	sealScapulaBreadth = .1349
	// sealHumerusBreadth is the humerus's "Width", 38.651, read as its
	// breadth at its widest end.
	sealHumerusBreadth = .0387
)

// sealBladeNeck is the share of its breadth a flat bone keeps at its
// narrow end, the scapula's neck at the glenoid or the hip's at the
// acetabulum; chosen.
const sealBladeNeck = .3

// sealShaftWaist is the share of its end radius a limb bone's shaft
// keeps at its middle; chosen.
const sealShaftWaist = .7

// sealWristLength is the carpus's length from the forearm to the
// knuckles, in metres; chosen.
const sealWristLength = .03

// sealForeDigits is each fore digit's metacarpal and phalanges, from the
// thumb, in metres. The thumb has two phalanges, the others three. The
// lengths are chosen, the first digit longest; no source was reached.
var sealForeDigits = [5][]float64{
	{.078, .052, .029},
	{.068, .044, .029, .023},
	{.062, .040, .026, .021},
	{.057, .036, .023, .020},
	{.052, .033, .021, .018},
}

// sealHindDigits is each hind digit's metatarsal and phalanges, from the
// first, in metres. Chosen, with the first and fifth longest so the
// flipper's outer edges lead; no source was reached.
var sealHindDigits = [5][]float64{
	{.094, .068, .036},
	{.081, .055, .036, .029},
	{.075, .049, .034, .026},
	{.081, .055, .036, .029},
	{.091, .065, .044, .035},
}

// Hind limb lengths, chosen: no record with them was reached.
const (
	sealFemurLength = .085
	sealTibiaLength = .19
)

// The limbs' joints, in radians, chosen unless a comment says otherwise.
var (
	sealShoulderBlade = skeleton.Limits(-.3, .3, -.2, .2, -.1, .1)
	sealShoulderJoint = skeleton.Limits(-.9, .9, -.9, .9, -.5, .5)
	sealElbow         = skeleton.Hinge(-.8, .8)
	sealWrist         = skeleton.Limits(-.6, .6, -.3, .3, -.2, .2)
	sealForeDigit     = skeleton.Limits(-.4, .4, -.1, .1, 0, 0)
	sealHipJoint      = skeleton.Limits(-.5, .5, -.4, .4, -.3, .3)
	sealKnee          = skeleton.Hinge(-.4, .4)
	// sealAnkle keeps the foot pointing aft: phocids "cannot position
	// their hind-flippers to move on land ... because of their massive
	// ankle bones and flatter heels" (Wikipedia, Pinniped, citing
	// Riedman 1990, p. 64). The feather of .8 is the rig's own bound.
	sealAnkle     = skeleton.Limits(-.35, .35, -.5, .5, -.8, .8)
	sealHindDigit = skeleton.Limits(-.4, .4, -.12, .12, 0, 0)
)

// sealGlenoid is the right shoulder joint at rest: beside the last neck
// vertebrae, inside the skin. Chosen.
var sealGlenoid = maths.V(.50, .067, .131)

// sealForeArm is the direction the rest fore limb runs from the glenoid,
// for the right side: back along the flank, down and a little out, so
// the forearm ends just under the skin below the flank's widest line
// behind the axillary site, 30 percent of the length behind the nose,
// and the hand trails back and down outside it. Chosen.
var sealForeArm = maths.V(-.2, -.088, .064)

// sealFlankOut is the way the right hand's back faces: out from the
// flank and half downward, so the blade shows from the side and from
// above; the hand's plane is square to this as nearly as the arm
// allows. Chosen.
var sealFlankOut = maths.V(0, -.75, .66)

// sealHindFoot is the direction of the rest hind flipper for the right
// side: straight back and a little toward the other foot, so the two
// lie side by side behind the tail. Chosen.
var sealHindFoot = maths.V(-1, 0, -.06)

// sealMirror is a right-side point or direction taken to the side.
func sealMirror(point maths.Vec, side float64) maths.Vec {
	return maths.V(point.X, point.Y, side*point.Z)
}

// sealForelimb is one side's scapula and, by the shared hand, its
// humerus, radius, ulna, carpals and five digits, the thumb leading.
func sealForelimb(side float64) func(*sealBones) error {
	return func(made *sealBones) error {
		prefix := "fore-" + sealSideName(side) + "-"
		glenoid := sealMirror(sealGlenoid, side)
		blade := glenoid.Add(sealMirror(maths.V(-.5, .7, -.45), side).
			Unit().Scale(sealScapulaLength))
		err := made.add(prefix+"scapula", "thoracic2", glenoid, blade,
			.003, skeleton.NewBlade(glenoid, blade, sealScapulaBreadth/2, .003,
				sealBladeNeck, maths.V(0, 0, side)), sealShoulderBlade)
		if err != nil {
			return err
		}
		along := sealMirror(sealForeArm, side).Unit()
		// The thumb's edge leads: forward and down, along the flank.
		lead := sealMirror(sealFlankOut.Cross(sealForeArm), side).Unit()
		hand := skeleton.NewHand(glenoid, along, lead, sealHumerusBreadth/2).
			Arm(sealHumerusLength, sealRadiusLength, sealWristLength)
		const spacing, splay = .014, .06 // chosen
		for digit, lengths := range sealForeDigits {
			from := float64(2 - digit)
			hand = hand.Digit(spacing*from, splay*from, lengths...)
		}
		if err := made.bones.Hand(prefix, prefix+"scapula", hand); err != nil {
			return err
		}
		made.jointHand(prefix)
		return nil
	}
}

// jointHand gives the shared hand's bones their joint ranges.
func (made *sealBones) jointHand(prefix string) {
	made.joints = made.joints.With(prefix+"humerus", sealShoulderJoint).
		With(prefix+"radius", sealElbow).With(prefix+"ulna", sealElbow)
	for digit := 1; digit <= len(sealForeDigits); digit++ {
		made.joints = made.joints.
			With(fmt.Sprintf("%scarpal%d", prefix, digit), sealWrist).
			With(fmt.Sprintf("%smetacarpal%d", prefix, digit), sealForeDigit)
		for bone := 1; bone < len(sealForeDigits[digit-1]); bone++ {
			made.joints = made.joints.With(
				fmt.Sprintf("%sphalanx%d-%d", prefix, digit, bone),
				sealForeDigit)
		}
	}
}

// sealPelvis is one side's innominate as two plates from the
// acetabulum: the ilium forward and up, the ischium and pubis back.
func sealPelvis(side float64) func(*sealBones) error {
	return func(made *sealBones) error {
		hip := sealAcetabulum(side)
		outward := maths.V(0, 0, side)
		forward := sealMirror(maths.V(.08, .035, -.01), side) // chosen
		err := made.add("ilium-"+sealSideName(side), "sacral1", hip,
			hip.Add(forward), .004, skeleton.NewBlade(hip, hip.Add(forward),
				.016, .004, sealBladeNeck, outward), skeleton.Fixed())
		if err != nil {
			return err
		}
		back := sealMirror(maths.V(-.11, -.01, -.012), side) // chosen
		return made.add("ischium-"+sealSideName(side), "sacral1", hip,
			hip.Add(back), .003, skeleton.NewBlade(hip, hip.Add(back), .012,
				.003, sealBladeNeck, outward), skeleton.Fixed())
	}
}

// sealAcetabulum is the hip socket, below and beside the column; chosen.
func sealAcetabulum(side float64) maths.Vec {
	const station, below, beside = -.55, .07, .045
	return sealSpinePoint(station).Add(maths.V(0, -below, side*beside))
}

// sealHindlimb is one side's femur, tibia and fibula, ankle, tarsals
// and five digits, the foot trailing behind the body.
func sealHindlimb(side float64) func(*sealBones) error {
	return func(made *sealBones) error {
		name := func(part string) string {
			return "hind-" + sealSideName(side) + "-" + part
		}
		hip := sealAcetabulum(side)
		thigh := sealMirror(maths.V(-.55, -.35, .6), side).Unit() // chosen
		knee := hip.Add(thigh.Scale(sealFemurLength))
		err := made.add(name("femur"), "ilium-"+sealSideName(side), hip,
			knee, .010, skeleton.NewRod(hip, knee, .013, sealShaftWaist),
			sealHipJoint)
		if err != nil {
			return err
		}
		aim := sealMirror(maths.V(-.80, -.02, .038), side) // chosen
		ankle := knee.Add(aim.Sub(knee).Unit().Scale(sealTibiaLength))
		err = made.add(name("tibia"), name("femur"), knee, ankle, .009,
			skeleton.NewRod(knee, ankle, .012, sealShaftWaist), sealKnee)
		if err != nil {
			return err
		}
		outside := maths.V(0, 0, side*.012) // the fibula's offset, chosen
		err = made.add(name("fibula"), name("tibia"), knee.Add(outside),
			ankle.Add(outside), .005, skeleton.NewRod(knee.Add(outside),
				ankle.Add(outside), .007, sealShaftWaist), skeleton.Fixed())
		if err != nil {
			return err
		}
		return sealFoot(made, name, ankle, side)
	}
}

// sealBlock is a short bone of length, width and thickness in metres,
// from start along way under parent.
func sealBlock(made *sealBones, name, parent string, start, way maths.Vec,
	size [3]float64, joint skeleton.Joint,
) error {
	end := start.Add(way.Scale(size[0]))
	const blocky = .9 // a short bone's middle over its ends, chosen
	return made.add(name, parent, start, end, size[2]/2,
		skeleton.NewRod(start, end, (size[1]+size[2])/4, blocky), joint)
}

// sealFoot is the ankle, the tarsals and five digits of one hind
// flipper, fanned up and down so the two feet's soles face each other,
// the first digit lowest.
func sealFoot(made *sealBones, name func(string) string, ankle maths.Vec,
	side float64,
) error {
	foot := sealMirror(sealHindFoot, side).Unit()
	outward := maths.Up.Sub(foot.Scale(maths.Up.Dot(foot))).Unit()
	err := sealBlock(made, name("astragalus"), name("tibia"), ankle, foot,
		[3]float64{.032, .022, .018}, sealAnkle) // chosen
	if err != nil {
		return err
	}
	if err := sealTarsals(made, name, ankle, foot, outward); err != nil {
		return err
	}
	return sealToes(made, name, ankle, foot, outward)
}

// sealToes is the five metatarsals and their phalanges, each on its
// tarsal, by the shared ray.
func sealToes(made *sealBones, name func(string) string, ankle, foot,
	outward maths.Vec,
) error {
	tarsals := [5]string{"cuneiform-1", "cuneiform-2", "cuneiform-3",
		"cuboid", "cuboid"}
	const base, spacing, fan, thick = .056, .020, .045, .005 // chosen
	for digit, lengths := range sealHindDigits {
		from := float64(digit - 2)
		start := ankle.Add(foot.Scale(base)).Add(outward.Scale(spacing * from))
		way := foot.Add(outward.Scale(fan * from))
		names, err := made.bones.Ray(name(fmt.Sprintf("digit%d-", digit+1)),
			name(tarsals[digit]), start, way, lengths, thick)
		if err != nil {
			return err
		}
		for _, bone := range names {
			made.joints = made.joints.With(bone, sealHindDigit)
		}
	}
	return nil
}

// sealTarsal is one tarsal: its name, its parent's, where it starts and
// its length, width and thickness in metres.
type sealTarsal struct {
	part, parent string
	at           maths.Vec
	size         [3]float64
}

// sealTarsals is the calcaneus, navicular, cuboid and three cuneiforms,
// held to the astragalus; sizes and places chosen.
func sealTarsals(made *sealBones, name func(string) string,
	ankle, foot, outward maths.Vec,
) error {
	row := ankle.Add(foot.Scale(.032))
	parts := []sealTarsal{
		{"calcaneus", "astragalus", ankle.Add(outward.Scale(.012)).
			Add(maths.V(0, -.008, 0)), [3]float64{.045, .018, .016}},
		{"navicular", "astragalus", row.Add(outward.Scale(-.006)),
			[3]float64{.010, .014, .012}},
		{"cuboid", "astragalus", row.Add(outward.Scale(.012)),
			[3]float64{.014, .016, .014}},
	}
	for index := range 3 {
		parts = append(parts, sealTarsal{
			fmt.Sprintf("cuneiform-%d", index+1), "navicular",
			row.Add(foot.Scale(.012)).
				Add(outward.Scale(-.014 + .008*float64(index))),
			[3]float64{.010, .010, .010}})
	}
	for _, each := range parts {
		err := sealBlock(made, name(each.part), name(each.parent), each.at,
			foot, each.size, skeleton.Fixed())
		if err != nil {
			return err
		}
	}
	return nil
}
