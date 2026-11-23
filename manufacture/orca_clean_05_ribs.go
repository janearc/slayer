package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

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
func orcaRibPoint(x, angle, side float64) maths.Vec {
	s := orcaTrunkProfile().at(x)
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
func orcaRibs(bones *skeleton.Skeleton) error {
	if err := orcaSternum(bones); err != nil {
		return err
	}
	for n := 1; orcaHas(bones, fmt.Sprintf("T%d", n)); n++ {
		for _, side := range orcaSides {
			if err := orcaRib(bones, n, side); err != nil {
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
func orcaRibCurve(bones *skeleton.Skeleton, n int,
	side orcaSide) [4]maths.Vec {
	vertebra := bones.Bone(orcaMust(bones, fmt.Sprintf("T%d", n)))
	thick, _ := vertebra.Thickness()
	head := vertebra.Head().Add(maths.V(0, 0, side.sign*orcaRibApart*thick))
	end := orcaRibEnd(n)
	curve := [4]maths.Vec{head}
	for k := 1; k <= 3; k++ {
		share := float64(k) / 3
		angle := orcaRibStart + share*(end-orcaRibStart)
		curve[k] = orcaRibPoint(head.X-orcaRibSweep*share, angle, side.sign)
	}
	for k := 1; k <= 2; k++ {
		s := orcaTrunkProfile().at(curve[k].X)
		centre := maths.V(curve[k].X, s.centre, 0)
		curve[k] = centre.Add(curve[k].Sub(centre).Scale(orcaRibBow))
	}
	return curve
}

// orcaRib adds rib n on side as one bone from the vertebra's transverse
// process to its lower end, and for a sternal rib a sternal rib on from
// there to the sternum.
func orcaRib(bones *skeleton.Skeleton, n int, side orcaSide) error {
	curve := orcaRibCurve(bones, n, side)
	name := fmt.Sprintf("rib%d-%s", n, side.name)
	if err := bones.Add(name, fmt.Sprintf("T%d", n), curve[0], curve[3],
		orcaRibThick, .6*orcaRibThick); err != nil {
		return err
	}
	if n > orcaRibSternal {
		return nil
	}
	foot := curve[3]
	s := orcaTrunkProfile().at(foot.X)
	low := maths.V(foot.X-.03, s.centre-orcaSternumLow*s.high,
		side.sign*orcaSternumGap)
	return bones.Add(name+"-sternal", name, foot, low, .6*orcaRibThick,
		.6*orcaRibThick)
}

// orcaSternum adds the breastbone on the belly's midline, under the
// sternal ribs, jointed to the first thoracic.
func orcaSternum(bones *skeleton.Skeleton) error {
	first := bones.Bone(orcaMust(bones, "T1")).Head()
	last := bones.Bone(orcaMust(bones,
		fmt.Sprintf("T%d", orcaRibSternal))).Head()
	front := orcaTrunkProfile().at(first.X)
	back := orcaTrunkProfile().at(last.X - orcaRibSweep)
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
func orcaProcesses(bones *skeleton.Skeleton) error {
	for i := 0; i < bones.Len(); i++ {
		bone := bones.Bone(i)
		region, n := orcaVertebraOf(bone.Name())
		if region == "" {
			continue
		}
		if err := orcaVertebralProcesses(bones, bone, region,
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
func orcaVertebralProcesses(bones *skeleton.Skeleton, vertebra skeleton.Bone,
	region string, n int,
) error {
	caudal := region == "Ca"
	if region == "T" || region == "L" || caudal && n <= orcaSpinedCaudals {
		if err := orcaNeuralSpine(bones, vertebra); err != nil {
			return err
		}
	}
	if region == "L" || caudal && n <= orcaWingedCaudals {
		if err := orcaTransverse(bones, vertebra); err != nil {
			return err
		}
	}
	if caudal && n >= orcaChevronFirst && n <= orcaChevronLast {
		return orcaChevron(bones, vertebra)
	}
	return nil
}

// orcaMiddle is the middle of a vertebra's body and its half thickness.
func orcaMiddle(vertebra skeleton.Bone) (maths.Vec, float64) {
	head, tail := vertebra.Thickness()
	return vertebra.Head().Add(vertebra.Tail()).Scale(.5), (head + tail) / 2
}

// orcaNeuralSpine adds the spine above vertebra, leaning back.
func orcaNeuralSpine(bones *skeleton.Skeleton, vertebra skeleton.Bone) error {
	middle, thick := orcaMiddle(vertebra)
	s := orcaTrunkProfile().at(middle.X)
	root := middle.Add(maths.V(0, thick, 0))
	rise := orcaSpineReach * (s.centre + s.high - root.Y)
	tip := root.Add(maths.V(-orcaSpineSlope*rise, rise, 0))
	return bones.Add(vertebra.Name()+"-spine", vertebra.Name(), root, tip,
		orcaProcessThick, .6*orcaProcessThick)
}

// orcaTransverse adds a transverse process each side of vertebra.
func orcaTransverse(bones *skeleton.Skeleton, vertebra skeleton.Bone) error {
	middle, thick := orcaMiddle(vertebra)
	s := orcaTrunkProfile().at(middle.X)
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
func orcaChevron(bones *skeleton.Skeleton, vertebra skeleton.Bone) error {
	middle, thick := orcaMiddle(vertebra)
	s := orcaTrunkProfile().at(middle.X)
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
