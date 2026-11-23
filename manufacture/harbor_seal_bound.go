package manufacture

import (
	"math"
	"strings"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
)

// sealBound is a rig's skin tied to the seal's skeleton: the bones with
// their joints, the sets of bones each part of the skin may follow, and
// each rest vertex's influences. A rig with one is posed by its bones.
type sealBound struct {
	made       *sealBones
	holders    [][]skeleton.Holder
	influences [][]skeleton.Influence
}

// sealReach is how far, in metres, a bone holds the skin by the shared
// nearness weighing; chosen, about a vertebra's length times two.
const sealReach = .08

// sealCaudalShare is how much of a turn of the hind body the tail takes,
// against the lumbar column and the sacrum, per unit of its joints'
// range: the tail is short and follows. Chosen.
const sealCaudalShare = .1

// sealShoulderShare is the share of a fore flipper's sweep the shoulder
// takes as a bend of the humerus; chosen.
const sealShoulderShare = .75

// trunkGroup is the binding set of the trunk, neck and head; the limbs'
// sets follow it in the order of the rig's openings.
const trunkGroup = 0

// NewSealRigOnBones builds the rest skin from the seal's skeleton: each
// ring of the trunk, neck and head reaches as far as the masses on the
// bones under it and the blubber over them, and every vertex is bound
// to the bones nearest it, so a pose turns joints and the skin follows.
func NewSealRigOnBones() (*SealRig, error) {
	made, err := harborSealBones()
	if err != nil {
		return nil, err
	}
	body, err := newSealBody(made)
	if err != nil {
		return nil, err
	}
	rig := buildSealRig(sealShoulder,
		body.shape((HarborSealPose{}).body()), false)
	return rig, rig.bindTo(made)
}

// bindTo ties every rest vertex to its bones: a flipper's vertices to
// that limb and the trunk beside it, the rest to the column and skull.
func (rig *SealRig) bindTo(made *sealBones) error {
	sets := sealBindingSets(made.names())
	holders := make([][]skeleton.Holder, len(sets))
	for group, names := range sets {
		held, err := made.bones.Holders(names, made.forms, sealReach)
		if err != nil {
			return err
		}
		holders[group] = held
	}
	influences := make([][]skeleton.Influence, len(rig.vertices))
	for index, vertex := range rig.vertices {
		group := trunkGroup
		if vertex.weight > 0 {
			group = vertex.limb + 1
		}
		influences[index] = skeleton.Nearness(rig.rest[index],
			holders[group])
	}
	rig.bound = &sealBound{made: made, holders: holders,
		influences: influences}
	return nil
}

// sealBindingSets is the trunk's set, then each opening's limb with the
// column beside it, in the order of sealOpenings.
func sealBindingSets(names []string) [][]string {
	trunk := sealNamesWith(names, "neck", "thoracic", "lumbar", "sacral",
		"caudal", "skull")
	shoulder := sealNamesWith(names, "thoracic1", "thoracic2",
		"thoracic3", "thoracic4")
	hips := sealNamesWith(names, "lumbar4", "lumbar5", "sacral",
		"caudal1", "caudal2")
	sets := [][]string{trunk}
	for _, opening := range sealOpenings {
		side := sealSideName(opening.side)
		near, limb := shoulder, sealNamesWith(names, "fore-"+side+"-")
		if opening.hind {
			near, limb = hips, sealNamesWith(names, "hind-"+side+"-")
		}
		sets = append(sets, append(append([]string{}, near...), limb...))
	}
	return sets
}

// sealNamesWith is the names that begin with any of the beginnings. A
// beginning that is a whole name, as "thoracic1", also takes the names
// it begins ("thoracic10" to "thoracic15"); the sets above allow it.
func sealNamesWith(names []string, beginnings ...string) []string {
	var found []string
	for _, name := range names {
		for _, beginning := range beginnings {
			if strings.HasPrefix(name, beginning) {
				found = append(found, name)
				break
			}
		}
	}
	return found
}

