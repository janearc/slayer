package whales

import (
	"fmt"
	"github.com/janearc/slayer/manufacture/geometry"
	"math"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// The vertebral column by region, its sources numbered as in
// doc/model/bioform/orca/SOURCES.md. Seven cervicals (the rule for
// mammals, and S1's Cervical 1 to 7), ten thoracics and twelve lumbars
// (S1's Thoracic 1 to 10 and Lumbar 1 to 12, and its ten ribs a side).
//
// Caudals make 53 in all, S2's total for one specimen; the caudal count
// is derived, not read. The relative lengths are chosen: short cervicals,
// the lumbars and first caudals longest, the fluke caudals small.
var orcaRegions = []skeleton.Region{
	skeleton.NewRegion("C", 7, .022, .022),
	skeleton.NewRegion("T", 10, .070, .085),
	skeleton.NewRegion("L", 12, .110, .120),
	skeleton.NewRegion("Ca", 24, .125, .035),
}

// Where the column runs, chosen: from the occipital condyle at the back
// of the skull to the last caudal at the flukes' notch, high in the body
// as a whale's spine is, falling to the middle of the tail stock.
const (
	orcaCondyleX     = 1.85  // x of the occipital condyle
	orcaColumnEnd    = -2.58 // x where the last caudal ends
	orcaSpineHigh    = .30   // spine's height in the chest, in half heights
	orcaSpineLow     = .04   // its height in the tail stock, in half heights
	orcaSpineFall    = -.60  // x behind which the spine begins to fall
	orcaSpineLowest  = -2.40 // x where it has fallen
	orcaCentrumShare = .12   // a vertebral body's half thickness, of trunk's
	orcaCentrumLeast = .008  // the thinnest a vertebral body is drawn
)

// orcaSpinePath is the column's path: the point a share of the way from
// the condyle to the last caudal, and a vertebral body's half thickness.
func (model *orcaModel) orcaSpinePath(share float64) (maths.Vec, float64) {
	x := orcaCondyleX + share*(orcaColumnEnd-orcaCondyleX)
	s := model.orcaTrunkProfile().at(x)
	lift := orcaSpineLow + (orcaSpineHigh-orcaSpineLow)*
		orcaStep(orcaSpineLowest, orcaSpineFall, x)
	thick := math.Max(orcaCentrumShare*math.Min(s.high, s.wide),
		orcaCentrumLeast)
	return maths.V(x, s.centre+lift*s.high, 0), thick
}

// The skull, chosen: a condylobasal length of 1.14 m, under a fifth of
// the body, with the braincase behind and the rostrum and the paired
// mandibles before it, so that the eye lies at the side of the
// braincase, in its orbit, above the gape's corner and ahead of the
// mandible's joint. No skull length from a source was reachable.
const (
	orcaCraniumFront = 2.30 // x where the braincase gives way to the rostrum
	orcaCraniumThick = .16  // half thickness of the braincase at the back
	orcaRostrumTip   = 2.99 // x of the rostrum's tip
	orcaRostrumRoot  = .05  // height of the rostrum's root
	orcaRostrumLow   = -.07 // height of its tip
	orcaRostrumThick = .08  // its half thickness at the root
	orcaJawBack      = 1.98 // x of the mandible's joint
	orcaJawThick     = .05  // half thickness of a mandible at its joint
	orcaFineThick    = .02  // half thickness of a slender bone's tip
	orcaCraniumTaper = .8   // the braincase's thickness at its front, of back
	orcaRostrumSunk  = .02  // how far the rostrum's root lies in the braincase
)

// orcaSkull adds the braincase, at the root of the skeleton, and the
// rostrum and the two mandibles jointed to it.
func (model *orcaModel) orcaSkull(bones *skeleton.Skeleton) error {
	condyle, _ := model.orcaSpinePath(0)
	front := maths.V(orcaCraniumFront, condyle.Y, 0)
	if err := bones.Add("cranium", "", condyle, front, orcaCraniumThick,
		orcaCraniumTaper*orcaCraniumThick); err != nil {
		return err
	}
	if err := bones.Add("rostrum", "cranium",
		maths.V(orcaCraniumFront-orcaRostrumSunk, orcaRostrumRoot, 0),
		maths.V(orcaRostrumTip, orcaRostrumLow, 0), orcaRostrumThick,
		orcaFineThick); err != nil {
		return err
	}
	for _, side := range orcaSides {
		err := bones.Add("mandible-"+side.name, "cranium",
			orcaMirror(orcaMandibleBack, side.sign),
			orcaMirror(orcaMandibleTip, side.sign), orcaJawThick,
			orcaFineThick)
		if err != nil {
			return err
		}
	}
	return nil
}

// The teeth in the bone: twelve in each half of each jaw, within S3's
// "10-12 large, recurved teeth in each half of both jaws" and S4's 10 to
// 14 a quadrant (SOURCES.md). Their spacing, length and lean are chosen.
const (
	orcaToothCount  = 12   // teeth in each half of each jaw
	orcaUpperFirst  = .12  // share of the rostrum to the rearmost upper tooth
	orcaUpperLast   = .90  // share to the foremost
	orcaLowerFirst  = .40  // share of the mandible to the rearmost lower tooth
	orcaLowerLast   = .96  // share to the foremost
	orcaToothLong   = .055 // length of a tooth's crown and root together
	orcaToothThick  = .011 // half thickness of a tooth at its root
	orcaToothCurve  = .30  // how far a tooth leans back, of its length
	orcaToothOutset = .85  // how far out on the rostrum the upper row lies
	orcaToothSunk   = .6   // how far into its bone's depth a root is set
)

// orcaToothRow adds a row of teeth on jaw, the root of the tooth at
// share s of the row at root(s), each pointing toward rise (up for the
// lower row, down for the upper), named after the side and row, as
// right-lower-tooth-3.
func orcaToothRow(bones *skeleton.Skeleton, jaw, row string, side orcaSide,
	rise float64, root func(float64) maths.Vec,
) error {
	for n := 0; n < orcaToothCount; n++ {
		base := root(float64(n) / float64(orcaToothCount-1))
		tip := base.Add(maths.V(-orcaToothCurve*orcaToothLong,
			rise*orcaToothLong, 0))
		name := fmt.Sprintf("%s-%s-tooth-%d", side.name, row, n+1)
		if err := bones.Add(name, jaw, base, tip, orcaToothThick,
			orcaToothThick/3); err != nil {
			return err
		}
	}
	return nil
}

// orcaUpperRoot is where the upper tooth at share s of side's row is
// rooted: along the rostrum's outer edge, on its underside.
func orcaUpperRoot(rostrum skeleton.Profile,
	side float64) func(float64) maths.Vec {
	return func(s float64) maths.Vec {
		t := orcaUpperFirst + s*(orcaUpperLast-orcaUpperFirst)
		wide, deep := rostrum.Size(t)
		return rostrum.At(t).Add(maths.V(0, -orcaToothSunk*deep,
			side*orcaToothOutset*wide))
	}
}

// orcaLowerRoot is where the lower tooth at share s of a row is rooted:
// along the top of the mandible.
func orcaLowerRoot(mandible skeleton.Arc) func(float64) maths.Vec {
	return func(s float64) maths.Vec {
		t := orcaLowerFirst + s*(orcaLowerLast-orcaLowerFirst)
		return mandible.At(t).Add(maths.V(0, orcaToothSunk*orcaMandibleDeep,
			0))
	}
}

// orcaToothRows adds the four rows: up from each mandible, down from
// each side of the rostrum, set in the bones' forms.
func orcaToothRows(bones *skeleton.Skeleton) error {
	rostrum := orcaRostrumForm(bones.Bone(orcaMust(bones, "rostrum")))
	for _, side := range orcaSides {
		if err := orcaToothRow(bones, "mandible-"+side.name, "lower", side,
			1, orcaLowerRoot(orcaMandibleForm(side.sign))); err != nil {
			return err
		}
		if err := orcaToothRow(bones, "rostrum", "upper", side, -1,
			orcaUpperRoot(rostrum, side.sign)); err != nil {
			return err
		}
	}
	return nil
}

// orcaSide is one side of the animal: its name and the sign of z there.
type orcaSide struct {
	name string
	sign float64
}

// orcaSides are the right (z positive) and the left.
var orcaSides = []orcaSide{{"right", 1}, {"left", -1}}

// orcaSkeleton is the animal's rest skeleton: skull, column, the
// processes of the vertebrae, ribs and sternum, the flippers' bones and
// the pelvic vestiges. There is no hind limb, and no bone in the dorsal
// fin or the flukes, which are fibrous.
func (model *orcaModel) orcaSkeleton() (*skeleton.Skeleton, error) {
	model.skeletonOnce.Do(func() {
		bones := skeleton.New()
		for _, add := range []func(*skeleton.Skeleton) error{model.orcaSkull,
			orcaToothRows, model.orcaColumn, model.orcaProcesses, model.orcaRibs,
			orcaFlippers, orcaPelvis} {
			if err := add(bones); err != nil {
				model.skeletonErr = fmt.Errorf("orca skeleton: %w", err)
				return
			}
		}
		model.skeleton = bones
	})
	return model.skeleton, model.skeletonErr
}

// orcaColumn adds the vertebrae, the first jointed to the skull.
func (model *orcaModel) orcaColumn(bones *skeleton.Skeleton) error {
	return bones.Column("cranium", orcaRegions, model.orcaSpinePath)
}

// orcaBoneMaterial is dry bone: pale, matte, its colour chosen.
func orcaBoneMaterial() *thing.Material {
	return &thing.Material{Name: "orca bone",
		Colour: maths.RGB(.62, .58, .50)}
}

// orcaBones is the animal's skeleton alone, at rest, every bone
// closed in its own form: a study of what the skin lies over.
func (model *orcaModel) orcaBones(at maths.Subspace) ([]thing.Thing, error) {
	bones, err := model.orcaSkeleton()
	if err != nil {
		return nil, err
	}
	points, faces := bones.Draw(bones.Frames(skeleton.Pose{}),
		model.orcaForms(bones))
	normals := geometry.UnitNormals(points, faces)
	drawn, err := mesh.New(points, faces, normals, true)
	if err != nil {
		return nil, err
	}
	return []thing.Thing{{Shape: drawn, Material: orcaBoneMaterial(),
		Motion: thing.Still(at)}}, nil
}

// The shapes of the skull's bones, chosen, as half sizes at evenly
// spaced stations from back to front: a braincase broad at the
// zygomatic arches behind the orbits, a rostrum broad at its root and
// flat, tapering to a blunt tip, as a killer whale's is short and broad
// among dolphins (S5 of doc/model/bioform/orca/SOURCES.md: "the snout
// is blunt").
//
// Every size is chosen; no skull measurement was reachable.
// The mandibles' points are chosen, the tips orcaJawShort behind the
// rostrum's.
var (
	orcaCraniumWide  = []float64{.12, .25, .27, .24, .20}
	orcaCraniumDeep  = []float64{.10, .17, .18, .15, .10}
	orcaRostrumWide  = []float64{.20, .17, .13, .09, .05}
	orcaRostrumDeep  = []float64{.07, .06, .045, .03, .02}
	orcaUp           = maths.V(0, 1, 0)
	orcaRibFlat      = maths.V(1, 0, 0) // a rib's broad face, along the body
	orcaMandibleBack = maths.V(orcaJawBack, -.17, .27)
	orcaMandibleBend = maths.V(2.30, -.23, .24)
	orcaMandibleFore = maths.V(2.65, -.17, .14)
	orcaMandibleTip  = maths.V(orcaRostrumTip-orcaJawShort, -.12, .03)
)

// orcaJawShort is how far the mandibles' tips lie behind the rostrum's.
const orcaJawShort = .02

// The sections of the other bones, chosen, in metres or as shares.
const (
	orcaMandibleWide = .022 // half thickness of a mandible across
	orcaMandibleDeep = .055 // half its height
	orcaRibWide      = .010 // half a rib's thickness, in and out
	orcaRibDeep      = .022 // half its breadth, along the body
	orcaPlateThick   = .006 // half thickness of a spine, process or chevron
	orcaSpineBreadth = .6   // a neural spine's breadth, of its vertebra's length
	orcaWingBreadth  = .5   // a transverse process's, of its vertebra's length
	orcaShaftShare   = .75  // the share of its ends a long bone's shaft keeps
	orcaDiscShare    = .85  // the share a vertebral body's middle keeps
	orcaBladeBreadth = .17  // half the shoulder blade's breadth at its top
	orcaBladeThick   = .012 // half its thickness
	orcaBladeFoot    = .25  // the share of its breadth at the shoulder joint
	orcaSternumWide  = .07  // half the sternum's breadth
	orcaSternumThick = .015 // half its thickness
)

// orcaCraniumForm is the braincase from the condyle forward.
func orcaCraniumForm(cranium skeleton.Bone) skeleton.Profile {
	return skeleton.NewProfile(cranium.Head(), cranium.Tail(), orcaUp,
		orcaCraniumWide, orcaCraniumDeep)
}

// orcaRostrumForm is the rostrum from its root to its tip.
func orcaRostrumForm(rostrum skeleton.Bone) skeleton.Profile {
	return skeleton.NewProfile(rostrum.Head(), rostrum.Tail(), orcaUp,
		orcaRostrumWide, orcaRostrumDeep)
}

// orcaMandibleForm is one side's mandible: a tall, thin jaw bowed down
// under the eye and converging on its fellow at the tip.
func orcaMandibleForm(side float64) skeleton.Arc {
	return skeleton.NewArc([4]maths.Vec{orcaMirror(orcaMandibleBack, side),
		orcaMirror(orcaMandibleBend, side), orcaMirror(orcaMandibleFore, side),
		orcaMirror(orcaMandibleTip, side)}, orcaMandibleWide,
		orcaMandibleDeep, orcaUp)
}

// orcaForms is the shape of every bone that has one more than a spindle:
// the skull's bones, the vertebrae, their processes as plates, the ribs
// as arcs, the sternum and shoulder blades as blades, the limb bones as
// rods. Teeth keep the spindle, a cone at each end.
func (model *orcaModel) orcaForms(bones *skeleton.Skeleton) map[string]skeleton.BoneForm {
	forms := map[string]skeleton.BoneForm{}
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		if form := model.orcaFormOf(bones, bone); form != nil {
			forms[bone.Name()] = form
		}
	}
	return forms
}

