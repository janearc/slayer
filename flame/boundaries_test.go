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

func TestFlameScaleHeatAndTimedEmissionStayCoherent(t *testing.T) {
	body := samplingBody(t)
	if body.Finest() <= 0 || body.Finest() >= body.fire.Radius {
		t.Fatal("unbounded eddy scale")
	}
	p := maths.V(.2, .7, .1)
	later := body.Tick(time.Second).(*Body)
	if body.Shine().Tick(time.Second).At(p) != later.Shine().At(p) {
		t.Fatal("emission time differs from density time")
	}
	fire := body.fire
	fire.Edge = fire.Hottest
	uniform, err := NewBody(fire, 1)
	if err != nil {
		t.Fatal(err)
	}
	if uniform.Shine().At(p) != uniform.Shine().At(maths.V(0, .2, 0)) {
		t.Fatal("uniform heat changed colour")
	}
	fire.Radius = math.MaxFloat64
	if _, err := NewBody(fire, 1); err == nil {
		t.Fatal("overflowing period accepted")
	}
}
