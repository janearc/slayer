package ninjafilm_test

import (
	"os"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
)

// anti-tests: the shelf holds no stock twice and none that is not a
// stock of the hills. Its names are in order and each is there once.
func TestTheShelfHoldsNoStockTwice(t *testing.T) {
	names, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	once := slices.Compact(slices.Clone(names))
	if !slices.IsSorted(names) || len(once) != len(names) ||
		slices.Contains(names, "celluloid") {
		t.Errorf("the shelf holds %v", names)
	}
}

// anti-tests: not every stock is cut to one gauge. Slides are cut to a
// gauge that differs from the one ash is cut to.
func TestNotEveryStockIsCutToOneGauge(t *testing.T) {
	slide, err := ninjafilm.Roll("lantern", seed)
	if err != nil {
		t.Fatal(err)
	}
	ash, err := ninjafilm.Roll("ash", seed)
	if err != nil {
		t.Fatal(err)
	}
	if slide.Gauge() == ash.Gauge() {
		t.Errorf("a slide and ash are both cut to %v", slide.Gauge())
	}
}

// anti-tests: no stock prints white as it prints black. White light
// makes every stock lighter than no light does, for these stocks are
// reversal stocks.
func TestNoStockPrintsWhiteAsItPrintsBlack(t *testing.T) {
	names, err := ninjafilm.Stocks()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		black, white := still(t, name, maths.Light{}), still(t, name,
			maths.Grey(lit))
		if !(white.R > black.R) {
			t.Errorf("%s prints white as %v and black as %v", name, white,
				black)
		}
	}
}

// anti-tests: stocks that see every colour do not print colour. White
// light prints as a grey, with the same in each lamp.
func TestABlackAndWhiteStockDoesNotPrintColour(t *testing.T) {
	for _, name := range []string{"smoke", "soot"} {
		got := still(t, name, maths.Grey(lit))
		if got.R != got.G || got.G != got.B {
			t.Errorf("%s prints white as %v", name, got)
		}
	}
}

// anti-tests: the mosaic does not pass most of a white light. It passes
// less than the clear stocks do.
func TestTheMosaicDoesNotPassMostOfAWhiteLight(t *testing.T) {
	white := mean(t, "pollen", maths.Grey(mosaicLamp))
	clear := still(t, "smoke", maths.Grey(lit))
	if !(white.R < clear.R/(lit+lit)) {
		t.Errorf("the mosaic passes %v, a clear stock %v", white, clear)
	}
}

// anti-tests: the guide does not give one exposure for every stock. The table
// of exposures holds exposures that differ.
func TestTheGuideDoesNotGiveOneExposureForEveryStock(t *testing.T) {
	text, err := os.ReadFile(exposuresFile)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[float64]bool{}
	for _, row := range rows {
		exposure, err := row.Float("exposure")
		if err != nil {
			t.Fatal(err)
		}
		seen[exposure] = true
	}
	if len(seen) < lit+lit {
		t.Errorf("the guide gives %d exposures", len(seen))
	}
}

// anti-tests: another roll of a stock does not print the same grain.
// The same flat light on two seeds does not print one frame.
func TestAnotherRollOfAStockDoesNotPrintTheSameGrain(t *testing.T) {
	shade := maths.Grey(grey())
	picture := uniform(spreadAcross, spreadDown, shade)
	firstRoll, err := ninjafilm.Roll("ash", seed)
	if err != nil {
		t.Fatal(err)
	}
	secondRoll, err := ninjafilm.Roll("ash", seed+uint64(lit))
	if err != nil {
		t.Fatal(err)
	}
	one, err := firstRoll.Print(first, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	other, err := secondRoll.Print(first, []*frame.Frame{picture})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(one.Pix, other.Pix) {
		t.Errorf("two rolls printed one frame")
	}
}
