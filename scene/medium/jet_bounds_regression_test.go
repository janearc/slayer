package medium

import (
	"math"

	"github.com/janearc/slayer/maths"
	"testing"
)

// The former one-width box clipped nonzero Gaussian gas at a flat wall.
func TestJetBoundsContainItsRadiatingSkirt(t *testing.T) {
	j := Jet{Length: 4, Base: .2, Spread: .25, Thick: 1}
	p := maths.V(1.4, 3, 0)
	if j.Density(p) <= 0 {
		t.Fatal("witness misses skirt")
	}
	b := j.Bounds()
	if p.X > b.Max.X || p.X < b.Min.X {
		t.Fatal("radiating gas is clipped by its volume bounds")
	}
	ray := maths.Ray{At: maths.V(1.4, 3, -4), Dir: maths.ZAxis}
	a, z, ok := j.Span(ray, 0, 8)
	if !ok || a >= 4 || z <= 4 {
		t.Fatal("ray through visible skirt is culled", a, z, ok)
	}
}

// Every recorded jet's former one-width limit excluded a positive portion of
// its Gaussian skirt. The current bounds and ray span must include that gas,
// and the independently calculated density must agree at the old and new
// radii. Knots are disabled here to isolate the common Gaussian envelope.
func TestJetRecordedSkirtsFollowTheGaussianSupport(t *testing.T) {
	for _, name := range []string{"jet", "jet, flowing", "jet, knotted", "jet, wide open"} {
		t.Run(name, func(t *testing.T) {
			j := recorded()[name].(Jet)
			j.Knots = 0
			old := j.Base/2 + j.Length*j.Spread
			outer := 2.5 * old
			if b := j.Bounds(); b.Min.X != -outer || b.Max.X != outer || b.Min.Z != -outer || b.Max.Z != outer {
				t.Fatalf("bounds do not cover 2.5 widths: %+v", b)
			}
			y := .75 * j.Length
			width := j.Base/2 + y*j.Spread
			for _, radius := range []float64{.5 * width, (old + 2.5*width) / 2} {
				point := maths.V(radius, y, 0)
				d := radius / width
				want := j.Thick * (1 - y/j.Length) * math.Exp(-2*d*d)
				if got := j.Density(point); !(want > 0) || math.Abs(got-want) > 1e-14 {
					t.Fatalf("density at radius %g = %g; Gaussian = %g", radius, got, want)
				}
				if radius <= old {
					continue
				}
				ray := maths.Ray{At: maths.V(radius, y, -4*outer), Dir: maths.ZAxis}
				enter, leave, ok := j.Span(ray, 0, 8*outer)
				if !ok || enter >= 4*outer || leave <= 4*outer {
					t.Fatalf("positive skirt outside old bounds is clipped: %g, %g, %v", enter, leave, ok)
				}
			}
		})
	}
}
