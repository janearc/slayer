package pinnipeds

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// Rest stations of the column on the rig's X axis, in metres, and the
// skull's length. Chosen to sit in the rest skin of a 1.67 m animal
// unless the comment names a source.
const (
	sealCondyle       = .60  // occipital condyles, chosen
	sealFirstThoracic = .38  // chosen
	sealLastCaudal    = -.92 // inside the grown tail, chosen
	// sealSkullLength is condylobasal length: 205 mm, the floor a search
	// summary gave for adult female Kuril harbor seals (Japanese Journal
	// of Veterinary Research 57(3)); the paper was not opened. Chosen.
	sealSkullLength = .205
)

// How high the column runs in the body, as a share of the section's half
// height above its centre: high in the trunk, lower through the head.
const (
	sealTrunkLift = .45 // chosen
	sealHeadLift  = .15 // chosen
	sealFallStart = .38 // where the column starts to fall to the head
	sealFallEnd   = .55 // where it reaches the head's height
)

// sealCentrum is a vertebral body's half thickness in the trunk, and
// sealTailCentrum at the tail's tip, in metres; chosen.
const sealCentrum, sealTailCentrum = .013, .004

// sealRibInset is the rib cage's share of the skin's half width and half
// height at its station, leaving the rest to muscle and blubber; chosen.
const sealRibInset = .65

// sealRibSlope is how far back a rib runs for each metre it falls; ribs
// lean toward the tail. Chosen.
const sealRibSlope = .35

// sealBones is the seal's skeleton with each bone's joint range and the
// form it is drawn with, beside the shared skeleton's bones.
type sealBones struct {
	profile pinnipedProfile
	bones   *skeleton.Skeleton
	joints  skeleton.Joints
	forms   map[string]skeleton.BoneForm
}

// add places a bone of half thickness thick with its form and its joint.
func (made *sealBones) add(name, parent string, head, tail maths.Vec,
	thick float64, form skeleton.BoneForm, joint skeleton.Joint,
) error {
	if err := made.bones.Add(name, parent, head, tail, thick,
		thick); err != nil {
		return err
	}
	made.joints = made.joints.With(name, joint)
	made.forms[name] = form
	return nil
}

// names is every bone's name, in the order they were added.
func (made *sealBones) names() []string {
	names := make([]string, made.bones.Len())
	for index := range names {
		names[index] = made.bones.Bone(index).Name()
	}
	return names
}

// ends is a bone's head and tail at rest, and whether it is here.
func (made *sealBones) ends(name string) (maths.Vec, maths.Vec, bool) {
	at, found := made.bones.Find(name)
	if !found {
		return maths.Vec{}, maths.Vec{}, false
	}
	bone := made.bones.Bone(at)
	return bone.Head(), bone.Tail(), true
}

// harborSealBones is the harbor seal's skeleton at rest in the rig's
// space: metres, +X toward the muzzle, +Y up, +Z to the right. The first
// thoracic vertebra is its root.
func harborSealBones() (*sealBones, error) {
	return newSealBones(defaultPinniped("harbor-seal"))
}

func newSealBones(profile pinnipedProfile) (*sealBones, error) {
	made := &sealBones{profile: profile, bones: skeleton.New(),
		forms: map[string]skeleton.BoneForm{}}
	steps := []func(*sealBones) error{sealTrunkColumn, sealNeckAndSkull,
		sealRibs, sealSternum, sealTeeth}
	for _, side := range [...]float64{-1, 1} {
		steps = append(steps, sealForelimb(side), sealPelvis(side),
			sealHindlimb(side))
	}
	for _, step := range steps {
		if err := step(made); err != nil {
			return nil, fmt.Errorf("harbor seal bones: %w", err)
		}
	}
	return made, nil
}

// sealProfileAt is the rest skin's section at x: the outline's, which
// the skeleton is laid inside.
func sealProfileAt(x float64) sealSection { return sealOutlineAt(x) }

// sealSpinePoint is the column's centre at rest at x.
func sealSpinePoint(x float64) maths.Vec {
	section := sealProfileAt(x)
	fall := maths.Smooth(sealFallStart, sealFallEnd, x)
	lift := sealTrunkLift + (sealHeadLift-sealTrunkLift)*fall
	return maths.V(x, section.y+lift*section.height, 0)
}

