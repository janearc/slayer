package colour_test

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/colour"
	"pgregory.net/rapid"
)

// anti-tests: half the light is not half the byte. The byte for a half is
// well above the middle of the bytes.
func TestHalfTheLightIsNotHalfTheByte(t *testing.T) {
	if got := colour.Encode(half); got <= byteMost/pair+whole {
		t.Errorf("half the light is the byte %d", got)
	}
}

// anti-tests: the curve is not a straight line. Half the byte decodes to
// less than half the light.
func TestHalfTheByteIsNotHalfTheLight(t *testing.T) {
	got := colour.Decode(byteMost / pair)
	if got >= half-within {
		t.Errorf("half the byte is %v of the light", got)
	}
}

// anti-tests: no two bytes are one light. Each byte decodes to more than
// the byte before it.
func TestNoTwoBytesAreOneLight(t *testing.T) {
	for code := range byteMost {
		if !(colour.Decode(uint8(code+whole)) > colour.Decode(uint8(code))) {
			t.Fatalf("bytes %d and %d are one light", code, code+whole)
		}
	}
}

// anti-tests: twice the light is not the same byte in the middle of the
// curve. Each stop more of light is a brighter byte.
func TestTwiceTheLightIsNotTheSameByte(t *testing.T) {
	rapid.Check(t, twiceTheLightIsBrighter)
}

func twiceTheLightIsBrighter(t *rapid.T) {
	light := rapid.Float64Range(dimLight/(pair*pair), half).Draw(t, "light")
	if colour.Encode(light) >= colour.Encode(pair*light) {
		t.Fatalf("%v and twice it are one byte", light)
	}
}

// anti-tests: the shoulder does not touch light under the knee. Grey
// light at or under the knee comes out as it went in.
func TestTheShoulderDoesNotTouchLightUnderTheKnee(t *testing.T) {
	rapid.Check(t, kneeLeavesLowLightAlone)
}

func kneeLeavesLowLightAlone(t *rapid.T) {
	knee := rapid.Float64Range(half/pair, whole-within).Draw(t, "knee")
	tone := colour.NewTone(none, knee, none, none)
	under := rapid.Float64Range(none, knee).Draw(t, "under")
	if got := tone.Squeeze(maths.Grey(under)); got != maths.Grey(under) {
		t.Fatalf("light of %v under a knee of %v came out %v", under, knee, got)
	}
}

// anti-tests: the shoulder never rolls light below the knee. Grey light
// above the knee comes out at the knee or above.
func TestTheShoulderNeverRollsLightBelowTheKnee(t *testing.T) {
	rapid.Check(t, shoulderStaysAboveTheKnee)
}

func shoulderStaysAboveTheKnee(t *rapid.T) {
	knee := rapid.Float64Range(half/pair, whole-within).Draw(t, "knee")
	tone := colour.NewTone(none, knee, none, none)
	over := rapid.Float64Range(knee+within, lightMost).Draw(t, "over")
	if got := tone.Squeeze(maths.Grey(over)); got.R < knee-within {
		t.Fatalf("light of %v over a knee of %v came out %v", over, knee, got)
	}
}

// anti-tests: exposure is not nothing. Mid light at twice the exposure
// is not the byte it is at the plain exposure.
func TestExposureChangesTheByteOfAMidLight(t *testing.T) {
	rapid.Check(t, exposureChangesTheByte)
}

func exposureChangesTheByte(t *rapid.T) {
	level := rapid.Float64Range(dimLight, half/pair).Draw(t, "level")
	grey := maths.Grey(snapped(level))
	seen := frameOf(grey)
	plain := colour.Develop(seen, whole).RGBAAt(none, none)
	doubled := colour.Develop(seen, pair).RGBAAt(none, none)
	if plain == doubled {
		t.Fatalf("light of %v came out %v at both exposures", level, plain)
	}
}

// anti-tests: a tone with a knee is not the clamp for bright light. The
// shoulder keeps detail the clamp loses.
func TestAToneWithAKneeIsNotTheClampForBrightLight(t *testing.T) {
	seen := frameOf(maths.Grey(brightLight))
	clamp := colour.Develop(seen, whole).RGBAAt(none, none)
	roll := colour.NewTone(none, kneeDrawn, none, none).Develop(seen)
	if roll.RGBAAt(none, none) == clamp {
		t.Errorf("a knee left bright light as the clamp does")
	}
}

// anti-tests: the scenes do not come out as one picture. The digests of
// the scenes differ where the scenes do.
func TestTheScenesDoNotAllComeOutAsOnePicture(t *testing.T) {
	seen := map[string]bool{}
	for _, row := range rows(t, scenarioFile) {
		seen[text(t, row, "digest")] = true
	}
	scenes := len(rows(t, scenarioFile))
	if len(seen) < scenes/pair {
		t.Errorf("%d scenes came out as %d pictures", scenes, len(seen))
	}
}
