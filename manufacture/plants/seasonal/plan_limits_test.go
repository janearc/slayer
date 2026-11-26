package seasonal

import (
	"math"
	"strings"
	"testing"
)

func TestPlantPlanBudgetsExposedRootsAndBranchesTogether(t *testing.T) {
	profile := plantProfiles["bruguiera-gymnorhiza"]
	roots := *profile.Roots
	profile.Roots = &roots
	profile.Roots.Habit = rootProp
	profile.Roots.Count.Minimum = 128
	profile.Roots.Count.Maximum = 128
	profile.Stems = 1
	profile.Children = 2
	profile.Orders = 7 // 255 scaffold axes, plus 256 prop-root segments.
	plan, err := profile.Plan(sow(1))
	if err != nil || len(plan.axes) != 255 || len(plan.roots) != 256 {
		t.Fatalf("near-budget plan: %d axes, %d roots, %v", len(plan.axes), len(plan.roots), err)
	}
	profile.Children = 4
	profile.Orders = 4 // 341 scaffold axes, plus 256 prop-root segments.
	if _, err := profile.Plan(sow(1)); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("expected combined woody-axis budget refusal, got %v", err)
	}
}

func TestPlantPlanRefusesDerivedNonfiniteGeometry(t *testing.T) {
	base := plantProfiles["valley-oak"]
	for _, tc := range []struct {
		name   string
		change func(*Profile)
	}{
		{"root radius overflows", func(p *Profile) {
			p.Scale.Minimum, p.Scale.Maximum = math.MaxFloat64, math.MaxFloat64
			p.Slenderness = 2
		}},
		{"child length overflows", func(p *Profile) {
			p.Scale.Minimum, p.Scale.Maximum = 3, 3
			p.Reach = math.MaxFloat64
			p.Orders = 1
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile := base
			tc.change(&profile)
			if _, err := profile.Plan(sow(1)); err == nil {
				t.Fatal("accepted nonfinite derived axis")
			}
		})
	}
}

func TestPlantPlanRefusesNonfiniteExposedRoot(t *testing.T) {
	profile := plantProfiles["bruguiera-gymnorhiza"]
	roots := *profile.Roots
	profile.Roots = &roots
	profile.Roots.Spread = math.MaxFloat64
	profile.Scale.Minimum, profile.Scale.Maximum = 3, 3
	if _, err := profile.Plan(sow(1)); err == nil {
		t.Fatal("accepted nonfinite exposed root")
	}
}
