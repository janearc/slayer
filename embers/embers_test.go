package embers

import (
	"math"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func fixture() Spec {
	return Spec{Rate: 100, Life: 3 * time.Second, Drag: 700 * time.Millisecond,
		Radius: measure.Metres(.02), Disk: measure.Metres(2), Height: measure.Metres(1),
		Velocity: maths.V(0, 9, 0), Jitter: maths.V(2, 2, 2), Air: maths.V(2, 6, 0),
		Gravity: 9.80665, Hot: measure.Kelvin(1400), Cool: measure.Kelvin(700), Seed: 31}
}

func TestFirebrandsHaveBoundedDeterministicBirths(t *testing.T) {
	source, err := New(fixture())
	if err != nil {
		t.Fatal(err)
	}
	a, b := source.At(5*time.Second), source.At(5*time.Second)
	if len(a) < 299 || len(a) > 301 || !reflect.DeepEqual(a, b) {
		t.Fatal("birth count or replay changed")
	}
	if source.At(-time.Nanosecond) != nil {
		t.Fatal("particles appeared before emission")
	}
	ids := map[int64]bool{}
	for _, p := range a {
		if ids[p.ID()] || p.Age() < 0 || p.Age() >= fixture().Life || p.Radius() != fixture().Radius {
			t.Fatal("invalid living particle")
		}
		ids[p.ID()] = true
	}
}

func TestFirebrandIDsPersistAndMoveWithoutStepping(t *testing.T) {
	source, _ := New(fixture())
	before := source.At(5 * time.Second)
	after := source.At(5040 * time.Millisecond)
	byID := map[int64]Particle{}
	for _, p := range before {
		byID[p.ID()] = p
	}
	shared, moved := 0, 0
	for _, p := range after {
		if old, ok := byID[p.ID()]; ok {
			shared++
			if p.Position() != old.Position() {
				moved++
			}
			if !(p.Heat().InKelvin() < old.Heat().InKelvin()) {
				t.Fatal("particle did not cool")
			}
		}
	}
	if shared < 290 || shared != moved {
		t.Fatal("stable IDs were redrawn or frozen")
	}
	source.At(time.Hour)
	if !reflect.DeepEqual(before, source.At(5*time.Second)) {
		t.Fatal("sampling order changed the source")
	}
}

func TestFirebrandDragHasAnalyticSpecialCases(t *testing.T) {
	at := maths.V(1, 2, 3)
	air := maths.V(2, 3, 4)
	if got := position(at, air, air, maths.Vec{}, 2*time.Second, time.Second); got != at.Add(air.Scale(2)) {
		t.Fatal("comoving particle moved relative to air")
	}
	got := position(maths.Vec{}, maths.Vec{}, maths.Vec{}, maths.V(0, -2, 0), time.Second, time.Second)
	if math.Abs(got.Y+2/math.E) > 1e-12 {
		t.Fatal("constant-gravity drag integral changed")
	}
	if position(at, air, maths.Vec{}, maths.YAxis, 0, time.Second) != at {
		t.Fatal("birth position moved")
	}
}

func TestFirebrandDragAgreesWithIndependentTimeIntegration(t *testing.T) {
	origin, velocity, air, gravity := maths.V(1, 2, 3), maths.V(2, 9, -1), maths.V(3, 5, 2), maths.V(0, -9.80665, 0)
	// Midpoint integration is independent of the closed form.
	p, v := origin, velocity
	const dt = 1.0 / 10000
	for range 10000 {
		a := air.Sub(v).Scale(1 / .7).Add(gravity)
		mid := v.Add(a.Scale(dt / 2))
		p = p.Add(mid.Scale(dt))
		v = v.Add(air.Sub(mid).Scale(1 / .7).Add(gravity).Scale(dt))
	}
	exact := position(origin, velocity, air, gravity, time.Second, 700*time.Millisecond)
	if exact.Sub(p).Len() > 1e-6 {
		t.Fatal("drag solution disagrees with time integration")
	}
}

func TestFirebrandInputsRejectInvalidOrUnboundedWork(t *testing.T) {
	for _, alter := range []func(*Spec){
		func(s *Spec) { s.Rate = 0 }, func(s *Spec) { s.Rate = math.NaN() },
		func(s *Spec) { s.Life = 0 }, func(s *Spec) { s.Drag = 0 },
		func(s *Spec) { s.Radius = measure.Metres(-1) }, func(s *Spec) { s.Disk = measure.Metres(-1) },
		func(s *Spec) { s.Height = measure.Metres(-1) }, func(s *Spec) { s.Velocity = maths.V(math.Inf(1), 0, 0) },
		func(s *Spec) { s.Jitter = maths.V(-1, 0, 0) }, func(s *Spec) { s.Gravity = math.NaN() },
		func(s *Spec) { s.Cool = measure.Kelvin(1800) }, func(s *Spec) { s.Rate = 1e9 },
	} {
		spec := fixture()
		alter(&spec)
		if _, err := New(spec); err == nil {
			t.Fatal("invalid particle source accepted")
		}
	}
}

func TestFirebrandSeedsChangePathsButNotCountOrThermalLaw(t *testing.T) {
	property := func(seed uint16) bool {
		spec := fixture()
		spec.Seed = uint64(seed)
		one, _ := New(spec)
		spec.Seed++
		other, _ := New(spec)
		a, b := one.At(5*time.Second), other.At(5*time.Second)
		if len(a) < 299 || len(a) > 301 || len(b) < 299 || len(b) > 301 {
			return false
		}
		different := false
		for i, p := range a {
			q := (p.Heat().InKelvin() - spec.Cool.InKelvin()) / (spec.Hot.InKelvin() - spec.Cool.InKelvin())
			if math.Abs(q-(1-p.Age().Seconds()/spec.Life.Seconds())) > 1e-12 {
				return false
			}
			different = different || (i < len(b) && p.Position() != b[i].Position())
		}
		return different
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 30}); err != nil {
		t.Fatal(err)
	}
}
