package assembly

import (
	"fmt"
	"slices"
	"time"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// AnimationInput pairs a clip and its bound surface with a root motion in
// the scene. Root receives scene time. Start is the scene time at which
// the pose clip begins. A nil Root keeps the figure at the scene origin.
type AnimationInput struct {
	Clip     *skeleton.PoseClip
	Skin     *skeleton.Skin
	Faces    [][3]int
	Paint    []maths.Vec // nil uses the binding's rest coordinates
	Material *thing.Material
	Root     thing.Motion
	Start    time.Duration
}

// Animation retains the surface's indexing and paint chart while a clip
// poses it. Animation can be sampled in any order. The caller keeps the
// supplied root motion and material read-only while using it.
type Animation struct {
	skin     *skeleton.SkinClip
	faces    [][3]int
	paint    []maths.Vec
	material *thing.Material
	root     thing.Motion
	start    time.Duration
}

// NewAnimation checks the clip and skin pairing and the surface indices.
func NewAnimation(input AnimationInput) (*Animation, error) {
	skin, err := input.Clip.WithSkin(input.Skin)
	if err != nil {
		return nil, err
	}
	if len(input.Faces) == 0 {
		return nil, fmt.Errorf("assembly: an animated surface needs faces")
	}
	rest := skin.RestPoints()
	for i, face := range input.Faces {
		for _, vertex := range face {
			if vertex < 0 || vertex >= len(rest) {
				return nil, fmt.Errorf("assembly: face %d names vertex %d outside %d bound points",
					i, vertex, len(rest))
			}
		}
	}
	paint := rest
	if input.Paint != nil {
		if len(input.Paint) != len(rest) {
			return nil, fmt.Errorf("assembly: %d paint points for %d bound points", len(input.Paint), len(rest))
		}
		for i, point := range input.Paint {
			if !finitePoint(point) {
				return nil, fmt.Errorf("assembly: paint point %d is not finite", i)
			}
		}
		paint = slices.Clone(input.Paint)
	}
	return &Animation{
		skin: skin, faces: slices.Clone(input.Faces), paint: paint,
		material: input.Material, root: input.Root, start: input.Start,
	}, nil
}

// AnimationSample holds a figure-space pose and the root placement that
// all geometry from the sample will share.
type AnimationSample struct {
	animation *Animation
	pose      *skeleton.SkinPose
	root      maths.Subspace
}

// At uses the same scene time for root placement and for the pose clip's
// clock, which is measured from Start. It holds the pose endpoints as
// PoseClip does. The root follows its own motion outside the clip.
func (animation *Animation) At(sceneTime time.Duration) (*AnimationSample, error) {
	if animation == nil || animation.skin == nil {
		return nil, fmt.Errorf("assembly: no animation to sample")
	}
	root := maths.Local()
	if animation.root != nil {
		root = animation.root.At(sceneTime)
	}
	if !validPlacement(root) {
		return nil, fmt.Errorf("assembly: invalid animation root at %v", sceneTime)
	}
	pose, err := animation.skin.At(animation.elapsed(sceneTime))
	if err != nil {
		return nil, fmt.Errorf("assembly: pose at %v: %w", sceneTime, err)
	}
	return &AnimationSample{animation: animation, pose: pose, root: root}, nil
}

func (animation *Animation) elapsed(sceneTime time.Duration) time.Duration {
	if sceneTime <= animation.start {
		return 0
	}
	elapsed := sceneTime - animation.start
	if elapsed < 0 {
		// The mathematical difference is positive and exceeds time.Duration.
		return animation.skin.Duration()
	}
	return elapsed
}

// Root gives the figure's sampled placement in scene space.
func (sample *AnimationSample) Root() maths.Subspace { return sample.root }

// Surface builds a scene thing at this instant. Its vertices and normals
// are in posed figure space. Its motion stores the sampled scene root so
// the original paint coordinates stay in their own chart.
func (sample *AnimationSample) Surface() (thing.Thing, error) {
	points := sample.pose.Points()
	faces := sample.animation.faces
	normals := geometry.AreaNormals(points, faces)
	surface, err := mesh.NewWithRestPoints(points, sample.animation.paint, faces, normals, true)
	if err != nil {
		return thing.Thing{}, fmt.Errorf("assembly: animated surface: %w", err)
	}
	return thing.Thing{
		Shape: surface, Material: sample.animation.material, Motion: thing.Still(sample.root),
	}, nil
}

// BoneAttachment is an object's placement in the skeleton's rest space
// and the name of the bone that carries it.
type BoneAttachment struct {
	Bone string
	Rest maths.Subspace
}

// Attachment gives a rigid object's placement in scene space. Its bone
// and the surface use the same pose and the same root.
func (sample *AnimationSample) Attachment(attachment BoneAttachment) (maths.Subspace, error) {
	frame, found := sample.pose.Frame(attachment.Bone)
	if !found {
		return maths.Subspace{}, fmt.Errorf("assembly: no attachment bone %q", attachment.Bone)
	}
	if !validPlacement(attachment.Rest) {
		return maths.Subspace{}, fmt.Errorf("assembly: invalid rest placement on bone %q", attachment.Bone)
	}
	world := frame.Place(attachment.Rest).Within(sample.root)
	if !validPlacement(world) || world.Scale == 0 {
		return maths.Subspace{}, fmt.Errorf("assembly: attachment on bone %q exceeds the placement range", attachment.Bone)
	}
	return world, nil
}
