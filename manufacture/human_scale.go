package manufacture

import (
	_ "embed"
	"math"

	"github.com/janearc/puzzlemouth/yamlfile"
	"github.com/janearc/slayer/maths"
)

//go:embed human-scale.yaml
var humanScaleRaw []byte

// scaleHuman is an authored standing mannequin, exactly two metres high.
// It is a scale reference, with no claim to measured human anatomy.
type scaleHuman struct{}

type humanOval struct {
	At, Size maths.Vec
	Paired   bool
}

type humanLimb struct {
	From, To maths.Vec
	Radius   float64
}

type humanRecipe struct {
	Material, Finish string
	Ovals            []humanOval
	Limbs            []humanLimb
}

// made uses the normal specimen detail.
func (row scaleHuman) made(from seed) (good, error) {
	return row.madeAt(from, "quick")
}

// madeAt constructs the same height and stance at every tessellation.
func (scaleHuman) madeAt(from seed, detail string) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	var recipe humanRecipe
	if err := yamlfile.Read("human-scale.yaml", humanScaleRaw, &recipe); err != nil {
		return good{}, err
	}
	return recipe.construct(detail)
}

// construct builds closed ellipsoids and rods, retaining the first refusal.
func (recipe humanRecipe) construct(detail string) (good, error) {
	look, err := surfaceOf("scale human", recipe.Material, recipe.Finish)
	if err != nil {
		return good{}, err
	}
	assembly := &instrumentAssembly{fine: cut{around: int(groundValue("sides-" + detail))},
		object: good{dressed: looks{body: look}}}
	for _, oval := range recipe.Ovals {
		assembly.add(scaleOval{oval.Size}, maths.NewSubspace(oval.At, maths.Identity, 1), body)
		if oval.Paired {
			oval.At.X = -oval.At.X
			assembly.add(scaleOval{oval.Size}, maths.NewSubspace(oval.At, maths.Identity, 1), body)
		}
	}
	for _, limb := range recipe.Limbs {
		assembly.rod(limb.From, limb.To, limb.Radius, body)
		limb.From.X, limb.To.X = -limb.From.X, -limb.To.X
		assembly.rod(limb.From, limb.To, limb.Radius, body)
	}
	if assembly.trouble != nil {
		return good{}, assembly.trouble
	}
	return assembly.object, nil
}

// scaleOval is a closed ellipsoid standing at local y=0.
type scaleOval struct{ size maths.Vec }

// skin revolves a latitude profile, then gives it independent width and depth.
func (oval scaleOval) skin(fine cut) surface {
	// Picked display budget: finer rings do not improve a scale silhouette.
	fine.around = min(fine.around, 48)
	steps := max(4, fine.around/2)
	points := make([][2]float64, steps+1)
	for index := 1; index < steps; index++ {
		angle := math.Pi * float64(index) / float64(steps)
		points[index] = [2]float64{math.Sin(angle) * oval.size.X / 2,
			(1 - math.Cos(angle)) * oval.size.Y / 2}
	}
	points[steps] = [2]float64{0, oval.size.Y}
	skin := (turnedProfile{points}).skin(fine)
	for index := range skin {
		for corner := range skin[index] {
			skin[index][corner].Z *= oval.size.Z / oval.size.X
		}
	}
	return skin
}
