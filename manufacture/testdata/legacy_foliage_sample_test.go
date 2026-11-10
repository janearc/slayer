package manufacture

import (
	"fmt"
	"github.com/janearc/celestial/measure"
	"testing"
)

func TestDiagnosticPrintLegacyFoliageSamples(t *testing.T) {
	axis, err := newTube(measure.Metres(1), measure.Metres(.1), measure.Metres(.02))
	if err != nil {
		t.Fatal(err)
	}
	for _, style := range []int{0, 1, 2} {
		spray := foliageSpray{envelope: axis, count: 16, seed: Sow(5), style: style}
		skin := spray.skin(cut{})
		for _, i := range []int{0, 3, 28, 31, 60, 63} {
			for j, p := range skin[i] {
				fmt.Printf("{%d, %d, %d, maths.V(%.17g, %.17g, %.17g)},\n", style, i, j, p.X, p.Y, p.Z)
			}
		}
	}
}
