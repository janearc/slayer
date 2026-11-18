package beamstep_test

import (
	"testing"

	"github.com/janearc/slayer/render/beamstep"
)

// The three budgets are the numbers authored in choices.yaml: a refinement
// envelope of 3 radii, 4 radial samples to a radius and at most 64 splits
// of one march cell.
func TestTheBudgetsAreTheAuthoredChoices(t *testing.T) {
	if got := beamstep.Radii(); got != 3 {
		t.Errorf("Radii is %v, want 3", got)
	}
	if got := beamstep.RadialSamples(); got != 4 {
		t.Errorf("RadialSamples is %v, want 4", got)
	}
	if got := beamstep.MaximumSplits(); got != 64 {
		t.Errorf("MaximumSplits is %v, want 64", got)
	}
}
