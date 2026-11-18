package manufacture

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestScaleHumanDefaultAndRecipeRefusals(t *testing.T) {
	if made, err := (scaleHuman{}).made(Sow(1)); err != nil || made.Bounds().Max.Y != 2 {
		t.Fatal("default human has wrong height", err)
	}
	for _, recipe := range []humanRecipe{
		{Material: "unknown", Finish: "polished"},
		{Material: "aluminium", Finish: "polished", Limbs: []humanLimb{{From: maths.V(0, 1, 0), To: maths.V(0, 1, 0), Radius: .1}}},
	} {
		if made, err := recipe.construct("quick"); err == nil || made.PartCount() != 0 {
			t.Fatal("invalid human recipe returned a partial mannequin")
		}
	}
	before := humanScaleRaw
	t.Cleanup(func() { humanScaleRaw = before })
	humanScaleRaw = []byte("ovals: [")
	if made, err := (scaleHuman{}).made(Sow(1)); err == nil || made.PartCount() != 0 {
		t.Fatal("malformed human recipe was accepted")
	}
}
