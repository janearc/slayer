package manufacture

import (
	"github.com/janearc/celestial/measure"
	"math"
	"testing"
)

func TestLegacyBladesStayLeafSizedAndNearTheirSupportingAxis(t *testing.T) {
	for _, length := range []float64{.4, 1, 2} {
		for _, radius := range []float64{.1, 1} {
			axis, err := newTube(measure.Metres(length), measure.Metres(radius), measure.Metres(radius/5))
			if err != nil {
				t.Fatal(err)
			}
			for _, style := range []int{0, 1, 2} {
				spray := foliageSpray{envelope: axis, count: 64, seed: Sow(5), style: style}
				skin := spray.skin(cut{})
				if len(skin) != 256 {
					t.Fatalf("style %d lost blade surfaces", style)
				}
				for _, face := range skin {
					for i, p := range face {
						// The authored needles are at most 25 mm long. Their vertices stay
						// within that margin of the supporting branch envelope.
						if math.IsNaN(p.X+p.Y+p.Z) || math.IsInf(p.X+p.Y+p.Z, 0) || p.Y < -.025 || p.Y > length+.025 || math.Hypot(p.X, p.Z) > radius+.025 {
							t.Fatalf("style %d blade left its support: %v", style, p)
						}
						if edge := p.Sub(face[(i+1)%3]).Len(); !(edge > 0 && edge <= .03) {
							t.Fatalf("style %d blade edge %g is no longer leaf sized", style, edge)
						}
					}
				}
			}
		}
	}
}