// orcaFormOf is bone's form, or nil where it keeps the spindle.
func (model *orcaModel) orcaFormOf(bones *skeleton.Skeleton,
	bone skeleton.Bone) skeleton.BoneForm {
	name := bone.Name()
	switch {
	case name == "cranium":
		return orcaCraniumForm(bone)
	case name == "rostrum":
		return orcaRostrumForm(bone)
	case strings.HasPrefix(name, "mandible-"):
		return orcaMandibleForm(orcaSignOf(name))
	case strings.HasPrefix(name, "rib") && !strings.HasSuffix(name, "sternal"):
		return model.orcaRibForm(bones, name)
	case strings.HasPrefix(name, "scapula-"):
		return orcaScapulaForm(bone, orcaSignOf(name))
	case name == "sternum":
		return skeleton.NewBlade(bone.Head(), bone.Tail(), orcaSternumWide,
			orcaSternumThick, 1, orcaUp)
	case strings.Contains(name, "tooth"):
		return nil
	}
	return orcaColumnFormOf(bones, bone)
}

// orcaColumnFormOf is the form of a vertebra or one of its processes, or
// a rod for any other bone.
func orcaColumnFormOf(bones *skeleton.Skeleton,
	bone skeleton.Bone) skeleton.BoneForm {
	name := bone.Name()
	thick, _ := bone.Thickness()
	if region, _ := orcaVertebraOf(name); region != "" {
		return skeleton.NewRod(bone.Head(), bone.Tail(), thick, orcaDiscShare)
	}
	if bone.Parent() < 0 {
		return skeleton.NewRod(bone.Head(), bone.Tail(), thick, orcaShaftShare)
	}
	length := bones.Bone(bone.Parent()).Length()
	switch {
	case strings.HasSuffix(name, "-spine"), strings.HasSuffix(name, "-chevron"):
		return skeleton.NewBlade(bone.Head(), bone.Tail(),
			orcaSpineBreadth*length/2, orcaPlateThick, 1, maths.V(0, 0, 1))
	case strings.HasSuffix(name, "-left"), strings.HasSuffix(name, "-right"):
		return skeleton.NewBlade(bone.Head(), bone.Tail(),
			orcaWingBreadth*length/2, orcaPlateThick, 1, orcaUp)
	}
	return skeleton.NewRod(bone.Head(), bone.Tail(), thick, orcaShaftShare)
}

