package learn

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal"
)

// cells of nil are an empty picture, not a panic.
func TestNoCellsAreAnEmptyPicture(t *testing.T) {
	if got := Chopped(nil); !got.Bounds().Empty() {
		t.Errorf("nil cells are %v", got.Bounds())
	}
}

// a cell of three colours comes back as the two a cell can hold: rows of
// red, green and blue, two pixels wide, are red and blue averaged in
// linear light, a half each, which srgb encodes as 188, above and below,
// and green between. worked by hand from Chop's linear mean.
func TestACellOfThreeColoursIsItsTwo(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	rows := []color.RGBA{{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}}
	for y, c := range rows {
		img.SetRGBA(0, y, c)
		img.SetRGBA(1, y, c)
	}
	shown := Chopped(terminal.Chop(img))
	want := []color.RGBA{{188, 0, 188, 255}, {0, 255, 0, 255}, {188, 0, 188, 255}}
	for y, w := range want {
		for x := range 2 {
			if got := shown.RGBAAt(x, y); got != w {
				t.Errorf("(%d,%d) is %v, want %v", x, y, got, w)
			}
		}
	}
}

// a picture not whole cells comes back whole cells: five by four is three
// cells by two, six by six.
func TestAPictureComesBackWholeCells(t *testing.T) {
	got := Chopped(terminal.Chop(image.NewRGBA(image.Rect(0, 0, 5, 4)))).Bounds()
	if got.Dx() != 6 || got.Dy() != 6 {
		t.Errorf("five by four came back %v", got)
	}
}
