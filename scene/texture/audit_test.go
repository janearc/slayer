package texture

import (
	"image"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a street of no storms, or fewer, is no spots, not a panic.
func TestAStreetOfNoStormsIsNone(t *testing.T) {
	for _, n := range []int{0, -1, -40} {
		if got := (Street{Count: n, Size: 0.1}).spots(); got != nil {
			t.Errorf("a street of %d is %v", n, got)
		}
	}
}

// a point that is no number has no number of light, on every machine:
// a banded giant and a thin disc both, where the language would leave a
// latitude's or a temperature's index to the machine.
func TestAPointOfNoNumberIsNoNumberOfLight(t *testing.T) {
	nan := math.NaN()
	b := Banded{Paints: []maths.Light{maths.Grey(0.2), maths.Grey(0.8)}}
	if got := b.At(maths.V(nan, nan, nan)); !math.IsNaN(got.R) {
		t.Errorf("a banded giant at no point is %v", got)
	}
	d := Thin{Inner: 1, Outer: 10, Hot: 6000, Bright: 1}
	if got := d.At(maths.V(nan, 0, 2)); !math.IsNaN(got.R) {
		t.Errorf("a thin disc at no radius is %v", got)
	}
}

// a picture of no pixels is dark, not a division by its width of zero.
func TestAPictureOfNoPixelsIsDark(t *testing.T) {
	for _, r := range []image.Rectangle{image.Rect(0, 0, 0, 4), image.Rect(0, 0, 4, 0)} {
		pic := Picture{Image: image.NewRGBA(r)}
		if got := pic.At(maths.V(1, 0, 0)); got != (maths.Light{}) {
			t.Errorf("a picture %v is %v", r, got)
		}
	}
}