// orcaSignOf is the sign of z on the side a bone's name says.
func orcaSignOf(name string) float64 {
	if strings.Contains(name, "left") {
		return -1
	}
	return 1
}

// orcaRibForm is the rib named name as an arc through its curve.
func (model *orcaModel) orcaRibForm(bones *skeleton.Skeleton, name string) skeleton.BoneForm {
	var n int
	side := orcaSides[0]
	if orcaSignOf(name) < 0 {
		side = orcaSides[1]
	}
	if _, err := fmt.Sscanf(name, "rib%d-"+side.name, &n); err != nil {
		return nil
	}
	return skeleton.NewArc(model.orcaRibCurve(bones, n, side), orcaRibWide,
		orcaRibDeep, orcaRibFlat)
}

// orcaScapulaForm is the shoulder blade: narrow at the joint, broad and
// thin at its top, its flat face to the flank.
func orcaScapulaForm(bone skeleton.Bone, side float64) skeleton.BoneForm {
	return skeleton.NewBlade(bone.Tail(), bone.Head(), orcaBladeBreadth,
		orcaBladeThick, orcaBladeFoot, maths.V(0, 0, side))
}

// The flipper's bones, laid out inside the paddle of orcaFlipper. The
// phalanges per digit are those the specimen record S1 of
// doc/model/bioform/orca/SOURCES.md lists for its right manus: digit
// one one phalanx, digit two four, digit three three, digit four two,
// digit five one. That specimen is a juvenile male.
//
// Every length, spread and splay is chosen, so that the hand fills the
// paddle as a cetacean's does, inside its outline.
const (
	orcaShoulderAlong = .10  // the shoulder joint, along the paddle from its root
	orcaHumerusLong   = .14  // the humerus
	orcaForearmLong   = .15  // the radius and the ulna
	orcaWristLong     = .06  // the carpals
	orcaHumerusThick  = .045 // half thickness of the humerus at the shoulder
	orcaScapulaThick  = .06  // half thickness of the shoulder blade's broad end
)

