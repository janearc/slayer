package terrain

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// PatchView is a caller-supplied visibility and screen-footprint assessment.
// Score is pixels per chosen sampling interval, or another declared error unit.
// The selector does not infer a geometric bound from skipped height samples.
type PatchView struct {
	Visible bool
	Score   float64
}

// SelectionBudget bounds retained patches, callback work and subdivision depth.
// Target uses the same declared metric as the view callback.
type SelectionBudget struct {
	Patches, Evaluations, Depth int
	Target                      float64
}

// Selection retains coarse coverage when detail cannot fit the chosen budget.
// Limited is explicit; it never claims the requested quality was attained.
type Selection struct {
	Patches     []Patch
	Evaluations int
	Limited     bool
}

// SelectPatches refines the highest-scoring visible source patch first. A
// budget refusal leaves its coarse parent, instead of dropping visible ground.
// The callback must provide conservative visibility and its declared score.
func SelectPatches(root Patch, budget SelectionBudget,
	view func(Patch) (PatchView, error),
) (Selection, error) {
	if !root.valid() || budget.Patches < 1 || budget.Evaluations < 1 || budget.Depth < root.Level ||
		budget.Depth >= strconv.IntSize || !finiteCoordinate(budget.Target) || budget.Target <= 0 || view == nil {
		return Selection{}, fmt.Errorf("terrain: invalid patch selection or work budget")
	}
	first, err := assessPatch(root, view)
	if err != nil {
		return Selection{}, err
	}
	selection := Selection{Evaluations: 1}
	if !first.visible {
		return selection, nil
	}
	front := patchHeap{first}
	leaves := map[Patch]patchCandidate{root: first}
	for len(front) > 0 {
		next := heap.Pop(&front).(patchCandidate)
		if next.score <= budget.Target {
			break
		}
		if err := refinePatch(next, budget, view, &selection, leaves, &front); err != nil {
			return Selection{}, err
		}
	}
	for patch := range leaves {
		selection.Patches = append(selection.Patches, patch)
	}
	sort.Slice(selection.Patches, func(one, other int) bool {
		return patchBefore(selection.Patches[one], selection.Patches[other])
	})
	return selection, nil
}

// refinePatch replaces a parent only after every visible child fits the budget.
func refinePatch(next patchCandidate, budget SelectionBudget,
	view func(Patch) (PatchView, error), selection *Selection,
	leaves map[Patch]patchCandidate, front *patchHeap,
) error {
	if next.patch.Level >= budget.Depth {
		selection.Limited = true
		return nil
	}
	children, err := next.patch.Children()
	if err != nil {
		return err
	}
	if len(children) == 0 {
		selection.Limited = true
		return nil
	}
	if len(children) > budget.Evaluations-selection.Evaluations {
		selection.Limited = true
		return nil
	}
	var visible []patchCandidate
	for _, child := range children {
		candidate, err := assessPatch(child, view)
		selection.Evaluations++
		if err != nil {
			return err
		}
		if candidate.visible {
			visible = append(visible, candidate)
		}
	}
	if len(visible) > budget.Patches-(len(leaves)-1) {
		selection.Limited = true
		return nil
	}
	delete(leaves, next.patch)
	for _, candidate := range visible {
		leaves[candidate.patch] = candidate
		heap.Push(front, candidate)
	}
	return nil
}

// assessPatch propagates mapper/view errors and refuses invalid score
// arithmetic.
func assessPatch(patch Patch, view func(Patch) (PatchView, error)) (patchCandidate, error) {
	result, err := view(patch)
	if err != nil {
		return patchCandidate{}, err
	}
	if math.IsNaN(result.Score) || math.IsInf(result.Score, 0) || result.Score < 0 {
		return patchCandidate{}, fmt.Errorf("terrain: invalid patch view score")
	}
	return patchCandidate{patch: patch, score: result.Score, visible: result.Visible}, nil
}
