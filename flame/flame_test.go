package flame

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
)

// near is whether a value is within a share of what it should be.
func near(got, want, share float64) bool {
	return math.Abs(got-want) <= share*math.Abs(want)
}

// a fire half a metre across puffs 1.5 over the square root of a half
// times a second, 2.1213 times a second, by cetegen and ahmed. a period
// is kept to the nanosecond, a few parts in a thousand million of it.
func TestAFlamePuffsForItsSize(t *testing.T) {
	period, err := puffPeriod(measure.Metres(0.5))
	if err != nil {
		t.Fatal(err)
	}
	if hertz := float64(time.Second) / float64(period); !near(hertz,
		1.5/math.Sqrt(0.5), 1e-8) {
		t.Errorf("half a metre puffs %v times a second", hertz)
	}
}

// a flame with no width is refused: none, less than none, not a number
// or past counting.
func TestAFlameWithNoWidthIsRefused(t *testing.T) {
	for _, width := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := puffPeriod(measure.Metres(width)); err == nil {
			t.Errorf("a flame %v m across puffed", width)
		}
	}
}
