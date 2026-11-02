package frame_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/ninjafilm/frame"
	"pgregory.net/rapid"
)

// anti-tests: a frame is not left as a new one once a pixel is set. The
// pixel is no longer one that met nothing.
func TestAFrameWithAPixelSetHasMetSomething(t *testing.T) {
	rapid.Check(t, setPixelHasMetSomething)
}

func setPixelHasMetSomething(t *rapid.T) {
	columns := rapid.IntRange(one, sideMost).Draw(t, "columns")
	rows := rapid.IntRange(one, sideMost).Draw(t, "rows")
	grid := frame.New(columns, rows)
	x := rapid.IntRange(none, columns-one).Draw(t, "x")
	y := rapid.IntRange(none, rows-one).Draw(t, "y")
	put := frame.NewPixel(one, one, one, one, one)
	grid.Set(x, y, put)
	if got := grid.At(x, y); math.IsInf(float64(got.Depth), endless) ||
		got != put {
		t.Fatalf("pixel %d,%d is still %+v after it was set", x, y, got)
	}
}

// anti-tests: a pixel set in one place is not read in another. Every
// other place is as it was.
func TestAPixelSetHereIsNotReadThere(t *testing.T) {
	rapid.Check(t, pixelIsNotReadElsewhere)
}

func pixelIsNotReadElsewhere(t *rapid.T) {
	columns := rapid.IntRange(one+one, sideMost).Draw(t, "columns")
	rows := rapid.IntRange(one, sideMost).Draw(t, "rows")
	grid := frame.New(columns, rows)
	x := rapid.IntRange(none, columns-one-one).Draw(t, "x")
	y := rapid.IntRange(none, rows-one).Draw(t, "y")
	put := frame.NewPixel(one, one, one, one, one)
	grid.Set(x, y, put)
	if grid.At(x+one, y) == put {
		t.Fatalf("a pixel set at %d,%d was read at %d,%d", x, y, x+one, y)
	}
}

// anti-tests: a pixel's light is not its cover or its depth. Pixels with
// only those have no light.
func TestAPixelsLightIsNotItsCoverOrDepth(t *testing.T) {
	rapid.Check(t, lightIgnoresCoverAndDepth)
}

func lightIgnoresCoverAndDepth(t *rapid.T) {
	lamp := rapid.Float64Range(one, lampMost)
	cover, depth := lamp.Draw(t, "cover"), lamp.Draw(t, "depth")
	made := frame.NewPixel(none, none, none, float32(cover), float32(depth))
	if got := made.Light(); got.R != none || got.G != none ||
		got.B != none {
		t.Fatalf("the light of %+v is %v", made, got)
	}
}

// anti-tests: a pixel does not swap its lamps. Red stays red. Green
// stays green. Blue stays blue. Cover is not depth.
func TestANewPixelDoesNotSwapItsParts(t *testing.T) {
	rapid.Check(t, newPixelKeepsPartsApart)
}

func newPixelKeepsPartsApart(t *rapid.T) {
	lamp := rapid.Float64Range(none, lampMost)
	red, green := float32(lamp.Draw(t, "red")), float32(lamp.Draw(t, "green"))
	blue, cover := float32(lamp.Draw(t, "blue")), float32(lamp.Draw(t, "cover"))
	depth := float32(lamp.Draw(t, "depth"))
	made := frame.NewPixel(red, green, blue, cover, depth)
	if made.R != red || made.G != green || made.B != blue ||
		made.Cover != cover || made.Depth != depth {
		t.Fatalf("%v %v %v %v %v came back as %+v", red, green, blue, cover,
			depth, made)
	}
}
