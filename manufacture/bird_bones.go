package manufacture

import (
	"errors"
	"fmt"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// A bird's skeleton at rest: the trunk level with its spine on y
// nought, the tail's base at x nought, the neck rising forward from the
// chest, the wings spread straight out to either side and the legs
// standing, the tarsus upright and the toes flat.

// Where the trunk's bones lie, each a share of the trunk's length from
// the tail's base forward, or of its depth below the spine, or of its
// half width out from the middle. Every share is picked.
const (
	birdSacrumBack    = .10 // the synsacrum's back end
	birdSacrumFront   = .50 // its front, where the thoracics begin
	birdThoraxFront   = .80 // the last thoracic's front, the neck's root
	birdPygostyle     = .02 // the pygostyle's root, behind the caudals
	birdPygostyleTip  = -.04
	birdHipAlong      = .32 // the hip joint, along
	birdHipDown       = .30 // below the spine
	birdHipOut        = .50 // out from the middle
	birdShoulderAlong = .78 // the shoulder joint, along
	birdShoulderDown  = .18 // below the spine
	birdShoulderOut   = .80 // out from the middle
	birdSternumFront  = .86 // the sternum's front, along
	birdSternumBack   = .36 // its back
	birdSternumDown   = .85 // its depth below the spine
	birdKeelDepth     = .20 // the keel's depth below the sternum
)

// The neck at rest rises forward from the chest at this angle, in
// radians; picked.
const birdNeckRise = .35

// Bone thicknesses, half, as shares of the bird's length; picked.
const (
	birdVertebra  = .005 // a vertebra
	birdLimbBone  = .006 // the humerus and the femur
	birdSmallBone = .003 // a rib, a digit's or a toe's bone
)

var (
	birdUp       = maths.V(0, 1, 0)
	birdAlong    = maths.V(1, 0, 0)
	birdAcross   = maths.V(0, 0, 1)
	errBirdShare = errors.New("manufacture: a bird's shares leave " +
		"nothing for one of its parts")
)

// birdSides are the bird's right and left: the sign of z, and the word
// its bones are named with.
var birdSides = []struct {
	name string
	sign float64
}{{"right", 1}, {"left", -1}}

// birdMeasures are one bird's sizes, worked from its row and the share
// its seed picks: lengths in metres, angles in radians.
type birdMeasures struct {
	length, wingspan, bill, tarsus    float64
	head, neck, trunk, tail           float64
	depth, width                      float64
	humerus, forearm, hand            float64
	tibia, femur, toe                 float64
	secondary                         float64
	billDepth, billWidth, spatula     float64
	stance, neckBend, billTilt        float64
	plumeLength, web                  float64
	cervicals, thoracics, caudals     int
	primaries, secondaries, rectrices int
	plumes                            int
}

// measured is the bird of species whose sizes stand share of the way
// through their ranges: a bigger bird is bigger in every length.
func measured(species birdSpecies, share float64) (birdMeasures, error) {
	pick := func(name string) float64 { return species.pick(name, share) }
	m := birdMeasures{length: pick("length"), wingspan: pick("wingspan"),
		bill: pick("bill"), tarsus: pick("tarsus")}
	rest := m.length - m.bill
	m.head, m.neck, m.tail = pick("head")*rest, pick("neck")*rest,
		pick("tail")*rest
	m.trunk = rest - m.head - m.neck - m.tail*species.Tail.longest()
	m.depth, m.width = pick("depth")*m.trunk, pick("width")*m.trunk
	outside := m.wingspan/2 - birdShoulderOut*m.width/2
	arm := pick("arm") * outside
	m.humerus, m.forearm = pick("humerus")*arm, pick("forearm")*arm
	m.hand = arm - m.humerus - m.forearm
	m.tibia, m.femur = pick("tibia")*m.tarsus, pick("femur")*m.tarsus
	m.toe, m.secondary = pick("toe")*m.tarsus, pick("secondary")
	m.billDepth, m.billWidth = pick("bill-depth")*m.bill,
		pick("bill-width")*m.bill
	m.spatula = pick("spatula") * m.bill
	m.stance, m.neckBend = pick("stance"), pick("neck-bend")
	m.billTilt, m.plumeLength = pick("bill-tilt"), pick("plume-length")
	m.countsOf(species, share)
	if m.trunk <= 0 || m.hand <= 0 || outside <= 0 || arm >= outside {
		return birdMeasures{}, errBirdShare
	}
	return m, nil
}

// countsOf sets the bird's counts of vertebrae and feathers, and how far
// along its front toes their web reaches, a share, nought for none.
func (m *birdMeasures) countsOf(species birdSpecies, share float64) {
	count := func(name string) int { return species.count(name, share) }
	m.web = species.pick("web", share)
	m.cervicals, m.thoracics = count("cervicals"), count("thoracics")
	m.caudals, m.primaries = count("caudals"), count("primaries")
	m.secondaries, m.rectrices = count("secondaries"), count("rectrices")
	m.plumes = count("plumes")
}

// birdBody is one bird laid out at rest: its measures, its skeleton,
// the form of each bone, and the feathers hung on the bones.
type birdBody struct {
	species  birdSpecies
	measures birdMeasures
	bones    *skeleton.Skeleton
	forms    map[string]skeleton.BoneForm
	feathers []birdFeather
	from     seed
}

// newBirdBody lays out the bird of species and measures, its feathers'
// small differences drawn from from.
func newBirdBody(species birdSpecies, measures birdMeasures,
	from seed) (*birdBody, error) {
	body := &birdBody{species: species, measures: measures,
		bones: skeleton.New(), forms: map[string]skeleton.BoneForm{},
		from: from}
	steps := []func() error{body.trunkBones, body.neckBones,
		body.skullBones, body.girdleBones, body.ribBones, body.wingBones,
		body.legBones, body.hangFeathers}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// at is the point along the trunk, down from the spine and out from
// the middle, each a share.
func (body *birdBody) at(along, down, out float64) maths.Vec {
	m := body.measures
	return maths.V(along*m.trunk, -down*m.depth, out*m.width/2)
}

// add puts a bone on the skeleton and gives it its form.
func (body *birdBody) add(name, parent string, head, tail maths.Vec,
	thick float64, form skeleton.BoneForm) error {
	err := body.bones.Add(name, parent, head, tail, thick, thick)
	if err == nil && form != nil {
		body.forms[name] = form
	}
	return err
}

// birdShaft is the share of its ends' radius that a long bone's shaft
// keeps; picked.
const birdShaft = .7

// rod is a long bone's form from head to tail.
func birdRod(head, tail maths.Vec, thick float64) skeleton.BoneForm {
	return skeleton.NewRod(head, tail, thick, birdShaft)
}

// trunkBones are the synsacrum, the thoracics before it and the
// caudals and pygostyle behind it.
func (body *birdBody) trunkBones() error {
	m := body.measures
	thick := birdVertebra * m.length
	front, back := body.at(birdSacrumFront, 0, 0), body.at(birdSacrumBack,
		0, 0)
	if err := body.add("synsacrum", "", front, back, 2*thick,
		birdRod(front, back, 2*thick)); err != nil {
		return err
	}
	_, err := body.chain("T", "synsacrum", m.thoracics, true, front,
		body.at(birdThoraxFront, 0, 0), thick)
	if err != nil {
		return err
	}
	last, err := body.chain("Ca", "synsacrum", m.caudals, false, back,
		body.at(birdPygostyle, 0, 0), thick)
	if err != nil {
		return err
	}
	return body.add("pygostyle", last, body.at(birdPygostyle, 0, 0),
		body.at(birdPygostyleTip, -.05, 0), thick,
		birdRod(body.at(birdPygostyle, 0, 0),
			body.at(birdPygostyleTip, -.05, 0), thick))
}

// chain adds count vertebrae of region evenly from from to to, the
// first jointed to parent, and gives back the last one's name. Counted
// toward the head, the first added is the region's last by number.
func (body *birdBody) chain(region, parent string, count int,
	towardHead bool, from, to maths.Vec, thick float64) (string, error) {
	const spine, reach = 1.2, .8 // processes, in thicknesses; picked
	if count < 1 {
		return "", fmt.Errorf("%w: no %s vertebrae", errBirdShare, region)
	}
	above := parent
	for j := 0; j < count; j++ {
		number := j + 1
		if towardHead {
			number = count - j
		}
		name := fmt.Sprintf("%s%d", region, number)
		step := to.Sub(from).Scale(1 / float64(count))
		head := from.Add(step.Scale(float64(j)))
		form := skeleton.NewCentrum(head, head.Add(step), thick,
			spine*thick, reach*thick)
		if err := body.add(name, above, head, head.Add(step), thick,
			form); err != nil {
			return "", err
		}
		above = name
	}
	return above, nil
}

// neckRoot and atlasEnd are where the neck begins and ends at rest.
func (body *birdBody) neckRoot() maths.Vec {
	return body.at(birdThoraxFront, 0, 0)
}

// atlasEnd is the end of the first cervical, where the skull sits.
func (body *birdBody) atlasEnd() maths.Vec {
	rise := maths.Rotation(birdAcross, birdNeckRise).Out(birdAlong)
	return body.neckRoot().Add(rise.Scale(body.measures.neck))
}

// neckBones are the cervicals, from the chest to the skull.
func (body *birdBody) neckBones() error {
	m := body.measures
	_, err := body.chain("C", "T1", m.cervicals, true, body.neckRoot(),
		body.atlasEnd(), birdVertebra*m.length)
	return err
}
