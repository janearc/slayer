package hud

// tests written for what game mutants found the other tests could not
// tell apart: each holds a rule at the edge a changed line would move.

import (
	"errors"
	"image"
	"image/color"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
)

// a frame of exactly two cells each way is four corners.
func TestAFrameTwoCellsSquareIsItsCorners(t *testing.T) {
	cover := covered(t, image.Rect(0, 0, 10, 10), 0,
		Frame{Box: Box{0, 0, 0.2, 0.2}, Ink: "ink"})
	var runes []rune
	for _, at := range cover.Mask() {
		cell, _ := cover.At(at.X, at.Y)
		runes = append(runes, cell.Rune)
	}
	if string(runes) != "┌┐└┘" {
		t.Errorf("a two by two frame is %q", string(runes))
	}
}

// a bar that starts away from the first column fills from its own
// left, and covers no cell past its right.
func TestABarFillsFromItsOwnLeft(t *testing.T) {
	cover := covered(t, image.Rect(0, 0, 10, 1), 0,
		Bar{Box: Box{0.2, 0, 0.6, 1}, Value: Fixed(0.5), Fill: "ok",
			Empty: "ground"})
	runes := ""
	for x := range 10 {
		if cell, ok := cover.At(x, 0); ok {
			runes += string(cell.Rune)
		} else {
			runes += "."
		}
	}
	if runes != "..██  ...." {
		t.Errorf("the bar is %q", runes)
	}
}

// a mask is every covered cell row by row from the top left.
func TestAMaskIsInReadingOrder(t *testing.T) {
	cover := covered(t, image.Rect(0, 0, 6, 4), 0,
		Frame{Box: Box{0, 0, 1, 1}, Ink: "ink"})
	mask := cover.Mask()
	sorted := slices.IsSortedFunc(mask, func(a, b image.Point) int {
		if a.Y != b.Y {
			return a.Y - b.Y
		}
		return a.X - b.X
	})
	if !sorted || mask[0] != image.Pt(0, 0) || mask[6] != image.Pt(0, 1) {
		t.Errorf("the mask is %v", mask)
	}
}

// a box of no width or height still takes one cell each way.
func TestABoxOfNoSizeTakesACell(t *testing.T) {
	on := &grid{cover: Cover{Rect: image.Rect(0, 0, 10, 10)}}
	if got := on.cells(Box{0.5, 0.5, 0.5, 0.5}); got !=
		image.Rect(5, 5, 6, 6) {
		t.Errorf("a box of no size takes %v", got)
	}
}

// a picture may be one row of cells, or rows two pixels high.
func TestAPictureTakesItsFewestAndItsMostRows(t *testing.T) {
	for _, rows := range []int{1, 50} {
		if err := (Hud{}).Picture(grey(200, 100), rows, 0, dusk,
			View{}); err != nil {
			t.Errorf("%d rows on a picture 100 high: %v", rows, err)
		}
	}
}

// inked is which pixels of a part of a picture are not the grey under
// it, as a string of rows, a hash for ink and a dot for grey.
func inked(img *image.RGBA, part image.Rectangle) string {
	plain := color.RGBA{R: 90, G: 90, B: 90, A: 255}
	text := ""
	for y := part.Min.Y; y < part.Max.Y; y++ {
		for x := part.Min.X; x < part.Max.X; x++ {
			if img.RGBAAt(x, y) == plain {
				text += "."
			} else {
				text += "#"
			}
		}
		text += "\n"
	}
	return text
}

// a frame's lines are drawn exactly: two pixels thick in a cell 24
// high and 12 wide, through its middle, the corner's two halves meeting
// there; nothing of the font drawn over them.
func TestAFramesLinesAreExact(t *testing.T) {
	img := grey(120, 96)
	err := Hud{Elements: []Element{Frame{Box: Box{0, 0, 1, 1},
		Ink: "ink"}}}.Picture(img, 4, 0, dusk, View{})
	if err != nil {
		t.Fatal(err)
	}
	across := ""
	for y := range 24 {
		row := "............"
		if y == 11 || y == 12 {
			row = "############"
		}
		across += row + "\n"
	}
	if got := inked(img, image.Rect(12, 0, 24, 24)); got != across {
		t.Errorf("a line across is\n%s", got)
	}
	corner := ""
	for y := range 24 {
		switch {
		case y < 11:
			corner += "............\n"
		case y < 13:
			corner += ".....#######\n"
		default:
			corner += ".....##.....\n"
		}
	}
	if got := inked(img, image.Rect(0, 0, 12, 24)); got != corner {
		t.Errorf("the top left corner is\n%s", got)
	}
}

