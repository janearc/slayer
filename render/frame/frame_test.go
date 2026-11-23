package frame_test

import (
	"math"
	"math/bits"
	"runtime"
	"slices"
	"testing"
	"unsafe"

	"github.com/janearc/slayer/render/frame"
)

// What the tests count in, each defined.
const (
	// none is no amount, and the first column and row. It is defined as zero.
	none = 0
	// one is a single step, a single pixel and the next place after the first.
	// It is defined as one.
	one = 1
	// pair is two, the least factor that makes the largest int wrap. It is
	// defined as two.
	pair = 2
	// endless is the sign math.Inf takes for plus infinity. It is defined by
	// that function.
	endless = 1
	// wrapping is two to the power of half a word's bits, so that two of
	// them multiplied wrap to nothing. It is defined by the arithmetic.
	wrapping = one << (bits.UintSize / pair)
	// wrappingQuarter is two to the power of a word's bits less two, so
	// that four of them multiplied wrap to nothing. It is defined by the
	// arithmetic.
	wrappingQuarter = one << (bits.UintSize - pair)
)

// What a refusal says, as the package words it.
const (
	sizeText    = "frame: a frame is no size below nought"
	outsideText = "frame: a pixel past the frame's edge"
	bigText     = "frame: more pixels than a frame can be sized for"
)

// What the tests choose, each picked in testdata/numbers.yaml.
var (
	wide      = pickedWhole("frame-wide")
	tall      = pickedWhole("frame-tall")
	sideMost  = pickedWhole("side-most")
	pastMost  = pickedWhole("past-most")
	lampRed   = pickedFloat("lamp-red")
	lampGreen = pickedFloat("lamp-green")
	lampBlue  = pickedFloat("lamp-blue")
)

// place is a column and a row.
type place struct{ x, y int }

// side is one edge of a frame.
type side string

// The four edges of a frame.
const (
	west  side = "west"
	east  side = "east"
	north side = "north"
	south side = "south"
)

// sides is each edge of a frame once.
var sides = []side{west, east, north, south}

// past is the place that many steps beyond this edge of the grid, level with
// the along'th row or column. Along is taken modulo the length of the edge. No
// steps is the place on the edge itself, which is inside the grid.
func (edge side) past(grid *frame.Frame, along, steps int) place {
	switch edge {
	case west:
		return place{x: -steps, y: along % grid.H}
	case east:
		return place{x: grid.W - one + steps, y: along % grid.H}
	case north:
		return place{x: along % grid.W, y: -steps}
	}
	return place{x: along % grid.W, y: grid.H - one + steps}
}

// action is a call that a test expects to panic, or not to.
type action interface{ run() }

// reading is a read of the pixel at a place.
type reading struct {
	grid *frame.Frame
	at   place
}

// run reads the place.
func (request reading) run() { request.grid.At(request.at.x, request.at.y) }

// writing is a write of a pixel to a place.
type writing struct {
	grid *frame.Frame
	at   place
	put  frame.Pixel
}

// run writes the pixel to the place.
func (request writing) run() {
	request.grid.Set(request.at.x, request.at.y, request.put)
}

// sizing is a frame asked for by its width and height.
type sizing struct{ width, height int }

// run makes the frame.
func (request sizing) run() { frame.New(request.width, request.height) }

// keep stores what a panic was given, and stops the panic.
func keep(into *any) { *into = recover() }

// panicked is what the action panics with, or nil if it does not.
func panicked(todo action) (caught any) {
	defer keep(&caught)
	todo.run()
	return nil
}

// said is the words of a refusal, or nothing if the panic was not an error.
func said(caught any) string {
	if refusal, ok := caught.(error); ok {
		return refusal.Error()
	}
	return ""
}

// reporter is what the helpers need of a test: a *testing.T or a *rapid.T.
type reporter interface {
	Errorf(format string, args ...any)
}

// refuses reports it if the action does not panic with the words given.
func refuses(test reporter, todo action, text string) {
	if got := said(panicked(todo)); got != text {
		test.Errorf("%T %+v panicked with %q, want %q", todo, todo, got, text)
	}
}

