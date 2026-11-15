package flame

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

func samplingBody(t *testing.T) *Body {
	t.Helper()
	body, err := NewBody(Fire{Radius: .8, Height: 1.8, Absorption: 2,
		Hottest: measure.Kelvin(1500), Edge: measure.Kelvin(900), Seed: 3}, 1e3)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestSampledFlameCachesDensityAndKeepsItsOpticalProperties(t *testing.T) {
	body := samplingBody(t)
	field, err := NewSampled(body, .08)
	if err != nil {
		t.Fatal(err)
	}
	p := maths.V(0, .2, 0)
	density := field.Density(p)
	if density <= 0 || field.Tint() != body.Tint() || field.Bounds() != body.Bounds() || field.Heat(p) != body.Heat(p) {
		t.Fatal("flame optics lost")
	}
	body.fire.Absorption = 0
	if field.Density(p) != density {
		t.Fatal("density recomputed instead of cached")
	}
	if field.Tick(0) != field {
		t.Fatal("unchanged time needlessly rebaked")
	}
	body.fire.Absorption = 2
	later := field.Tick(time.Second).(*Sampled)
	if later.Density(p) == density {
		t.Fatal("cached flame frozen")
	}
	if field.Density(p) != density {
		t.Fatal("later frame mutated earlier cache")
	}
	for _, p := range []maths.Vec{maths.V(100, 0, 0), maths.V(math.NaN(), 0, 0), field.Bounds().Max} {
		if got := field.Density(p); got != 0 {
			t.Fatal("outside density", p, got)
		}
	}
}
func TestSampledFlameRefusesUnboundedOrInvalidGrids(t *testing.T) {
	body := samplingBody(t)
	for _, step := range []float64{0, -1, math.NaN(), math.Inf(1), 1e-12} {
		if _, err := NewSampled(body, step); err == nil {
			t.Fatal("invalid grid accepted", step)
		}
	}
	if _, err := NewSampled(nil, .1); err == nil {
		t.Fatal("nil body accepted")
	}
}