// seven eighths of a cell five pixels wide is four pixels, the share
// rounded to the nearest pixel.
func TestSevenEighthsOfACellIsRounded(t *testing.T) {
	img := drawnOn(t, Bar{Box: Box{0, 0, 0.025, 0.1},
		Value: Fixed(7.0 / 8), Fill: "ok", Empty: "ground"})
	if got := count(img, image.Rect(0, 0, 5, 10), rgba("#7fb069")); got !=
		40 {
		t.Errorf("seven eighths fills %d pixels, want 40", got)
	}
}

// a letter on a ground is not drawn as a block: its cell's left column
// is not all ink.
func TestALetterIsNotABlock(t *testing.T) {
	img := drawnOn(t, Text{Words: Words("o"), At: Point{0, 0}, Ink: "ink",
		Ground: "ground"})
	if count(img, image.Rect(0, 0, 1, 10), rgba("#e8dcc8")) == 10 {
		t.Error("the letter's cell is inked down its left edge")
	}
}

// a picture of the cover's height but not its width is refused too.
func TestAPictureOfTheWrongWidthIsRefused(t *testing.T) {
	cover := covered(t, image.Rect(0, 0, 19, 10), 0)
	if _, err := cover.Over(sextants()); !errors.Is(err, ErrHud) {
		t.Errorf("a hud 19 wide over a picture 20 wide: %v", err)
	}
}

// four squares of six set is the larger part: the cell is its ink.
func TestFourSquaresOfSixIsTheLargerPart(t *testing.T) {
	fg, bg := color.RGBA{R: 200, A: 255}, color.RGBA{B: 200, A: 255}
	four := terminal.Cell{Fg: fg, Bg: bg, Pattern: 0b001111}
	if got := behind(four); got != fg {
		t.Errorf("four squares of six set: %v", got)
	}
}

// a ramp over a nanosecond starts at its from, and a ramp from ten to
// twenty is fifteen halfway.
func TestARampRunsBetweenItsEnds(t *testing.T) {
	if got := (Ramp{From: 3, To: 7, Over: 1}).At(0); got != 3 {
		t.Errorf("a nanosecond ramp at its start is %v", got)
	}
	ramp := Ramp{From: 10, To: 20, Over: 2 * time.Second}
	if got := ramp.At(time.Second); got != 15 {
		t.Errorf("halfway from ten to twenty is %v", got)
	}
}

// a view one pixel square projects; a point level with the eye, or a
// hair behind it, is not ahead.
func TestAViewOfOnePixelProjectsAndLevelIsNotAhead(t *testing.T) {
	eye := maths.Subspace{Origin: maths.V(0, 0, 10),
		Orientation: maths.LookAt(maths.V(0, 0, 10), maths.Vec{})}
	view := View{Eye: eye, Width: 1, Height: 1}
	if _, _, ahead, err := view.Project(maths.Vec{}); !ahead || err != nil {
		t.Errorf("a one pixel view: %v, %v", ahead, err)
	}
	for _, p := range []maths.Vec{maths.V(3, 0, 10), maths.V(0, 1, 10.5)} {
		if _, _, ahead, _ := view.Project(p); ahead {
			t.Errorf("%v is ahead of an eye at 0, 0, 10", p)
		}
	}
}

// the scene says how far along a ray a thing is met, in the world's
// units: a ball of no scale said, of twice the scale, and one past near.
func TestTheSceneSaysHowFarAThingIsMet(t *testing.T) {
	ray := maths.Ray{Dir: maths.V(0, 0, -1)}
	for name, each := range map[string]struct {
		scale, near, want float64
	}{"unsaid": {0, 0, 9}, "twice": {2, 0, 8}, "past near": {2, 9, 12}} {
		world := thing.NewWorld()
		world.Add(thing.Thing{Shape: shape.Sphere{Radius: 1},
			Motion: thing.Still(maths.Subspace{
				Origin: maths.V(0, 0, -10),
				Scale:  each.scale})})
		t2, hit := Seen(world, 0).Hit(ray, each.near, 100)
		if !hit || math.Abs(t2-each.want) > 1e-9 {
			t.Errorf("%s: met at %v, %v; want %v", name, t2, hit,
				each.want)
		}
	}
}

// in a cell under 24 pixels high a frame's line is one pixel thick.
func TestAFramesLineIsAPixelInASmallCell(t *testing.T) {
	img := grey(60, 48)
	err := Hud{Elements: []Element{Frame{Box: Box{0, 0, 1, 1},
		Ink: "ink"}}}.Picture(img, 4, 0, dusk, View{})
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for y := range 12 {
		if count(img, image.Rect(6, y, 12, y+1), rgba("#e8dcc8")) > 0 {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("a line across a cell 12 high is %d rows thick", rows)
	}
}
