package frame_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"pgregory.net/rapid"
)

// anti-test: a new pixel is not a zero-value pixel. One has depth zero, a hit
// at the camera. A new one has met nothing. A frame built as a literal is
// all zero-value pixels, which is why a new frame is not the same as one.
func TestANewPixelIsNotAZeroValuePixel(t *testing.T) {
	var zero frame.Pixel
	grid := frame.New(wide, tall)
	for _, each := range grid.Pix {
		if each == zero {
			t.Errorf("a new pixel is the zero value %+v", each)
		}
	}
	literal := frame.Frame{W: wide, H: tall, Pix: make([]frame.Pixel,
		wide*tall)}
	if got := literal.At(none, none).Depth; math.IsInf(float64(got), endless) {
		t.Errorf("a literal frame's pixel has depth %v, not zero", got)
	}
}

// anti-test: a pixel set in one place is not found in any other. Every other
// place of the frame is still a pixel that met nothing.
func TestAPixelSetHereIsNotFoundElsewhere(t *testing.T) {
	grid := frame.New(wide, tall)
	here := place{x: one, y: one}
	grid.Set(here.x, here.y, marked(lampRed))
	for y := none; y < tall; y++ {
		for x := none; x < wide; x++ {
			got := grid.At(x, y)
			missed := math.IsInf(float64(got.Depth), endless)
			if (place{x: x, y: y}) != here && !missed {
				t.Errorf("the pixel at column %d row %d is %+v", x, y, got)
			}
		}
	}
}

// anti-test: a pixel's light is not its cover or its depth. A pixel with only
// those has no light.
func TestAPixelsLightIsNotItsCoverOrDepth(t *testing.T) {
	covered := frame.Pixel{Cover: float32(lampRed), Depth: float32(lampGreen)}
	if got := covered.Light(); got.R != none || got.G != none ||
		got.B != none {
		t.Errorf("the light of %+v is %+v", covered, got)
	}
}

// anti-test: a place on the edge of the frame is inside it. It is read and
// written, not refused, on each side.
func TestAPixelOnTheEdgeIsNotRefused(t *testing.T) {
	grid := frame.New(wide, tall)
	for _, edge := range sides {
		for along := none; along < wide+tall; along++ {
			at := edge.past(grid, along, none)
			allows(t, reading{grid: grid, at: at})
			allows(t, writing{grid: grid, at: at, put: marked(lampRed)})
		}
	}
}

// anti-test: a size of zero is not a size below zero. A frame of no width or
// no height is made, with the width and height it was asked for.
func TestAFrameOfNothingIsNotRefused(t *testing.T) {
	for _, request := range nothing() {
		allows(t, request)
		grid := frame.New(request.width, request.height)
		if grid.W != request.width || grid.H != request.height {
			t.Errorf("a %v frame is %d by %d", request, grid.W, grid.H)
		}
	}
}

// anti-test: a frame that can be counted is not refused as too big, even with
// the largest width beside no height. The two refusals are not each other: a
// size below zero is not too big, and one too big is not below zero.
func TestAFrameThatCountsIsNotRefusedAsTooBig(t *testing.T) {
	allows(t, sizing{width: wide, height: tall})
	allows(t, sizing{width: math.MaxInt, height: none})
	allows(t, sizing{width: none, height: math.MaxInt})
	if got := said(panicked(sizing{width: -one, height: one})); got == bigText {
		t.Errorf("a negative size was refused as %q", got)
	}
	for _, request := range tooBig() {
		if got := said(panicked(request)); got == sizeText {
			t.Errorf("%v was refused as %q", request, got)
		}
	}
}

// anti-test: a frame of one pixel is not a frame of nothing. Its one place is
// read, and the places beside it are past the edge.
func TestAFrameOfOnePixelIsNotAFrameOfNothing(t *testing.T) {
	grid := frame.New(one, one)
	allows(t, reading{grid: grid, at: place{x: none, y: none}})
	refuses(t, reading{grid: grid, at: place{x: one, y: none}}, outsideText)
	refuses(t, reading{grid: grid, at: place{x: none, y: one}}, outsideText)
}

