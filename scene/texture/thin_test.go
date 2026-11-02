package texture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A thin disc is hottest a little out from its inner edge, blue-white
// there and orange at a rim twenty-five inner radii out, far brighter
// inside than out; with no streaks it is the same all round, with
// streaks it is not.
func TestThinDiscIsHotInsideAndCoolOutside(t *testing.T) {
	d := Thin{Inner: 1, Outer: 25, Hot: 25000, Bright: 1}
	if k := d.Temperature(49.0 / 36); math.Abs(k-25000) > 1 {
		t.Errorf("the hottest ring is %v kelvin", k)
	}
	if k := d.Temperature(25); k > 5000 || k < 3500 {
		t.Errorf("the rim is %v kelvin", k)
	}
	inner, rim := d.At(maths.V(1.4, 0, 0)), d.At(maths.V(24.5, 0, 0))
	if !(inner.B > inner.R) {
		t.Errorf("the inner disc is %v, not blue-white", inner)
	}
	if !(rim.R > rim.B) {
		t.Errorf("the rim is %v, not orange", rim)
	}
	if !(inner.Luminance() > 20*rim.Luminance()) {
		t.Errorf("the inner disc is %v bright, the rim %v", inner.Luminance(), rim.Luminance())
	}
	if d.At(maths.V(0.5, 0, 0)) != (maths.Light{}) || d.At(maths.V(26, 0, 0)) != (maths.Light{}) {
		t.Error("light beyond the disc")
	}
	a, b := d.At(maths.V(3, 0, 0)), d.At(maths.V(0, 0, 3))
	if a != b {
		t.Errorf("a smooth disc differs round the circle: %v and %v", a, b)
	}
	d.Streaks = 0.6
	a, b = d.At(maths.V(3, 0, 0)), d.At(maths.V(0, 0, 3))
	if a == b {
		t.Error("a streaked disc is the same round the circle")
	}
}
