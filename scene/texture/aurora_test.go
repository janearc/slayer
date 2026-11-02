package texture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// The aurora's colour runs fringe, bottom, top with height.
func TestAuroraColourRunsWithHeight(t *testing.T) {
	a := Aurora{Radius: 10, Low: 1, High: 3, Fringe: maths.RGB(0.6, 0.3, 1),
		Bottom: maths.RGB(0.3, 1, 0.35), Top: maths.RGB(1, 0.2, 0.2)}
	if c := a.At(maths.V(11.0, 0, 0)); !(c.B > c.G) {
		t.Errorf("the fringe is %v", c)
	}
	if c := a.At(maths.V(11.4, 0, 0)); !(c.G > c.R && c.G > c.B) {
		t.Errorf("the bottom is %v", c)
	}
	if c := a.At(maths.V(12.9, 0, 0)); !(c.R > c.G) {
		t.Errorf("the top is %v", c)
	}
}
