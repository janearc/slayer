package primates

import (
	"fmt"
	"slices"

	"github.com/janearc/slayer/bioforms/pigment"
	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/geometry"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/mesh"
	"github.com/janearc/slayer/scene/thing"
)

// AdultAnatomy selects the reference chest and external pelvic form.
// Head, skin, hair, body proportions and pose are configured independently.
type AdultAnatomy uint8

const (
	FemaleAnatomy AdultAnatomy = iota
	MaleAnatomy
)

// HumanSpec describes an adult reference. Stature is in metres.
// LegLength scales the ground-to-pelvis distance before stature scaling.
// Torso supplies global breadth, depth, and chest shape. Proportions
// adjusts regional dimensions. Joint motion belongs to Pose.
type HumanSpec struct {
	Anatomy          AdultAnatomy
	Torso            Torso
	Stature          float64
	LegLength        float64
	Proportions      AdultProportions
	Breasts          BreastForm
	Vulvar           VulvarForm
	skinMottle       pigment.Mottle
	skinDiffusion    thing.SubsurfaceProfile
	skinTransmission thing.ThinTissue
	auricles         *AuricleForm
}

// AdultHuman is the 1.70 m female reference in a neutral standing pose.
func AdultHuman() HumanSpec {
	return HumanSpec{Torso: Adult(), Stature: humanHeight, LegLength: 1,
		Vulvar:     VulvarForm{Asymmetry: .064},
		skinMottle: pigment.Mottle{Scale: .062, Strength: .13}}
}

// AdultMaleHuman uses the same frame and face with a male chest and pelvis.
func AdultMaleHuman() HumanSpec {
	spec := AdultHuman()
	spec.Anatomy = MaleAnatomy
	spec.Torso.Left = Side{
		Width: .080, Projection: .032, Height: .007,
		AreolaRadius: .013, PapillaRadius: .0032, PapillaProjection: .0027,
	}
	spec.Torso.Right = Side{
		Width: .078, Projection: .030, Height: .005,
		AreolaRadius: .0126, PapillaRadius: .0030, PapillaProjection: .0025,
	}
	return spec
}

const (
	humanHeight = 1.70
	pelvisY     = .80
)

func (spec HumanSpec) check() error {
	if spec.Anatomy > MaleAnatomy {
		return fmt.Errorf("adult human: unknown anatomy %d", spec.Anatomy)
	}
	if !between(spec.Stature, 1.45, 2.05) || !between(spec.LegLength, .90, 1.10) {
		return fmt.Errorf("adult human: stature or leg proportion outside range")
	}
	if !spec.skinMottle.Valid() {
		return fmt.Errorf("adult human: invalid skin mottle scale or strength")
	}
	torso := spec.Torso
	if torso.LegacyContour || torso.ProximalArms || torso.LateralAttachment ||
		torso.LeftArm != 0 || torso.RightArm != 0 || torso.Pitch != 0 || torso.Twist != 0 {
		return fmt.Errorf("adult human: use native geometry and named joint poses")
	}
	if spec.Anatomy == MaleAnatomy {
		if spec.Breasts != (BreastForm{}) {
			return fmt.Errorf("adult human: breast form requires female anatomy")
		}
		return torso.Check()
	}
	if !spec.Breasts.valid() {
		return fmt.Errorf("adult human: breast form outside sculpting bounds")
	}
	return torso.check(0)
}

// Human holds an indexed exterior and its persistent rest binding.
type Human struct {
	spec        HumanSpec
	points      []maths.Vec
	chart       []maths.Vec
	faces       [][3]int
	lashMargins [2]lidMargin
	digitHair   []digitHairPatch
	earFaces    [2]int
	tissue      []tissueMotion
	binding     *skeleton.Skin
	diffusion   thing.Subsurface
}

