package manufacture

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/maths"
)

// plantAxis is a seeded woody scaffold, shared by every surface detail.
type plantAxis struct {
	root, end   maths.Vec
	radius, tip float64
	order       int
	seed        seed
}

// plantPlan separates the permanent scaffold from the foliage representation.
type plantPlan struct {
	axes []plantAxis
	tips []int
}

// plan applies bounded parametric branching rules directly, without strings.
// Length contraction and child-radius powers adapt ABOP chapter2 and
// Weber-Penn.
func (profile plantProfile) plan(from seed) (plantPlan, error) {
	plan := plantPlan{}
	scale := profile.Scale.Minimum +
		(profile.Scale.Maximum-profile.Scale.Minimum)*from.draw(picking, spaceZero, spaceZero)
	if profile.Orders < spaceZero || profile.Children < spaceOne ||
		profile.Stems < spaceOne || !positive(scale) {
		return plan, fmt.Errorf("manufacture: invalid branching profile %q", profile.Kind)
	}
	if err := profile.roots(&plan, from, scale); err != nil {
		return plan, err
	}
	for index := spaceZero; index < len(plan.axes); index++ {
		axis := plan.axes[index]
		if axis.order >= profile.Orders {
			plan.tips = append(plan.tips, index)
			continue
		}
		if len(plan.axes)+profile.Children > int(plantValue("maximum-axes")) {
			return plantPlan{}, fmt.Errorf("manufacture: plant scaffold exceeds its axis budget")
		}
		for child := range profile.Children {
			next, err := profile.child(axis, index, child, scale)
			if err != nil {
				return plantPlan{}, err
			}
			plan.axes = append(plan.axes, next)
		}
	}
	return plan, nil
}

// roots gives multi-stem shrubs their own bases instead of scaled tree trunks.
func (profile plantProfile) roots(plan *plantPlan, from seed, scale float64) error {
	if profile.Stems > int(plantValue("maximum-axes")) {
		return fmt.Errorf("manufacture: excessive plant stem count")
	}
	for index := range profile.Stems {
		angle := fullTurn * float64(index) / float64(profile.Stems)
		sine, cosine := math.Sincos(angle)
		radius := scale * profile.Slenderness
		root := maths.V(profile.StemSpread*cosine, spaceZero,
			profile.StemSpread*sine)
		bend := scale * plantValue("root-lean") * from.swing(picking, index, spaceOne)
		end := root.Add(maths.V(bend*cosine, scale*profile.Trunk, bend*sine))
		plan.axes = append(plan.axes, plantAxis{root: root, end: end,
			radius: radius, tip: radius * profile.Taper, seed: from.sprout(index, spaceZero)})
	}
	return nil
}

// child retains local branch planes and gives a leader a different contraction.
func (profile plantProfile) child(parent plantAxis, index, child int,
	scale float64,
) (plantAxis, error) {
	from := parent.seed.sprout(index, child)
	run := parent.end.Sub(parent.root)
	length := run.Len() * profile.Contraction
	if parent.order == spaceZero {
		length = scale * profile.Reach
	}
	angle := profile.Fork * maths.Degree
	if profile.Leader && child == spaceZero {
		angle *= plantValue("leader-angle-share")
		length *= profile.LeaderGain
	}
	length *= spaceOne + plantValue("length-variation")*from.swing(growing, index, child)
	turn := fullTurn*float64(child)/float64(profile.Children) +
		profile.Twist*maths.Degree*from.swing(growing, child, spaceOne)
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
	inset := direction.Scale(radius * plantValue("branch-inset"))
	root := parent.end.Sub(inset)
	return plantAxis{root: root, end: parent.end.Add(direction.Scale(length)),
		radius: radius, tip: radius * profile.Taper,
		order: parent.order + spaceOne, seed: from}, nil
}
