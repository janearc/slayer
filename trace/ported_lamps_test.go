package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/film"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// floorOf is a world of a plane of the colour given and nothing else.
func floorOf(colour maths.Light) *thing.World {
	world := thing.NewWorld()
	world.Add(thing.Thing{Shape: shape.Plane{},
		Material: &thing.Material{Colour: colour}})
	return world
}

// down is a ray from over a point of the floor straight down onto it.
func down(x float64) maths.Ray {
	return maths.Ray{At: maths.V(x, 5, 0), Dir: maths.V(0, -1, 0)}
}

// lightIs holds a pixel's light to one wanted, channel by channel, and
// fails on a NaN.
func lightIs(t *testing.T, what string, got, want maths.Light, tol float64) {
	t.Helper()
	if !within(got.R, want.R, tol) || !within(got.G, want.G, tol) ||
		!within(got.B, want.B, tol) {
		t.Errorf("%s: %v, want %v", what, got, want)
	}
}

// A white floor under a spot is lit by the spot's strength over the
// square of the distance, times the cosine of the angle the light lands
// at, over pi: the spot two up reads 2/4/pi straight under it, and
// at three to one side and four up (distance 5, cosine 0.8) it reads
// 2/25 times 0.8 over pi. A point outside the spot's cone is dark.
func TestPortedAFloorUnderASpotIsLitByTheInverseSquareAndTheCosine(t *testing.T) {
	spot := func(height, cone float64) film.Spot {
		return film.Spot{At: maths.V(0, height, 0), Aim: maths.V(0, -1, 0),
			Strength: maths.Grey(2), Cone: cone}
	}
	for _, c := range []struct {
		height, across, cone, want float64
	}{
		{2, 0, 1.5, 2.0 / 4 / math.Pi},
		{4, 0, 1.5, 2.0 / 16 / math.Pi},
		{4, 3, 1.5, 2.0 / 25 * 0.8 / math.Pi},
		{2, 5, 0.1, 0},
	} {
		world := floorOf(maths.Grey(1))
		world.Lamp(spot(c.height, c.cone))
		got := lone(world, down(c.across), 0)
		lightIs(t, "a spot at the height and across given", got.Light(),
			maths.Grey(c.want), 1e-6)
	}
}

// A floor under a sun reads its colour times the sun's, times the
// cosine of the angle the sun is off straight up, over pi. A sun below
// the floor lights nothing seen from above and lights the face a ray
// from below sees squarely: the surface turns to face the ray.
func TestPortedAFloorUnderASunIsLitByTheCosine(t *testing.T) {
	chalk, sun := maths.RGB(0.9, 0.6, 0.3), maths.RGB(1, 0.8, 0.5)
	full := maths.RGB(chalk.R*sun.R, chalk.G*sun.G, chalk.B*sun.B)
	for _, degrees := range []float64{0, 30, 60, 85} {
		angle := degrees * maths.Degree
		world := floorOf(chalk)
		world.Lamp(thing.Sun(maths.V(math.Sin(angle), math.Cos(angle), 0),
			sun))
		got := lone(world, down(0), 0).Light()
		scale := math.Cos(angle) / math.Pi
		want := maths.RGB(full.R*scale, full.G*scale, full.B*scale)
		lightIs(t, "the sun off straight up by the angle given", got, want,
			0.01*want.R+1e-6)
	}
	world := floorOf(chalk)
	world.Lamp(thing.Sun(maths.V(0, -1, 0), sun))
	if got := lone(world, down(0), 0).Light(); got != (maths.Light{}) {
		t.Errorf("a sun under the floor lit its top with %v", got)
	}
	up := maths.Ray{At: maths.V(0, -5, 0), Dir: maths.V(0, 1, 0)}
	got := lone(world, up, 0).Light()
	lightIs(t, "the underside facing a sun under it", got,
		maths.RGB(full.R/math.Pi, full.G/math.Pi, full.B/math.Pi), 1e-6)
}

// A lamp's light is stopped by what stands between it and the surface
// and by nothing else: a ball over a point of the floor shades it, a ball
// beside it does not, and a ball past a spot that is nearer than the ball
// does not shade what the spot lights.
func TestPortedWhatStandsBetweenTheFloorAndALampStopsIt(t *testing.T) {
	ball := func(radius, scale float64, at maths.Vec) thing.Thing {
		return thing.Thing{Shape: shape.Sphere{Radius: radius},
			Motion: thing.Still(maths.Subspace{Origin: at, Scale: scale})}
	}
	under := maths.Ray{At: maths.V(4, 2, 0), Dir: maths.V(-4, -2, 0).Unit()}
	beside := maths.Ray{At: maths.V(7, 2, 0), Dir: maths.V(-4, -2, 0).Unit()}
	world := floorOf(maths.Grey(1))
	world.Add(ball(0.2, 0, maths.V(0, 0.25, 0)))
	world.Lamp(thing.Sun(maths.V(0, 1, 0), maths.Grey(1)))
	if got := lone(world, under, 0).Light(); !within(got.R, 0, 1e-6) {
		t.Errorf("under a ball the floor reads %v, want black", got)
	}
	got := lone(world, beside, 0).Light()
	lightIs(t, "the floor a ball's width off", got, maths.Grey(1/math.Pi),
		1e-6)

	spotted := floorOf(maths.Grey(1))
	spotted.Lamp(film.Spot{At: maths.V(0, 2, 0), Aim: maths.V(0, -1, 0),
		Strength: maths.Grey(1), Cone: 1.5})
	spotted.Add(ball(1, 1, maths.V(0, 5, 0)))
	eye := maths.Ray{At: maths.V(0, 1, -6), Dir: maths.V(0, -1, 6).Unit()}
	got = lone(spotted, eye, 0).Light()
	lightIs(t, "a floor under a spot with a ball behind the spot", got,
		maths.Grey(0.25/math.Pi), 1e-6)

	shaded := floorOf(maths.Grey(1))
	shaded.Lamp(film.Spot{At: maths.V(0, 2, 0), Aim: maths.V(0, -1, 0),
		Strength: maths.Grey(1), Cone: 1.5})
	shaded.Add(ball(0.4, 2, maths.V(0, 1, 0)))
	side := maths.Ray{At: maths.V(5, 0.5, 0), Dir: maths.V(-5, -0.5, 0).Unit()}
	got = lone(shaded, side, 0).Light()
	if !within(got.R, 0, 1e-6) {
		t.Errorf("a scaled ball between a spot and the floor leaves %v",
			got)
	}
}

