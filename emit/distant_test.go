package emit

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a distant light comes the same way, as strong, to every point, from
// an endless distance: a sun seen from a planet.
func TestADistantLightIsTheSameEverywhere(t *testing.T) {
	sun, err := NewDistant(maths.V(1, 2, 2), maths.Grey(1.5))
	if err != nil {
		t.Fatal(err)
	}
	want := maths.V(1, 2, 2).Unit()
	for _, at := range []maths.Vec{{}, maths.V(1e6, -3, 7),
		maths.V(-6.4e6, 0, 0)} {
		toward, far, light := sun.Light(at, maths.V(0, 1, 0))
		if !toward.Near(want, 1e-15) || !math.IsInf(far, 1) ||
			light != maths.Grey(1.5) {
			t.Errorf("at %v: toward %v, %g away, light %v", at, toward,
				far, light)
		}
	}
}

// a distant light needs a way to come from and a light that is not
// less than none.
func TestADistantLightWithNoWayIsRefused(t *testing.T) {
	for _, wrong := range []struct {
		toward maths.Vec
		light  maths.Light
	}{{maths.Vec{}, maths.Grey(1)}, {maths.V(0, 1, 0), maths.Grey(-1)},
		{maths.V(math.NaN(), 1, 0), maths.Grey(1)},
		{maths.V(0, 1, 0), maths.Grey(math.Inf(1))}} {
		if _, err := NewDistant(wrong.toward, wrong.light); !errors.Is(
			err, ErrDistant) {
			t.Errorf("%v of %v is a distant light: %v", wrong.toward,
				wrong.light, err)
		}
	}
}
