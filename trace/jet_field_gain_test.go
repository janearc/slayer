package trace

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// In the flat limit, the field's total shift is the ordinary relativistic
// Doppler factor. Both paths must use the same cube in either direction.
func TestJetFieldGainAgreesWithFlatBeaming(t *testing.T) {
	beam := thing.Beam{Speed: .9}
	gamma := 1 / math.Sqrt(1-beam.Speed*beam.Speed)
	for _, toward := range []maths.Vec{maths.YAxis, maths.YAxis.Neg()} {
		delta := 1 / (gamma * (1 - beam.Speed*maths.YAxis.Dot(toward)))
		want := beam.Boost(toward, maths.YAxis)
		if got := specificShiftGain(delta); got != want {
			t.Fatalf("field boost %g, flat boost %g", got, want)
		}
	}
}

// The cube applies to the whole shift and is not clipped by a picture guard.
func TestJetFieldGainIsCubicWithoutClipping(t *testing.T) {
	for _, example := range []struct{ shift, want float64 }{
		{.5, .125}, {1, 1}, {2, 8}, {4, 64}, {10, 1000},
	} {
		if got := specificShiftGain(example.shift); got != example.want {
			t.Errorf("shift %g: got %g, want %g", example.shift, got, example.want)
		}
	}
	for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := specificShiftGain(invalid); got != 0 {
			t.Errorf("invalid shift %g produced %g", invalid, got)
		}
	}
}
