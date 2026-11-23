package manufacture

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// harborSealKind is a harbor seal with its anatomy and coat: the rigged
// skin, hair as blades along the coat's flow, five-digit flippers with
// claws, a short tail, ear openings, shut or open nostrils, vibrissae in
// rows, and a spotted pelage that reads wet or dry. It is a static
// specimen in a pose, not a clock.
type harborSealKind struct {
	look SealLook
	pose SealRigPose
	// onBones builds the skin from the skeleton's masses and binds it to
	// the bones, instead of the drawn profile and the rig's own frames.
	onBones bool
}

// rig is the kind's rig: the drawn profile's, or one built on bones.
func (kind harborSealKind) rig() (*SealRig, error) {
	if kind.onBones {
		return NewSealRigOnBones()
	}
	return NewHarborSealRig(), nil
}

// harborSealRelaxed is a seal at rest with the head a little tucked, so
// the neck shows its rolls, and the foreflippers swept back.
var harborSealRelaxed = SealRigPose{
	Yaw: .10, Pitch: .05, HeadPitch: -.50,
	Fore: [2]float64{-.30, -.22}, Hind: [2]float64{.10, -.10},
}

// made is the harbor seal in the kind's look, seeded from the seed. It
// refuses a seed that was not made.
func (kind harborSealKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	look := kind.look
	look.Seed = from.number
	rig, err := kind.rig()
	if err != nil {
		return good{}, err
	}
	skin, err := rig.Skin(kind.pose)
	if err != nil {
		return good{}, err
	}
	fur, err := rig.Fur(kind.pose, look)
	if err != nil {
		return good{}, err
	}
	anatomy, err := rig.Parts(kind.pose, look)
	if err != nil {
		return good{}, err
	}
	skinLook, hairLook := rig.Materials(look)
	parts := []thing.Thing{
		{Shape: skin, Material: skinLook, Motion: thing.Still(maths.Local())},
		{Shape: fur, Material: hairLook, Motion: thing.Still(maths.Local())},
	}
	parts = append(parts, anatomy...)
	// Conservative static extent includes the whiskers and the flippers.
	box := maths.Box{Min: maths.V(-1.3, -.8, -.8), Max: maths.V(1.3, .8, .8)}
	return good{native: parts, extent: &box}, nil
}
