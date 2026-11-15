package thing

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// A crinkle tilts a face's normal, a little and differently from place
// to place, keeping it a unit long and turned toward the eye. The
// anti-test: a surface with none, or none of any size or slope, keeps
// its normal.
func TestACrinkleTiltsTheFaceFromPlaceToPlace(t *testing.T) {
	up, look := maths.V(0, 1, 0), maths.V(0.3, 0.9, 0.1).Unit()
	wrinkles := &Crinkle{Size: 0.05, Slope: 0.4, Seed: 7}
	moved, largest := 0, 0.0
	for i := range 400 {
		point := maths.V(float64(i%20)*0.013, 0, float64(i/20)*0.017)
		got := wrinkles.Crinkled(point, up, look)
		if !(math.Abs(got.Len()-1) <= 1e-12) || !(got.Dot(look) > 0.05) {
			t.Fatalf("at %v the normal is %v", point, got)
		}
		tilt := math.Acos(math.Min(1, got.Dot(up)))
		largest = math.Max(largest, tilt)
		if tilt > 1e-3 {
			moved++
		}
	}
	if moved < 300 || !(largest < 1.2) {
		t.Errorf("%d of 400 places tilted, the most by %v radians", moved, largest)
	}
	var none *Crinkle
	for _, flat := range []*Crinkle{none, {Size: 0, Slope: 1}, {Size: 1, Slope: 0}} {
		if got := flat.Crinkled(maths.V(1, 2, 3), up, look); got != up {
			t.Errorf("%+v tilts the normal to %v", flat, got)
		}
	}
}

// A tilt that would turn the face from the eye is not taken. Seen from
// behind, every tilt of a face turns it away, so the normal is left as
// it was at every place; the anti-test is the same places seen from in
// front, where most are tilted.
func TestACrinkleNeverTurnsAFaceFromTheEye(t *testing.T) {
	up := maths.V(0, 1, 0)
	wrinkles := &Crinkle{Size: 0.05, Slope: 0.4, Seed: 7}
	var tilted int
	for i := range 100 {
		point := maths.V(float64(i%10)*0.013, 0, float64(i/10)*0.017)
		if got := wrinkles.Crinkled(point, up, up.Scale(-1)); got != up {
			t.Fatalf("at %v, seen from behind, the normal is %v", point, got)
		}
		if wrinkles.Crinkled(point, up, up) != up {
			tilted++
		}
	}
	if tilted < 75 {
		t.Errorf("%d of 100 places tilted seen from in front", tilted)
	}
}

// The same wrinkles at the same place tilt the same way, every time.
// The anti-test: another seed tilts another way.
func TestACrinkleIsTheSameEveryTime(t *testing.T) {
	up, look := maths.V(0, 1, 0), maths.V(0, 1, 0)
	point := maths.V(0.21, 0, 0.34)
	one := (&Crinkle{Size: 0.05, Slope: 0.4, Seed: 7}).Crinkled(point, up, look)
	two := (&Crinkle{Size: 0.05, Slope: 0.4, Seed: 7}).Crinkled(point, up, look)
	other := (&Crinkle{Size: 0.05, Slope: 0.4, Seed: 8}).Crinkled(point, up, look)
	if one != two || one == other {
		t.Errorf("seed 7 gives %v then %v; seed 8 gives %v", one, two, other)
	}
}
