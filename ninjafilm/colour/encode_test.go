package colour_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/colour"
	"pgregory.net/rapid"
)

// Half the light is 188 and not half the byte. Below 0.0031308 of linear
// light the srgb curve is a straight line of slope 12.92 (IEC 61966-2-1)
// and not the power curve above it. The power curve carried down below
// the toe would give 1 and 6 for the first two of the toe's bytes.
func TestEncodeIsTheCurveOfTheScreen(t *testing.T) {
	for _, row := range rows(t, encodeFile) {
		got := colour.Encode(amount(t, row, "light"))
		if want := uint8(amount(t, row, "code")); got != want {
			t.Errorf("%s: %v is %d, want %d", text(t, row, "label"),
				amount(t, row, "light"), got, want)
		}
	}
}

// anti-tests: parts that are no number are black by rule. That is the
// same on every machine.
func TestNoNumberEncodesAsBlack(t *testing.T) {
	if got := colour.Encode(math.NaN()); got != none {
		t.Errorf("no number encodes as %d", got)
	}
	if got := colour.Encode(math.Inf(whole)); got != byteMost {
		t.Errorf("endless light encodes as %d", got)
	}
	if got := colour.Encode(math.Inf(-whole)); got != none {
		t.Errorf("endless dark encodes as %d", got)
	}
}

// Decode is the curve undone at the ends and in the middle.
func TestDecodeIsTheCurveUndone(t *testing.T) {
	for _, row := range rows(t, decodeFile) {
		code := uint8(amount(t, row, "code"))
		want := amount(t, row, "light")
		if got := colour.Decode(code); math.Abs(got-want) > amount(t, row,
			"within") {
			t.Errorf("%s: byte %d is %v, want %v", text(t, row, "label"),
				code, got, want)
		}
	}
}

// Decode undoes Encode for every byte there is.
func TestDecodeUndoesEncodeForEveryByte(t *testing.T) {
	for code := range bytes {
		byte8 := uint8(code)
		decoded := colour.Decode(byte8)
		if got := colour.Encode(decoded); got != byte8 {
			t.Fatalf("byte %d came back as %d", code, got)
		}
	}
}

// More light is never a darker byte.
func TestEncodeNeverGetsDarkerWithLight(t *testing.T) {
	rapid.Check(t, encodeFollowsLight)
}

func encodeFollowsLight(t *rapid.T) {
	dim := rapid.Float64Range(-whole, lightMost).Draw(t, "dim")
	bright := rapid.Float64Range(dim, lightMost).Draw(t, "bright")
	if colour.Encode(dim) > colour.Encode(bright) {
		t.Fatalf("%v is brighter than %v", dim, bright)
	}
}
