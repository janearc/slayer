package embers

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func TestSourceAcceptsSmallPositiveLifetimesAndThermalEndpoints(t *testing.T) {
	for _, change := range []func(*Spec){
		func(s *Spec) { s.Life = time.Nanosecond },
		func(s *Spec) { s.Drag = time.Nanosecond },
		func(s *Spec) { s.Hot = measure.Kelvin(.5); s.Cool = measure.Kelvin(.25) },
		func(s *Spec) { s.Hot = s.Cool },
	} {
		spec := fixture()
		change(&spec)
		if _, err := New(spec); err != nil {
			t.Fatalf("positive endpoint refused: %v", err)
		}
	}
	for _, change := range []func(*Spec){
		func(s *Spec) { s.Radius = measure.Metres(0) },
		func(s *Spec) { s.Cool = measure.Kelvin(0) },
	} {
		spec := fixture()
		change(&spec)
		if _, err := New(spec); err == nil {
			t.Fatal("zero physical endpoint accepted")
		}
	}
}

func TestSourceRejectsEveryNonfiniteMotionComponent(t *testing.T) {
	for _, bad := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
		for component := range 3 {
			for field := range 2 {
				spec := fixture()
				point := maths.Vec{}
				switch component {
				case 0:
					point.X = bad
				case 1:
					point.Y = bad
				case 2:
					point.Z = bad
				}
				if field == 0 {
					spec.Velocity = point
				} else {
					spec.Air = point
				}
				if _, err := New(spec); err == nil {
					t.Fatalf("field %d component %d accepted %g", field, component, bad)
				}
			}
		}
	}
}

func TestExactBirthAndExpiryAreHalfOpen(t *testing.T) {
	spec := fixture()
	spec.Seed = 0
	spec.Rate = 2
	source, err := New(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(source.At(-time.Nanosecond)) != 0 {
		t.Fatal("negative clock has births")
	}
	for _, p := range source.At(spec.Life) {
		if p.ID() == 0 {
			t.Fatal("birth zero survived its exact expiry")
		}
	}
}