// turns is the rig's pose as turns of the bones: the trunk's yaw and
// pitch shared over the column aft of the flex start, the nod over the
// neck, a fore flipper's sweep at the shoulder, and a hind flipper's
// feather at the ankle. A turn the joints refuse is an error.
func (bound *sealBound) turns(pose SealRigPose,
	start float64,
) (skeleton.Pose, error) {
	turns, err := bound.spine(skeleton.Pose{}, pose, start)
	if err != nil {
		return turns, err
	}
	neck := sealNamesWith(bound.made.names(), "neck")
	for _, name := range neck {
		turns, err = bound.turn(turns, name,
			pose.HeadPitch/float64(len(neck)), 0, 0)
		if err != nil {
			return turns, err
		}
	}
	for limb, opening := range sealOpenings {
		side := sealSideName(opening.side)
		angle := opening.side * pose.angle(limb)
		if opening.hind {
			turns, err = bound.turn(turns, "hind-"+side+"-astragalus", 0, 0,
				angle)
		} else {
			turns, err = bound.turn(turns, "fore-"+side+"-humerus", 0,
				sealShoulderShare*angle, 0)
		}
		if err != nil {
			return turns, err
		}
	}
	return turns, nil
}

// turn is the pose with one more bone turned within its joint.
func (bound *sealBound) turn(pose skeleton.Pose, name string, flex, bend,
	twist float64,
) (skeleton.Pose, error) {
	return bound.made.joints.Turned(bound.made.bones, pose, name, flex,
		bend, twist)
}

// spine shares the trunk's yaw, as a bend, and its pitch, as a flex,
// over the column's bones aft of start, each by its joint's range. The
// rig pitches its tail down where a bone's flex lifts it, so the sign
// of the pitch is turned.
func (bound *sealBound) spine(turns skeleton.Pose, pose SealRigPose,
	start float64,
) (skeleton.Pose, error) {
	names := bound.aft(start)
	var flexes, bends float64
	for _, name := range names {
		flex, bend := bound.ranges(name)
		flexes, bends = flexes+flex, bends+bend
	}
	for _, name := range names {
		flex, bend := bound.ranges(name)
		var err error
		turns, err = bound.turn(turns, name, -pose.Pitch*flex/flexes,
			pose.Yaw*bend/bends, 0)
		if err != nil {
			return turns, err
		}
	}
	return turns, nil
}

// aft is the column's bones whose heads lie behind start.
func (bound *sealBound) aft(start float64) []string {
	var names []string
	for _, name := range sealNamesWith(bound.made.names(), "thoracic",
		"lumbar", "sacral", "caudal") {
		head, _, _ := bound.made.ends(name)
		if head.X < start {
			names = append(names, name)
		}
	}
	return names
}

// ranges is how far a column bone flexes and bends, the tail's scaled
// down by the share of a turn it takes.
func (bound *sealBound) ranges(name string) (float64, float64) {
	_, flex, _ := bound.made.joints.Limit(name, skeleton.Flex)
	_, bend, _ := bound.made.joints.Limit(name, skeleton.Bend)
	if strings.HasPrefix(name, "caudal") {
		return flex * sealCaudalShare, bend * sealCaudalShare
	}
	return flex, bend
}

// frames is each bone's motion for the rig's pose.
func (bound *sealBound) frames(pose SealRigPose,
	start float64,
) ([]skeleton.Frame, error) {
	turns, err := bound.turns(pose, start)
	if err != nil {
		return nil, err
	}
	return bound.made.bones.Frames(turns), nil
}

// posedOnBones is every rest vertex carried by its bones in the pose,
// then the folds the bend bunches up. A pose the joints refuse gives
// points of no number, which the mesh then refuses where it is made.
func (rig *SealRig) posedOnBones(pose SealRigPose) []maths.Vec {
	frames, err := rig.bound.frames(pose, rig.flexStart)
	if err != nil {
		return sealUnplaced(len(rig.rest))
	}
	points := make([]maths.Vec, len(rig.rest))
	for index, rest := range rig.rest {
		points[index] = skeleton.Carry(rest, frames,
			rig.bound.influences[index]).Add(rig.fold(index, pose))
	}
	return points
}

// headPointOnBones is a rest landmark carried by the trunk's bones
// nearest it, as the skin there is; of no number if the pose is refused.
func (rig *SealRig) headPointOnBones(pose SealRigPose,
	rest maths.Vec,
) maths.Vec {
	bound := rig.bound
	frames, err := bound.frames(pose, rig.flexStart)
	if err != nil {
		return sealUnplaced(1)[0]
	}
	return skeleton.Carry(rest, frames,
		skeleton.Nearness(rest, bound.holders[trunkGroup]))
}

// sealUnplaced is count points of no number.
func sealUnplaced(count int) []maths.Vec {
	points := make([]maths.Vec, count)
	for index := range points {
		points[index] = maths.V(math.NaN(), math.NaN(), math.NaN())
	}
	return points
}
