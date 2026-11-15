package bodies_test

import (
	"math"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/scene/bodies"
)

// measure.Length stores natural units and loses a last digit on the
// way back; Span holds kilometres as said.
func TestSpanKeepsTheDigitMeasureLoses(test *testing.T) {
	through := measure.Metres(64*1000).InMetres() / 1000
	if through != 64.000000000000014 {
		test.Fatalf("64 km through measure.Length read %v; the test "+
			"expects 64.000000000000014", through)
	}
	if got := bodies.Kilometres(64).InKilometres(); got != 64 {
		test.Fatalf("64 km through Span read %v", got)
	}
}

func TestExtinctionSaysLossPerKilometre(test *testing.T) {
	for _, amount := range []float64{0, 0.012, 1, 40} {
		got, err := bodies.PerKilometre(amount).Extinction()
		if err != nil || got != amount {
			test.Fatalf("PerKilometre(%v) said %v, %v", amount,
				got, err)
		}
	}
}

func TestExtinctionRefusesWhatNoMediumDoes(test *testing.T) {
	for _, amount := range []float64{-0.01, math.NaN(), math.Inf(1)} {
		if _, err := bodies.PerKilometre(amount).Extinction(); err == nil {
			test.Fatalf("PerKilometre(%v) passed; no medium "+
				"loses that", amount)
		}
	}
}
