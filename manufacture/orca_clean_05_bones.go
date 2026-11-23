package manufacture

import (
	"fmt"
	"math"
	"sync"

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
func orcaSpinePath(share float64) (maths.Vec, float64) {
	x := orcaCondyleX + share*(orcaColumnEnd-orcaCondyleX)
	s := orcaTrunkProfile().at(x)
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
func orcaSkull(bones *skeleton.Skeleton) error {
	condyle, _ := orcaSpinePath(0)
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

// orcaSkeletonOnce builds the rest skeleton once; it is shared after.
var (
	orcaSkeletonOnce   sync.Once
	orcaSkeletonShared *skeleton.Skeleton
	orcaSkeletonErr    error
)

// orcaSkeleton is the animal's rest skeleton: skull, column, the
// processes of the vertebrae, ribs and sternum, the flippers' bones and
// the pelvic vestiges. There is no hind limb, and no bone in the dorsal
// fin or the flukes, which are fibrous.
func orcaSkeleton() (*skeleton.Skeleton, error) {
	orcaSkeletonOnce.Do(func() {
		bones := skeleton.New()
		for _, add := range []func(*skeleton.Skeleton) error{orcaSkull,
			orcaToothRows, orcaColumn, orcaProcesses, orcaRibs,
			orcaFlippers, orcaPelvis} {
			if err := add(bones); err != nil {
				orcaSkeletonErr = fmt.Errorf("orca skeleton: %w", err)
				return
			}
		}
		orcaSkeletonShared = bones
	})
	return orcaSkeletonShared, orcaSkeletonErr
}

// orcaColumn adds the vertebrae, the first jointed to the skull.
func orcaColumn(bones *skeleton.Skeleton) error {
	return bones.Column("cranium", orcaRegions, orcaSpinePath)
}

// orcaBoneMaterial is dry bone: pale, matte, its colour chosen.
func orcaBoneMaterial() *thing.Material {
	return &thing.Material{Name: "orca bone",
		Colour: maths.RGB(.62, .58, .50)}
}

// OrcaClean05Bones is the animal's skeleton alone, at rest, every bone
// closed in its own form: a study of what the skin lies over.
func OrcaClean05Bones(at maths.Subspace) ([]thing.Thing, error) {
	bones, err := orcaSkeleton()
	if err != nil {
		return nil, err
	}
	points, faces := bones.Draw(bones.Frames(skeleton.Pose{}),
		orcaForms(bones))
	normals := orcaVertexNormals(points, faces)
	drawn, err := mesh.New(points, faces, normals, true)
	if err != nil {
		return nil, err
	}
	return []thing.Thing{{Shape: drawn, Material: orcaBoneMaterial(),
		Motion: thing.Still(at)}}, nil
}
