package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/medium"
)

// lobeSlack is how near two lobes taken two ways must agree. Picked:
// float64 rounding over a handful of operations.
const lobeSlack = 1e-12

// TestLobedSendsLightOn: a forward lobe sends a lamp behind the point,
// seen through it, on more than evenly, and one in front back less; and
// is gas's lobe over the even one, four pi times PhaseAt.
func TestLobedSendsLightOn(t *testing.T) {
	dir := maths.V(0, 0, -1)
	ahead := lobed(0.85, dir, dir)
	back := lobed(0.85, dir.Neg(), dir)
	if !(ahead > 1) || !(back < 1) {
		t.Errorf("forward lobe: on %v, back %v; want over and under one",
			ahead, back)
	}
	// the lobe at nought between the ways, by its own formula
	want := (1 - 0.85*0.85) / math.Pow(1+0.85*0.85-2*0.85, 1.5)
	if math.Abs(ahead-want) > lobeSlack*want {
		t.Errorf("forward lobe on: %v, want %v", ahead, want)
	}
}

// TestAsymmetryOfSaysNoughtForAnEvenMedium: a medium that does not say
// an asymmetry, and a layer of even weather, are taken as even; a lobed
// layer as its lobe.
func TestAsymmetryOfSaysNoughtForAnEvenMedium(t *testing.T) {
	if got := asymmetryOf(nil); got != 0 {
		t.Errorf("no medium: %v", got)
	}
	box := maths.Box{Min: maths.V(-5, -5, -5), Max: maths.V(5, 5, 5)}
	ground := func(x, z float64) float64 { return -10 }
	for _, want := range []float64{0, 0.85} {
		layer, err := medium.NewLayer(medium.GroveWeather{Extinction: .05,
			Depth: 10, Coverage: 1, Asymmetry: want}, box, ground)
		if err != nil {
			t.Fatal(err)
		}
		if got := asymmetryOf(layer); got != want {
			t.Errorf("layer of %v: %v", want, got)
		}
	}
}
