// Package branching expands bounded recursive structures without prescribing
// their shape, materials, randomness, or the meaning of a node.
package branching

import (
	"errors"
	"fmt"
	"slices"
)

// ErrBudget says the expansion would exceed its total node budget.
var ErrBudget = errors.New("branching: node budget exceeded")

// Expand grows roots breadth first. Count says how many children a node has;
// Child builds each child in order, receiving the parent's index in the
// returned traversal. That index can change if an earlier branch changes.
// A node with zero children is a leaf. The returned leaf indexes refer to
// the returned nodes. The input slice is not modified. No partial result is
// returned on error. The budget counts nodes, not geometry or other resources
// a callback may allocate, including root or mesh geometry; callers validate
// their own domain values and resource use.
// A node can carry its own semantic path or seed: Child need not use the
// traversal index to derive identity.
func Expand[T any](roots []T, limit int, count func(T, int) (int, error), child func(T, int, int) (T, error)) ([]T, []int, error) {
	if len(roots) == 0 || limit < 1 || count == nil || child == nil {
		return nil, nil, fmt.Errorf("branching: roots, budget and rules are required")
	}
	if len(roots) > limit {
		return nil, nil, ErrBudget
	}
	nodes := slices.Clone(roots)
	var leaves []int
	for index := 0; index < len(nodes); index++ {
		parent := nodes[index]
		n, err := count(parent, index)
		if err != nil {
			return nil, nil, err
		}
		if n < 0 {
			return nil, nil, fmt.Errorf("branching: negative child count at node %d", index)
		}
		if n > limit-len(nodes) {
			return nil, nil, ErrBudget
		}
		if n == 0 {
			leaves = append(leaves, index)
			continue
		}
		for j := 0; j < n; j++ {
			next, err := child(parent, index, j)
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, next)
		}
	}
	return nodes, leaves, nil
}