// orcaScapulaTop is where the shoulder blade's broad end lies on the
// right flank, above and behind the shoulder, outside the ribs; chosen.
var orcaScapulaTop = maths.V(1.62, .08, .44)

// orcaHandFrame is the right paddle's shoulder joint, the direction
// along its middle (trailing as the paddle's outline does) and the
// direction across it toward the leading edge.
func orcaHandFrame() (shoulder, along, across maths.Vec) {
	root := maths.V(orcaFlipperRootX, orcaFlipperRootY, orcaFlipperRootZ)
	shoulder = root.Add(orcaFlipperSpan.Scale(orcaShoulderAlong))
	along = orcaFlipperSpan.Sub(orcaFlipperChordway.Scale(
		orcaFlipperSweep)).Unit()
	across = orcaFlipperChordway.Sub(along.Scale(
		orcaFlipperChordway.Dot(along))).Unit()
	return shoulder, along, across
}

// orcaArmAndDigits is hand with the arm and five digits, from the
// leading edge to the trailing: each its knuckle's place across the
// wrist, its splay in radians, its metacarpal and then its phalanges.
func orcaArmAndDigits(hand skeleton.Hand) skeleton.Hand {
	return hand.Arm(orcaHumerusLong, orcaForearmLong, orcaWristLong).
		Digit(.10, .25, .07, .06).
		Digit(.045, .06, .10, .10, .09, .08, .06).
		Digit(0, -.02, .10, .10, .09, .08).
		Digit(-.05, -.12, .09, .08, .07).
		Digit(-.10, -.28, .07, .06)
}

