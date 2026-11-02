package learn

import (
	"image"
	"image/color"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal"
)

// a picture that is already two colours a cell comes back unchanged:
// Picture undoes Chop where Chop loses nothing. it is not compared with a
// terminal; interfaces/terminal's pty tests are.
func TestTwoColoursACellComeBackAsTheyWere(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	a, b := color.RGBA{200, 40, 40, 255}, color.RGBA{30, 60, 210, 255}
	for y := range 3 {
		for x := range 4 {
			c := a
			if (x+y)%2 == 0 {
				c = b
			}
			img.SetRGBA(x, y, c)
		}
	}
	shown := Chopped(terminal.Chop(img))
	for y := range 3 {
		for x := range 4 {
			if got, want := shown.RGBAAt(x, y), img.RGBAAt(x, y); got != want {
				t.Fatalf("(%d,%d) is %v, want %v", x, y, got, want)
			}
		}
	}
}
