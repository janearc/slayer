package ocean

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"math"
	"testing"
)

func TestInfiniteContinuationExactPlaneRequiresCoreAndCollarAgreement(t *testing.T) {
	for _, row := range []struct {
		height, mean float64
		exact        bool
	}{
		{0, 0, true}, {2.5, 2.5, true}, {0, .1, false}, {math.Nextafter(2.5, 3), 2.5, false},
	} {
		core, err := shape.NewHeightField([]float64{row.height, row.height, row.height, row.height}, 2, 20)
		if err != nil {
			t.Fatal(err)
		}
		surface, err := NewInfiniteMeanContinuation(core, row.mean, 4)
		if err != nil {
			t.Fatal(err)
		}
		level, ok := surface.ExactPlaneLevel()
		if ok != row.exact || ok && level != row.mean {
			t.Fatalf("height%g mean%g plane(%g,%t)", row.height, row.mean, level, ok)
		}
		if ok {
			for _, x := range []float64{0, 11, 13, 1e6} {
				if normal := surface.Normal(maths.V(x, level, 0)); normal != maths.Up {
					t.Fatalf("plane normal at%g=%v", x, normal)
				}
			}
		}
	}
}
