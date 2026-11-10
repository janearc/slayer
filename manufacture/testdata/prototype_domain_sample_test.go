package manufacture

import (
	"math"
	"testing"
)

func TestDiagnosticCheckTheCompletePrototypeLengthDomain(t *testing.T) {
	if shootValue("shoot-minimum-cell") != .025 || shootValue("shoot-cell-length") != .4 {
		t.Fatal("public prototype size domain changed; review this exhaustive producer")
	}
	plans, axes, equalities := 0, 0, 0
	minimumCount := math.MaxInt
	for _, style := range []int{1, 2} {
		for bin := range 5 {
			for variant := range 2 {
				length := .025 * math.Exp2(float64(bin))
				plan := planShoot(style, length, Sow(uint64(variant)))
				lowest := math.Inf(1)
				for i, axis := range plan.axes {
					axes++
					count := int(axis.length / needleSpacing(style))
					minimumCount = min(minimumCount, count)
					if i > 0 {
						lowest = math.Min(lowest, axis.length)
						if axis.length == 2*needleSpacing(style) {
							equalities++
						}
					}
				}
				t.Logf("style=%d bin=%d variant=%d length=%.17g axes=%d shortest-child=%.17g",
					style, bin, variant, length, len(plan.axes), lowest)
				plans++
			}
		}
	}
	t.Logf("plans=%d axes=%d exact-cutoff-equalities=%d minimum-needle-count=%d",
		plans, axes, equalities, minimumCount)
}
