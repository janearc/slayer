package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// shadowEdge is the angle off the way to the hole, in radians, at which a
// still eye at re masses from a hole sees the edge of its shadow, by the
// schwarzschild critical impact parameter, 3 root 3 masses.
//
// the light of that impact, which a static eye at radius r sees at an
// angle whose sine is b root (1 - 2 / r) over r, winds round the photon
// sphere and neither falls in nor gets out.
//
// from outside the photon sphere the shadow is the angles inward of it;
// from inside, the angles short of a half turn less it.
func shadowEdge(re float64) float64 {
	x := math.Sqrt(27) * math.Sqrt(1-2/re) / re
	x = math.Min(x, 1)
	if re < 3 {
		return math.Pi - math.Asin(x)
	}
	return math.Asin(x)
}

// a still eye inside the reach sees a shadow whose edge is at the angle
// the critical impact parameter, root 27 M, gives: a ray a hundredth of
// the angle inside it falls into the hole, and one a hundredth outside it
// gets out.
//
// this is the exact schwarzschild form for a static eye, worked at 4, 8,
// 20 and 30 masses out, where the edge is 66.7, 34.2, 14.3 and 9.6
// degrees.
//
// at 3 masses, the photon sphere, the edge is a quarter turn exactly,
// since light sent along the sphere stays on it. at 2.5, inside the
// photon sphere, it is a half turn less 68.4 degrees, since light sent
// outward gets out and only a ray well inward falls.
func TestAStillEyeSeesTheShadowWhereTheCriticalImpactSays(t *testing.T) {
	w := hole(nil)
	for _, re := range []float64{2.5, 3, 4, 8, 20, 30} {
		edge := shadowEdge(re)
		look := func(theta float64) bool {
			p := Trace(w, maths.Ray{At: maths.V(0, 0, -re),
				Dir: maths.V(0, math.Sin(theta), math.Cos(theta))}, 0)
			return p.Light() == (maths.Light{})
		}
		if !look(edge * 0.99) {
			t.Errorf("eye at %v: a ray at %.3f degrees, inside the edge at %.3f, got out", re,
				edge*0.99/maths.Degree, edge/maths.Degree)
		}
		if look(edge * 1.01) {
			t.Errorf("eye at %v: a ray at %.3f degrees, outside the edge at %.3f, fell in", re,
				edge*1.01/maths.Degree, edge/maths.Degree)
		}
	}
	if edge := shadowEdge(3); !near(edge, math.Pi/2, 1e-7) {
		t.Fatalf("the worked edge at the photon sphere is %v, want a quarter turn", edge)
	}
}

// from far away the shadow's edge is at the critical impact parameter,
// root 27 M, 5.196 M, from the line through the hole's middle.
//
// rays parallel to the line, 100 out, are entered at the reach of 40 M,
// where the edge for an eye held still at the reach would be at the root
// of 27 times the root of one less 2 over 40, 5.065 M, and the edge here
// is at 5.194, between them, which this test cannot tell apart.
//
// it holds that the edge lies within 5 percent of 5.196, and that it is
// a ring: a ray a twentieth inside falls, a twentieth outside gets out.
func TestAShadowFromFarAwayIsRootTwentySevenMassesWide(t *testing.T) {
	w := hole(nil)
	edge := math.Sqrt(27)
	for _, c := range []struct {
		scale float64
		falls bool
	}{{0.95, true}, {1.05, false}} {
		for _, bearing := range []float64{0, 1, 2, 3, 4, 5} {
			// the same ring all the way round, since the hole is round
			y, x := edge*c.scale*math.Cos(bearing), edge*c.scale*math.Sin(bearing)
			p := Trace(w, maths.Ray{At: maths.V(x, y, -100), Dir: maths.V(0, 0, 1)}, 0)
			if fell := p.Light() == (maths.Light{}); fell != c.falls {
				t.Errorf("a ray at %.3f M, bearing %v: fell %v, want %v", edge*c.scale, bearing, fell, c.falls)
			}
		}
	}
}

// a ray that comes in at under the critical impact falls whatever way the
// sums round at the reach: a ray entered a hair outside the reach, as
// rounding may put one, is as bent as one a hair inside.
//
// two hundred rays that pass from 0.3 to 2.2 out from the line through
// the hole are all inside the shadow, and some of them, by rounding,
// begin the march a hair outside the reach; every one is black.
func TestARayEnteredAHairOutsideTheReachStillFalls(t *testing.T) {
	w := hole(nil)
	sc := stage(w, 0)
	hair := 0
	for i := 0; i < 200; i++ {
		x, y := 0.31*float64(i%7), 0.3+0.000237*float64(i)
		r := maths.Ray{At: maths.V(x, y, -100), Dir: maths.V(0, 0, 1)}
		_, enter, _ := entering(sc.fs, r)
		m := &marcher{sc: sc, f: sc.fs[0], keep: maths.Grey(1)}
		m.start(r, enter)
		if sc.fs[0].hole.Radius(m.ray.Space()) > 40 {
			hair++
		}
		if got := Trace(w, r, 0); got.Light() != (maths.Light{}) {
			t.Errorf("a ray at %v, %v out came back %v", x, y, got.Light())
		}
	}
	if hair == 0 {
		t.Fatal("no ray began outside the reach; the test says nothing")
	}
}

