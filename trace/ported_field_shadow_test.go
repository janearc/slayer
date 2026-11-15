package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// shadowEdge is the angle off the way to a hole, in radians, at which a
// still eye re masses from a hole of unit mass sees the edge of its
// shadow, by the Schwarzschild critical impact parameter b of root 27
// masses.
//
// Light of that impact winds round the photon sphere and neither falls in
// nor gets out. A static eye at radius r sees it at an angle whose sine is
// b root (1 - 2 / r) over r. From outside the photon sphere, at 3 masses,
// the shadow is the angles inward of that; from inside it, the angles
// short of a half turn less it.
func shadowEdge(re float64) float64 {
	sine := math.Min(1, math.Sqrt(27)*math.Sqrt(1-2/re)/re)
	if re < 3 {
		return math.Pi - math.Asin(sine)
	}
	return math.Asin(sine)
}

// A still eye inside a hole's reach sees a shadow whose edge is where
// the critical impact parameter puts it: a ray a hundredth of the angle
// inside the edge falls in, and one a hundredth outside it gets out.
//
// Worked by hand at 4, 8, 20 and 30 masses out the edge is 66.7, 34.2,
// 14.3 and 9.6 degrees. At 3 masses it is a quarter turn, since light
// sent along the photon sphere stays on it. At 2.5, inside the photon
// sphere, it is a half turn less 68.4 degrees, since light sent outward
// gets out and only a ray well inward falls.
func TestPortedAStillEyeSeesTheShadowWhereTheCriticalImpactSays(t *testing.T) {
	if got := shadowEdge(3); !(math.Abs(got-math.Pi/2) <= 1e-7) {
		t.Fatalf("the edge at the photon sphere is %v, want a quarter turn",
			got)
	}
	for _, c := range []struct{ at, degrees float64 }{
		{4, 66.7}, {8, 34.2}, {20, 14.3}, {30, 9.6}, {2.5, 180 - 68.4},
	} {
		if got := shadowEdge(c.at) / maths.Degree; !(math.Abs(got-c.degrees) <= 0.1) {
			t.Fatalf("the worked edge at %v masses is %.2f degrees, want %.1f",
				c.at, got, c.degrees)
		}
	}
	world := holeWorld(nil)
	for _, re := range []float64{2.5, 3, 4, 8, 20, 30} {
		edge := shadowEdge(re)
		falls := func(theta float64) bool {
			ray := maths.Ray{At: maths.V(0, 0, -re),
				Dir: maths.V(0, math.Sin(theta), math.Cos(theta))}
			return lone(world, ray, 0).Light() == (maths.Light{})
		}
		if !falls(edge * 0.99) {
			t.Errorf("eye at %v masses: a ray at %.3f degrees, inside the edge at %.3f, got out",
				re, edge*0.99/maths.Degree, edge/maths.Degree)
		}
		if falls(edge * 1.01) {
			t.Errorf("eye at %v masses: a ray at %.3f degrees, outside the edge at %.3f, fell in",
				re, edge*1.01/maths.Degree, edge/maths.Degree)
		}
	}
}

// From far away the shadow's edge is at the critical impact parameter,
// root 27 masses or 5.196, from the line through the hole's middle, and it
// is a ring: a ray 5 percent inside it falls and one 5 percent outside it
// gets out, at every bearing round the hole.
//
// Rays parallel to the line, 100 out, enter the reach of 40 masses, where
// an eye held still would put the edge at 5.065; the test holds the edge
// to within 5 percent of 5.196 and cannot tell the two apart.
func TestPortedAShadowFromFarAwayIsRootTwentySevenMassesWide(t *testing.T) {
	world := holeWorld(nil)
	edge := math.Sqrt(27)
	for _, c := range []struct {
		scale float64
		falls bool
	}{{0.95, true}, {1.05, false}} {
		for _, bearing := range []float64{0, 1, 2, 3, 4, 5} {
			y, x := edge*c.scale*math.Cos(bearing), edge*c.scale*math.Sin(bearing)
			ray := maths.Ray{At: maths.V(x, y, -100), Dir: maths.V(0, 0, 1)}
			if fell := lone(world, ray, 0).Light() == (maths.Light{}); fell != c.falls {
				t.Errorf("a ray at %.3f masses, bearing %v: fell %v, want %v",
					edge*c.scale, bearing, fell, c.falls)
			}
		}
	}
}
