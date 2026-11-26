package conifers

import (
	"github.com/janearc/slayer/manufacture/branching"
	"github.com/janearc/slayer/manufacture/geometry"
)

// Round describes a sleeve before selecting a material role.
type Round struct {
	Joints []branching.Joint
	Sides  int
}

func (shape Round) in(role int) (branching.Sleeve, error) {
	if shape.Sides < 3 || shape.Sides > 64 {
		return branching.Sleeve{}, geometry.ErrCut
	}
	return branching.NewSleeve(shape.Joints, shape.Sides, role)
}

// Limb describes the division and variation of a branch.
type Limb struct {
	Pieces        int
	Droop, Wander float64
}

// Branchlets describe the shoots carried by each branch.
type Branchlets struct {
	Per   [2]float64
	Shoot Limb
	Set   branching.Habit
	Wear  Round
}

// Architecture is a stem with whorled branches and optional branchlets.
type Architecture struct {
	Trunk, Bark Round
	Branch      Limb
	Clad        Round
	Shoots      *Branchlets
}

// Grown builds the shared branching structure for the selected counts.
func (tree Architecture) Grown(count, lowest, sprays int, set branching.Habit, body, leaf int) (branching.Growth, error) {
	trunk, err := tree.Trunk.in(body)
	if err != nil {
		return branching.Growth{}, err
	}
	bark, err := tree.Bark.in(body)
	if err != nil {
		return branching.Growth{}, err
	}
	clad, err := tree.Clad.in(leaf)
	if err != nil {
		return branching.Growth{}, err
	}
	branch, err := tree.Branching(bark, clad, sprays, leaf)
	if err != nil {
		return branching.Growth{}, err
	}
	return branching.TrunkCarrying(trunk, count, lowest, set, branch)
}

// Branching builds one branch and its optional terminal branchlets.
func (tree Architecture) Branching(bark, clad branching.Sleeve, sprays, leaf int) (branching.Growth, error) {
	if tree.Shoots == nil {
		bough, err := branching.NewLimb(tree.Branch.Pieces, 1, tree.Branch.Droop, bark, clad)
		if err != nil {
			return branching.Growth{}, err
		}
		bough.Wander = tree.Branch.Wander
		return branching.NewGrowth(bough)
	}
	shoots, err := branching.NewWhorls(sprays, 1, tree.Shoots.Set)
	if err != nil {
		return branching.Growth{}, err
	}
	worn, err := tree.Shoots.Wear.in(leaf)
	if err != nil {
		return branching.Growth{}, err
	}
	worn.Terminal = true
	carrying, err := branching.StemCarrying(tree.Branch.Pieces, tree.Branch.Droop,
		[]branching.Sleeve{bark, clad}, shoots, tree.Shoots.Shoot.Pieces,
		tree.Shoots.Shoot.Droop, []branching.Sleeve{worn})
	carrying.Stem.Wander = tree.Branch.Wander
	return carrying, err
}

// Resampled eases a sleeve's joints to a new number of pieces.
func Resampled(joints []branching.Joint, count int) []branching.Joint {
	out := make([]branching.Joint, count)
	last := float64(len(joints) - 1)
	for at := range out {
		place := float64(at) / float64(count-1) * last
		low := min(int(place), len(joints)-2)
		f := place - float64(low)
		a, b := joints[low], joints[low+1]
		out[at] = branching.Joint{OfGirth: a.OfGirth + f*(b.OfGirth-a.OfGirth),
			OfLength: a.OfLength + f*(b.OfLength-a.OfLength)}
	}
	return out
}

// Flowing spreads the original droop over more, gently wandering pieces.
func (tree Architecture) Flowing(pieces int, wander float64) Architecture {
	if tree.Clad.Joints == nil || tree.Branch.Pieces < 2 {
		return tree
	}
	joints := tree.Branch.Pieces - 1
	tree.Branch = Limb{Pieces: pieces, Droop: tree.Branch.Droop * float64(joints) / float64(pieces-1), Wander: wander}
	tree.Bark.Joints = Resampled(tree.Bark.Joints, pieces+1)
	tree.Clad.Joints = Resampled(tree.Clad.Joints, pieces+1)
	return tree
}