// sealPath is the column's path from one station to another, for the
// shared column: the centre and a vertebral body's half thickness,
// narrowing through the tail.
func sealPath(from, to float64) skeleton.Path {
	return func(share float64) (maths.Vec, float64) {
		x := from + (to-from)*share
		tail := maths.Smooth(-.58, sealLastCaudal, x)
		return sealSpinePoint(x), sealCentrum +
			(sealTailCentrum-sealCentrum)*tail
	}
}

// The column's joints, in radians. "Phocids have a rigid thoracic region
// and a highly flexible lumbar region" (Pierce and others 2011, its
// abstract); the angles themselves are chosen.
var (
	sealThoracicJoint = skeleton.Limits(-.03, .03, -.03, .03, -.02, .02)
	sealLumbarJoint   = skeleton.Limits(-.15, .15, -.23, .23, -.05, .05)
	sealCaudalJoint   = skeleton.Limits(-.2, .2, -.2, .2, 0, 0)
	sealNeckJoint     = skeleton.Limits(-.25, .25, -.25, .25, -.15, .15)
)

// sealVertebrae is each region's joint and the form of its vertebra at a
// share of the way along the region; sizes chosen. Long lumbar
// transverse processes: the lumbar region's "long muscular lever arms"
// (Pierce and others 2011).
var sealVertebrae = map[string]struct {
	joint skeleton.Joint
	sizes func(share float64) (radius, spine, reach float64)
}{
	"thoracic": {sealThoracicJoint, func(float64) (float64, float64,
		float64) {
		return .013, .024, .012
	}},
	"lumbar": {sealLumbarJoint, func(float64) (float64, float64, float64) {
		return .016, .020, .035
	}},
	"sacral": {sealLumbarJoint, func(float64) (float64, float64, float64) {
		return .015, .018, .030
	}},
	"caudal": {sealCaudalJoint, func(share float64) (float64, float64,
		float64) {
		narrow := 1 - .7*share
		return .010 * narrow, .006 * narrow, .012 * (1 - share)
	}},
	"neck": {sealNeckJoint, func(float64) (float64, float64, float64) {
		return .011, .008, .016
	}},
}

// sealTrunkColumn is the thoracic, lumbar, sacral and caudal vertebrae
// by the shared column, from the root at the first thoracic back to the
// tip of the tail, each given its region's joint and form.
func sealTrunkColumn(made *sealBones) error {
	regions := []skeleton.Region{
		skeleton.NewRegion("thoracic", made.profile.Thoracic, 1, 1),
		skeleton.NewRegion("lumbar", made.profile.Lumbar, 1.25, 1.25),
		skeleton.NewRegion("sacral", made.profile.Sacral, 2.75, 2.75),
		skeleton.NewRegion("caudal", made.profile.Caudal, .85, .57),
	}
	err := made.bones.Column("", regions,
		sealPath(sealFirstThoracic, sealLastCaudal))
	if err != nil {
		return err
	}
	made.dressColumn("thoracic", "lumbar", "sacral", "caudal")
	return nil
}

// dressColumn gives each vertebra of the named regions its joint and
// form, the shared column having named them region and number.
func (made *sealBones) dressColumn(regions ...string) {
	for _, region := range regions {
		count := made.bones.Count(region)
		for number := 1; number <= count; number++ {
			name := fmt.Sprintf("%s%d", region, number)
			share := float64(number-1) / math.Max(1, float64(count-1))
			head, tail, _ := made.ends(name)
			radius, spine, reach := sealVertebrae[region].sizes(share)
			made.joints = made.joints.With(name, sealVertebrae[region].joint)
			made.forms[name] = skeleton.NewCentrum(head, tail, radius, spine,
				reach)
		}
	}
}

// sealNeckAndSkull is the cervical vertebrae from the first thoracic
// forward to the skull, the skull, and the two halves of the mandible.
func sealNeckAndSkull(made *sealBones) error {
	regions := []skeleton.Region{skeleton.NewRegion("neck", made.profile.Cervical, 1, 1)}
	err := made.bones.Column("thoracic1", regions,
		sealPath(sealFirstThoracic, sealCondyle))
	if err != nil {
		return err
	}
	made.dressColumn("neck")
	condyle := sealSpinePoint(sealCondyle)
	snout := sealSpinePoint(sealCondyle + sealSkullLength)
	err = made.add("skull", fmt.Sprintf("neck%d", made.profile.Cervical), condyle,
		snout, .04, sealSkullForm(condyle, snout),
		skeleton.Limits(-.6, .6, -.3, .3, -.3, .3))
	if err != nil {
		return err
	}
	return sealMandibles(made, condyle)
}

