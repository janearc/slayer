package manufacture

import (
	"errors"
	"math"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// errSealCarrier is a flipper whose carrying bone is not in the
// skeleton.
var errSealCarrier = errors.New("seal: a flipper has no bone to carry it")

// sealForm is one species of seal as data for the lofted maker: its
// outline's stations in the shared rest space, the measured animal the
// stations' radii come from and its own mean radius at the axillary
// site, its length nose to tail tip over the rest space's, and its
// coat. A second seal is a second form.
type sealForm struct {
	outline  sealOutline
	source   sealSource
	axillary float64
	scale    float64
	pelt     sealPelt
}

// harborSealForm is the harbor seal, Phoca vitulina, at the rest
// space's own size.
var harborSealForm = sealForm{outline: harborSealOutline,
	source: sealWeddellSource, axillary: harborSealAxillary, scale: 1,
	pelt: harborSealPelt}

// sealLoftedKind is a seal whose head, neck and trunk are one surface
// lofted through its form's outline, over the seal's skeleton, with a
// cushion of flesh over each flipper's digits, the skin held by the
// bones and posed by turning their joints. It is a still specimen in a
// pose.
type sealLoftedKind struct {
	// form is the species.
	form sealForm
	// pose is the joints' turns, in the rig's words.
	pose SealRigPose
	// lean turns the whole animal nose up about its shoulders, radians.
	lean float64
	// rests is whether it lies on the ground, pressed where it bears;
	// the ground is then at height nought.
	rests bool
	// wet is the coat from dry at nought to soaked at one.
	wet float64
}

var (
	// sealBanana is a harbor seal hauled out with head and hind
	// flippers held up off the ground, the fore flippers out on the
	// ground beside the chest. Angles picked.
	sealBanana = sealLoftedKind{form: harborSealForm, rests: true,
		wet: .45, lean: .15,
		pose: SealRigPose{Yaw: .06, Pitch: -.62, HeadPitch: .55,
			Fore: [2]float64{.35, .35}, Hind: [2]float64{.6, .6}}}
	// sealStroke is a harbor seal swimming: the hind body swept to one
	// side, the hind flippers turned the same way, the fore flippers a
	// little off the flanks. Angles picked.
	sealStroke = sealLoftedKind{form: harborSealForm, wet: 1,
		pose: SealRigPose{Yaw: .38, Pitch: .04, HeadPitch: -.06,
			Fore: [2]float64{.2, .2}, Hind: [2]float64{.4, -.4}}}
)

// sealWhiskerReach pads the kind's extent past its skin for the
// whiskers, in metres at rest: the longest strand and a little.
const sealWhiskerReach = .1

// stance is the kind's pose over the skeleton: every bone's motion and
// the whole animal's lean about the first thoracic vertebra's head.
func (kind sealLoftedKind) stance(made *sealBones) (sealStance, error) {
	frames, err := (&sealBound{made: made}).frames(kind.pose,
		sealFirstThoracic)
	if err != nil {
		return sealStance{}, err
	}
	shoulder := sealSpinePoint(sealFirstThoracic)
	return sealStance{frames: frames, rests: kind.rests,
		scale: kind.form.scale,
		lean:  skeleton.TurnAbout(shoulder, maths.ZAxis, kind.lean)}, nil
}

// made is the seal as one lofted body over its bones. It refuses a seed
// that was not made.
func (kind sealLoftedKind) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	made, err := harborSealBones()
	if err != nil {
		return good{}, err
	}
	flippers, err := sealHeldFlippers(made)
	if err != nil {
		return good{}, err
	}
	held := append([]sealHeld{sealHeldTrunk(made, kind.form.outline)},
		flippers...)
	stance, err := kind.stance(made)
	if err != nil {
		return good{}, err
	}
	stance = stance.onGround(held)
	return kind.dressed(made, held, stance, from.number)
}

// dressed is the held surfaces in the stance as the kind's parts: each
// a mesh in the coat, then the eyes and whiskers on the skull.
func (kind sealLoftedKind) dressed(made *sealBones, held []sealHeld,
	stance sealStance, number uint64,
) (good, error) {
	form := kind.form
	hide := sealHide{pelt: form.pelt, outline: form.outline, seed: number,
		wet: kind.wet}
	var parts []thing.Thing
	box := maths.Box{}
	for index, part := range held {
		skin, err := part.posed(stance)
		if err != nil {
			return good{}, err
		}
		bounds := skin.Bounds()
		hide.limb = index > 0
		if index == 0 {
			box = bounds
		}
		box = maths.Box{Min: sealLowCorner(box.Min, bounds.Min),
			Max: sealHighCorner(box.Max, bounds.Max)}
		parts = append(parts, thing.Thing{Shape: skin,
			Material: hide.look(), Motion: thing.Still(maths.Local())})
	}
	skull, _ := made.bones.Find("skull")
	head := []skeleton.Influence{skeleton.NewInfluence(skull, 1)}
	parts = append(parts, form.outline.face(func(rest maths.Vec) maths.Vec {
		return stance.placed(rest, head)
	}, form.scale)...)
	reach := maths.V(1, 1, 1).Scale(sealWhiskerReach * form.scale)
	box = maths.Box{Min: box.Min.Sub(reach), Max: box.Max.Add(reach)}
	return good{native: parts, extent: &box}, nil
}

// posed is the surface in a stance as a mesh that keeps its rest points
// for the coat.
func (part sealHeld) posed(stance sealStance) (*mesh.Mesh, error) {
	points := make([]maths.Vec, len(part.surface.points))
	for index, rest := range part.surface.points {
		points[index] = stance.placed(rest, part.holds[index])
	}
	return mesh.NewWithRestPoints(points, part.surface.points,
		part.surface.faces, sealSmoothNormals(points, part.surface.faces),
		true)
}

// sealLowCorner is the least of two points' coordinates, each on its
// own.
func sealLowCorner(a, b maths.Vec) maths.Vec {
	return maths.V(math.Min(a.X, b.X), math.Min(a.Y, b.Y),
		math.Min(a.Z, b.Z))
}

// sealHighCorner is the most of two points' coordinates, each on its
// own.
func sealHighCorner(a, b maths.Vec) maths.Vec {
	return maths.V(math.Max(a.X, b.X), math.Max(a.Y, b.Y),
		math.Max(a.Z, b.Z))
}
