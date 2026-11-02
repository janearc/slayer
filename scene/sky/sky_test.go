package sky

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

var g = Gradient{Horizon: maths.Grey(0.9), Zenith: maths.RGB(0.3, 0.5, 0.9),
	Ground: maths.Grey(0.2)}

// Level with the horizon it is the horizon's colour, straight up the
// zenith's, and below the horizon the ground's.
func TestGradient(t *testing.T) {
	if got := g.Light(maths.V(1, 0, 0)); got != g.Horizon {
		t.Errorf("level is %v", got)
	}
	if got := g.Light(maths.V(0, 5, 0)); got != g.Zenith {
		t.Errorf("up is %v", got)
	}
	if got := g.Light(maths.V(0, -1, 1)); got != g.Ground {
		t.Errorf("down is %v", got)
	}
}

// Up can be anywhere: a sky tipped on its side is the zenith's colour
// along the new up.
func TestUpCanMove(t *testing.T) {
	tipped := g
	tipped.Up = maths.V(1, 0, 0)
	if got := tipped.Light(maths.V(3, 0, 0)); got != g.Zenith {
		t.Errorf("along the new up it is %v", got)
	}
}