// sealSkullForm is the skull from the condyles to the incisors: the
// braincase broad and high, narrow between the orbits, the rostrum
// short. Half widths, half heights and lifts in metres, chosen.
func sealSkullForm(condyle, snout maths.Vec) skeleton.Loft {
	return skeleton.NewLoft(condyle, snout, maths.Up,
		skeleton.NewLoftSection(.04, .032, .028, .010),
		skeleton.NewLoftSection(.20, .052, .042, .018),
		skeleton.NewLoftSection(.38, .058, .034, .010),
		skeleton.NewLoftSection(.52, .026, .028, .008),
		skeleton.NewLoftSection(.68, .027, .025, .000),
		skeleton.NewLoftSection(.85, .022, .019, -.004),
		skeleton.NewLoftSection(.96, .013, .011, -.005))
}

// sealMandibles is the two halves of the lower jaw, from the jaw joint
// under the braincase to the chin, each opening on a hinge; chosen.
func sealMandibles(made *sealBones, condyle maths.Vec) error {
	const jaw, chin, drop, spread, narrow = .35, .95, .035, .042, .010
	for _, side := range [...]float64{-1, 1} {
		joint := condyle.Add(maths.V(jaw*sealSkullLength, -drop,
			side*spread))
		tip := condyle.Add(maths.V(chin*sealSkullLength, -drop,
			side*narrow))
		err := made.add("mandible-"+sealSideName(side), "skull", joint, tip,
			.008, skeleton.NewBlade(joint, tip, .011, .004, .6,
				maths.V(0, 0, side)), skeleton.Hinge(-.6, 0))
		if err != nil {
			return err
		}
	}
	return nil
}

// The teeth on each side by the dental formula: "The dental formula is
// I 3/2, C 1/1, PC 5/5." (Duke OBIS-SEAMAP, harbor seal species page),
// 34 in all. Places along the jaws, sizes and forms are chosen.
var (
	sealUpperTeeth = []sealTooth{
		{"incisor-1", .975, .004, .007, .0014},
		{"incisor-2", .96, .008, .007, .0014},
		{"incisor-3", .945, .012, .008, .0016},
		{"canine", .90, .016, .020, .0040},
		{"postcanine-1", .84, .018, .008, .0030},
		{"postcanine-2", .79, .019, .009, .0034},
		{"postcanine-3", .74, .020, .009, .0036},
		{"postcanine-4", .69, .021, .009, .0036},
		{"postcanine-5", .64, .022, .008, .0032},
	}
	sealLowerTeeth = []sealTooth{
		{"incisor-1", .955, .005, .006, .0013},
		{"incisor-2", .94, .009, .006, .0013},
		{"canine", .89, .014, .016, .0036},
		{"postcanine-1", .83, .016, .008, .0030},
		{"postcanine-2", .78, .017, .009, .0034},
		{"postcanine-3", .73, .018, .009, .0036},
		{"postcanine-4", .68, .019, .009, .0036},
		{"postcanine-5", .63, .020, .008, .0032},
	}
)

// sealTooth is one tooth: its name, where it stands as a share of the
// skull's length from the condyles, how far out from the midline, and
// its crown's height and radius, in metres.
type sealTooth struct {
	name                      string
	share, beside, tall, wide float64
}

// sealGum is how far below the skull's axis the upper gum runs, and
// sealJaw how far the lower gum runs below that, in metres; chosen.
const sealGum, sealJaw = .020, .012

// sealTeeth is the upper teeth on the skull, pointing down, and the
// lower on each half of the mandible, pointing up.
func sealTeeth(made *sealBones) error {
	for _, side := range [...]float64{-1, 1} {
		name := sealSideName(side)
		err := sealToothRow(made, side, "upper", "skull", sealUpperTeeth,
			-sealGum, -1)
		if err != nil {
			return err
		}
		err = sealToothRow(made, side, "lower", "mandible-"+name,
			sealLowerTeeth, -sealGum-sealJaw, 1)
		if err != nil {
			return err
		}
	}
	return nil
}

// sealToothRow is one jaw's teeth on one side, rooted gum metres below
// the skull's axis and pointing along up, minus one down or one up.
func sealToothRow(made *sealBones, side float64, jaw, parent string,
	teeth []sealTooth, gum, up float64,
) error {
	condyle := sealSpinePoint(sealCondyle)
	snout := sealSpinePoint(sealCondyle + sealSkullLength)
	for _, tooth := range teeth {
		root := condyle.Add(snout.Sub(condyle).Scale(tooth.share)).
			Add(maths.V(0, gum, side*tooth.beside))
		tip := root.Add(maths.V(0, up*tooth.tall, 0))
		name := "tooth-" + sealSideName(side) + "-" + jaw + "-" + tooth.name
		err := made.add(name, parent, root, tip, tooth.wide,
			sealToothForm(root, tip, tooth.wide), skeleton.Fixed())
		if err != nil {
			return err
		}
	}
	return nil
}

