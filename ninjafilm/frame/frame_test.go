package frame_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/frame"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	none    = 0 // defined: no amount at all
	endless = 1 // defined: math.Inf's code for infinity above all numbers
	one     = 1 // defined: a place on, or a single pixel
)

// what these tests choose, each picked in picked-numbers.yml
var (
	sideMost  = data.Whole("frame-side-most")
	pastMost  = data.Whole("frame-past-most")
	belowMost = data.Whole("frame-below-most")
	lampMost  = data.Picked("frame-lamp-most")
)

// recovered keeps what a panic was given, and stops the panic.
func recovered(into *any) { *into = recover() }

// panics is what a call panics with, or nothing if it does not.
func panics(call func()) (message any) {
	defer recovered(&message)
	call()
	return message
}

// toucher is a place in a frame to read or to write.
type toucher struct {
	grid *frame.Frame
	x, y int
}

// touching is a place in a frame.
func touching(grid *frame.Frame, x, y int) *toucher {
	var touch toucher
	touch.grid, touch.x, touch.y = grid, x, y
	return &touch
}

// read reads the place.
func (touch *toucher) read() { touch.grid.At(touch.x, touch.y) }

// write writes to the place.
func (touch *toucher) write() {
	var lit frame.Pixel
	lit.R = one
	touch.grid.Set(touch.x, touch.y, lit)
}

// drawnSize is the size of a drawn frame.
func drawnSize(t *rapid.T) (columns, rows int) {
	size := rapid.IntRange(none, sideMost)
	return size.Draw(t, "columns"), size.Draw(t, "rows")
}

// said is whether a panic gave an error that says a thing.
func said(message any, words string) bool {
	err, ok := message.(error)
	return ok && strings.Contains(err.Error(), words)
}

// New frames are every pixel meeting nothing. They are dark. They are
// uncovered and infinitely far. There are w by h of them.
func TestANewFrameHasMetNothing(t *testing.T) {
	rapid.Check(t, newFrameHasMetNothing)
}

func newFrameHasMetNothing(t *rapid.T) {
	columns, rows := drawnSize(t)
	made := frame.New(columns, rows)
	if made.W != columns || made.H != rows ||
		len(made.Pix) != columns*rows {
		t.Fatalf("a %d by %d frame is %d by %d of %d", columns, rows,
			made.W, made.H, len(made.Pix))
	}
	for _, each := range made.Pix {
		if each.R != none || each.G != none || each.B != none ||
			each.Cover != none || !math.IsInf(float64(each.Depth), endless) {
			t.Fatalf("a new pixel is %+v", each)
		}
	}
}

// Pixels set at a column and row are the ones read back there. No other
// is touched. Rows count from the top left, row by row.
func TestAPixelIsWhereItIsSet(t *testing.T) {
	rapid.Check(t, pixelIsWhereItIsSet)
}

func pixelIsWhereItIsSet(t *rapid.T) {
	columns := rapid.IntRange(one, sideMost).Draw(t, "columns")
	rows := rapid.IntRange(one, sideMost).Draw(t, "rows")
	grid := frame.New(columns, rows)
	before := append([]frame.Pixel(nil), grid.Pix...)
	x := rapid.IntRange(none, columns-one).Draw(t, "x")
	y := rapid.IntRange(none, rows-one).Draw(t, "y")
	var put frame.Pixel
	put.G = float32(rapid.Float64Range(one, lampMost).Draw(t, "green"))
	grid.Set(x, y, put)
	if grid.At(x, y) != put || grid.Pix[y*columns+x] != put {
		t.Fatalf("%v set at %d,%d came back as %v", put, x, y, grid.At(x, y))
	}
	for index, each := range grid.Pix {
		if index != y*columns+x && each != before[index] {
			t.Fatalf("setting %d,%d touched pixel %d", x, y, index)
		}
	}
}

