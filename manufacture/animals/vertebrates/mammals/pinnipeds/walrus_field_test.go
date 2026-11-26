package pinnipeds

import (
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

func TestWalrusHeightIsNoughtAtTheThreshold(t *testing.T) {
	if at := walrusHeight(walrusThreshold); !(math.Abs(at) <= 1e-12) {
		t.Fatalf("the height at the threshold is %g", at)
	}
	if inside := walrusHeight(2 * walrusThreshold); !(inside < 0) {
		t.Fatalf("a denser place has the height %g", inside)
	}
	if outside := walrusHeight(walrusThreshold / 2); !(outside > 0) {
		t.Fatalf("a fainter place has the height %g", outside)
	}
}

func TestWalrusHeightOfNoDensityIsANumber(t *testing.T) {
	for _, density := range []float64{0, -1, walrusFaint / 2} {
		height := walrusHeight(density)
		if math.IsNaN(height) || math.IsInf(height, 0) || !(height > 0) {
			t.Fatalf("a density of %g has the height %g", density, height)
		}
	}
}

func TestWalrusHeightIsTheDistanceFromTheOutline(t *testing.T) {
	outline := testWalrusOutline(t)
	section := outline.at(0)
	for _, off := range []float64{-.05, -.01, 0, .01, .05} {
		p := maths.V(0, section.centre, section.wide+off)
		height := walrusHeight(walrusBodyDensity(outline, nil, p))
		if want := outline.distance(p); !(math.Abs(height-want) <= 1e-9) {
			t.Fatalf("%.2f m off the flank the height is %.6f, the "+
				"outline's distance %.6f", off, height, want)
		}
	}
}

func TestWalrusCutDepthMovesTheHideIn(t *testing.T) {
	if none := walrusCutDepth(0); none != 0 {
		t.Fatalf("no cut moves the hide %g", none)
	}
	shallow, deep := walrusCutDepth(.05), walrusCutDepth(.2)
	if !(shallow > 0 && deep > shallow) {
		t.Fatalf("cuts of .05 and .2 move the hide %g and %g", shallow, deep)
	}
	// Where the density is the threshold and the cut together, the cut
	// hide lies.
	for _, cut := range []float64{.02, .1, .45} {
		at := walrusHeight(walrusThreshold+cut) + walrusCutDepth(cut)
		if !(math.Abs(at) <= 1e-12) {
			t.Fatalf("a cut of %g leaves the hide %g off its place", cut, at)
		}
	}
}

func TestWalrusNothingLiesUnderTheGround(t *testing.T) {
	const deepInside = -.5
	under := maths.V(0, walrusFloor-.05, 0)
	if field := walrusOnGround(deepInside, under); !(field > 0) {
		t.Fatalf("under the ground the field is %g, inside the body", field)
	}
	over := maths.V(0, walrusFloor+.3, 0)
	if field := walrusOnGround(deepInside, over); !(field < 0) {
		t.Fatalf("over the ground, inside the body, the field is %g", field)
	}
	outside := walrusOnGround(.2, over)
	if !(math.Abs(outside-.2) <= .001) {
		t.Fatalf("well over the ground a height of .2 became %g", outside)
	}
}