// orcaMirror is p reflected across the midline when side is minus one.
func orcaMirror(p maths.Vec, side float64) maths.Vec {
	return maths.V(p.X, p.Y, side*p.Z)
}

// orcaHand is one side's flipper bones; the left is the right reflected.
func orcaHand(side float64) skeleton.Hand {
	shoulder, along, across := orcaHandFrame()
	return orcaArmAndDigits(skeleton.NewHand(orcaMirror(shoulder, side),
		orcaMirror(along, side), orcaMirror(across, side),
		orcaHumerusThick))
}

// orcaFlippers adds each side's shoulder blade, jointed to the second
// thoracic, and the flipper's bones jointed to the shoulder blade.
func orcaFlippers(bones *skeleton.Skeleton) error {
	shoulder, _, _ := orcaHandFrame()
	for _, side := range orcaSides {
		top := orcaMirror(orcaScapulaTop, side.sign)
		joint := orcaMirror(shoulder, side.sign)
		blade := "scapula-" + side.name
		if err := bones.Add(blade, "T2", top, joint, orcaScapulaThick,
			orcaHumerusThick); err != nil {
			return err
		}
		if err := bones.Hand(side.name+"-", blade,
			orcaHand(side.sign)); err != nil {
			return err
		}
	}
	return nil
}

