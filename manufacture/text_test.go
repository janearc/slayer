package manufacture

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// testText is a few words in the font the typography package carries by the
// name go, a metre to the em and a tenth deep, standing at the origin and
// facing north.
func testText() Text {
	return Text{Name: "sign", Words: "AB", Font: "go", Size: 1, Depth: 0.1,
		Smooth: 0.5, Colour: maths.Grey(0.5), Bearing: 0}
}

func TestATextsBoxIsAsWideAsItsWordsAndAsDeepAsItsDepth(t *testing.T) {
	box, err := testText().Extent()
	if err != nil {
		t.Fatal(err)
	}
	if !(box.Max.X > 0.5 && box.Min.X < -0.5) {
		t.Errorf("a line of two capitals has a box %+v under an em wide", box)
	}
	if math.Abs(box.Max.Z-0.05) > 1e-9 || math.Abs(box.Min.Z+0.05) > 1e-9 {
		t.Errorf("a text a tenth deep has a box %+v not a tenth deep", box)
	}
	if box.Max.Y < 0.7 || box.Min.Y > 0 {
		t.Errorf("a text's box %+v does not stand over its baseline", box)
	}
}

func TestATextThatTurnsHasABoxRoundWholeTurn(t *testing.T) {
	still := testText()
	turning := testText()
	turning.Spin = 30
	standing, err := still.Extent()
	if err != nil {
		t.Fatal(err)
	}
	round, err := turning.Extent()
	if err != nil {
		t.Fatal(err)
	}
	if !(round.Max.Z-round.Min.Z > standing.Max.Z-standing.Min.Z) {
		t.Errorf("a turning text's box %+v is no wider than %+v", round,
			standing)
	}
}

func TestATextWithNoSizeOrNoDepthHasNoBox(t *testing.T) {
	flat := testText()
	flat.Depth = 0
	small := testText()
	small.Size = 0
	for name, text := range map[string]Text{"no depth": flat,
		"no size": small} {
		if _, err := text.Extent(); err == nil {
			t.Errorf("a text with %s was given a box", name)
		}
	}
}

func TestATextInAFontThatIsNotThereIsNotStaged(t *testing.T) {
	missing := testText()
	missing.Font = "no-such-font.ttf"
	world := thing.NewWorld()
	if _, err := missing.Staged(world); err == nil || len(world.Things) != 0 {
		t.Errorf("a text in no font left %d things (%v)", len(world.Things),
			err)
	}
}

func TestATextBehindAnEyeIsNotSeen(t *testing.T) {
	behind := maths.Subspace{Origin: maths.V(0, 0, 5000),
		Orientation: maths.Identity, Scale: 1}
	got, err := testText().seen(sow(0), behind, lookingDown())
	if err != nil {
		t.Fatal(err)
	}
	if got.Visible {
		t.Errorf("a text behind is %+v", got)
	}
}