// What the lamp is not reached by takes nothing: a painted volume the
// ray to a lamp never meets (a box beside it, or one short of a lamp that
// is nearer than the box) lets all of it through, and from inside a
// block the way out is what is left of it, e to the minus its density.
func TestPortedNothingIsTakenFromWhereNothingIsMet(t *testing.T) {
	block := thing.Volume{Medium: slab{density: 0.5, box: maths.Box{
		Min: maths.V(-1, -1, -1), Max: maths.V(1, 1, 1)}}}
	volumes := staging(block).volumes
	for _, c := range []struct {
		what string
		from maths.Vec
		dir  maths.Vec
		far  float64
		want float64
	}{
		{"a ray beside the block", maths.V(5, 0, -50), maths.V(0, 0, 1), 100, 1},
		{"a lamp nearer than the block", maths.V(0, 0, -50), maths.V(0, 0, 1), 10, 1},
		{"from the middle out", maths.V(0, 0, 0), maths.V(0, 1, 0), 100, math.Exp(-0.5)},
	} {
		got := paintedClear(volumes, c.from, c.dir, c.far)
		lightIs(t, c.what, got, maths.Grey(c.want), 1e-9)
	}
}

// partial is the exact Schwarzschild integral of the polar angle a ray
// sweeps from its periapsis out to radius r, for the impact parameter b
// and a mass of one: the integral of du over the root of 1/b^2 - u^2
// (1 - 2u), from u = 1/r to the root u0 of that, which is the
// periapsis' 1/r. It is done with u = u0 (1 - t^2), which has no edge to
// the integrand, by the midpoint rule over 400000 pieces.
func partial(b, r float64) float64 {
	f := func(u float64) float64 { return 1/(b*b) - u*u + 2*u*u*u }
	lo, hi := 0.0, 1.0/3.0
	for range 200 {
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
	for i := range pieces {
		t := tR * (float64(i) + 0.5) / pieces
		sum += 2 * u0 * t / math.Sqrt(f(u0*(1-t*t))) * tR / pieces
	}
	return sum
}

// A ray in the field sweeps the polar angle the exact integral says, to
// 1e-6 of a radian: from where it enters at 40 M, in to its periapsis
// and out to where it is let go, which is the first step past 12 M for
// the rays that come in as close as that. Worked at four impacts, 6, 8,
// 12 and 20.
func TestPortedARaySweepsTheAngleTheIntegralSays(t *testing.T) {
	scene := stage(holeWorld(nil), 0)
	for _, y := range []float64{6, 8, 12, 20} {
		ray := maths.Ray{At: maths.V(0, y, -100), Dir: maths.V(0, 0, 1)}
		into, ok := entering(scene.fields, ray)
		if !ok {
			t.Fatalf("the ray at %v does not enter the reach", y)
		}
		walker := newMarcher(scene, look{ray: ray}, into)
		x0, p := parts3(walker.ray.Space()), parts4(walker.ray.Momentum())
		cross := [3]float64{x0[1]*p[3] - x0[2]*p[2],
			x0[2]*p[1] - x0[0]*p[3], x0[0]*p[2] - x0[1]*p[1]}
		b := math.Sqrt(cross[0]*cross[0]+cross[1]*cross[1]+
			cross[2]*cross[2]) / math.Abs(p[0])
		done := false
		for i := 0; i < 40000 && !done; i++ {
			_, done = walker.march(i)
		}
		if !done {
			t.Fatalf("the ray at %v did not end", y)
		}
		xe := parts3(walker.ray.Space())
		r0 := math.Sqrt(x0[0]*x0[0] + x0[1]*x0[1] + x0[2]*x0[2])
		re := math.Sqrt(xe[0]*xe[0] + xe[1]*xe[1] + xe[2]*xe[2])
		swept := math.Acos((x0[0]*xe[0] + x0[1]*xe[1] + x0[2]*xe[2]) / (r0 * re))
		exact := math.Mod(partial(b, r0)+partial(b, re), 2*math.Pi)
		if exact > math.Pi {
			exact = 2*math.Pi - exact
		}
		if !within(swept, exact, 1e-6) {
			t.Errorf("y=%v (b=%.4f): swept %.7f, the integral says %.7f",
				y, b, swept, exact)
		}
		if y <= 12 && !(re > 12 && re < 12.5) {
			t.Errorf("y=%v: let go at %v M, the first step past 12", y, re)
		}
	}
}
