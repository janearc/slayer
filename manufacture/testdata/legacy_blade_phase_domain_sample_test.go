package manufacture

import (
	"math"
	"testing"
)

// This finite phase domain complements the analytic envelope bounds. It is
// exhaustive over blade indices, not sampling over procedural tree seeds.
func TestDiagnosticLegacyBladePhaseDomain(t *testing.T) {
	if foliageBudget != 12000 {
		t.Fatal("legacy blade budget changed; review the domain argument")
	}
	for _, row := range []conifer{spruceRow(), norwayRow()} {
		if row.FoliageStyle != 0 || row.Tall[0] < 4 || row.Tall[1] > 13 ||
			row.Hangs.Reach != .30 || row.Hangs.Shortening != .88 ||
			row.Hangs.Uneven != .15 || row.Grows.Branch.Pieces != 3 {
			t.Fatal("legacy primary envelope changed; review the domain argument")
		}
		minimum := row.Tall[0] * .30 * (1 - .88) * (1 - .15)
		maximum := row.Tall[1] * .30 * (1 + .15)
		check := func(sleeve round, low, high float64, pieces int) {
			if high/float64(pieces) >= 2 {
				t.Fatal("legacy segment exceeds coordinate bound")
			}
			for i, joint := range sleeve.Joints {
				if joint.OfGirth != 0 || joint.OfLength < 0 || joint.OfLength*high >= 1 ||
					(i < len(sleeve.Joints)-1 && joint.OfLength*low <= .002) {
					t.Fatal("legacy sleeve radii exceed analytic bounds", i, joint)
				}
			}
		}
		check(row.Grows.Clad, minimum, maximum, 3)
		if shoots := row.Grows.Shoots; shoots != nil {
			if shoots.Set.Reach != 1.1 || shoots.Set.Shortening != .4 ||
				shoots.Set.Uneven != .2 || shoots.Shoot.Pieces != 2 {
				t.Fatal("legacy terminal envelope changed; review the domain argument")
			}
			check(shoots.Wear, minimum/3*1.1*(1-.4)*(1-.2), maximum/3*1.1*(1+.2), 2)
		}
	}
	minimum, index := math.Inf(1), -1
	for blade := range foliageBudget {
		value := math.Abs(math.Cos(float64(blade) * 2.4))
		if value > 1.02 || !(math.Abs(math.Sin(float64(blade)*2.4)) <= 1.02) {
			t.Fatal("phase exceeds coordinate envelope", blade)
		}
		if value < minimum {
			minimum, index = value, blade
		}
	}
	if !(minimum > 5e-5) {
		t.Fatalf("phase lower bound no longer holds: %.17g at %d", minimum, index)
	}
	t.Logf("indices=12000 minimum-abs-cos=%.17g at-index=%d", minimum, index)
}