// allows reports it if the action panics.
func allows(test reporter, todo action) {
	if caught := panicked(todo); caught != nil {
		test.Errorf("%T %+v panicked with %v", todo, todo, caught)
	}
}

// same is whether two pixels hold the same bits. NaN is the same as itself
// here, as it is not under ==.
func same(first, second frame.Pixel) bool {
	return math.Float32bits(first.R) == math.Float32bits(second.R) &&
		math.Float32bits(first.G) == math.Float32bits(second.G) &&
		math.Float32bits(first.B) == math.Float32bits(second.B) &&
		math.Float32bits(first.Cover) == math.Float32bits(second.Cover) &&
		math.Float32bits(first.Depth) == math.Float32bits(second.Depth)
}

// snapshot is a copy of the frame's pixels.
func snapshot(grid *frame.Frame) []frame.Pixel { return slices.Clone(grid.Pix) }

// unchanged is whether the frame's pixels are the ones in the snapshot.
func unchanged(grid *frame.Frame, before []frame.Pixel) bool {
	return slices.EqualFunc(before, grid.Pix, same)
}

// rowMajor is where the pixel of a place is in a frame of that width: a row
// of pixels after row, from the top left.
func rowMajor(width int, at place) int { return at.y*width + at.x }

// marked is a pixel with every part set to the amount.
func marked(amount float64) frame.Pixel {
	part := float32(amount)
	return frame.Pixel{R: part, G: part, B: part, Cover: part, Depth: part}
}

// tooBig is the sizes whose pixels cannot be counted in an int, or whose
// memory cannot be. Most wrapped to a small size before they were refused.
func tooBig() []sizing {
	largest := math.MaxInt / int(unsafe.Sizeof(frame.Pixel{}))
	return []sizing{
		{width: wrapping, height: wrapping},
		{width: wrappingQuarter, height: pair * pair},
		{width: math.MaxInt, height: math.MaxInt},
		{width: math.MaxInt, height: pair},
		{width: pair, height: math.MaxInt},
		{width: largest + one, height: one},
		{width: one, height: largest + one},
	}
}

// nothing is the sizes of a frame with no width or no height.
func nothing() []sizing {
	return []sizing{
		{width: none, height: none},
		{width: wide, height: none},
		{width: none, height: tall},
	}
}

// a new frame is every pixel meeting nothing: dark, uncovered and infinitely
// far, w by h of them.
func TestANewFrameHasMetNothing(t *testing.T) {
	grid := frame.New(wide, tall)
	if grid.W != wide || grid.H != tall || len(grid.Pix) != wide*tall {
		t.Fatalf("a %d by %d frame is %d by %d of %d", wide, tall, grid.W,
			grid.H, len(grid.Pix))
	}
	for _, each := range grid.Pix {
		if each.R != none || each.G != none || each.B != none ||
			each.Cover != none || !math.IsInf(float64(each.Depth), endless) {
			t.Errorf("a new pixel is %+v", each)
		}
	}
}

// a pixel set at a column and row is the one read back there: rows count from
// the top left, row by row. The place of each in Pix is worked out here, not
// by the package.
func TestAPixelIsWhereItIsSet(t *testing.T) {
	grid := frame.New(wide, tall)
	right := place{x: wide - one, y: none}
	below := place{x: none, y: one}
	grid.Set(right.x, right.y, frame.Pixel{R: float32(lampRed)})
	grid.Set(below.x, below.y, frame.Pixel{G: float32(lampGreen)})
	if grid.At(right.x, right.y).R != float32(lampRed) ||
		grid.At(below.x, below.y).G != float32(lampGreen) {
		t.Errorf("set and read back: %+v", grid.Pix)
	}
	if grid.Pix[rowMajor(wide, right)].R != float32(lampRed) ||
		grid.Pix[rowMajor(wide, below)].G != float32(lampGreen) {
		t.Errorf("rows are not laid out from the top left: %+v", grid.Pix)
	}
}