// NewHuman constructs and binds one native adult skin.
func NewHuman(spec HumanSpec) (*Human, error) {
	if err := spec.check(); err != nil {
		return nil, err
	}
	diffusion, err := thing.NewSubsurface(spec.skinDiffusion)
	if err != nil {
		return nil, err
	}
	proportions, err := spec.Proportions.resolved()
	if err != nil {
		return nil, err
	}
	spec.Proportions = proportions
	vulvar, err := spec.Vulvar.resolved()
	if err != nil {
		return nil, err
	}
	spec.Vulvar = vulvar
	if spec.auricles != nil {
		auricles, err := spec.auricles.resolved()
		if err != nil {
			return nil, err
		}
		spec.auricles = &auricles
	}
	anatomy := newHumanAnatomy(spec)
	bones, err := anatomy.skeleton()
	if err != nil {
		return nil, err
	}
	model := anatomy.build()
	points := make([]maths.Vec, len(model.vertices))
	chart := make([]maths.Vec, len(model.vertices))
	tissue := make([]tissueMotion, len(model.vertices))
	for i, vertex := range model.vertices {
		points[i] = spec.measure(vertex.point, vertex.site)
		chart[i], tissue[i] = vertex.chart, vertex.tissue
	}
	binding, err := bones.BindSkin(skeleton.SkinInput{
		Points: points, Blend: skeleton.RigidSkin,
		Weigh: func(index int, _ maths.Vec) []skeleton.Influence {
			return model.vertices[index].weights.influences(bones)
		},
	})
	if err != nil {
		return nil, err
	}
	return &Human{spec: spec, points: points, chart: chart, faces: model.faces,
		lashMargins: model.lashMargins, digitHair: model.digitHair, earFaces: model.earFaces,
		tissue: tissue, binding: binding, diffusion: diffusion}, nil
}

// RestPoints returns the measured neutral geometry in stable vertex order.
func (human *Human) RestPoints() []maths.Vec { return slices.Clone(human.points) }

// PaintPoints returns the canonical material chart in the same vertex order
// as RestPoints. Place pigment masks against these coordinates. Every pose
// interpolates the same points on its corresponding triangles.
func (human *Human) PaintPoints() []maths.Vec { return slices.Clone(human.chart) }

// Faces returns the triangles shared by all poses.
func (human *Human) Faces() [][3]int { return slices.Clone(human.faces) }

// HumanPose holds the final skin points and the concurrent attachment frames.
type HumanPose struct {
	human  *Human
	points []maths.Vec
	bones  *skeleton.SkinPose
}

// Pose binds the head, arms, legs and digits to the named joint turns and
// then adds soft-tissue displacement to the chest.
func (human *Human) Pose(pose skeleton.Pose) (*HumanPose, error) {
	placed, err := human.binding.Pose(pose)
	if err != nil {
		return nil, err
	}
	return human.poseFromSkin(placed), nil
}

func (human *Human) poseFromSkin(placed *skeleton.SkinPose) *HumanPose {
	points := placed.Points()
	human.moveTissue(placed, points)
	return &HumanPose{human: human, points: points, bones: placed}
}

// Points returns a copy of the posed positions for surface attachments.
func (pose *HumanPose) Points() []maths.Vec { return slices.Clone(pose.points) }

// Frame returns the named rest-to-posed frame for a rigid attachment.
func (pose *HumanPose) Frame(name string) (skeleton.Frame, bool) {
	return pose.bones.Frame(name)
}

// Surface builds a smooth mesh with stable native paint coordinates. A nil
// material uses the human's authored pigment and scattering responses.
func (pose *HumanPose) Surface(material *thing.Material) (thing.Thing, error) {
	faces := pose.human.faces
	shape, err := mesh.NewWithRestPoints(pose.points, pose.human.chart, faces,
		geometry.AreaNormals(pose.points, faces), true)
	if err != nil {
		return thing.Thing{}, err
	}
	if material == nil {
		skin := newHumanSkin(pose.human.spec)
		material = &thing.Material{Name: "adult human skin", Gloss: .025, Index: 1.40,
			Satin: .72, Texture: skin, Finish: skin, Subsurface: pose.human.diffusion,
			ThinTissue: pose.human.spec.skinTransmission}
	}
	return thing.Thing{Shape: shape, Material: material}, nil
}
