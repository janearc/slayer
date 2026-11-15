package colour

import (
	"math"
	"math/rand"
	"testing"

	"github.com/janearc/libtheme-css/spaces/srgb"
)

// oldEncode is Encode as it was written, by the curve on every call.
func oldEncode(v float64) uint8 {
	if math.IsNaN(v) {
		return 0
	}
	v = math.Max(0, math.Min(1, v))
	return uint8(math.Round(srgb.FromLinear(v) * 255))
}

// every step of the table is where the curve's byte steps, and the floats
// either side of it, a thousand each way, fall on the right sides.
func TestEncodeStepsAreWhereTheCurveSteps(t *testing.T) {
	for i, s := range steps {
		if got := oldEncode(s); int(got) < i+1 {
			t.Fatalf("step %d at %v: curve gives %d", i+1, s, got)
		}
		lo, hi := s, s
		for n := 0; n < 1000; n++ {
			lo, hi = math.Nextafter(lo, -1), math.Nextafter(hi, 2)
			for _, v := range []float64{lo, hi} {
				if got, want := Encode(v), oldEncode(v); got != want {
					t.Fatalf("step %d: Encode(%v) = %d, curve gives %d", i+1, v, got, want)
				}
			}
		}
		if below := math.Nextafter(s, -1); oldEncode(below) >= uint8(i+1) {
			t.Fatalf("step %d at %v is not the first: %v gives %d", i+1, s, below, oldEncode(below))
		}
	}
}

// Encode is the curve's byte for the ends, the special values, and a few
// million seeded parts across the range, spread evenly and by exponent.
func TestEncodeIsTheCurveEverywhere(t *testing.T) {
	for _, v := range []float64{0, math.Copysign(0, -1), 1, 2, -1, math.Inf(1),
		math.Inf(-1), math.NaN(), math.SmallestNonzeroFloat64, 5e-324, 1e-300,
		math.Nextafter(1, 0), math.Nextafter(0, 1), 0.0031308, 0.04045, 0.5} {
		if got, want := Encode(v), oldEncode(v); got != want {
			t.Errorf("Encode(%v) = %d, want %d", v, got, want)
		}
	}
	r := rand.New(rand.NewSource(20260910))
	n := 4_000_000
	if testing.Short() {
		n = 200_000
	}
	for i := 0; i < n; i++ {
		var v float64
		switch i % 3 {
		case 0:
			v = r.Float64()
		case 1:
			v = math.Ldexp(r.Float64(), -r.Intn(40))
		default:
			v = math.Float64frombits(r.Uint64())
		}
		if got, want := Encode(v), oldEncode(v); got != want {
			t.Fatalf("Encode(%v) = %d, curve gives %d", v, got, want)
		}
	}
	// and every byte's decoded value, which the round trip must keep
	for b := 0; b < 256; b++ {
		if got := Encode(Decode(uint8(b))); got != uint8(b) {
			t.Errorf("Encode(Decode(%d)) = %d", b, got)
		}
	}
}
