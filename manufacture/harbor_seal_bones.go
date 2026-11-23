package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// The harbor seal's vertebrae and ribs by region. Where a count is
// sourced the comment quotes it; the others are chosen.
const (
	// sealCervicals is seven, the mammalian count; chosen, as no seal
	// source giving it was reached.
	sealCervicals = 7
	// sealThoracics: "In pinnipeds there are typically 15 thoracic
	// (rib-bearing) vertebrae and five lumbar vertebrae", Pierce, Clack
	// and Hutchinson 2011, Journal of Anatomy 219:502, PMC3196755.
	sealThoracics = 15
	// sealLumbars is five, from the same sentence.
	sealLumbars = 5
	// sealCaudals is twelve; chosen, as no source was reached.
	sealCaudals = 12
	// sealSternalRibs is how many ribs reach the sternum; chosen.
	sealSternalRibs = 10
	// sealSternebrae is how many bones make the sternum; chosen.
	sealSternebrae = 8
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

// Each region's vertebrae's length, first and last, as a share of a
// thoracic vertebra's: the lumbars longer, the sacrum one fused bone of
// three, the tail shortening to its tip. Chosen.
var sealTrunkRegions = []skeleton.Region{
	skeleton.NewRegion("thoracic", sealThoracics, 1, 1),
	skeleton.NewRegion("lumbar", sealLumbars, 1.25, 1.25),
	skeleton.NewRegion("sacral", 1, 2.75, 2.75),
	skeleton.NewRegion("caudal", sealCaudals, .85, .57),
}

// sealNeckRegion is the cervicals, laid by the shared column from the
// thorax forward: neck1 is the seventh cervical and neck7 the atlas.
var sealNeckRegion = []skeleton.Region{
	skeleton.NewRegion("neck", sealCervicals, 1, 1),
}

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
	bones  *skeleton.Skeleton
	joints skeleton.Joints
	forms  map[string]skeleton.BoneForm
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
	made := &sealBones{bones: skeleton.New(),
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
	err := made.bones.Column("", sealTrunkRegions,
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
	err := made.bones.Column("thoracic1", sealNeckRegion,
		sealPath(sealFirstThoracic, sealCondyle))
	if err != nil {
		return err
	}
	made.dressColumn("neck")
	condyle := sealSpinePoint(sealCondyle)
	snout := sealSpinePoint(sealCondyle + sealSkullLength)
	err = made.add("skull", fmt.Sprintf("neck%d", sealCervicals), condyle,
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
	for rib := 1; rib <= sealThoracics; rib++ {
		head, tail, _ := made.ends(fmt.Sprintf("thoracic%d", rib))
		middle := head.Add(tail).Scale(.5)
		for _, side := range [...]float64{-1, 1} {
			if err := sealRib(made, rib, side, middle); err != nil {
				return err
			}
		}
	}
	return nil
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
	if rib > sealSternalRibs {
		bottom += sealRibFloat * float64(rib-sealSternalRibs)
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
	last, _, _ := made.ends(fmt.Sprintf("thoracic%d", sealSternalRibs))
	parent := "thoracic1"
	var previous maths.Vec
	for index := 0; index <= sealSternebrae; index++ {
		share := float64(index) / float64(sealSternebrae)
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
