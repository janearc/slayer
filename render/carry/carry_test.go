package carry_test

import (
	"math"
	"testing"

	ninjaframe "github.com/janearc/slayer/ninjafilm/frame"
	"github.com/janearc/slayer/render/carry"
	"github.com/janearc/slayer/render/frame"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	nothing = 0 // defined: no pixels, no size, no length
	one     = 1 // defined: a size of one pixel
)

// aFrame is a frame of slayer's with any size up to the most picked and
// any bits in its pixels, odd ones included.
func aFrame(test *rapid.T) *frame.Frame {
	most := pickedWhole("side-most")
	width := rapid.IntRange(nothing, most).Draw(test, "width")
	height := rapid.IntRange(nothing, most).Draw(test, "height")
	drawn := frame.New(width, height)
	part := rapid.Float32()
	for index := range drawn.Pix {
		drawn.Pix[index] = frame.Pixel{R: part.Draw(test, "r"),
			G: part.Draw(test, "g"), B: part.Draw(test, "b"),
			Cover: part.Draw(test, "cover"),
			Depth: part.Draw(test, "depth")}
	}
	return drawn
}

// same is whether two pixels hold the same bits, so that a NaN is the
// same as itself.
func same(first, second [5]float32) bool {
	for index := range first {
		if math.Float32bits(first[index]) != math.Float32bits(second[index]) {
			return false
		}
	}
	return true
}

// parts are a slayer pixel's five numbers.
func parts(pixel frame.Pixel) [5]float32 {
	return [5]float32{pixel.R, pixel.G, pixel.B, pixel.Cover, pixel.Depth}
}

// ninjaParts are a ninjafilm pixel's five numbers.
func ninjaParts(pixel ninjaframe.Pixel) [5]float32 {
	return [5]float32{pixel.R, pixel.G, pixel.B, pixel.Cover, pixel.Depth}
}

// Frames carried to ninjafilm's and back are the same frame, size and
// every bit of every pixel.
func TestAFrameCarriedThereAndBackIsTheSameFrame(test *testing.T) {
	rapid.Check(test, carriedThereAndBack)
}

func carriedThereAndBack(test *rapid.T) {
	drawn := aFrame(test)
	back := carry.Slayer(carry.Ninja(drawn))
	if back.W != drawn.W || back.H != drawn.H || len(back.Pix) != len(drawn.Pix) {
		test.Fatalf("a %d by %d frame came back %d by %d", drawn.W, drawn.H,
			back.W, back.H)
	}
	for index := range drawn.Pix {
		if !same(parts(drawn.Pix[index]), parts(back.Pix[index])) {
			test.Fatalf("pixel %d changed: %v became %v", index,
				drawn.Pix[index], back.Pix[index])
		}
	}
}

// Frames made by ninjafilm's roll and lens are carried the other way
// with every pixel the same.
func TestAFrameOfNinjafilmsIsCarriedPixelForPixel(test *testing.T) {
	rapid.Check(test, carriedFromNinja)
}

func carriedFromNinja(test *rapid.T) {
	exposed := carry.Ninja(aFrame(test))
	back := carry.Slayer(exposed)
	for index := range exposed.Pix {
		if !same(ninjaParts(exposed.Pix[index]),
			parts(back.Pix[index])) {
			test.Fatalf("pixel %d changed", index)
		}
	}
}

// Pixels and lists of them are carried in order.
func TestPixelsAreCarriedInOrder(test *testing.T) {
	rapid.Check(test, pixelsInOrder)
}

func pixelsInOrder(test *rapid.T) {
	drawn := aFrame(test)
	carried := carry.NinjaPixels(drawn.Pix)
	if len(carried) != len(drawn.Pix) {
		test.Fatalf("%d pixels became %d", len(drawn.Pix), len(carried))
	}
	for index := range carried {
		if !same(parts(drawn.Pix[index]), ninjaParts(carried[index])) {
			test.Fatalf("pixel %d is out of place", index)
		}
	}
}

// A list of frames is carried frame for frame, in order.
func TestFramesAreCarriedInOrder(test *testing.T) {
	rapid.Check(test, framesInOrder)
}

func framesInOrder(test *rapid.T) {
	many := rapid.IntRange(nothing, pickedWhole("frames-most")).Draw(test, "many")
	var drawn []*frame.Frame
	for range many {
		drawn = append(drawn, aFrame(test))
	}
	carried := carry.NinjaFrames(drawn)
	if len(carried) != len(drawn) {
		test.Fatalf("%d frames became %d", len(drawn), len(carried))
	}
	for index := range carried {
		if carried[index].W != drawn[index].W ||
			carried[index].H != drawn[index].H {
			test.Fatalf("frame %d is out of place", index)
		}
	}
}

// Anti-tests: a carried frame shares nothing with the frame it came from.
// Changing the one does not change the other.
func TestACarriedFrameSharesNothingWithItsOriginal(test *testing.T) {
	drawn := frame.New(one, one)
	drawn.Pix[nothing] = frame.Pixel{R: pickedFloat("light")}
	carried := carry.Ninja(drawn)
	carried.Pix[nothing].R = nothing
	if drawn.Pix[nothing].R != pickedFloat("light") {
		test.Error("changing the carried frame changed the original")
	}
	back := carry.Slayer(carried)
	back.Pix[nothing].R = pickedFloat("light")
	if carried.Pix[nothing].R != nothing {
		test.Error("changing the frame carried back changed the carried one")
	}
}

// Anti-tests: no frame is carried to no frame, not to an empty one.
func TestNoFrameIsCarriedToNoFrame(test *testing.T) {
	if carry.Ninja(nil) != nil {
		test.Error("no frame became a frame of ninjafilm's")
	}
	if carry.Slayer(nil) != nil {
		test.Error("no frame became a frame of slayer's")
	}
	if got := carry.NinjaFrames(nil); len(got) != nothing {
		test.Errorf("no frames became %d", len(got))
	}
	if got := carry.NinjaPixels(nil); len(got) != nothing {
		test.Errorf("no pixels became %d", len(got))
	}
}

// Anti-tests: a frame with no pixels is carried as one with no pixels,
// and keeps the size it had.
func TestAnEmptyFrameStaysEmptyAtItsSize(test *testing.T) {
	wide := frame.New(pickedWhole("side-most"), nothing)
	carried := carry.Ninja(wide)
	if carried.W != wide.W || carried.H != nothing || len(carried.Pix) != nothing {
		test.Errorf("an empty frame came across as %d by %d with %d pixels",
			carried.W, carried.H, len(carried.Pix))
	}
}

// Anti-tests: a pixel that met nothing is still infinitely far after the
// crossing, not zero.
func TestAPixelThatMetNothingStaysInfinitelyFar(test *testing.T) {
	far := frame.New(one, one).Pix[nothing]
	if !math.IsInf(float64(far.Depth), one) {
		test.Fatal("a new frame's pixel is not infinitely far")
	}
	if got := carry.NinjaPixel(far).Depth; !math.IsInf(float64(got), one) {
		test.Errorf("depth crossed as %v", got)
	}
	if got := carry.SlayerPixel(carry.NinjaPixel(far)).Depth; !math.IsInf(float64(got), one) {
		test.Errorf("depth came back as %v", got)
	}
}
