package hud

import (
	"errors"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
)

// covered is a hud's cover in dusk over a rectangle, or the test fails.
func covered(t *testing.T, rect image.Rectangle, at time.Duration,
	elements ...Element) Cover {
	t.Helper()
	cover, err := Hud{Elements: elements}.Cover(rect, at, dusk, View{})
	if err != nil {
		t.Fatal(err)
	}
	return cover
}

// words are set a letter a cell along their row: from their place when
// aligned left, centred on it, or ending on it; and cut where the
// rectangle ends.
func TestWordsAreAlignedOnTheirPlace(t *testing.T) {
	rect := image.Rect(0, 0, 20, 4)
	for align, first := range map[Align]int{Left: 10, Centre: 8,
		Right: 6} {
		cover := covered(t, rect, 0, Text{Words: Words("abcd"),
			At: Point{0.5, 0.5}, Align: align, Ink: "ink"})
		mask := cover.Mask()
		if len(mask) != 4 || mask[0] != image.Pt(first, 2) {
			t.Errorf("aligned %d: %v", align, mask)
		}
		cell, _ := cover.At(first+2, 2)
		if cell.Rune != 'c' || !cell.Clear {
			t.Errorf("aligned %d: the third letter is %+v",
				align, cell)
		}
	}
	cut := covered(t, rect, 0, Text{Words: Words("abcdef"),
		At: Point{0.9, 0}, Ink: "ink"})
	if got := len(cut.Mask()); got != 2 {
		t.Errorf("words past the edge cover %d cells, want 2", got)
	}
}

// a rectangle that does not start at zero, as carrie hands its drawers, is laid
// out in its own cells: its top left is a box's zero, zero, and nothing is set
// outside it.
func TestAHudFillsTheRectangleItIsGiven(t *testing.T) {
	rect := image.Rect(30, 10, 50, 20)
	cover := covered(t, rect, 0, Frame{Box: Box{0, 0, 1, 1}, Ink: "ink"})
	for _, at := range cover.Mask() {
		if !at.In(rect) {
			t.Errorf("cell %v is outside %v", at, rect)
		}
	}
	if cell, ok := cover.At(30, 10); !ok || cell.Rune != '┌' {
		t.Errorf("the top left is %+v, %v", cell, ok)
	}
	if cell, ok := cover.At(49, 19); !ok || cell.Rune != '┘' {
		t.Errorf("the bottom right is %+v, %v", cell, ok)
	}
}

// a frame covers exactly its edge, every cell of it, the picture
// showing behind; its middle is the picture's. one less than two cells
// either way is refused.
func TestAFrameCoversItsEdge(t *testing.T) {
	cover := covered(t, image.Rect(0, 0, 10, 6), 0,
		Frame{Box: Box{0.1, 0, 0.9, 1}, Ink: "ink"})
	box := image.Rect(1, 0, 9, 6)
	for y := 0; y < 6; y++ {
		for x := 0; x < 10; x++ {
			rim := x == 1 || x == 8 || y == 0 || y == 5
			edge := image.Pt(x, y).In(box) && rim
			cell, ok := cover.At(x, y)
			if ok != edge || ok && !cell.Clear {
				t.Errorf("cell %d, %d: covered %v, %+v",
					x, y, ok, cell)
			}
		}
	}
	thin := Hud{Elements: []Element{Frame{Box: Box{0, 0, 0.1, 1},
		Ink: "ink"}}}
	if _, err := thin.Cover(image.Rect(0, 0, 10, 6), 0, dusk,
		View{}); !errors.Is(err, ErrHud) {
		t.Errorf("a frame one cell wide: %v", err)
	}
}

// filled is how many eighths of a bar's cells are filled, of how many.
func filled(cover Cover) (int, int) {
	got, all := 0, 0
	for _, at := range cover.Mask() {
		cell, _ := cover.At(at.X, at.Y)
		for share, block := range eighths {
			if block == cell.Rune {
				got += share
			}
		}
		all += 8
	}
	return got, all
}

// a bar at zero fills nothing, at a half fills half, and at one fills all, to
// the eighth of a cell, in every row of its box; a bar of three cells at a
// third fills one cell exactly.
func TestABarFillsItsShare(t *testing.T) {
	rect := image.Rect(0, 0, 20, 4)
	for value, want := range map[float64]float64{0: 0, 0.5: 0.5, 1: 1,
		0.3: 0.3} {
		cover := covered(t, rect, 0, Bar{Box: Box{0, 0, 1, 0.5},
			Value: Fixed(value), Fill: "ok", Empty: "ground"})
		got, all := filled(cover)
		if all != 20*2*8 || float64(got) != want*float64(all) {
			t.Errorf("a bar at %v fills %d eighths of %d",
				value, got, all)
		}
	}
	third := covered(t, image.Rect(0, 0, 3, 1), 0, Bar{Box: Box{0, 0, 1, 1},
		Value: Fixed(1.0 / 3), Fill: "ok", Empty: "ground"})
	runes := ""
	for x := range 3 {
		cell, _ := third.At(x, 0)
		runes += string(cell.Rune)
	}
	if runes != "█  " {
		t.Errorf("a third of three cells is %q", runes)
	}
}