// a ray that gets out comes out the same way to a small difference
// however the sums round where it enters.
//
// three hundred rays a millionth of a unit apart, at 8 out from the line
// through the hole, which are bent about two radians, come out a few
// millionths of a radian apart, and none jumps, as one would that was let
// go at once for starting a hair outside the reach.
func TestAnEscapingRayDoesNotJumpWhereItEntersAHairOutside(t *testing.T) {
	w := hole(nil)
	sc := stage(w, 0)
	var last maths.Vec
	hair, worst := 0, 0.0
	for i := 0; i < 300; i++ {
		r := maths.Ray{At: maths.V(0.2, 8+1e-6*float64(i), -100), Dir: maths.V(0, 0, 1)}
		_, enter, _ := entering(sc.fs, r)
		m := &marcher{sc: sc, f: sc.fs[0], keep: maths.Grey(1)}
		m.start(r, enter)
		if sc.fs[0].hole.Radius(m.ray.Space()) > 40 {
			hair++
		}
		p := Trace(w, r, 0)
		heading := maths.V(float64(p.R), float64(p.G), float64(p.B))
		if i > 0 {
			worst = math.Max(worst, heading.Sub(last).Len())
		}
		last = heading
	}
	if hair == 0 {
		t.Fatal("no ray began outside the reach; the test says nothing")
	}
	if !(worst < 1e-4) {
		t.Errorf("neighbouring rays came out %v apart", worst)
	}
}

// partial is the exact schwarzschild integral of the polar angle a ray
// sweeps from its periapsis out to radius r, for the impact parameter b
// and a mass of one: the integral of du over the root of 1/b^2 - u^2
// (1 - 2u), from u = 1/r to the root u0 of that, the periapsis' 1/r.
//
// it is done with u = u0 (1 - t^2), which has no edge to the integrand,
// by the midpoint rule over 400000 pieces.
func partial(b, r float64) float64 {
	f := func(u float64) float64 { return 1/(b*b) - u*u + 2*u*u*u }
	lo, hi := 0.0, 1.0/3.0
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		if f(mid) > 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	u0 := lo
	tR := math.Sqrt(1 - 1/(r*u0))
	const pieces = 400000
	sum := 0.0
	for i := 0; i < pieces; i++ {
		t := tR * (float64(i) + 0.5) / pieces
		sum += 2 * u0 * t / math.Sqrt(f(u0*(1-t*t))) * tR / pieces
	}
	return sum
}

// orbit is a ray marched in a field until the march lets it go: the
// impact parameter it has, the radii it begins and ends at, the angle it
// has swept about the hole between them, the angle the exact integral
// says it should have, and the way the exact orbit is heading there, as
// a direction in the world.
type orbit struct {
	b, r0, re, swept, exact float64
	heading                 maths.Vec
}