func TestAPixelsLightIsItsLamps(t *testing.T) {
	rapid.Check(t, pixelLightIsItsLamps)
}

func pixelLightIsItsLamps(t *rapid.T) {
	lamp := rapid.Float64Range(none, lampMost)
	var made frame.Pixel
	made.R = float32(lamp.Draw(t, "red"))
	made.G = float32(lamp.Draw(t, "green"))
	made.B = float32(lamp.Draw(t, "blue"))
	want := maths.RGB(float64(made.R), float64(made.G), float64(made.B))
	if got := made.Light(); got != want {
		t.Fatalf("the light of %+v is %v, want %v", made, got, want)
	}
}

// anti-tests: places past the edge of the frame are refused by name on
// a read. They are refused on a write too. They are never read or
// written as the next row's first.
func TestAPixelPastTheEdgeIsRefused(t *testing.T) {
	rapid.Check(t, pixelPastTheEdgeIsRefused)
}

func pixelPastTheEdgeIsRefused(t *rapid.T) {
	columns := rapid.IntRange(one, sideMost).Draw(t, "columns")
	rows := rapid.IntRange(one, sideMost).Draw(t, "rows")
	grid := frame.New(columns, rows)
	before := append([]frame.Pixel(nil), grid.Pix...)
	past := rapid.IntRange(one, pastMost).Draw(t, "past")
	inside := rapid.IntRange(none, columns-one).Draw(t, "inside")
	places := map[string]*toucher{
		"right": touching(grid, columns-one+past, none),
		"left":  touching(grid, -past, none),
		"below": touching(grid, none, rows-one+past),
		"above": touching(grid, inside, -past)}
	for name, touch := range places {
		if got := panics(touch.read); !said(got, "past the frame's edge") {
			t.Fatalf("read %s: %v", name, got)
		}
		if got := panics(touch.write); !said(got, "past the frame's edge") {
			t.Fatalf("write %s: %v", name, got)
		}
	}
	if !reflect.DeepEqual(grid.Pix, before) {
		t.Fatal("a write past the edge landed on a pixel")
	}
}

// anti-tests: a frame of a negative size is refused by name.
func TestANegativeFrameIsRefused(t *testing.T) {
	rapid.Check(t, negativeFrameIsRefused)
}

// maker is a frame to be made of a size.
type maker struct{ columns, rows int }

// making is a frame to be made of a size.
func making(columns, rows int) maker {
	var made maker
	made.columns, made.rows = columns, rows
	return made
}

// build makes the frame, to see whether it is refused.
func (made maker) build() { frame.New(made.columns, made.rows) }

func negativeFrameIsRefused(t *rapid.T) {
	below := -rapid.IntRange(one, belowMost).Draw(t, "below")
	side := rapid.IntRange(none, sideMost).Draw(t, "side")
	builds := []maker{making(below, side), making(side, below),
		making(below, below)}
	for _, build := range builds {
		if got := panics(build.build); !said(got, "no size below nought") {
			t.Fatalf("a frame %v: %v", build, got)
		}
	}
}

// New pixels hold what they are given. That is the light, the cover and
// the depth.
func TestANewPixelHoldsWhatItIsGiven(t *testing.T) {
	rapid.Check(t, newPixelHoldsWhatItIsGiven)
}

func newPixelHoldsWhatItIsGiven(t *rapid.T) {
	lamp := rapid.Float32Range(none, float32(lampMost))
	red, green, blue := lamp.Draw(t, "red"), lamp.Draw(t, "green"),
		lamp.Draw(t, "blue")
	cover, depth := lamp.Draw(t, "cover"), lamp.Draw(t, "depth")
	made := frame.NewPixel(red, green, blue, cover, depth)
	if made.R != red || made.G != green || made.B != blue ||
		made.Cover != cover || made.Depth != depth {
		t.Fatalf("a pixel of %v %v %v %v %v is %+v", red, green, blue,
			cover, depth, made)
	}
}
