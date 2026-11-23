package thing

import (
	"testing"

	"github.com/janearc/celestial/measure"
)

// The tables the ray-reference test reads, each row named in its name
// column.
const (
	rayReferenceAccepted = "testdata/ray-reference-accepted.csv"
	rayReferenceRefused  = "testdata/ray-reference-refused.csv"
)

// A ray-origin reference is a length above nought and not endless, kept
// in metres. The anti-tests: nought, a length below it, an endless one
// and one that is no number are each refused; and the zero reference,
// the choice made from the geometry, is nought.
func TestARayOriginReferenceIsAPositiveLength(t *testing.T) {
	for _, row := range readSheet(t, rayReferenceAccepted) {
		metres := cell(t, row, "metres")
		made, err := NewRayOriginReference(measure.Metres(metres))
		if err != nil || made.Metres() != metres {
			t.Errorf("%v metres: %v, %v", metres, made.Metres(), err)
		}
	}
	for _, row := range readSheet(t, rayReferenceRefused) {
		metres := cell(t, row, "metres")
		made, err := NewRayOriginReference(measure.Metres(metres))
		if err == nil || made.Metres() != nought {
			t.Errorf("%v metres were taken: %v, %v", metres, made.Metres(), err)
		}
	}
	if got := (RayOriginReference{}).Metres(); got != nought {
		t.Errorf("the zero reference is %v metres", got)
	}
}
