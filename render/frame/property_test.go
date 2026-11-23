package frame_test

import (
	"math"
	"testing"

	"github.com/janearc/slayer/render/frame"
	"pgregory.net/rapid"
)

// drawnFrame is a new frame of a drawn width and height, each at least one.
func drawnFrame(t *rapid.T) *frame.Frame {
	columns := rapid.IntRange(one, sideMost).Draw(t, "columns")
	rows := rapid.IntRange(one, sideMost).Draw(t, "rows")
	return frame.New(columns, rows)
}

// drawnPlace is a drawn place inside the grid.
func drawnPlace(t *rapid.T, grid *frame.Frame) place {
	x := rapid.IntRange(none, grid.W-one).Draw(t, "x")
	y := rapid.IntRange(none, grid.H-one).Draw(t, "y")
	return place{x: x, y: y}
}

// drawnPixel is a pixel whose every part is drawn as a whole word of bits, so
// that NaN, infinities, negatives and amounts past one are among them.
func drawnPixel(t *rapid.T) frame.Pixel {
	word := rapid.Uint32()
	return frame.Pixel{
		R:     math.Float32frombits(word.Draw(t, "red")),
		G:     math.Float32frombits(word.Draw(t, "green")),
		B:     math.Float32frombits(word.Draw(t, "blue")),
		Cover: math.Float32frombits(word.Draw(t, "cover")),
		Depth: math.Float32frombits(word.Draw(t, "depth")),
	}
}

// drawnEdge is a drawn edge of a frame.
func drawnEdge(t *rapid.T) side {
	return rapid.SampledFrom(sides).Draw(t, "edge")
}

// drawnAlong is a drawn row or column to be level with, for past.
func drawnAlong(t *rapid.T) int {
	return rapid.IntRange(none, sideMost).Draw(t, "along")
}

// a pixel set at a place is read back there bit for bit, whatever it holds:
// the frame stores what it is given.
func TestAPixelSetIsReadBack(t *testing.T) {
	rapid.Check(t, setIsReadBack)
}

// setIsReadBack is one draw of the property.
func setIsReadBack(t *rapid.T) {
	grid := drawnFrame(t)
	at := drawnPlace(t, grid)
	put := drawnPixel(t)
	grid.Set(at.x, at.y, put)
	if got := grid.At(at.x, at.y); !same(got, put) {
		t.Fatalf("%+v set at %v was read as %+v", put, at, got)
	}
}

// a pixel set at one place changes no other pixel of the frame.
func TestASetChangesNoNeighbour(t *testing.T) {
	rapid.Check(t, setChangesOnlyItsPlace)
}

// setChangesOnlyItsPlace is one draw of the property.
func setChangesOnlyItsPlace(t *rapid.T) {
	grid := drawnFrame(t)
	at := drawnPlace(t, grid)
	before := snapshot(grid)
	put := drawnPixel(t)
	grid.Set(at.x, at.y, put)
	for index, each := range grid.Pix {
		if index != rowMajor(grid.W, at) && !same(each, before[index]) {
			t.Fatalf("a set at %v changed the pixel at index %d", at, index)
		}
	}
}

// a place past any edge of any frame, by any distance, is refused on a read
// and on a write, and a refused write changes nothing.
func TestAPlacePastAnEdgeIsRefused(t *testing.T) {
	rapid.Check(t, pastIsRefused)
}

// pastIsRefused is one draw of the property.
func pastIsRefused(t *rapid.T) {
	grid := drawnFrame(t)
	before := snapshot(grid)
	steps := rapid.IntRange(one, pastMost).Draw(t, "steps")
	edge := drawnEdge(t)
	along := drawnAlong(t)
	put := drawnPixel(t)
	at := edge.past(grid, along, steps)
	refuses(t, reading{grid: grid, at: at}, outsideText)
	refuses(t, writing{grid: grid, at: at, put: put}, outsideText)
	if !unchanged(grid, before) {
		t.Fatalf("a refused write at %v changed the frame", at)
	}
}