// The ribs. Ten a side, one on each thoracic, as the specimen record of
// doc/model/bioform/orca/SOURCES.md has (Rib First to Rib 10, left).
//
// The first five a side reach the sternum by a sternal rib; that record
// lists eight sternal rib elements, and five pairs is chosen from them.
// The shape is chosen: each rib follows the trunk's own section a little
// inside it, from the spine down round the flank, its lower end swept
// back.
const (
	orcaRibInside  = .78  // share of the trunk's section the ribs lie at
	orcaRibStart   = .60  // angle down from the back a rib meets it, radians
	orcaRibLongest = 2.35 // angle where a sternal rib's bony part ends
	orcaRibShorten = .12  // angle each floating rib ends short of the last
	orcaRibSternal = 5    // ribs a side that reach the sternum
	orcaRibBow     = 1.06 // how far a rib's curve bows out past the section
	orcaRibSweep   = .18  // how far back a rib's lower end lies
	orcaRibThick   = .018 // half thickness of a rib at the spine
	orcaRibApart   = 1.8  // where it leaves the spine, in centrum thicknesses
	orcaSternumLow = .74  // depth of the sternum, in the trunk's half heights
	orcaSternumGap = .05  // a sternal rib's lower end from the midline
)

// orcaRibPoint is the point on the rib section at x, angle down from the
// back, on side: the trunk's ellipse drawn in to the ribs.
func (model *orcaModel) orcaRibPoint(x, angle, side float64) maths.Vec {
	s := model.orcaTrunkProfile().at(x)
	return maths.V(x, s.centre+orcaRibInside*s.high*math.Cos(angle),
		side*orcaRibInside*s.wide*math.Sin(angle))
}

// orcaRibEnd is the angle where the bony part of rib n (from one) ends.
func orcaRibEnd(n int) float64 {
	if n <= orcaRibSternal {
		return orcaRibLongest
	}
	return orcaRibLongest - orcaRibShorten*float64(n-orcaRibSternal)
}

// orcaRibs adds both sides' ribs, the sternum and the sternal ribs.
func (model *orcaModel) orcaRibs(bones *skeleton.Skeleton) error {
	if err := model.orcaSternum(bones); err != nil {
		return err
	}
	for n := 1; orcaHas(bones, fmt.Sprintf("T%d", n)); n++ {
		for _, side := range orcaSides {
			if err := model.orcaRib(bones, n, side); err != nil {
				return err
			}
		}
	}
	return nil
}

// orcaRibCurve is the four points of rib n's curve on side, the middle
// two the control points of a cubic: from the vertebra's transverse
// process, out to the section and round it to the rib's lower end, the
// middle bowed out so the cubic follows the section.
func (model *orcaModel) orcaRibCurve(bones *skeleton.Skeleton, n int,
	side orcaSide) [4]maths.Vec {
	vertebra := bones.Bone(orcaMust(bones, fmt.Sprintf("T%d", n)))
	thick, _ := vertebra.Thickness()
	head := vertebra.Head().Add(maths.V(0, 0, side.sign*orcaRibApart*thick))
	end := orcaRibEnd(n)
	curve := [4]maths.Vec{head}
	for k := 1; k <= 3; k++ {
		share := float64(k) / 3
		angle := orcaRibStart + share*(end-orcaRibStart)
		curve[k] = model.orcaRibPoint(head.X-orcaRibSweep*share, angle, side.sign)
	}
	for k := 1; k <= 2; k++ {
		s := model.orcaTrunkProfile().at(curve[k].X)
		centre := maths.V(curve[k].X, s.centre, 0)
		curve[k] = centre.Add(curve[k].Sub(centre).Scale(orcaRibBow))
	}
	return curve
}

