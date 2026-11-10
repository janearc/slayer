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
		if row.foliageStyle != 0 || row.tall.least < 4 || row.tall.most > 13 ||
			row.hangs.reach != .30 || row.hangs.shortening != .88 ||
			row.hangs.uneven != .15 || row.grows.branch.pieces != 3 {
			t.Fatal("legacy primary envelope changed; review the domain argument")
		}
		minimum := row.tall.least * .30 * (1 - .88) * (1 - .15)
		maximum := row.tall.most * .30 * (1 + .15)
		check := func(wear foliage, low, high float64, pieces int) {
			sleeve := wear.(sleeveFoliage)
			if high/float64(pieces) >= 2 {
				t.Fatal("legacy segment exceeds coordinate bound")
			}
			for i, joint := range sleeve.joints {
				if joint.ofGirth != 0 || joint.ofLength < 0 || joint.ofLength*high >= 1 ||
					(i < len(sleeve.joints)-1 && joint.ofLength*low <= .002) {
					t.Fatal("legacy sleeve radii exceed analytic bounds", i, joint)
				}
			}
		}
		check(row.grows.clad, minimum, maximum, 3)
		if shoots := row.grows.shoots; shoots != nil {
			if shoots.set.reach != 1.1 || shoots.set.shortening != .4 ||
				shoots.set.uneven != .2 || shoots.shoot.pieces != 2 {
				t.Fatal("legacy terminal envelope changed; review the domain argument")
			}
			check(shoots.wear, minimum/3*1.1*(1-.4)*(1-.2), maximum/3*1.1*(1+.2), 2)
		}
	}
	minimum, index := math.Inf(1), -1
	for blade := range foliageBudget {
		value := math.Abs(math.Cos(float64(blade) * 2.4))
		if value > 1.02 || math.Abs(math.Sin(float64(blade)*2.4)) > 1.02 {
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
