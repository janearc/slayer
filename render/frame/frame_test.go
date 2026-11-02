package frame

import (
	"errors"
	"math"
	"testing"

	"github.com/janearc/slayer/maths"
)

// a new frame is every pixel meeting nothing: dark, uncovered and
// infinitely far, w by h of them.
func TestANewFrameHasMetNothing(t *testing.T) {
	f := New(3, 2)
	if f.W != 3 || f.H != 2 || len(f.Pix) != 6 {
		t.Fatalf("a 3 by 2 frame is %d by %d of %d", f.W, f.H, len(f.Pix))
	}
	for _, p := range f.Pix {
		if p.R != 0 || p.G != 0 || p.B != 0 || p.Cover != 0 ||
			!math.IsInf(float64(p.Depth), 1) {
			t.Errorf("a new pixel is %+v", p)
		}
	}
	if empty := New(0, 0); len(empty.Pix) != 0 {
		t.Errorf("a frame of nothing holds %d", len(empty.Pix))
	}
}

// a pixel set at a column and row is the one read back there, and no
// other is touched: rows count from the top left, row by row.
func TestAPixelIsWhereItIsSet(t *testing.T) {
	f := New(3, 2)
	f.Set(2, 0, Pixel{R: 9})
	f.Set(0, 1, Pixel{G: 7})
	if f.At(2, 0).R != 9 || f.At(0, 1).G != 7 || f.Pix[2].R != 9 ||
		f.Pix[3].G != 7 {
		t.Errorf("set and read back: %+v", f.Pix)
	}
	if got := (Pixel{R: 0.5, G: 0.25, B: 2}).Light(); got !=
		maths.RGB(0.5, 0.25, 2) {
		t.Errorf("a pixel's light is %v", got)
	}
}

// a pixel past the frame's edge, either way, is refused by name: never
// read or written as the next row's first.
func TestAPixelPastTheEdgeIsRefused(t *testing.T) {
	f := New(3, 2)
	for _, at := range [][2]int{{3, 0}, {-1, 1}, {0, 2}, {0, -1}} {
		for name, touch := range map[string]func(){
			"read":  func() { f.At(at[0], at[1]) },
			"write": func() { f.Set(at[0], at[1], Pixel{R: 1}) },
		} {
			if err := refused(touch); !errors.Is(err, errOutside) {
				t.Errorf("%s %v: %v, want %v", name, at, err, errOutside)
			}
		}
	}
	if f.Pix[3].R != 0 {
		t.Error("a write past the edge landed on the next row")
	}
}

// a frame of a negative size is refused by name.
func TestANegativeFrameIsRefused(t *testing.T) {
	for _, size := range [][2]int{{-1, 2}, {2, -1}, {-2, -2}} {
		if err := refused(func() { New(size[0], size[1]) }); !errors.Is(err,
			errSize) {
			t.Errorf("a frame %v: %v, want %v", size, err, errSize)
		}
	}
}

// refused is the error a call panics with, or nil.
func refused(call func()) (err error) {
	defer func() {
		if caught := recover(); caught != nil {
			err, _ = caught.(error)
		}
	}()
	call()
	return nil
}
