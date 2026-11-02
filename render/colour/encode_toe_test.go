package colour

import (
	"math"
	"testing"
)

// below 0.0031308 of linear light the srgb curve is a straight line of
// slope 12.92 (IEC 61966-2-1), not the power curve above it. each byte
// is 255 * 12.92 * light, rounded:
//
//	0.001     -> 3.2946 -> 3
//	0.002     -> 6.5892 -> 7
//	0.0031308 -> 10.3147 -> 10, where the line meets the power curve
//
// the power curve carried down below the toe would give 1 and 6 for
// the first two. 0.25 is above the toe, and is 137 on the power curve.
func TestTheToeOfTheCurveIsAStraightLine(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want uint8
	}{{0.001, 3}, {0.002, 7}, {0.0031308, 10}, {0.25, 137}} {
		if got := Encode(c.in); got != c.want {
			t.Errorf("Encode(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

// decode is the curve undone. the byte 1 is 1/255 = 0.0039216 of the
// curve's range, under its own toe of 0.04045, so its light is that
// over 12.92: 0.00030353. the byte 128 is ((128/255 + 0.055)/1.055)^2.4,
// which is 0.21586050 as the standard's table gives it.
func TestDecodeOfTheEndsAndTheMiddle(t *testing.T) {
	for _, c := range []struct {
		in   uint8
		want float64
	}{{0, 0}, {1, 1.0 / 255 / 12.92}, {128, 0.21586050}, {255, 1}} {
		if got := Decode(c.in); math.Abs(got-c.want) > 1e-8 {
			t.Errorf("Decode(%d) = %v, want %v", c.in, got, c.want)
		}
	}
}
