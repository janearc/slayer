package hud

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// grey is a picture of one grey, so what the hud draws on it is seen.
func grey(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 90, G: 90,
		B: 90, A: 255}), image.Point{}, draw.Src)
	return img
}

// drawnOn is a hud drawn on a grey picture 200 by 100, ten rows of
// cells ten pixels high and five wide, forty across.
func drawnOn(t *testing.T, elements ...Element) *image.RGBA {
	t.Helper()
	img := grey(200, 100)
	err := Hud{Elements: elements}.Picture(img, 10, 0, dusk, View{})
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// count is how many pixels of a part of a picture are one colour.
func count(img *image.RGBA, part image.Rectangle, colour color.RGBA) int {
	n := 0
	for y := part.Min.Y; y < part.Max.Y; y++ {
		for x := part.Min.X; x < part.Max.X; x++ {
			if img.RGBAAt(x, y) == colour {
				n++
			}
		}
	}
	return n
}

// in a picture a bar at zero fills none of its pixels, at a half half, at one
// all, and at three eighths of one cell's width three eighths to the pixel, in
// the theme's ok over its ground.
func TestABarInAPictureFillsItsShare(t *testing.T) {
	ok, ground := rgba("#7fb069"), rgba("#1d1b26")
	part := image.Rect(0, 0, 200, 10)
	for value, want := range map[float64]int{0: 0, 0.5: 1000, 1: 2000} {
		img := drawnOn(t, Bar{Box: Box{0, 0, 1, 0.1},
			Value: Fixed(value), Fill: "ok", Empty: "ground"})
		got := count(img, part, ok)
		if got != want || got+count(img, part, ground) != 2000 {
			t.Errorf("a bar at %v fills %d pixels, want %d", value,
				got, want)
		}
	}
	img := drawnOn(t, Bar{Box: Box{0, 0, 0.2, 0.1},
		Value: Fixed(0.375 / 8), Fill: "ok", Empty: "ground"})
	if got := count(img, image.Rect(0, 0, 40, 10), ok); got != 20 {
		t.Errorf("three eighths of a cell five wide fills %d pixels",
			got)
	}
}

// what a hud does not cover is the picture's, every pixel; a frame's
// lines are in its ink in its edge cells, and its middle is untouched;
// words are inked inside their own cells.
func TestAPictureIsTheHudsOnlyWhereItsCellsAre(t *testing.T) {
	ink := rgba("#e8dcc8")
	img := drawnOn(t, Frame{Box: Box{0, 0, 0.5, 0.5}, Ink: "ink"},
		Text{Words: Words("ok"), At: Point{0.75, 0.75}, Ink: "ink"})
	both := Hud{Elements: []Element{
		Frame{Box: Box{0, 0, 0.5, 0.5}, Ink: "ink"},
		Text{Words: Words("ok"), At: Point{0.75, 0.75}, Ink: "ink"}}}
	cover, _ := both.Cover(image.Rect(0, 0, 40, 10), 0, dusk, View{})
	plain := color.RGBA{R: 90, G: 90, B: 90, A: 255}
	for y := range 100 {
		for x := range 200 {
			_, ok := cover.At(x/5, y/10)
			if !ok && img.RGBAAt(x, y) != plain {
				t.Fatalf("pixel %d, %d moved, in no hud cell",
					x, y)
			}
		}
	}
	if count(img, image.Rect(5, 10, 95, 40), plain) != 90*30 {
		t.Error("the frame's middle changed")
	}
	if count(img, image.Rect(0, 0, 100, 10), ink) == 0 {
		t.Error("the frame's top edge has no ink")
	}
	inked := 0
	for x := 150; x < 160; x++ {
		for y := 70; y < 80; y++ {
			if img.RGBAAt(x, y) != plain {
				inked++
			}
		}
	}
	if inked == 0 {
		t.Error("the words have no ink in their cells")
	}
}

// a picture too short for its rows, or with none, is refused; so is a
// hud whose role the theme has not.
func TestAPictureThatCannotBeDrawnSaysSo(t *testing.T) {
	for _, rows := range []int{0, 60} {
		err := Hud{}.Picture(grey(200, 100), rows, 0, dusk, View{})
		if !errors.Is(err, ErrHud) {
			t.Errorf("%d rows on a picture 100 high: %v", rows, err)
		}
	}
	err := Hud{Elements: []Element{Text{Words: Words("x"),
		Ink: "mauve"}}}.Picture(grey(200, 100), 10, 0, dusk, View{})
	if !errors.Is(err, ErrHud) {
		t.Errorf("an unknown role: %v", err)
	}
}
