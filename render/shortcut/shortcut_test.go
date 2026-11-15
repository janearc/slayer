package shortcut

import (
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
)

// The cutoff's threshold is where what is left cannot show: for the
// clip tone and a top of one it is the linear light that encodes under
// half a step, about a seventh of a thousandth; a brighter top lowers
// it; a filmic shoulder and more exposure move it; the remainder bounds
// the error; and the words parse.
func TestTheCutoffThreshold(t *testing.T) {
	clip := Threshold(colour.Tone{}, 1)
	if !(clip > 1e-5 && clip < 1e-3) {
		t.Errorf("the clip tone's threshold at top one is %g", clip)
	}
	if bright := Threshold(colour.Tone{}, 100); !(bright < clip/50) {
		t.Errorf("a top of a hundred gives %g against %g", bright, clip)
	}
	if exposed := Threshold(colour.Tone{Exposure: 4}, 1); !(exposed < clip) {
		t.Errorf("two stops up gives %g against %g", exposed, clip)
	}
	if Remainder(clip, 1) != clip {
		t.Error("the remainder is keep times top")
	}
	if Brightest([]maths.Light{maths.RGB(1.1, 1.1, 1.1), maths.Grey(26)}) != 26 || Brightest(nil) != 1 {
		t.Error("the brightest lamp is the top")
	}
	s, err := Parse("cutoff baked")
	if err != nil || !s.Cutoff || s.String() != "baked cutoff" {
		t.Errorf("cutoff parses as %v, %v", s, err)
	}
	if !All().Cutoff {
		t.Error("all has no cutoff")
	}
}