// marched marches a ray, set off the way a picture sets it off, until
// the march lets it go.
//
// the impact parameter is the ray's own, its angular momentum over its
// energy, whatever the frame it was set up in. the heading at the end is
// the exact orbit's: it has swept the exact angle in the plane the ray
// began in, and there it heads at the angle off straight out whose
// tangent is r dphi/dr, which is 1 over r root (1/b^2 - u^2 (1 - 2u)).
func marched(t *testing.T, sc staged, r maths.Ray) orbit {
	t.Helper()
	_, enter, ok := entering(sc.fs, r)
	if !ok {
		t.Fatalf("the ray %v does not enter the reach", r)
	}
	f := sc.fs[0]
	m := &marcher{sc: sc, f: f, keep: maths.Grey(1)}
	m.start(r, enter)
	x0, p := parts3(m.ray.Space()), parts4(m.ray.Momentum())
	v0 := parts4(f.hole.Velocity(m.ray))
	cross := [3]float64{x0[1]*p[3] - x0[2]*p[2], x0[2]*p[1] - x0[0]*p[3], x0[0]*p[2] - x0[1]*p[1]}
	var o orbit
	o.b = math.Sqrt(cross[0]*cross[0]+cross[1]*cross[1]+cross[2]*cross[2]) / math.Abs(p[0])
	done := false
	for i := 0; i < 40000 && !done; i++ {
		_, done = m.march(i)
	}
	if !done {
		t.Fatalf("the ray %v did not end", r)
	}
	xe := parts3(m.ray.Space())
	o.r0 = math.Sqrt(x0[0]*x0[0] + x0[1]*x0[1] + x0[2]*x0[2])
	o.re = math.Sqrt(xe[0]*xe[0] + xe[1]*xe[1] + xe[2]*xe[2])
	o.swept = math.Acos((x0[0]*xe[0] + x0[1]*xe[1] + x0[2]*xe[2]) / (o.r0 * o.re))
	whole := partial(o.b, o.r0) + partial(o.b, o.re)
	o.exact = math.Mod(whole, 2*math.Pi)
	if o.exact > math.Pi {
		o.exact = 2*math.Pi - o.exact
	}

	// the plane of the orbit is the place and the way the ray began in
	var e1, e2 [3]float64
	toward := v0[1]*x0[0]/o.r0 + v0[2]*x0[1]/o.r0 + v0[3]*x0[2]/o.r0
	side := [3]float64{v0[1] - toward*x0[0]/o.r0, v0[2] - toward*x0[1]/o.r0, v0[3] - toward*x0[2]/o.r0}
	long := math.Sqrt(side[0]*side[0] + side[1]*side[1] + side[2]*side[2])
	for k := range e1 {
		e1[k], e2[k] = x0[k]/o.r0, side[k]/long
	}
	u := 1 / o.re
	slope := 1 / (o.re * math.Sqrt(1/(o.b*o.b)-u*u*(1-2*u)))
	psi := math.Atan(slope)
	var d [3]float64
	for k := range d {
		out := math.Cos(whole)*e1[k] + math.Sin(whole)*e2[k]
		round := -math.Sin(whole)*e1[k] + math.Cos(whole)*e2[k]
		d[k] = math.Cos(psi)*out + math.Sin(psi)*round
	}
	o.heading = f.f.FromHole(d)
	return o
}

// a ray in the field sweeps the polar angle the exact integral says, to
// the last digits the integral has: from where it enters at 40 M, in to
// its periapsis and out to where it is let go, which is the first step
// past 12 M for the rays that come in as close as that.
//
// worked at four impacts, 6, 8, 12 and 20; the sweep is held to 1e-6 of
// a radian. the part of the bend beyond 12 M is not here: that is as it was
// cutoff (see loose), and the exact sweep is of the same stretch of the
// path, so it holds all the same.
func TestARaySweepsTheAngleTheIntegralSays(t *testing.T) {
	sc := stage(hole(nil), 0)
	for _, y := range []float64{6, 8, 12, 20} {
		o := marched(t, sc, maths.Ray{At: maths.V(0, y, -100), Dir: maths.V(0, 0, 1)})
		if !near(o.swept, o.exact, 1e-6) {
			t.Errorf("y=%v (b=%.4f): swept %.7f, the integral says %.7f", y, o.b, o.swept, o.exact)
		}
		if y <= 12 && !(o.re > 12 && o.re < 12.5) {
			t.Errorf("y=%v: the ray was let go at %v M, the first step past 12", y, o.re)
		}
	}
}

// a field that reaches less than 12 masses lets a ray go at its reach,
// and not at the 12 masses. a hole of mass 1 reaching 8 lets rays that
// pass it at 6.5, 7 and 7.5 go at the first step past 8, having swept
// what the exact integral says.
//
// there the ray is outside the field and the picture is the sky it
// heads for, which the compass sky gives as its own heading: the exact
// orbit's heading there, to a few parts in 10^7, float32 light.
func TestAFieldThatReachesLessThanTwelveMassesLetsARayRunToItsReach(t *testing.T) {
	w := thing.NewWorld()
	w.Sky = compass{}
	w.AddField(thing.Field{Mass: 1, Reach: 8})
	sc := stage(w, 0)
	for _, y := range []float64{6.5, 7, 7.5} {
		r := maths.Ray{At: maths.V(0, y, -100), Dir: maths.V(0, 0, 1)}
		o := marched(t, sc, r)
		if !(o.re > 8 && o.re < 8.6) {
			t.Errorf("y=%v: the ray was let go at %v M, the first step past the reach of 8", y, o.re)
		}
		if !near(o.swept, o.exact, 1e-6) {
			t.Errorf("y=%v (b=%.4f): swept %.7f, the integral says %.7f", y, o.b, o.swept, o.exact)
		}
		colours(t, "the ray's heading", Trace(w, r, 0),
			maths.RGB(o.heading.X, o.heading.Y, o.heading.Z), 1e-6)
	}
}
