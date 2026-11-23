package manufacture

import (
	"fmt"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// Shares of the skull's bones, of the head's length; picked.
const (
	birdSkullWide  = .30 // the braincase's half width
	birdSkullDeep  = .32 // its half depth
	birdSkullRaise = .30 // how far its middle line lies above the condyle
	birdJawBack    = .30 // the lower jaw's joint, from the condyle forward
	birdJawDown    = .55 // and below it, of the braincase's half depth
)

// The wing's bones, as shares of the hand or of the bird's length;
// picked.
const (
	birdCarpus       = .60  // the carpometacarpus, of the hand
	birdMajorFirst   = .25  // the major digit's first phalanx
	birdMajorSecond  = .15  // its second
	birdMinorDigit   = .12  // the minor digit
	birdAlulaLength  = .14  // the alular digit
	birdForearmApart = .004 // radius and ulna either side of the line
)

// The leg's directions at rest, in the bird's side plane: the femur
// forward and down, the tibiotarsus back and down; picked so the knee
// lies inside the belly and the tarsus stands upright under the hip.
var (
	birdFemurWay = maths.V(.6, -.8, 0)
	birdTibiaWay = maths.V(-.26, -.966, 0)
)

// birdToes are the four toes: which way each points from the foot, out
// being toward the bird's own side, its length as a share of the middle
// toe, and its count of phalanges, the last its claw. The ways and
// lengths are picked; the counts are the usual two, three, four and
// five of a perching or wading bird, not read tonight.
var birdToes = []struct {
	forward, out, length float64
	phalanges            int
}{{-1, 0, .55, 2}, {.88, -.47, .78, 3}, {1, 0, 1, 4}, {.88, .47, .85, 5}}

// skullBones are the braincase, the upper bill and the lower jaw.
func (body *birdBody) skullBones() error {
	m := body.measures
	condyle := body.atlasEnd()
	front := condyle.Add(birdAlong.Scale(m.head))
	deep := birdSkullDeep * m.head
	lift := birdUp.Scale(birdSkullRaise * deep)
	braincase := skeleton.NewProfile(condyle.Add(lift), front.Add(lift),
		birdUp, birdScaled(birdBraincaseWide, birdSkullWide*m.head),
		birdScaled(birdBraincaseDeep, deep))
	if err := body.add("cranium", "C1", condyle, front,
		birdVertebra*m.length, braincase); err != nil {
		return err
	}
	tip := front.Add(birdAlong.Scale(m.bill))
	wide, deep2 := body.billStations(birdBoneShare)
	if err := body.add("premaxilla", "cranium", front, tip,
		birdSmallBone*m.length, skeleton.NewProfile(front, tip, birdUp,
			wide, deep2)); err != nil {
		return err
	}
	joint := condyle.Add(birdAlong.Scale(birdJawBack * m.head)).
		Add(birdUp.Scale(-birdJawDown * deep))
	under := tip.Add(birdUp.Scale(-m.billDepth / 4))
	return body.add("mandible", "cranium", joint, under,
		birdSmallBone*m.length, skeleton.NewProfile(joint, under, birdUp,
			wide, birdScaled(deep2, birdJawShare)))
}

// girdleBones are the coracoids, the shoulder blades, the wishbone,
// the sternum and its keel, and the hip bones.
func (body *birdBody) girdleBones() error {
	m := body.measures
	thin := birdSmallBone * m.length
	for _, side := range birdSides {
		shoulder := body.at(birdShoulderAlong, birdShoulderDown,
			side.sign*birdShoulderOut)
		foot := body.at(birdSternumFront, birdSternumDown, side.sign*.25)
		blade := body.at(.45, .05, side.sign*.35)
		if err := body.add("coracoid-"+side.name, "T1", shoulder, foot,
			thin, birdRod(shoulder, foot, thin)); err != nil {
			return err
		}
		if err := body.add("scapula-"+side.name, "T1", shoulder, blade,
			thin, skeleton.NewBlade(shoulder, blade, .04*m.trunk, thin/2,
				.5, birdUp)); err != nil {
			return err
		}
	}
	if err := body.furcula(thin); err != nil {
		return err
	}
	if err := body.sternum(thin); err != nil {
		return err
	}
	return body.pelvis(thin)
}

// furcula is the wishbone, one bone bowed forward and down between the
// shoulders.
func (body *birdBody) furcula(thin float64) error {
	right := body.at(birdShoulderAlong, birdShoulderDown, birdShoulderOut)
	left := body.at(birdShoulderAlong, birdShoulderDown, -birdShoulderOut)
	curve := [4]maths.Vec{right, body.at(.92, .55, .2),
		body.at(.92, .55, -.2), left}
	return body.add("furcula", "T1", right, left, thin,
		skeleton.NewArc(curve, thin, thin, birdAlong))
}

// sternum is the breastbone, flat under the chest, and its keel.
func (body *birdBody) sternum(thin float64) error {
	m := body.measures
	front := body.at(birdSternumFront, birdSternumDown, 0)
	back := body.at(birdSternumBack, birdSternumDown, 0)
	if err := body.add("sternum", "T1", front, back, thin,
		skeleton.NewBlade(front, back, .45*m.width/2, thin/2, 1,
			birdUp)); err != nil {
		return err
	}
	half := birdKeelDepth * m.depth / 2
	drop := birdUp.Scale(-half)
	return body.add("keel", "sternum", front.Add(drop), back.Add(drop),
		thin, skeleton.NewBlade(front.Add(drop), back.Add(drop), half,
			thin/2, .6, birdAcross))
}

// pelvis is the ilium over the hip each side and the pubis behind it.
func (body *birdBody) pelvis(thin float64) error {
	m := body.measures
	for _, side := range birdSides {
		front := body.at(.50, .02, side.sign*.18)
		back := body.at(.12, .10, side.sign*.35)
		if err := body.add("ilium-"+side.name, "synsacrum", front, back,
			thin, skeleton.NewBlade(front, back, .05*m.trunk, thin/2, .6,
				birdUp)); err != nil {
			return err
		}
		hip := body.hip(side.sign)
		end := body.at(0, .55, side.sign*.3)
		if err := body.add("pubis-"+side.name, "synsacrum", hip, end,
			thin/2, birdRod(hip, end, thin/2)); err != nil {
			return err
		}
	}
	return nil
}

// hip is the hip joint on the side of sign.
func (body *birdBody) hip(sign float64) maths.Vec {
	return body.at(birdHipAlong, birdHipDown, sign*birdHipOut)
}

// ribBones are a rib each side of each thoracic, bowed down and back
// round the chest to the sternum's edge.
func (body *birdBody) ribBones() error {
	m := body.measures
	thin := birdSmallBone * m.length
	for n := 1; n <= m.thoracics; n++ {
		name := fmt.Sprintf("T%d", n)
		at, _ := body.bones.Find(name)
		along := body.bones.Bone(at).Head().X / m.trunk
		for _, side := range birdSides {
			s := side.sign
			curve := [4]maths.Vec{body.at(along, .03, .2*s),
				body.at(along-.03, .32, .85*s), body.at(along-.05, .62, .8*s),
				body.at(along-.07, birdSternumDown, .35*s)}
			rib := fmt.Sprintf("rib%d-%s", n, side.name)
			if err := body.add(rib, name, curve[0], curve[3], thin,
				skeleton.NewArc(curve, thin, thin*1.5, birdAlong)); err != nil {
				return err
			}
		}
	}
	return nil
}

// wingBones are each wing's humerus, radius and ulna, the small wrist
// bones, the carpometacarpus and the digits, laid straight out.
func (body *birdBody) wingBones() error {
	for _, side := range birdSides {
		if err := body.arm(side.name, side.sign); err != nil {
			return err
		}
		if err := body.digits(side.name, side.sign); err != nil {
			return err
		}
	}
	return nil
}

// shoulder is the shoulder joint on the side of sign.
func (body *birdBody) shoulder(sign float64) maths.Vec {
	return body.at(birdShoulderAlong, birdShoulderDown,
		sign*birdShoulderOut)
}

// wrist is the wrist on the side of sign, at rest.
func (body *birdBody) wrist(sign float64) maths.Vec {
	m := body.measures
	return body.shoulder(sign).Add(birdAcross.Scale(sign *
		(m.humerus + m.forearm)))
}

// arm adds the humerus, the radius and the ulna on one side.
func (body *birdBody) arm(side string, sign float64) error {
	m := body.measures
	out := birdAcross.Scale(sign)
	thick := birdLimbBone * m.length
	shoulder := body.shoulder(sign)
	elbow := shoulder.Add(out.Scale(m.humerus))
	if err := body.add("humerus-"+side, "T1", shoulder, elbow, thick,
		birdRod(shoulder, elbow, thick)); err != nil {
		return err
	}
	apart := birdAlong.Scale(birdForearmApart * m.length)
	for _, bone := range []struct {
		name string
		lead float64
	}{{"radius-", 1}, {"ulna-", -1}} {
		head := elbow.Add(apart.Scale(bone.lead))
		tail := head.Add(out.Scale(m.forearm))
		if err := body.add(bone.name+side, "humerus-"+side, head, tail,
			thick*.7, birdRod(head, tail, thick*.7)); err != nil {
			return err
		}
	}
	return nil
}

// digits adds the wrist's bones and the hand's on one side: the
// carpometacarpus, the alular digit, the two phalanges of the major
// digit and the minor digit.
func (body *birdBody) digits(side string, sign float64) error {
	m := body.measures
	out := birdAcross.Scale(sign)
	thin := birdSmallBone * m.length
	wrist := body.wrist(sign)
	end := wrist.Add(out.Scale(birdCarpus * m.hand))
	first := end.Add(out.Scale(birdMajorFirst * m.hand))
	alula := wrist.Add(out.Scale(.05 * m.hand))
	bones := []struct {
		name, parent string
		head, tail   maths.Vec
	}{{"carpometacarpus-", "ulna-", wrist, end},
		{"alula-", "carpometacarpus-", alula, alula.Add(maths.V(.4, 0,
			sign).Unit().Scale(birdAlulaLength * m.hand))},
		{"major-digit1-", "carpometacarpus-", end, first},
		{"major-digit2-", "major-digit1-", first,
			first.Add(out.Scale(birdMajorSecond * m.hand))},
		{"minor-digit-", "carpometacarpus-", end.Sub(birdAlong.Scale(
			birdForearmApart * m.length)), end.Add(out.Scale(
			birdMinorDigit * m.hand)).Sub(birdAlong.Scale(2 *
			birdForearmApart * m.length))}}
	for _, bone := range bones {
		if err := body.add(bone.name+side, bone.parent+side, bone.head,
			bone.tail, thin, birdRod(bone.head, bone.tail, thin)); err != nil {
			return err
		}
	}
	return nil
}

// legBones are each leg's femur, tibiotarsus, tarsometatarsus and toes,
// the leg standing.
func (body *birdBody) legBones() error {
	m := body.measures
	thick := birdLimbBone * m.length
	for _, side := range birdSides {
		hip := body.hip(side.sign)
		knee := hip.Add(birdFemurWay.Unit().Scale(m.femur))
		ankle := knee.Add(birdTibiaWay.Unit().Scale(m.tibia))
		foot := ankle.Add(birdUp.Scale(-m.tarsus))
		joints := []maths.Vec{hip, knee, ankle, foot}
		names := []string{"femur-", "tibiotarsus-", "tarsometatarsus-"}
		parent := "synsacrum"
		for i, name := range names {
			bone := name + side.name
			if err := body.add(bone, parent, joints[i], joints[i+1], thick,
				birdRod(joints[i], joints[i+1], thick)); err != nil {
				return err
			}
			parent = bone
		}
		if err := body.toes(side.name, side.sign, foot); err != nil {
			return err
		}
	}
	return nil
}

// toes adds the four toes of one foot, each from the foot outward.
func (body *birdBody) toes(side string, sign float64, foot maths.Vec) error {
	m := body.measures
	thin := birdSmallBone * m.length
	for n, toe := range birdToes {
		way := maths.V(toe.forward, 0, sign*toe.out).Unit()
		step := way.Scale(toe.length * m.toe / float64(toe.phalanges))
		head, parent := foot, "tarsometatarsus-"+side
		for k := 1; k <= toe.phalanges; k++ {
			name := fmt.Sprintf("toe%d-%d-%s", n+1, k, side)
			taper := thin * (1 - .5*float64(k)/float64(toe.phalanges))
			if err := body.add(name, parent, head, head.Add(step), taper,
				birdRod(head, head.Add(step), taper)); err != nil {
				return err
			}
			head, parent = head.Add(step), name
		}
	}
	return nil
}

// birdScaled is stations each times size.
func birdScaled(stations []float64, size float64) []float64 {
	scaled := make([]float64, len(stations))
	for i, station := range stations {
		scaled[i] = station * size
	}
	return scaled
}
