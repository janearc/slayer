package manufacture

import (
	"fmt"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

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
func orcaForms(bones *skeleton.Skeleton) map[string]skeleton.BoneForm {
	forms := map[string]skeleton.BoneForm{}
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		if form := orcaFormOf(bones, bone); form != nil {
			forms[bone.Name()] = form
		}
	}
	return forms
}

// orcaFormOf is bone's form, or nil where it keeps the spindle.
func orcaFormOf(bones *skeleton.Skeleton,
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
		return orcaRibForm(bones, name)
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
func orcaRibForm(bones *skeleton.Skeleton, name string) skeleton.BoneForm {
	var n int
	side := orcaSides[0]
	if orcaSignOf(name) < 0 {
		side = orcaSides[1]
	}
	if _, err := fmt.Sscanf(name, "rib%d-"+side.name, &n); err != nil {
		return nil
	}
	return skeleton.NewArc(orcaRibCurve(bones, n, side), orcaRibWide,
		orcaRibDeep, orcaRibFlat)
}

// orcaScapulaForm is the shoulder blade: narrow at the joint, broad and
// thin at its top, its flat face to the flank.
func orcaScapulaForm(bone skeleton.Bone, side float64) skeleton.BoneForm {
	return skeleton.NewBlade(bone.Tail(), bone.Head(), orcaBladeBreadth,
		orcaBladeThick, orcaBladeFoot, maths.V(0, 0, side))
}
