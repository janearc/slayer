package roll_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/ninjafilm/roll"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// anti-tests: a roll is not cut to the gauge of another stock. Two
// stocks cut to gauges that differ load rolls that differ in them.
func TestARollIsNotCutToTheGaugeOfAnotherStock(t *testing.T) {
	rapid.Check(t, rollKeepsItsOwnGauge)
}

func rollKeepsItsOwnGauge(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	narrow := rapid.Float64Range(chipLeast, chipMost).Draw(t, "narrow")
	wider := narrow + rapid.Float64Range(chipLeast, chipMost).Draw(t, "more")
	small := roll.NewStock(chip(narrow), stacked(t), nil)
	large := roll.NewStock(chip(wider), stacked(t), nil)
	first := loaded(t, ctrl, small, grainySeed)
	second := loaded(t, ctrl, large, grainySeed)
	if first.Gauge() == second.Gauge() || first.Gauge() != chip(narrow) {
		t.Fatalf("rolls of %v and %v are cut to %v and %v", narrow, wider,
			first.Gauge(), second.Gauge())
	}
}

// anti-tests: a print is not the picture it was made from. Film takes
// light, so the frame that comes back is not the frame given.
func TestAPrintIsNotThePictureItWasMadeFrom(t *testing.T) {
	rapid.Check(t, printIsNotThePicture)
}

func printIsNotThePicture(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	columns := rapid.IntRange(frameLeast, frameMost).Draw(t, "columns")
	rows := rapid.IntRange(frameLeast, frameMost).Draw(t, "rows")
	stock := stockOn(t, drawnGauge(t))
	made := loaded(t, ctrl, stock, rapid.Uint64().Draw(t, "seed"))
	picture := frameOf(t, columns, rows)
	if reflect.DeepEqual(printed(t, made, picture).Pix, picture.Pix) {
		t.Fatal("a print came back as the picture it was made from")
	}
}

// anti-tests: a glimpse that differs changes the print. Two frames do not
// print as the first alone, where the film tells their mean from the
// first. Where it cannot, their difference is in light it does not see,
// and that proves nothing.
func TestAGlimpseThatDiffersChangesThePrint(t *testing.T) {
	rapid.Check(t, differentGlimpseChangesPrint)
}

func differentGlimpseChangesPrint(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	columns := rapid.IntRange(frameLeast, frameMost).Draw(t, "columns")
	rows := rapid.IntRange(frameLeast, frameMost).Draw(t, "rows")
	stock := stockOn(t, drawnGauge(t))
	made := loaded(t, ctrl, stock, rapid.Uint64().Draw(t, "seed"))
	both := glimpses(t, columns, rows, single+single)
	alone := printed(t, made, both[first])
	if sameLight(alone, printed(t, made, meanOf(both))) {
		t.Skip("glimpses whose mean the film prints as the first")
	}
	together, err := made.Print(first, both)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(together.Pix, alone.Pix) {
		t.Fatal("a glimpse that differs left the print as it was")
	}
}

// anti-tests: a stop over is not a stop under. The roll rated one stop
// each way prints the middle grey as two frames that differ. It is one
// fixed frame, not drawn: a drawn frame of a pixel or two, of few
// crystals, may develop the same crystals a stop either way, as film does.
func TestAStopOverIsNotAStopUnder(t *testing.T) {
	ctrl := gomock.NewController(t)
	made := loaded(t, ctrl, grainyStock(t, largeWidth), grainySeed)
	flat := uniform(spreadAcross, spreadDown, maths.Grey(grey()))
	over := printed(t, made.Rated(whole), flat)
	under := printed(t, made.Rated(-whole), flat)
	if reflect.DeepEqual(over.Pix, under.Pix) {
		t.Error("a stop over printed the middle grey as a stop under")
	}
}

// anti-tests: another roll does not make the same grain. Not every pixel
// of a frame matches the frame of another roll.
func TestAnotherRollDoesNotMakeTheSameGrain(t *testing.T) {
	ctrl := gomock.NewController(t)
	shade := maths.Grey(grey())
	middle := uniform(grainyAcross, grainyDown, shade)
	stock := grainyStock(t, grainyWidth)
	made := loaded(t, ctrl, stock, grainySeed)
	other := loaded(t, ctrl, stock, grainySeed+once)
	base := printedAs(t, made, middle, grainyFrame)
	if alike := same(base, printedAs(t, other, middle, grainyFrame)); alike ==
		len(base.Pix) {
		t.Errorf("another roll matched at all %d pixels", alike)
	}
}

// anti-tests: twice the middle grey does not land on the middle of the
// curve. One stop more light prints denser than the middle.
func TestTwiceTheMiddleGreyIsNotTheMiddleOfTheCurve(t *testing.T) {
	ctrl := gomock.NewController(t)
	made := loaded(t, ctrl, grainyStock(t, largeWidth), grainySeed)
	middle := (grainyLeast + grainyMost) * halfway
	shade := maths.Grey(grey() + grey())
	bright := uniform(frameLeast, frameLeast, shade)
	got := printed(t, made, bright).At(first, first).Light()
	if within(got.R, passed(middle), smoothError) {
		t.Errorf("twice the middle grey prints as %v, the middle's", got)
	}
}

// anti-tests: flat light has grain. Prints of one grey have pixels that differ.
// So the spread is greater than zero.
func TestFlatLightIsNotPrintedFlat(t *testing.T) {
	ctrl := gomock.NewController(t)
	made := loaded(t, ctrl, grainyStock(t, largeWidth), grainySeed)
	shade := maths.Grey(grey())
	flat := uniform(spreadAcross, spreadDown, shade)
	shot := printed(t, made, flat)
	if got := spread(densities(shot)); !(got > none) || math.IsNaN(got) {
		t.Errorf("flat light printed with a spread of %v", got)
	}
}

// sameLight is whether two frames hold the same light in every pixel,
// whatever their cover and depth.
func sameLight(one, other *frame.Frame) bool {
	for index, pixel := range one.Pix {
		if pixel.Light() != other.Pix[index].Light() {
			return false
		}
	}
	return true
}
