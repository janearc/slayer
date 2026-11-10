package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// This producer exhausts the same twenty compiled prototype plans. It checks
// only the sign used to orient closed needles; volume magnitudes are private.
func TestDiagnosticNeedleVolumeSignsAcrossThePrototypeDomain(t *testing.T) {
	if shootValue("shoot-minimum-cell") != .025 || shootValue("shoot-cell-length") != .4 {
		t.Fatal("prototype size domain changed; review this exhaustive producer")
	}
	needles, surfaces := 0, 0
	minimum := math.Inf(1)
	for _, style := range []int{1, 2} {
		for bin := range 5 {
			for variant := range 2 {
				plan := planShoot(style, .025*math.Exp2(float64(bin)), Sow(uint64(variant)))
				for _, needle := range plan.needles {
					needles++
					for _, detail := range []string{"quick", "demo"} {
						skin := needle.skin(detail)
						for _, origin := range []maths.Vec{skin[0][0], skin[1][0], skin[0][1], skin[0][2]} {
							sum := 0.0
							for _, face := range skin {
								sum += face[0].Sub(origin).Dot(
									face[1].Sub(origin).Cross(face[2].Sub(origin)))
							}
							if !(sum > 0) || math.IsInf(sum, 0) {
								t.Fatalf("style %d bin %d variant %d detail %s volume %g",
									style, bin, variant, detail, sum)
							}
							minimum = math.Min(minimum, sum)
						}
						surfaces++
					}
				}
			}
		}
	}
	t.Logf("needles=%d closed-surfaces=%d minimum-six-volume=%.17g",
		needles, surfaces, minimum)
}
