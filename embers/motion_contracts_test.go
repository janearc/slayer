package embers

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// integrateBrand solves dv/dt=(air-v)/drag+gravity and dx/dt=v using RK4.
// It does not use the renderer's exponential closed form.
func integrateBrand(at, velocity, air, gravity maths.Vec, seconds, drag float64) maths.Vec {
	const steps = 1024
	dt := seconds / steps
	acceleration := func(v maths.Vec) maths.Vec { return air.Sub(v).Scale(1 / drag).Add(gravity) }
	for range steps {
		a := acceleration(velocity)
		vb := velocity.Add(a.Scale(dt / 2))
		ab := acceleration(vb)
		vc := velocity.Add(ab.Scale(dt / 2))
		ac := acceleration(vc)
		vd := velocity.Add(ac.Scale(dt))
		ad := acceleration(vd)
		at = at.Add(velocity.Add(vb.Scale(2)).Add(vc.Scale(2)).Add(vd).Scale(dt / 6))
		velocity = velocity.Add(a.Add(ab.Scale(2)).Add(ac.Scale(2)).Add(ad).Scale(dt / 6))
	}
	return at
}

func TestFirebrandMotionFollowsAnIndependentForceIntegrator(t *testing.T) {
	spec := fixture()
	spec.Rate = 2
	spec.Disk = measure.Metres(0)
	spec.Jitter = maths.Vec{}
	spec.Velocity = maths.V(3, 9, -2)
	spec.Air = maths.V(-1, 4, 2)
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	particles := source.At(2 * time.Second)
	if len(particles) < 3 {
		t.Fatal("trajectory fixture has too few particle ages")
	}
	for _, particle := range particles {
		expected := integrateBrand(maths.YAxis.Scale(spec.Height.InMetres()), spec.Velocity, spec.Air, maths.V(0, -spec.Gravity, 0), particle.Age().Seconds(), spec.Drag.Seconds())
		if distance := particle.Position().Sub(expected).Len(); !(distance < 1e-9) {
			t.Fatalf("particle %d age %s is %g m from independent force solution", particle.ID(), particle.Age(), distance)
		}
	}
}

func TestLaunchJitterIsCentredBoundedAndIndependentByAxisAndBirth(t *testing.T) {
	spec := fixture()
	spec.Disk, spec.Height = measure.Metres(0), measure.Metres(0)
	spec.Velocity, spec.Air = maths.Vec{}, maths.Vec{}
	spec.Gravity = 0
	spec.Jitter = maths.V(2, 3, 5)
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	const count = 2048
	at := time.Second
	unit := integrateBrand(maths.Vec{}, maths.XAxis, maths.Vec{}, maths.Vec{}, at.Seconds(), spec.Drag.Seconds()).X
	if !(unit > 0) {
		t.Fatal("reference response vanished")
	}
	var sums, squares [3]float64
	var products [3]float64
	low, high := [3]float64{1, 1, 1}, [3]float64{-1, -1, -1}
	distinct := map[maths.Vec]bool{}
	for id := range int64(count) {
		point := source.particle(id, at).Position()
		if distance := point.Len(); math.IsNaN(distance) || math.IsInf(distance, 0) {
			t.Fatal("nonfinite launch jitter")
		}
		values := [3]float64{point.X / (2 * unit), point.Y / (3 * unit), point.Z / (5 * unit)}
		distinct[point] = true
		for axis, value := range values {
			if value < -1-1e-10 || value > 1+1e-10 {
				t.Fatalf("jitter axis %d outside launch range: %g", axis, value)
			}
			sums[axis] += value
			squares[axis] += value * value
			low[axis], high[axis] = math.Min(low[axis], value), math.Max(high[axis], value)
		}
		products[0] += values[0] * values[1]
		products[1] += values[1] * values[2]
		products[2] += values[2] * values[0]
	}
	if len(distinct) != count {
		t.Fatalf("only %d launch trajectories for %d births", len(distinct), count)
	}
	for axis := range 3 {
		mean, variance := sums[axis]/count, squares[axis]/count-math.Pow(sums[axis]/count, 2)
		if math.Abs(mean) > .06 || math.Abs(variance-1.0/3) > .04 || low[axis] > -.95 || high[axis] < .95 {
			t.Fatalf("axis %d mean %g variance %g range [%g,%g]", axis, mean, variance, low[axis], high[axis])
		}
	}
	for index, pair := range [][2]int{{0, 1}, {1, 2}, {2, 0}} {
		covariance := products[index]/count - sums[pair[0]]*sums[pair[1]]/(count*count)
		if math.Abs(covariance) > .03 {
			t.Fatalf("launch axes %v share correlated jitter: %g", pair, covariance)
		}
	}
}
