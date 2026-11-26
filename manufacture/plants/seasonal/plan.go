package seasonal

import (
	"errors"
	"fmt"
	"math"

	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/maths"
)

// plantAxis is a seeded woody scaffold, shared by every surface detail.
type plantAxis struct {
	root, end   maths.Vec
	radius, tip float64
	order       int
	seed        branching.Seed
}

// Plan separates the permanent scaffold from the foliage representation.
type Plan struct {
	axes  []plantAxis
	tips  []int
	roots []plantAxis
}

// woody is every woody axis the plan draws: the scaffold, then the roots
// that show above the ground. Only the scaffold's tips carry foliage.
func (plan Plan) woody() []plantAxis {
	return append(append([]plantAxis{}, plan.axes...), plan.roots...)
}

// plan applies bounded parametric branching rules directly, without strings.
// Length contraction and child-radius powers adapt ABOP chapter2 and
// Weber-Penn.
func (profile Profile) Plan(from branching.Seed) (Plan, error) {
	plan := Plan{}
	scale := profile.Scale.Minimum +
		(profile.Scale.Maximum-profile.Scale.Minimum)*from.Draw(picking, spaceZero, spaceZero)
	if profile.Orders < spaceZero || profile.Children < spaceOne ||
		profile.Stems < spaceOne || !positive(scale) {
		return plan, fmt.Errorf("manufacture: invalid branching profile %q", profile.Kind)
	}
	if err := profile.validRoots(); err != nil {
		return Plan{}, err
	}
	plan.roots = profile.exposed(from, scale)
	budget := int(profile.value("maximum-axes"))
	if len(plan.roots) >= budget {
		return Plan{}, fmt.Errorf("manufacture: plant scaffold exceeds its axis budget")
	}
	for _, axis := range plan.roots {
		if !validPlantAxis(axis) {
			return Plan{}, fmt.Errorf("manufacture: invalid exposed root axis")
		}
	}
	if err := profile.roots(&plan, from, scale); err != nil {
		return plan, err
	}
	for _, axis := range plan.axes {
		if !validPlantAxis(axis) {
			return Plan{}, fmt.Errorf("manufacture: invalid plant stem axis")
		}
	}
	axes, tips, err := branching.Expand(plan.axes, budget-len(plan.roots),
		func(axis plantAxis, _ int) (int, error) {
			if axis.order >= profile.Orders {
				return 0, nil
			}
			return profile.Children, nil
		},
		func(axis plantAxis, index, child int) (plantAxis, error) {
			next, err := profile.child(axis, index, child, scale)
			if err != nil {
				return plantAxis{}, err
			}
			if !validPlantAxis(next) {
				return plantAxis{}, fmt.Errorf("manufacture: invalid plant branch axis")
			}
			return next, nil
		})
	if errors.Is(err, branching.ErrBudget) {
		return Plan{}, fmt.Errorf("manufacture: plant scaffold exceeds its axis budget")
	}
	if err != nil {
		return Plan{}, err
	}
	plan.axes, plan.tips = axes, tips
	return plan, nil
}

// validPlantAxis refuses nonfinite or degenerate derived geometry before it
// reaches the mesh builder. Root tips may legitimately taper to zero.
func validPlantAxis(axis plantAxis) bool {
	for _, value := range []float64{axis.root.X, axis.root.Y, axis.root.Z,
		axis.end.X, axis.end.Y, axis.end.Z, axis.tip} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return positive(axis.radius) && axis.tip >= 0 && positive(axis.end.Sub(axis.root).Len())
}

// roots gives multi-stem shrubs their own bases instead of scaled tree trunks.
func (profile Profile) roots(plan *Plan, from branching.Seed, scale float64) error {
	if profile.Stems > int(profile.value("maximum-axes")) {
		return fmt.Errorf("manufacture: excessive plant stem count")
	}
	for index := range profile.Stems {
		angle := fullTurn * float64(index) / float64(profile.Stems)
		sine, cosine := math.Sincos(angle)
		radius := scale * profile.Slenderness
		root := maths.V(profile.StemSpread*cosine, spaceZero,
			profile.StemSpread*sine)
		bend := scale * profile.value("root-lean") * from.Swing(picking, index, spaceOne)
		end := root.Add(maths.V(bend*cosine, scale*profile.Trunk, bend*sine))
		plan.axes = append(plan.axes, plantAxis{root: root, end: end,
			radius: radius, tip: radius * profile.Taper, seed: from.Sprout(index, spaceZero)})
	}
	return nil
}

// child retains local branch planes and gives a leader a different contraction.
func (profile Profile) child(parent plantAxis, index, child int,
	scale float64,
) (plantAxis, error) {
	from := parent.seed.Sprout(index, child)
	run := parent.end.Sub(parent.root)
	length := run.Len() * profile.Contraction
	if parent.order == spaceZero {
		length = scale * profile.Reach
	}
	angle := profile.Fork * maths.Degree
	if profile.Leader && child == spaceZero {
		angle *= profile.value("leader-angle-share")
		length *= profile.LeaderGain
	}
	length *= spaceOne + profile.value("length-variation")*from.Swing(growing, index, child)
	turn := fullTurn*float64(child)/float64(profile.Children) +
		profile.Twist*maths.Degree*from.Swing(growing, child, spaceOne)
	sinTurn, cosTurn := math.Sincos(turn)
	sinAngle, cosAngle := math.Sincos(angle)
	local := maths.V(sinAngle*cosTurn, cosAngle, -sinAngle*sinTurn)
	orientation := maths.Upright(run)
	direction := orientation.Out(local)
	direction = direction.Scale(spaceOne - profile.Upward).
		Add(maths.YAxis.Scale(profile.Upward)).Unit()
	radius := parent.radius * math.Pow(length/run.Len(), profile.RadiusPower)
	if !positive(radius) || !positive(length) {
		return plantAxis{}, fmt.Errorf("manufacture: invalid contracted plant axis")
	}
	inset := direction.Scale(radius * profile.value("branch-inset"))
	root := parent.end.Sub(inset)
	return plantAxis{root: root, end: parent.end.Add(direction.Scale(length)),
		radius: radius, tip: radius * profile.Taper,
		order: parent.order + spaceOne, seed: from}, nil
}
