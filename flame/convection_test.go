package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
)

// bonfire is a fire ten metres tall, the size of melancholia's: its heat
// to the two fifths 50 m, so its tongues rise at 1.9 times the square root
// of 50, about 13.4 metres a second.
var bonfire = Fire{Radius: 3, Height: 10, Absorption: 1,
	Hottest: measure.Kelvin(1600), Edge: measure.Kelvin(900), Seed: 3}

// the gas rises up the axis as mccaffrey measured it in each region: the
// speed a height's travel time gives back, one over its slope, is his
// speed there, within a part in a thousand. and an eddy rises with it:
// a moment later, the noise a height reads is the noise from that much
// lower.
func TestTheGasRisesAsMcCaffreyFound(t *testing.T) {
	flame := body(t, bonfire)
	scale := bonfire.Height / plumeFrom
	fifth := math.Sqrt(scale)
	speeds := map[float64]float64{
		0.04 * scale: continuousSpeed * fifth * math.Sqrt(0.04),
		0.12 * scale: tongueSpeed * fifth,
		0.40 * scale: plumeSpeed * fifth * math.Pow(0.40, -1.0/3),
	}
	for height, want := range speeds {
		const step = 1e-4
		slope := (flame.travel(height+step) - flame.travel(height-step)) /
			(2 * step)
		if got := 1 / slope; math.Abs(got-want) > 1e-3*want {
			t.Errorf("at %g m the gas rises %g m/s, not %g", height, got,
				want)
		}
	}
	later := body(t, bonfire).Tick(100 * time.Millisecond).(*Body)
	now, then := flame.lifted(6), later.lifted(6)
	if moved := now - then; math.Abs(moved-0.1*tongueSpeed*fifth) > 1e-9 {
		t.Errorf("in a tenth of a second the noise moved %g m", moved)
	}
}

// the regions meet: the gas's speed is the same either side of each
// border, within a part in fifty, as mccaffrey's fits nearly do.
func TestTheRegionsMeet(t *testing.T) {
	flame := body(t, bonfire)
	scale := bonfire.Height / plumeFrom
	for _, border := range []float64{tonguesFrom * scale, plumeFrom * scale} {
		const step = 1e-3
		below := step / (flame.travel(border) - flame.travel(border-step))
		above := step / (flame.travel(border+step) - flame.travel(border))
		if math.Abs(below-above) > 0.02*below {
			t.Errorf("at %g m the gas rises %g below and %g above", border,
				below, above)
		}
	}
}