// orcaRib adds rib n on side as one bone from the vertebra's transverse
// process to its lower end, and for a sternal rib a sternal rib on from
// there to the sternum.
func (model *orcaModel) orcaRib(bones *skeleton.Skeleton, n int, side orcaSide) error {
	curve := model.orcaRibCurve(bones, n, side)
	name := fmt.Sprintf("rib%d-%s", n, side.name)
	if err := bones.Add(name, fmt.Sprintf("T%d", n), curve[0], curve[3],
		orcaRibThick, .6*orcaRibThick); err != nil {
		return err
	}
	if n > orcaRibSternal {
		return nil
	}
	foot := curve[3]
	s := model.orcaTrunkProfile().at(foot.X)
	low := maths.V(foot.X-.03, s.centre-orcaSternumLow*s.high,
		side.sign*orcaSternumGap)
	return bones.Add(name+"-sternal", name, foot, low, .6*orcaRibThick,
		.6*orcaRibThick)
}

// orcaSternum adds the breastbone on the belly's midline, under the
// sternal ribs, jointed to the first thoracic.
func (model *orcaModel) orcaSternum(bones *skeleton.Skeleton) error {
	first := bones.Bone(orcaMust(bones, "T1")).Head()
	last := bones.Bone(orcaMust(bones,
		fmt.Sprintf("T%d", orcaRibSternal))).Head()
	front := model.orcaTrunkProfile().at(first.X)
	back := model.orcaTrunkProfile().at(last.X - orcaRibSweep)
	return bones.Add("sternum", "T1",
		maths.V(first.X, front.centre-orcaSternumLow*front.high, 0),
		maths.V(last.X-orcaRibSweep, back.centre-orcaSternumLow*back.high,
			0), 1.6*orcaRibThick, orcaRibThick)
}

// orcaMust is the place of the bone named name, which this package's
// order of building puts there before it is asked for. A name it does
// not find is a fault in that code: it stops, naming the bone, rather
// than becoming the skull's place without a word.
func orcaMust(bones *skeleton.Skeleton, name string) int {
	at, ok := bones.Find(name)
	if !ok {
		panic("orca skeleton: no bone named " + name)
	}
	return at
}

// orcaHas is whether the skeleton has a bone named name.
func orcaHas(bones *skeleton.Skeleton, name string) bool {
	_, ok := bones.Find(name)
	return ok
}

// The processes of the vertebrae, chosen: a neural spine above each
// thoracic, lumbar and the first fourteen caudals, sloped back; a
// transverse process each side of each lumbar and the first ten caudals;
// a chevron below the second to the fourteenth caudal, thirteen, as many
// as the specimen record of SOURCES.md shows (Chevron 1 to 13).
const (
	orcaSpineReach    = .55  // a neural spine's height, of the room above it
	orcaSpineSlope    = .30  // how far back it leans, of its height
	orcaProcessReach  = .45  // a transverse process's reach, of the half width
	orcaChevronReach  = .35  // a chevron's depth, of the room below it
	orcaProcessThick  = .012 // half thickness of a process at its root
	orcaSpinedCaudals = 14   // caudals that carry a neural spine
	orcaWingedCaudals = 10   // caudals that carry transverse processes
	orcaChevronFirst  = 2    // the first caudal with a chevron
	orcaChevronLast   = 14   // the last
)

// orcaProcesses adds the spines, transverse processes and chevrons.
func (model *orcaModel) orcaProcesses(bones *skeleton.Skeleton) error {
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		region, n := orcaVertebraOf(bone.Name())
		if region == "" {
			continue
		}
		if err := model.orcaVertebralProcesses(bones, bone, region,
			n); err != nil {
			return err
		}
	}
	return nil
}