// a pixel's light is its red, green and blue, each as it was, widened. The
// amounts are written here, not got from the maths package's constructor.
func TestAPixelsLightIsItsRGB(t *testing.T) {
	lit := frame.Pixel{R: float32(lampRed), G: float32(lampGreen),
		B: float32(lampBlue)}
	got := lit.Light()
	if got.R != lampRed || got.G != lampGreen || got.B != lampBlue {
		t.Errorf("a pixel of %v, %v, %v has light %+v", lampRed, lampGreen,
			lampBlue, got)
	}
}

// a pixel past the frame's edge, on any side and by any distance, is refused
// by name: never read or written as the next row's first. A refused write
// leaves every pixel as it was.
func TestAPixelPastTheEdgeIsRefused(t *testing.T) {
	grid := frame.New(wide, tall)
	before := snapshot(grid)
	for _, edge := range sides {
		for steps := one; steps <= pastMost; steps++ {
			for along := none; along < wide+tall; along++ {
				at := edge.past(grid, along, steps)
				refuses(t, reading{grid: grid, at: at}, outsideText)
				put := marked(lampRed)
				refuses(t, writing{grid: grid, at: at, put: put}, outsideText)
			}
		}
	}
	if !unchanged(grid, before) {
		t.Error("a write past the edge landed in the frame")
	}
}

// a frame of a size below zero is refused by name, whichever side is below.
func TestANegativeFrameIsRefused(t *testing.T) {
	refuses(t, sizing{width: -one, height: tall}, sizeText)
	refuses(t, sizing{width: wide, height: -one}, sizeText)
	refuses(t, sizing{width: -one, height: -one}, sizeText)
	refuses(t, sizing{width: math.MinInt, height: tall}, sizeText)
}

// a frame whose pixels cannot be counted is refused by name. It is not
// made with a small Pix beside a W and an H that say otherwise.
func TestAFrameTooBigIsRefused(t *testing.T) {
	for _, request := range tooBig() {
		refuses(t, request, bigText)
	}
}

// a frame of no width or no height holds no pixels, and every place in it is
// past the edge.
func TestAFrameOfNothingHoldsNothing(t *testing.T) {
	for _, request := range nothing() {
		grid := frame.New(request.width, request.height)
		if len(grid.Pix) != none {
			t.Errorf("a %v frame holds %d pixels", request, len(grid.Pix))
		}
		at := place{x: none, y: none}
		refuses(t, reading{grid: grid, at: at}, outsideText)
		refuses(t, writing{grid: grid, at: at, put: marked(lampRed)},
			outsideText)
	}
}

// a frame built as a literal, with W, H and Pix that agree, is read by row and
// column as a new one is: the pixel at a place is the one at its row-major
// place.
func TestAFrameLiteralIsReadByItsSize(t *testing.T) {
	made := make([]frame.Pixel, wide*tall)
	for index := range made {
		made[index].R = float32(index + one)
	}
	literal := &frame.Frame{W: wide, H: tall, Pix: made}
	for y := none; y < tall; y++ {
		for x := none; x < wide; x++ {
			want := float32(rowMajor(wide, place{x: x, y: y}) + one)
			if got := literal.At(x, y).R; got != want {
				t.Errorf("column %d row %d of a literal is %v, want %v", x, y,
					got, want)
			}
		}
	}
	allows(t, writing{grid: literal, at: place{x: none, y: none},
		put: marked(lampRed)})
}

// a frame literal with fewer pixels than its size says is the builder's
// mistake. A read inside its size that has no pixel panics from the runtime,
// as the doc says, not with the refusal of a place past the edge.
func TestAShortFrameLiteralPanicsFromTheRuntime(t *testing.T) {
	short := make([]frame.Pixel, wide*tall-one)
	literal := &frame.Frame{W: wide, H: tall, Pix: short}
	last := place{x: wide - one, y: tall - one}
	caught := panicked(reading{grid: literal, at: last})
	if _, ok := caught.(runtime.Error); !ok {
		t.Errorf("the last place of a short literal panicked with %v", caught)
	}
}
