package live

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
)

// the ladder climbs: an eighth, a quarter, a half and the whole size at
// one ray, then the whole at two rays and at four, each rung no cheaper
// than the one before.
func TestLadderClimbsInOrder(t *testing.T) {
	rungs := ladderOf(80, 48)
	want := []Size{{10, 6, 1}, {20, 12, 1}, {40, 24, 1}, {80, 48, 1},
		{80, 48, 2}, {80, 48, 4}}
	if len(rungs) != len(want) {
		t.Fatalf("%d rungs, want %d", len(rungs), len(want))
	}
	for i, each := range rungs {
		if each.size != want[i] {
			t.Errorf("rung %d is %+v, want %+v", i, each.size, want[i])
		}
	}
	if rungs[1].words != "1/4 size, 1 ray" ||
		rungs[5].words != "pane size, 4 rays" {
		t.Errorf("rungs say %q and %q", rungs[1].words, rungs[5].words)
	}
}

// anti-test of TestLadderClimbsInOrder
// a picture too small to divide does not draw the same rung
// twice, and no rung is below a pixel.
func TestLadderOfATinyPictureRepeatsNothing(t *testing.T) {
	rungs := ladderOf(1, 1)
	if len(rungs) != 3 {
		t.Fatalf("a pixel's ladder has %d rungs: %+v", len(rungs), rungs)
	}
	for _, each := range rungs {
		if each.size.width < one || each.size.height < one {
			t.Errorf("a rung of %+v", each.size)
		}
	}
}

// a rung predicted past the ceiling is not started; one within it is.
func TestCeilingStopsTheLadder(t *testing.T) {
	small := rung{size: Size{10, 10, 1}}
	big := rung{size: Size{20, 20, 1}}
	finer := rung{size: Size{20, 20, 2}}
	if got := predicted(small, time.Second, big); got != 4*time.Second {
		t.Errorf("four times the pixels predicted %v", got)
	}
	if got := predicted(big, time.Second, finer); got != 4*time.Second {
		t.Errorf("twice the rays predicted %v", got)
	}
	if admitted(small, Ceiling/2, big) {
		t.Error("a rung of twice the ceiling was admitted")
	}
}

// anti-test of TestCeilingStopsTheLadder
// a rung predicted at the ceiling exactly is started.
func TestCeilingAdmitsARungAtIt(t *testing.T) {
	small := rung{size: Size{10, 10, 1}}
	big := rung{size: Size{20, 20, 1}}
	if !admitted(small, Ceiling/4, big) {
		t.Error("a rung at the ceiling was not admitted")
	}
}

// a portrait job in a wide pane is a tall picture, centred, with bars
// either side; its pixels are drawn square and shown on the pane's,
// which are three wide to four tall, so it is shown wider than drawn.
func TestPlacedFitsAndKeepsTheShape(t *testing.T) {
	at := pane{width: 120, height: 60, pixel: 0.75}
	place := placed(1080, 1920, at)
	if place.height != 60 || place.width != 34 {
		t.Errorf("drawn at %d by %d, want 34 by 60", place.width,
			place.height)
	}
	if place.shown.Dx() != 45 || place.shown.Dy() != 60 {
		t.Errorf("shown %v, want 45 by 60", place.shown)
	}
	if place.shown.Min.X != (120-45)/2 || place.shown.Min.Y != 0 {
		t.Errorf("not centred: %v", place.shown)
	}
}

// anti-test of TestPlacedFitsAndKeepsTheShape
// a picture of the pane's own shape fills it with no bars,
// and no picture is placed past the pane.
func TestPlacedFillsAPaneOfItsShape(t *testing.T) {
	at := pane{width: 80, height: 60, pixel: 1}
	place := placed(160, 120, at)
	if place.shown != image.Rect(0, 0, 80, 60) {
		t.Errorf("shown %v, want the whole pane", place.shown)
	}
	wide := placed(10000, 1, at)
	if !wide.shown.In(image.Rect(0, 0, 80, 60)) || wide.height < one {
		t.Errorf("a long picture is placed at %v", wide.shown)
	}
}

// the pane is the screen less the foot and the status row.
func TestPaneIsTheScreenLessTheFoot(t *testing.T) {
	at := paneOf(testScreen())
	rows := testRows - FootRows - statusRows
	if at.width != testCols*pair || at.height != rows*3 ||
		at.pixel != testPixel {
		t.Errorf("the pane is %+v", at)
	}
}

// anti-test of TestPaneIsTheScreenLessTheFoot
// a screen too short for the foot keeps a row of picture, and
// a screen that says no pixel shape has square ones.
func TestPaneOfATinyScreen(t *testing.T) {
	at := paneOf(terminal.Screen{Cols: 4, Rows: 2, W: 8, H: 6})
	if at.height != 3 || at.pixel != one {
		t.Errorf("the pane is %+v", at)
	}
}

// a coarse picture grows as blocks; a fine one shrinks to the mean of
// what each pixel covers; the canvas is black where it is not filled.
func TestCanvasResamplesAndBars(t *testing.T) {
	two := image.NewRGBA(image.Rect(0, 0, 2, 1))
	two.SetRGBA(0, 0, color.RGBA{R: 200, A: 255})
	two.SetRGBA(1, 0, color.RGBA{G: 200, A: 255})
	place := placing{shown: image.Rect(2, 0, 6, 2), paneW: 8, paneH: 2}
	out := canvas(two, place)
	if got := out.RGBAAt(3, 1); got.R != 200 || got.G != 0 {
		t.Errorf("the left block is %v", got)
	}
	if got := out.RGBAAt(5, 0); got.G != 200 || got.R != 0 {
		t.Errorf("the right block is %v", got)
	}
	for _, x := range []int{0, 1, 6, 7} {
		if got := out.RGBAAt(x, 0); got != (color.RGBA{A: 255}) {
			t.Errorf("the bar at %d is %v", x, got)
		}
	}
	shrunk := canvas(two, placing{shown: image.Rect(0, 0, 1, 1),
		paneW: 1, paneH: 1})
	if got := shrunk.RGBAAt(0, 0); got.R != 100 || got.G != 100 {
		t.Errorf("two pixels shrunk to one are %v, want their mean", got)
	}
}

// exposure scales the colour of a copy; coverage and depth are kept and
// the light drawn is not touched.
func TestScaledChangesOnlyTheCopysColour(t *testing.T) {
	light := lit(2, 1)
	light.Pix[0].Depth = 3
	bright := scaled(light, gainOf(thirdsInStop))
	if bright.Pix[0].R != 1 || bright.Pix[0].Cover != 1 ||
		bright.Pix[0].Depth != 3 {
		t.Errorf("a stop up is %+v", bright.Pix[0])
	}
	if light.Pix[0].R != 0.5 {
		t.Errorf("the light drawn was changed to %+v", light.Pix[0])
	}
	if gainOf(none) != one {
		t.Errorf("no thirds gain %v", gainOf(none))
	}
}

// a size says its parts; a part below nought is refused by name.
func TestNewSizeRefusesBelowNought(t *testing.T) {
	size, err := NewSize(3, 2, 1)
	if err != nil || size.Width() != 3 || size.Height() != 2 ||
		size.Rays() != 1 {
		t.Errorf("a size of 3 by 2 at 1 is %+v, %v", size, err)
	}
	if _, err := NewSize(3, -one, 1); err == nil {
		t.Error("a height below nought was taken")
	}
	if zero, err := NewSize(none, none, none); err != nil ||
		zero != (Size{}) {
		t.Errorf("the job's own size was refused: %v", err)
	}
}
