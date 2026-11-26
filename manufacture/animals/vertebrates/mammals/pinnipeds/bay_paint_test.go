package pinnipeds

import (
	"github.com/janearc/slayer/maths"
	"testing"
)

func TestBaySealRetainsOriginalPaint(t *testing.T) {
	result, err := Bay()
	if err != nil {
		t.Fatal(err)
	}
	coat := result.Parts[0].Material
	// Check retained paint at separated body points.
	original := baySealWetPaint{spots: HarborSealCoat(407)}
	for _, p := range []maths.Vec{maths.V(0, .2, 0), maths.V(-.4, .1, .12), maths.V(.81, .059, .02)} {
		if coat.Texture.At(p) != original.At(p) {
			t.Fatalf("paint differs at %v", p)
		}
	}
}
