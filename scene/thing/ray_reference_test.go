package thing

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
)

// A ray-origin reference is a length above nought and not endless, kept
// in metres. The anti-tests: nought, a length below it, an endless one
// and one that is no number are each refused; and the zero reference,
// the choice made from the geometry, is nought.
func TestARayOriginReferenceIsAPositiveLength(t *testing.T) {
	for _, metres := range []float64{1e-9, 1, 6.371e6} {
		made, err := NewRayOriginReference(measure.Metres(metres))
		if err != nil || made.Metres() != metres {
			t.Errorf("%v metres: %v, %v", metres, made.Metres(), err)
		}
	}
	for _, metres := range []float64{0, -1, math.Inf(1), math.Inf(-1), math.NaN()} {
		made, err := NewRayOriginReference(measure.Metres(metres))
		if err == nil || made.Metres() != 0 {
			t.Errorf("%v metres were taken: %v, %v", metres, made.Metres(), err)
		}
	}
	if got := (RayOriginReference{}).Metres(); got != 0 {
		t.Errorf("the zero reference is %v metres", got)
	}
}
