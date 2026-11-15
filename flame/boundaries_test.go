package flame

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"testing"
	"time"
)

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
	soot, err := NewSoot(measure.Kelvin(1300), 1, measure.Metres(550e-9), 1)
	if err != nil || soot.Heat() != measure.Kelvin(1300) {
		t.Fatal("soot lost its measured heat", err)
	}
}