// a bar follows its gauge from frame to frame; one past one or below zero, or
// not a number, is refused.
func TestABarFollowsItsGauge(t *testing.T) {
	ramp := Ramp{From: 0, To: 1, Over: 4 * time.Second}
	bar := Bar{Box: Box{0, 0, 1, 1}, Value: ramp, Fill: "ok",
		Empty: "ground"}
	for at, want := range map[time.Duration]int{0: 0, time.Second: 20,
		2 * time.Second: 40, 4 * time.Second: 80} {
		if got, _ := filled(covered(t, image.Rect(0, 0, 10, 1), at,
			bar)); got != want {
			t.Errorf("at %v the bar fills %d eighths, want %d",
				at, got, want)
		}
	}
	for _, value := range []float64{-0.1, 1.1, nan()} {
		bad := Hud{Elements: []Element{Bar{Box: Box{0, 0, 1, 1},
			Value: Fixed(value), Fill: "ok", Empty: "ground"}}}
		if _, err := bad.Cover(image.Rect(0, 0, 4, 1), 0, dusk,
			View{}); !errors.Is(err, ErrHud) {
			t.Errorf("a bar at %v: %v", value, err)
		}
	}
}

// nan is not a number.
func nan() float64 {
	zero := 0.0
	return zero / zero
}

// a picture of sextant cells, each its own colour, so any cell moved or
// changed is seen.
func sextants() *terminal.Cells {
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := range 30 {
		for x := range 40 {
			img.Set(x, y, color.RGBA{R: uint8(x * 6),
				G: uint8(y * 8), B: uint8((x + y) * 3), A: 255})
		}
	}
	return terminal.Chop(img)
}

// over the terminal's sextants the hud covers exactly the cells its
// mask says, each the hud's character in the hud's colours, a clear one
// on the picture's colour; every other cell is the picture's, as it was.
func TestTheHudCoversExactlyItsMask(t *testing.T) {
	picture := sextants()
	rect := image.Rect(0, 0, picture.W, picture.H)
	cover := covered(t, rect, 0,
		Frame{Box: Box{0, 0, 1, 1}, Ink: "ink"},
		Text{Words: Words("hud"), At: Point{0.5, 0.5}, Align: Centre,
			Ink: "ink", Ground: "ground"},
		Bar{Box: Box{0.2, 0.8, 0.8, 0.9}, Value: Fixed(0.25),
			Fill: "ok", Empty: "ground"})
	drawn, err := cover.Over(picture)
	if err != nil {
		t.Fatal(err)
	}
	masked := map[image.Point]bool{}
	for _, at := range cover.Mask() {
		masked[at] = true
	}
	for y := range picture.H {
		for x := range picture.W {
			got, was := drawn.At(x, y), picture.At(x, y)
			cell, ok := cover.At(x, y)
			switch {
			case ok != masked[image.Pt(x, y)]:
				t.Fatalf("cell %d, %d: At and Mask disagree",
					x, y)
			case !ok && got != was:
				t.Errorf("cell %d, %d is not the picture's",
					x, y)
			case ok && (got != terminal.Cell{Rune: cell.Rune,
				Fg: cell.Fg, Bg: under(cell, was)}):
				t.Errorf("cell %d, %d is %+v, the hud's %+v",
					x, y, got, cell)
			}
		}
	}
	if len(masked) == 0 || len(masked) == picture.W*picture.H {
		t.Errorf("the hud covers %d of %d cells", len(masked),
			picture.W*picture.H)
	}
}

// under is the colour behind a hud cell over a picture cell: its own
// ground, or the picture's for a clear one.
func under(cell Cell, was terminal.Cell) color.RGBA {
	if cell.Clear {
		return behind(was)
	}
	return cell.Bg
}

// a clear cell keeps the colour of the larger part of the picture cell
// under it; a picture of another size than the hud's rectangle is
// refused.
func TestAClearCellKeepsThePicturesColour(t *testing.T) {
	fg, bg := color.RGBA{R: 200, A: 255}, color.RGBA{B: 200, A: 255}
	five := terminal.Cell{Fg: fg, Bg: bg, Pattern: 0b111011}
	if got := behind(five); got != fg {
		t.Errorf("five squares of six set: %v", got)
	}
	three := terminal.Cell{Fg: fg, Bg: bg, Pattern: 0b000111}
	if got := behind(three); got != bg {
		t.Errorf("three squares of six set: %v", got)
	}
	cover := covered(t, image.Rect(0, 0, 5, 5), 0)
	if _, err := cover.Over(sextants()); !errors.Is(err, ErrHud) {
		t.Errorf("a hud of 5x5 over a picture of 20x10: %v", err)
	}
}

// a frame or a bar in a role the theme has not is not drawn, and says
// which role.
func TestAFrameOrABarInAnUnknownRoleIsRefused(t *testing.T) {
	for name, element := range map[string]Element{
		"frame": Frame{Box: Box{0, 0, 1, 1}, Ink: "mauve"},
		"bar": Bar{Box: Box{0, 0, 1, 1}, Value: Fixed(0.5), Fill: "ok",
			Empty: "mauve"}} {
		_, err := Hud{Elements: []Element{element}}.Cover(
			image.Rect(0, 0, 8, 4), 0, dusk, View{})
		if !errors.Is(err, ErrHud) {
			t.Errorf("a %s in mauve: %v", name, err)
		}
	}
}
