package manufacture

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
)

func TestLegacySpinesKeepTheirAuthoredSlenderRadiusAndCap(t *testing.T) {
	for _, test := range []struct{ foot, root, tip float64 }{{.008, .001, .0003}, {.1, .006, .0018}} {
		axis, err := newTube(measure.Metres(1), measure.Metres(test.foot), measure.Metres(test.foot/5))
		if err != nil {
			t.Fatal(err)
		}
		result := individualFoliage(good{parts: []part{{of: leaf, shape: axis}}}, Sow(5), 0)
		found := false
		for _, piece := range result.parts {
			if spine, ok := piece.shape.(tube); ok {
				found = true
				if !(math.Abs(spine.foot.InMetres()-test.root) <= 1e-15) || !(math.Abs(spine.tip.InMetres()-test.tip) <= 1e-15) {
					t.Fatalf("spine from %g m axis: %g/%g m, want %g/%g", test.foot, spine.foot.InMetres(), spine.tip.InMetres(), test.root, test.tip)
				}
			}
		}
		if !found {
			t.Fatal("foliage lost its woody spine")
		}
	}
}

func TestLegacyBladesReserveTheActualScaffoldFaceCost(t *testing.T) {
	axis, err := newTube(measure.Metres(1), measure.Metres(.1), measure.Metres(.02))
	if err != nil {
		t.Fatal(err)
	}
	wood := make(fixedSurface, 99376)
	for i := range wood {
		wood[i] = face{maths.Vec{}, maths.XAxis, maths.YAxis}
	}
	// This picked preview budget leaves room for 100 blades after accounting
	// for this scaffold and one leaf's supporting twig geometry.
	result := individualFoliage(good{parts: []part{{of: body, shape: wood}, {of: leaf, shape: axis}}}, Sow(5), 0)
	blades := 0
	for _, piece := range result.parts {
		if spray, ok := piece.shape.(foliageSpray); ok {
			blades += spray.count
		}
	}
	if blades != 100 {
		t.Fatalf("scaffold left %d blades, want 100", blades)
	}
}