// orcaVertebraOf is the region and number of the vertebra named name, or
// an empty region if name is not a vertebra's.
func orcaVertebraOf(name string) (string, int) {
	for _, region := range []string{"Ca", "C", "T", "L"} {
		var n int
		_, err := fmt.Sscanf(name, region+"%d", &n)
		if err == nil && fmt.Sprintf("%s%d", region, n) == name {
			return region, n
		}
	}
	return "", 0
}

// orcaVertebralProcesses adds the processes vertebra n of region has.
func (model *orcaModel) orcaVertebralProcesses(bones *skeleton.Skeleton, vertebra skeleton.Bone,
	region string, n int,
) error {
	caudal := region == "Ca"
	if region == "T" || region == "L" || caudal && n <= orcaSpinedCaudals {
		if err := model.orcaNeuralSpine(bones, vertebra); err != nil {
			return err
		}
	}
	if region == "L" || caudal && n <= orcaWingedCaudals {
		if err := model.orcaTransverse(bones, vertebra); err != nil {
			return err
		}
	}
	if caudal && n >= orcaChevronFirst && n <= orcaChevronLast {
		return model.orcaChevron(bones, vertebra)
	}
	return nil
}

// orcaMiddle is the middle of a vertebra's body and its half thickness.
func orcaMiddle(vertebra skeleton.Bone) (maths.Vec, float64) {
	head, tail := vertebra.Thickness()
	return vertebra.Head().Add(vertebra.Tail()).Scale(.5), (head + tail) / 2
}

// orcaNeuralSpine adds the spine above vertebra, leaning back.
func (model *orcaModel) orcaNeuralSpine(bones *skeleton.Skeleton, vertebra skeleton.Bone) error {
	middle, thick := orcaMiddle(vertebra)
	s := model.orcaTrunkProfile().at(middle.X)
	root := middle.Add(maths.V(0, thick, 0))
	rise := orcaSpineReach * (s.centre + s.high - root.Y)
	tip := root.Add(maths.V(-orcaSpineSlope*rise, rise, 0))
	return bones.Add(vertebra.Name()+"-spine", vertebra.Name(), root, tip,
		orcaProcessThick, .6*orcaProcessThick)
}

// orcaTransverse adds a transverse process each side of vertebra.
func (model *orcaModel) orcaTransverse(bones *skeleton.Skeleton, vertebra skeleton.Bone) error {
	middle, thick := orcaMiddle(vertebra)
	s := model.orcaTrunkProfile().at(middle.X)
	for _, side := range orcaSides {
		root := middle.Add(maths.V(0, 0, side.sign*thick))
		reach := math.Max(orcaProcessReach*s.wide-thick, thick)
		tip := root.Add(maths.V(0, 0, side.sign*reach))
		if err := bones.Add(vertebra.Name()+"-"+side.name, vertebra.Name(),
			root, tip, orcaProcessThick, .6*orcaProcessThick); err != nil {
			return err
		}
	}
	return nil
}

// orcaChevron adds the chevron below vertebra.
func (model *orcaModel) orcaChevron(bones *skeleton.Skeleton, vertebra skeleton.Bone) error {
	middle, thick := orcaMiddle(vertebra)
	s := model.orcaTrunkProfile().at(middle.X)
	root := middle.Sub(maths.V(0, thick, 0))
	drop := orcaChevronReach * (root.Y - (s.centre - s.high))
	tip := root.Add(maths.V(-orcaSpineSlope*drop, -drop, 0))
	return bones.Add(vertebra.Name()+"-chevron", vertebra.Name(), root, tip,
		orcaProcessThick, .6*orcaProcessThick)
}

// The pelvic vestiges: a slender bone each side in the body wall near
// the genital slit, jointed to nothing. A cetacean keeps them; there is
// no hind limb. Place and size chosen.
var (
	orcaPelvisFront = maths.V(-.30, -.36, .07)
	orcaPelvisBack  = maths.V(-.52, -.40, .05)
)

// orcaPelvis adds the two pelvic vestiges.
func orcaPelvis(bones *skeleton.Skeleton) error {
	for _, side := range orcaSides {
		front := orcaMirror(orcaPelvisFront, side.sign)
		back := orcaMirror(orcaPelvisBack, side.sign)
		if err := bones.Add("pelvis-"+side.name, "", front, back,
			orcaProcessThick, .6*orcaProcessThick); err != nil {
			return err
		}
	}
	return nil
}
