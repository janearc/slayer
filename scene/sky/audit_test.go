package sky

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/hexes"
)

// straight up is straight up at any length: a vector of 1e200 or 1e-200
// is the zenith, not the horizon Unit's limits made of it. and a way
// that is nowhere is no number of light, not the horizon or the dark.
func TestADirectionAtAnySizeAndNoneAtAll(t *testing.T) {
	g := Gradient{Horizon: maths.Grey(0.9), Zenith: maths.RGB(0, 0, 1),
		Ground: maths.Grey(0.1)}
	for _, up := range []float64{1e200, 1e-200, 1} {
		if got := g.Light(maths.V(0, up, 0)); got != maths.RGB(0, 0, 1) {
			t.Errorf("straight up at %v is %v", up, got)
		}
	}
	if got := (Gradient{Zenith: maths.Grey(1), Up: maths.V(0, 0, 1e300)}).Light(
		maths.V(0, 0, 3)); got != maths.Grey(1) {
		t.Errorf("an up of 1e300 along z, looking along z: %v", got)
	}
	nan, inf := math.NaN(), math.Inf(1)
	for name, sky := range map[string]Sky{"gradient": g, "stars": Stars{},
		"hexed": Hexed{Of: g, Tiling: hexes.Sphere{N: 4}}} {
		for _, way := range []maths.Vec{{}, maths.V(nan, 0, 1), maths.V(inf, 0, 0)} {
			if got := sky.Light(way); !math.IsNaN(got.R) {
				t.Errorf("%s toward %v is %v, not no number", name, way, got)
			}
		}
	}
}

// halfway up, at 45 degrees, the gradient is the horizon and the zenith
// blended by the sine, root a half: worked by hand, 0.9 by 0.29289 and
// the zenith's part by 0.70711.
func TestTheGradientBlendsByTheSine(t *testing.T) {
	g := Gradient{Horizon: maths.RGB(0.9, 0.9, 0.9), Zenith: maths.RGB(0.3, 0.5, 0.9)}
	got := g.Light(maths.V(1, 1, 0))
	h := math.Sqrt(0.5)
	want := maths.RGB(0.9*(1-h)+0.3*h, 0.9*(1-h)+0.5*h, 0.9)
	if math.Abs(got.R-want.R) > 1e-12 || math.Abs(got.G-want.G) > 1e-12 ||
		math.Abs(got.B-want.B) > 1e-12 {
		t.Errorf("at 45 degrees the sky is %v, want %v", got, want)
	}
}
