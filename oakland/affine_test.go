package oakland

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestAffineForwardInverseKnownPoint(t *testing.T) {
	rotation := maths.Rotation(maths.YAxis, 0.37)
	axes := [3]maths.Vec{
		rotation.Out(maths.XAxis), rotation.Out(maths.YAxis), rotation.Out(maths.ZAxis),
	}
	transform := affine{offset: maths.V(2, -3, 4)}
	for row := range 3 {
		transform.linear[row*3] = component(axes[0], row) * 2
		transform.linear[row*3+1] = component(axes[1], row) * 0.75
		transform.linear[row*3+2] = component(axes[2], row) * 1.25
	}
	want := maths.V(0.5, -1.25, 2)
	got, err := transform.inversePoint(transform.point(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.Sub(want).Len() > 1e-12 {
		t.Fatalf("inverse(forward(%v)) = %v", want, got)
	}
}

func TestAffineRejectsSingularAndNonFiniteTransforms(t *testing.T) {
	singular := affine{linear: [9]float64{1, 0, 0, 0, 0, 0, 0, 0, 1}}
	if _, err := singular.inversePoint(maths.V(1, 2, 3)); err == nil {
		t.Fatal("singular transform had an inverse")
	}
	if _, err := singular.normal(maths.YAxis); err == nil {
		t.Fatal("singular transform had a normal transform")
	}
	if validNumber(math.Inf(1)) || validNumber(math.NaN()) {
		t.Fatal("non-finite glTF scalar was accepted")
	}
}
