package arid

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
)

func TestInfiniteOceanSmoothNormalsKeepTheSameInterface(t *testing.T) {
	grid, err := shape.NewHeightField([]float64{0, .2, 0, 0, .4, 0, 0, .2, 0}, 3, 8)
	if err != nil {
		t.Fatal(err)
	}
	exact, err := NewInfiniteMeanContinuation(grid, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	smooth, err := exact.WithApproximateSmoothNormals()
	if err != nil {
		t.Fatal(err)
	}
	core := maths.V(.25, 0, .25)
	offshore := maths.V(1e2, 0, 1e2)
	if smooth.Normal(core) == exact.Normal(core) ||
		smooth.Normal(offshore) != exact.Normal(offshore) {
		t.Fatal("only the core's optical normal should change")
	}
	for _, at := range []maths.Vec{core, offshore} {
		ray := maths.Ray{At: at.Add(maths.V(0, 2, 0)), Dir: maths.Up.Neg()}
		a, ha := exact.Hit(ray, 0, 4)
		b, hb := smooth.Hit(ray, 0, 4)
		if !ha || !hb || a != b {
			t.Fatalf("surface hit changed from %g/%v to %g/%v", a, ha, b, hb)
		}
	}
}
