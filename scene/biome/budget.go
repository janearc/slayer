package biome

import (
	_ "embed"
	"fmt"
	"math"

	"github.com/janearc/puzzlemouth/numbers"
)

//go:embed selection-policy.yaml
var policyRaw []byte

// DefaultBudget reads authored per-query work limits through Puzzlemouth.
// A streamed-world manager must still cap its aggregate work across regions.
func DefaultBudget() (Budget, error) {
	policy, err := numbers.Read("selection-policy.yaml", policyRaw)
	if err != nil {
		return Budget{}, err
	}
	var budget Budget
	for name, target := range map[string]*int{"cells": &budget.Cells, "candidates": &budget.Candidates} {
		value, err := policy.Get(name)
		if err != nil {
			return Budget{}, err
		}
		if !finite(value.Value) || value.Value < 1 ||
			value.Value >= float64(math.MaxInt) || value.Value != math.Trunc(value.Value) {
			return Budget{}, fmt.Errorf("biome: invalid selection policy %s", name)
		}
		*target = int(value.Value)
	}
	return budget, nil
}
