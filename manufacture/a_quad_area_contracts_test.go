package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestQuadAreaFloorKeepsItsExactGeometricEndpoint(t *testing.T) {
	// A unit-height rectangle's triangle cross-product has magnitude
	// equal to its width. These are real positive-area inputs at either
	// side of the authored rejection floor; no constructor is replaced.
	const floor = 1e-14
	lower := math.Nextafter(math.Nextafter(floor, 0), 0)
	higher := math.Nextafter(math.Nextafter(floor, math.Inf(1)), math.Inf(1))
	for _, test := range []struct {
		width float64
		faces int
	}{{lower, 0}, {floor, 0}, {higher, 2}} {
		skin := quad(nil, maths.V(0, 0, 0), maths.V(test.width, 0, 0),
			maths.V(test.width, 1, 0), maths.V(0, 1, 0))
		if len(skin) != test.faces {
			t.Fatalf("rectangle width %.17g: %d faces, want %d",
				test.width, len(skin), test.faces)
		}
	}
}
