package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestClearCloudExplicitHGValidatesAndDistinguishesZero(t *testing.T) {
	field := Puff{Radius: 2, Thick: 3}
	legacy, err := NewClearCloud(field)
	if err != nil {
		t.Fatal(err)
	}
	if g, explicit := legacy.PhaseAsymmetry(); g != 0 || explicit {
		t.Fatal("omitted phase acquired an override")
	}
	for _, g := range []float64{-.9, 0, .8, math.Nextafter(1, 0)} {
		cloud, err := NewClearCloudWithHenyeyGreenstein(field, g)
		if err != nil {
			t.Fatal(err)
		}
		got, explicit := cloud.PhaseAsymmetry()
		if !explicit || got != g {
			t.Fatalf("phase %g read (%g,%t)", g, got, explicit)
		}
		if cloud.Density(maths.Vec{}) != legacy.Density(maths.Vec{}) {
			t.Fatal("phase changed density")
		}
	}
	for _, g := range []float64{-1, 1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := NewClearCloudWithHenyeyGreenstein(field, g); err == nil {
			t.Fatalf("invalid phase %g accepted", g)
		}
	}
	if _, err := NewClearCloudWithHenyeyGreenstein(nil, .8); err == nil {
		t.Fatal("phase bypassed field bounds validation")
	}
}

type phaseClockField struct{ seconds float64 }

func (field phaseClockField) Bounds() maths.Box {
	return maths.Box{Min: maths.V(-1, -1, -1), Max: maths.V(1, 1, 1)}
}
func (field phaseClockField) Density(maths.Vec) float64 { return field.seconds + 1 }
func (field phaseClockField) Tick(at time.Duration) Medium {
	field.seconds = at.Seconds()
	return field
}

func TestClearCloudTickRetainsExplicitHGAndTicksUnderlyingField(t *testing.T) {
	cloud, err := NewClearCloudWithHenyeyGreenstein(phaseClockField{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	ticked := cloud.Tick(2 * time.Second).(ClearCloud)
	if g, explicit := ticked.PhaseAsymmetry(); g != 0 || !explicit {
		t.Fatal("Tick lost explicit isotropic HG")
	}
	if ticked.Density(maths.Vec{}) != 3 || cloud.Density(maths.Vec{}) != 1 {
		t.Fatal("phase wrapper lost immutable field clock")
	}
}