// sealToothForm is a crown narrowing from its root to a blunt point.
func sealToothForm(root, tip maths.Vec, radius float64) skeleton.Loft {
	return skeleton.NewLoft(root, tip, maths.XAxis,
		skeleton.NewLoftSection(.15, radius*.8, radius, 0),
		skeleton.NewLoftSection(.5, radius*.7, radius*.85, 0),
		skeleton.NewLoftSection(.92, radius*.25, radius*.3, 0))
}

// sealSideName is the side a sign of Z is on: +Z is the right.
func sealSideName(side float64) string {
	if side > 0 {
		return "right"
	}
	return "left"
}

// sealRibs is a pair of ribs on each thoracic vertebra, each one bone
// curved round an ellipse inside the skin's section, the first ten
// reaching the breastbone and the rest floating shorter.
func sealRibs(made *sealBones) error {
	return Ribs(made.profile.Thoracic, func(rib int, side float64) error {
		head, tail, _ := made.ends(fmt.Sprintf("thoracic%d", rib))
		middle := head.Add(tail).Scale(.5)
		return sealRib(made, rib, side, middle)
	})
}

// The rib's shape: where it leaves the vertebra's side, the angle round
// the section its arc starts at, how much shorter each floating rib is,
// and its half sizes, radial and along the body, in metres. Chosen.
const (
	sealRibFoot  = .012
	sealRibTop   = 70 * math.Pi / 180
	sealRibFloat = .26
	sealRibThin  = .0025
	sealRibBroad = .006
)

// sealRib is one rib from beside its vertebra round to its lowest point,
// as the shared arc through four points of the ellipse: its cubic's
// control points are worked so the curve meets them at thirds.
func sealRib(made *sealBones, rib int, side float64, spine maths.Vec) error {
	bottom := -math.Pi / 2
	if rib > made.profile.SternalRibs {
		bottom += sealRibFloat * float64(rib-made.profile.SternalRibs)
	}
	section := sealProfileAt(spine.X)
	around := func(angle float64) maths.Vec {
		y := section.y + sealRibInset*section.height*math.Sin(angle)
		z := side * sealRibInset * section.width * math.Cos(angle)
		return maths.V(spine.X-sealRibSlope*(spine.Y-y), y, z)
	}
	foot := spine.Add(maths.V(0, 0, side*sealRibFoot))
	through := [4]maths.Vec{foot, around(sealRibTop),
		around(sealRibTop + (bottom-sealRibTop)/2), around(bottom)}
	arc := skeleton.NewArc(sealThroughFour(through), sealRibThin,
		sealRibBroad, maths.XAxis)
	return made.add(fmt.Sprintf("rib-%s-%d", sealSideName(side), rib),
		fmt.Sprintf("thoracic%d", rib), foot, through[3], sealRibThin,
		arc, skeleton.Fixed())
}

// sealThroughFour is the control points of the cubic that passes through
// four points at nought, a third, two thirds and one.
func sealThroughFour(at [4]maths.Vec) [4]maths.Vec {
	inner := func(a, b, c, d float64) maths.Vec {
		return at[0].Scale(a).Add(at[1].Scale(b)).Add(at[2].Scale(c)).
			Add(at[3].Scale(d)).Scale(1. / 6)
	}
	return [4]maths.Vec{at[0], inner(-5, 18, -9, 2), inner(2, -9, 18, -5),
		at[3]}
}

// sealSternum is the breastbone along the belly under the sternal ribs.
func sealSternum(made *sealBones) error {
	first, _, _ := made.ends("thoracic1")
	last, _, _ := made.ends(fmt.Sprintf("thoracic%d", made.profile.SternalRibs))
	parent := "thoracic1"
	var previous maths.Vec
	for index := 0; index <= made.profile.Sternebrae; index++ {
		share := float64(index) / float64(made.profile.Sternebrae)
		x := first.X + (last.X-first.X)*share
		section := sealProfileAt(x)
		point := maths.V(x, section.y-sealRibInset*section.height, 0)
		if index > 0 {
			name := fmt.Sprintf("sternebra%d", index)
			err := made.add(name, parent, previous, point, .006,
				skeleton.NewRod(previous, point, .008, .8),
				skeleton.Fixed()) // chosen
			if err != nil {
				return err
			}
			parent = name
		}
		previous = point
	}
	return nil
}

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

