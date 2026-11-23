package cities

import (
	"testing"

	"github.com/janearc/slayer/maths"
)

// testBlock is a glass block of a pale glass in white joints, the tests'
// choice.
var testBlock = glassBlock{Glass: maths.RGB(0.6, 0.8, 0.9),
	Joint: maths.Grey(0.9), Side: 0.2, Width: 0.02, Rim: 0.5}

func TestAGlassBlockIsBrightAtItsMiddleAndSetInJoints(t *testing.T) {
	middle := testBlock.At(maths.V(0.1, 0.1, 0))
	rim := testBlock.At(maths.V(0.03, 0.03, 0))
	joint := testBlock.At(maths.V(0.2, 0.1, 0))
	if middle != testBlock.Glass {
		t.Errorf("a block's middle is %v, not its glass", middle)
	}
	if !(rim.G < middle.G) {
		t.Errorf("a block's rim %v is not darker than its middle %v", rim,
			middle)
	}
	if joint != testBlock.Joint {
		t.Errorf("a joint is %v", joint)
	}
}

func TestAGlassBlockWithNoRimIsEvenAndOneOfNoSideIsGlass(t *testing.T) {
	even := testBlock
	even.Rim = 0
	if got := even.At(maths.V(0.03, 0.03, 0)); got != even.Glass {
		t.Errorf("a block with no rim is %v at its rim", got)
	}
	none := testBlock
	none.Side = 0
	if got := none.At(maths.V(0.2, 0.1, 0)); got != none.Glass {
		t.Errorf("glass block of no side is %v", got)
	}
}

func TestAFarGlassBlockFadesToItsAverage(t *testing.T) {
	far := testBlock.Within(maths.V(0.2, 0.1, 0), 10)
	if far == testBlock.Joint || far == testBlock.Glass {
		t.Errorf("glass block a pixel wider than ten metres is %v", far)
	}
}

func TestMarbleHasVeinsAndNoneWhenTheyHaveNoWidth(t *testing.T) {
	marble := marbled{Stone: maths.Grey(0.8), Vein: maths.Grey(0.3),
		Apart: 0.7, Width: 0.12, Turbulence: 1.6, Seed: 1}
	var veined, clear int
	for x := range 200 {
		at := maths.V(float64(x)*0.013, 0.4, 0)
		if marble.At(at) == marble.Stone {
			clear++
		} else {
			veined++
		}
	}
	if veined == 0 || clear == 0 {
		t.Errorf("%d points veined and %d clear", veined, clear)
	}
	marble.Width = 0
	for x := range 200 {
		point := maths.V(float64(x)*0.013, 0.4, 0)
		if got := marble.At(point); got !=
			marble.Stone {
			t.Fatalf("marble with veins of no width is %v", got)
		}
	}
}

func TestOnlyADemoVaporwaveTowerHasItsBlocksAndVeins(t *testing.T) {
	pins := plainPins(skinVaporwave)
	for level, want := range map[Detail]bool{"quick": false, "demo": true} {
		drawn := drawnTower(t, towerOf(t, 3, 0, pins), level, 0)
		blocks := named(drawn, "glass block")
		if len(blocks) == 0 {
			t.Fatalf("%s: no glass block", level)
		}
		_, laidIn := blocks[0].Material.Texture.(glassBlock)
		bands := named(drawn, "pastel band")
		_, veined := bands[0].Material.Texture.(marbled)
		if laidIn != want || veined != want {
			t.Errorf("%s: glass block laid in blocks %v, bands veined %v",
				level, laidIn, veined)
		}
	}
}
