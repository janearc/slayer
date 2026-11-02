package film

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestTheMiddleOfThePictureLooksForward(t *testing.T) {
	from, to := maths.V(0, 2, 6), maths.V(0, 1, 0)
	eye := maths.Subspace{Origin: from,
		Orientation: maths.LookAt(from, to)}
	r := Look(eye, nil, 50, 50, 101, 101, maths.Band{})
	if !r.Dir.Near(eye.Forward(), 1e-9) || !r.At.Near(from, 1e-9) {
		t.Fatalf("middle ray %v, forward %v from %v", r,
			eye.Forward(), from)
	}
}

func TestTheTopRowLooksUp(t *testing.T) {
	r := Pinhole{}.Ray(5, 0, 11, 11, maths.Band{})
	if r.Dir.Y <= 0 {
		t.Fatalf("row 0 looks along %v, which is not up", r.Dir)
	}
}

// the step between neighbouring columns, over the step between
// neighbouring rows, has to be the pixel's own width over height, or
// circles are not round.
func TestPixelShapeKeepsCirclesRound(t *testing.T) {
	p := Pinhole{Wide: 40 * maths.Degree, Pixel: 0.75}
	w, h := 358, 150
	slope := func(r maths.Ray) (float64, float64) {
		return r.Dir.X / -r.Dir.Z, r.Dir.Y / -r.Dir.Z
	}
	x0, y0 := slope(p.Ray(100, 70, w, h, maths.Band{}))
	x1, _ := slope(p.Ray(101, 70, w, h, maths.Band{}))
	_, y1 := slope(p.Ray(100, 71, w, h, maths.Band{}))
	if got := (x1 - x0) / (y0 - y1); math.Abs(got-0.75) > 1e-9 {
		t.Fatalf("across over down is %v, want 0.75", got)
	}
}