// sealDigits is how many digits a seal's flipper has, thumb to fifth.
const sealDigits = 5

// sealHand lays out one flipper's five digits. reach is how far each
// digit runs past the wrist, in metres, from the rear chord station to
// the front one. splay is half the fan's spread in metres. web is the
// share of its two digits' mean reach that the skin between them keeps,
// and thin the share of a digit's thickness that web keeps. All of them
// are chosen to read as a fan with a scalloped edge, not measured.
type sealHand struct {
	reach            [sealDigits]float64
	splay, web, thin float64
}

// foreHand has the thumb at the front station and the longest, each
// digit after it shorter.
var foreHand = sealHand{
	reach: [sealDigits]float64{.040, .048, .057, .065, .072},
	splay: .040, web: .70, thin: .45,
}

// hindHand has the first and fifth digits longest, which is the shape of
// a seal's trailing flipper.
var hindHand = sealHand{
	reach: [sealDigits]float64{.080, .062, .054, .060, .082},
	splay: .052, web: .68, thin: .45,
}

// sealDigitStations names each outline vertex of a flipper opening by
// the chord station it lies on, from zero at the rear to sealDigits-1 at
// the front. The outline runs along the rear edge, down the lower face,
// along the front edge and back up the upper face; every opening is
// sealDigits-1 rows long.
func sealDigitStations(loop int) []int {
	rows := sealDigits - 1
	columns := loop/2 - rows
	stations := make([]int, loop)
	for index := range stations {
		switch {
		case index < columns:
			stations[index] = 0
		case index < columns+rows:
			stations[index] = index - columns
		case index < 2*columns+rows:
			stations[index] = rows
		default:
			stations[index] = rows - (index - 2*columns - rows)
		}
	}
	return stations
}

// labels is each vertex of a digit ring: the digit it ends, or minus one
// where the vertex is web between two digits. Even entries are outline
// vertices and odd ones the web vertices inserted between them.
func (hand sealHand) labels(loop int) []int {
	stations := sealDigitStations(loop)
	labels := make([]int, 2*loop)
	for index := range labels {
		labels[index] = -1
		if index%2 == 0 {
			labels[index] = stations[index/2]
		}
	}
	return labels
}

// reachAt is how far a point at a chord station runs past the wrist. A
// station between two digits is web, which falls short of both.
func (hand sealHand) reachAt(station float64) float64 {
	low := int(math.Floor(station))
	if float64(low) == station {
		return hand.reach[low]
	}
	return hand.web * (hand.reach[low] + hand.reach[low+1]) / 2
}

// thickness is the share of a section's thickness kept at a station.
func (hand sealHand) thickness(station float64) float64 {
	if station == math.Floor(station) {
		return 1
	}
	return hand.thin
}

// spread is a point's offset in the flipper's chord, thickness and span
// axes, at a station and a fan share. The span axis points away from the
// body, so side signs it.
func (hand sealHand) spread(station, fan, side float64) maths.Vec {
	middle := float64(sealDigits-1) / 2
	chord := (station - middle) / middle
	return maths.V(chord*hand.splay*fan, 0, side*hand.reachAt(station)*fan)
}

// fanRing places one digit ring: every outline vertex of the opening and
// a web vertex halfway to the next. Digits spread across the chord and
// run out by their reach, and web is thinner and shorter between them.
func (rig *SealRig) fanRing(limb int, boundary []int, section sealBranchRing,
	hand sealHand,
) ([]int, maths.Vec) {
	joint := rig.limbs[limb]
	to := joint.root.Add(maths.V(section.dx, section.dy, joint.side*section.dz))
	stations := sealDigitStations(len(boundary))
	ring := make([]int, 0, 2*len(boundary))
	for index := range boundary {
		next := (index + 1) % len(boundary)
		for half := range 2 {
			base := rig.vertices[boundary[index]].point
			station := float64(stations[index])
			if half == 1 {
				base = base.Add(rig.vertices[boundary[next]].point).Scale(.5)
				station = (station + float64(stations[next])) / 2
			}
			point, spine := rig.sectionPoint(limb, base, section,
				hand.thickness(station), hand.spread(station, section.fan, joint.side))
			ring = append(ring, rig.add(point, spine, limb, section.influence))
		}
	}
	return ring, to
}