// anti-test: a frame literal is read by its W, not by the length of its Pix.
// The same pixels in a frame a column wider give another pixel at one place.
func TestAFrameLiteralIsNotReadByItsLength(t *testing.T) {
	made := make([]frame.Pixel, (wide+one)*tall)
	for index := range made {
		made[index].R = float32(index + one)
	}
	narrow := frame.Frame{W: wide, H: tall, Pix: made}
	broad := frame.Frame{W: wide + one, H: tall, Pix: made}
	if narrow.At(none, one) == broad.At(none, one) {
		t.Errorf("frames of widths %d and %d read the same pixel at row one",
			wide, wide+one)
	}
}

// anti-test: a short frame literal is not refused for a place past its edge by
// the runtime. A place past the edge of its size is refused by name, and the
// first place, which it does hold, is read.
func TestAShortFrameLiteralIsStillRefusedPastItsEdge(t *testing.T) {
	short := make([]frame.Pixel, wide*tall-one)
	literal := &frame.Frame{W: wide, H: tall, Pix: short}
	refuses(t, reading{grid: literal, at: place{x: wide, y: none}}, outsideText)
	allows(t, reading{grid: literal, at: place{x: none, y: none}})
}

// anti-test: a frame is not left as a new one once a pixel is set. The place
// is no longer one that met nothing.
func TestAFrameWithAPixelSetHasMetSomething(t *testing.T) {
	rapid.Check(t, setPixelHasMetSomething)
}

// setPixelHasMetSomething is one draw of the property.
func setPixelHasMetSomething(t *rapid.T) {
	grid := drawnFrame(t)
	at := drawnPlace(t, grid)
	grid.Set(at.x, at.y, marked(lampRed))
	if got := grid.At(at.x, at.y); math.IsInf(float64(got.Depth), endless) {
		t.Fatalf("the pixel at %v is still a miss after it was set: %+v", at,
			got)
	}
}

// anti-test: a pixel set at a place is not read beside it. The places next to
// it, where the frame has them, are as they were.
func TestAPixelSetHereIsNotReadBesideIt(t *testing.T) {
	rapid.Check(t, setPixelIsNotReadBeside)
}

// setPixelIsNotReadBeside is one draw of the property.
func setPixelIsNotReadBeside(t *rapid.T) {
	grid := drawnFrame(t)
	at := drawnPlace(t, grid)
	put := marked(lampRed)
	grid.Set(at.x, at.y, put)
	for _, beside := range at.neighbours() {
		if beside.within(grid) && same(grid.At(beside.x, beside.y), put) {
			t.Fatalf("a pixel set at %v was read at %v", at, beside)
		}
	}
}

// neighbours is the four places a step from this one, whether or not a frame
// has them.
func (at place) neighbours() []place {
	return []place{
		{x: at.x - one, y: at.y},
		{x: at.x + one, y: at.y},
		{x: at.x, y: at.y - one},
		{x: at.x, y: at.y + one},
	}
}

// within is whether the grid has this place.
func (at place) within(grid *frame.Frame) bool {
	return at.x >= none && at.x < grid.W && at.y >= none && at.y < grid.H
}

// anti-test: a place on the edge of any frame is not refused, on any side.
func TestAPlaceOnAnEdgeIsNotRefused(t *testing.T) {
	rapid.Check(t, edgeIsNotRefused)
}

// edgeIsNotRefused is one draw of the property.
func edgeIsNotRefused(t *rapid.T) {
	grid := drawnFrame(t)
	edge := drawnEdge(t)
	along := drawnAlong(t)
	at := edge.past(grid, along, none)
	allows(t, reading{grid: grid, at: at})
	allows(t, writing{grid: grid, at: at, put: drawnPixel(t)})
}

// anti-test: goroutines that write different pixels leave no pixel as a new
// frame made it, and no two with the same number. A writer that missed a pixel
// or wrote another's place would leave one or the other.
func TestAfterTheWritersNoPixelIsLeftAsNewMadeIt(t *testing.T) {
	grid := shared()
	seen := map[float32]place{}
	for y := none; y < grid.H; y++ {
		for x := none; x < grid.W; x++ {
			at := place{x: x, y: y}
			got := grid.At(x, y)
			if math.IsInf(float64(got.Depth), endless) {
				t.Errorf("the pixel at %v was never written", at)
			}
			if first, twice := seen[got.R]; twice {
				t.Errorf("the pixels at %v and %v hold one number", first, at)
			}
			seen[got.R] = at
		}
	}
}
