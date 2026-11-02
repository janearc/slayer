package shortcut

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// a resolution past largest is refused, by Parse with an error and by
// Bake by name, before a grid of terabytes is asked for.
func TestABakeHasACeiling(t *testing.T) {
	for _, w := range []string{"baked(257)", "baked(5000)", "baked(3000000)"} {
		if _, err := Parse(w); err == nil || !strings.Contains(err.Error(), "256") {
			t.Errorf("%s: %v", w, err)
		}
	}
	if s, err := Parse("baked(256)"); err != nil || s.Resolution != 256 {
		t.Errorf("baked(256): %+v %v", s, err)
	}
	func() {
		defer func() {
			got, _ := recover().(string)
			if !strings.Contains(got, "past 256") {
				t.Errorf("Bake at 300: %q", got)
			}
		}()
		Bake(medium.Puff{Radius: 1, Thick: 1}, 300)
	}()
}

// a point of no number is outside, zero; and a box flat in a way, whose cells
// are 0/0 apart, is zero too, not no number.
func TestNoNumberIsOutside(t *testing.T) {
	b := Bake(medium.Puff{Radius: 1, Thick: 1}, 4)
	for _, p := range []maths.Vec{maths.V(math.NaN(), 0, 0), maths.V(0, math.NaN(), 0)} {
		if got := b.Density(p); got != 0 {
			t.Errorf("at %v the bake is %v", p, got)
		}
	}
	flat := &Baked{Of: medium.Puff{}, Resolution: 4,
		box: maths.Box{Min: maths.V(0, 0, 0), Max: maths.V(0, 1, 1)}, cells: make([]float64, 64)}
	if got := flat.Density(maths.V(0, 0.5, 0.5)); got != 0 {
		t.Errorf("a flat box gives %v", got)
	}
}
