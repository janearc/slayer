package sheet

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// the darkroom grey and the paler ink, written out here so a changed
// ground or ink is seen. basicfont's mask is one bit, so an inked pixel
// is the ink exactly.
var (
	darkroom = color.RGBA{38, 38, 40, 255}
	paler    = color.RGBA{200, 200, 196, 255}
)

// A frame no pixels tall still has its label strip: one 4 by 0 frame is
// 12+0+18+12 = 42 high.
func TestAFrameOfNoHeightKeepsItsStrip(t *testing.T) {
	s := New(1)
	s.Add("a", flat(4, 0, red), 1)
	if b := s.Image().Bounds(); b != image.Rect(0, 0, 28, 42) {
		t.Errorf("a sheet of one 4 by 0 frame is %v, wanted 28 by 42", b)
	}
}

// A Sheet made without New prints one frame to a row, not a division by zero:
// two 4 by 4 frames are 12+4+12 = 28 wide and 12+2*(4+18+12) = 80 high.
func TestASheetWithoutNewIsOneColumn(t *testing.T) {
	var s Sheet
	s.Add("a", flat(4, 4, red), 1)
	s.Add("b", flat(4, 4, red), 1)
	if b := s.Image().Bounds(); b != image.Rect(0, 0, 28, 80) {
		t.Errorf("a sheet without New is %v, wanted 28 by 80", b)
	}
}

// New with zero or fewer columns is one column, as with none at all.
func TestNoughtColumnsIsOne(t *testing.T) {
	for _, n := range []int{0, -3} {
		s := New(n)
		s.Add("a", flat(4, 4, red), 1)
		s.Add("b", flat(4, 4, red), 1)
		if b := s.Image().Bounds(); b != image.Rect(0, 0, 28, 80) {
			t.Errorf("New(%d) with two frames is %v, wanted 28 by 80", n, b)
		}
	}
}

// A nil picture panics with its label: a nil interface, a nil *RGBA and
// a nil *NRGBA alike.
func TestNoPictureIsRefusedByName(t *testing.T) {
	var none *image.RGBA
	var noneN *image.NRGBA
	for _, img := range []image.Image{nil, none, noneN} {
		func() {
			defer func() {
				r, _ := recover().(string)
				if !strings.Contains(r, `"smoke 7"`) {
					t.Errorf("adding no picture said %q", r)
				}
			}()
			New(1).Add("smoke 7", img, 1)
		}()
	}
}

// Two 10 by 10 frames at two to a row: the second's corner is at
// 12+10+12 = 34, the gap between at 22 is the ground, and so is the
// sheet's own corner.
func TestTheSecondColumnSitsAGapOver(t *testing.T) {
	s := New(2)
	s.Add("a", flat(10, 10, red), 1)
	s.Add("b", flat(10, 10, red), 1)
	img := s.Image()
	if c := img.RGBAAt(34, 12); c != red {
		t.Errorf("the second frame's corner is %v", c)
	}
	if c := img.RGBAAt(33, 12); c != darkroom {
		t.Errorf("just before the second frame is %v", c)
	}
	if c := img.RGBAAt(22, 12); c != darkroom {
		t.Errorf("the gap between frames is %v", c)
	}
	if c := img.RGBAAt(0, 0); c != darkroom {
		t.Errorf("the sheet's corner is %v", c)
	}
}

// A label's baseline is five above the strip's foot: an 80 by 20 frame at
// 12 has its strip from 32 to 50, so the baseline is 45. basicfont's 7
// by 13 face has an ascent of 11, so "a", one advance of 7 wide and no
// descender, inks only x 12 to 18 and y 34 to 44.
func TestALabelSitsUnderItsFrame(t *testing.T) {
	s := New(1)
	s.Add("a", flat(80, 20, red), 1)
	img := s.Image()
	inked := 0
	for y := 32; y < 50; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y) != paler {
				continue
			}
			inked++
			if x < 12 || x > 18 || y < 34 || y > 44 {
				t.Errorf("ink at %d,%d, outside the label's place", x, y)
			}
		}
	}
	if inked == 0 {
		t.Error("the label is not printed")
	}
	bottom := false
	for x := 12; x <= 18; x++ {
		bottom = bottom || img.RGBAAt(x, 44) == paler
	}
	if !bottom {
		t.Error("no ink on the row above the baseline, where an a sits")
	}
}

// PNG writes the sheet as Image prints it, pixel for pixel, and a path
// that cannot be written is an error.
func TestThePNGIsTheSheet(t *testing.T) {
	s := New(2)
	s.Add("a", flat(10, 10, red), 1)
	s.Add("b", flat(10, 10, red), 0.75)
	path := filepath.Join(t.TempDir(), "sheet.png")
	if err := s.PNG(path); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	want := s.Image()
	if got.Bounds() != want.Bounds() {
		t.Fatalf("the png is %v, the sheet %v", got.Bounds(), want.Bounds())
	}
	for y := 0; y < want.Bounds().Dy(); y++ {
		for x := 0; x < want.Bounds().Dx(); x++ {
			r, g, b, a := got.At(x, y).RGBA()
			w := want.RGBAAt(x, y)
			if c := (color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8),
				uint8(a >> 8)}); c != w {
				t.Fatalf("the png at %d,%d is %v, the sheet %v", x, y, c, w)
			}
		}
	}
	if err := s.PNG(filepath.Join(t.TempDir(), "none", "sheet.png")); err == nil {
		t.Error("a sheet into a folder that is not there wrote without error")
	}
}

// Len counts the frames added.
func TestLenCountsFrames(t *testing.T) {
	s := New(3)
	if s.Len() != 0 {
		t.Errorf("an empty sheet has %d", s.Len())
	}
	s.Add("a", flat(2, 2, red), 1)
	s.Add("b", flat(2, 2, red), 1)
	if s.Len() != 2 {
		t.Errorf("two frames are %d", s.Len())
	}
}
