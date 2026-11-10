package flame

import (
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestSamplesDescribeTheTimedFlame(t *testing.T) {
	made := body(t, campfire).Tick(137 * time.Millisecond).(*Body)
	fire, at, period, glows := made.Samples()
	wanted, _ := Period(measure.Metres(2 * campfire.Radius))
	if fire != campfire || at != 137*time.Millisecond || period != wanted {
		t.Fatalf("wrong field description: %+v, %v, %v", fire, at, period)
	}
	if len(glows) != 256 {
		t.Fatalf("got %d table entries", len(glows))
	}
	for i, heat := range []float64{fire.Edge.InKelvin(), fire.Hottest.InKelvin()} {
		index := i * (len(glows) - 1)
		expected := made.glows.at(measure.Kelvin(heat))
		if glows[index] != expected {
			t.Fatalf("wrong table endpoint %d", index)
		}
	}
	light, ok := made.Shine().(interface{ Flame() *Body })
	if !ok || light.Flame() != made {
		t.Fatal("standard glow lost its field")
	}
}

func TestSampleCopiesCannotChangeTheFlameOrItsEmission(t *testing.T) {
	made := body(t, campfire)
	point := low(campfire)
	before := made.Shine().At(point)
	fire, _, _, glows := made.Samples()
	if len(glows) == 0 {
		t.Fatal("missing light table")
	}
	fire.Radius = 99
	for i := range glows {
		glows[i] = maths.Light{}
	}
	original, _, _, again := made.Samples()
	if original.Radius != campfire.Radius || len(again) != 256 || made.Shine().At(point) != before {
		t.Fatal("sample mutation changed the model")
	}
}
