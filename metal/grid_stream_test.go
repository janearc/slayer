//go:build draw && darwin

package metal

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// floats is the little-endian bytes of float32 values, for an input.
func floats(values ...float32) []byte {
	var out []byte
	for _, value := range values {
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(value))
	}
	return out
}

// TestAStreamRedrawsWithOnlyTheChangedInput opens a stream on the where
// kernel, draws, changes the one input, and draws again: the second
// picture shows the new value at every pixel, with nothing re-added.
func TestAStreamRedrawsWithOnlyTheChangedInput(t *testing.T) {
	kernel, err := Open(where, "where")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	width, height, rows := 13, 10, 4
	stream, err := kernel.OpenGrid(width, height, rows, 4,
		[][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	for _, given := range []float32{7, 11} {
		if given != 7 {
			if err := stream.Update(0, floats(given)); err != nil {
				t.Fatal(err)
			}
		}
		out, err := stream.Draw()
		if err != nil {
			t.Fatal(err)
		}
		for row := range height {
			for column := range width {
				pixel := [4]float32(out[(row*width+column)*4:][:4])
				want := expected(column, row, rows)
				want[3] = given
				if pixel != want {
					t.Fatalf("with %v, pixel %d,%d is %v, want %v",
						given, column, row, pixel, want)
				}
			}
		}
	}
}

// TestAStreamGrowsAnInputWhenItsLengthChanges updates an input to a
// longer slice, which must land in a fresh buffer, and draws.
func TestAStreamGrowsAnInputWhenItsLengthChanges(t *testing.T) {
	kernel, err := Open(where, "where")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	stream, err := kernel.OpenGrid(3, 2, 2, 4, [][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if err := stream.Update(0, floats(3, 9)); err != nil {
		t.Fatal(err)
	}
	out, err := stream.Draw()
	if err != nil {
		t.Fatal(err)
	}
	if out[3] != 3 {
		t.Fatalf("the grown input reads %v at its first value, want 3",
			out[3])
	}
}

// TestAStreamRefusesWhatItCannotDo covers the refusals: an input that
// is not there, a closed stream, and sizes with nothing in them.
func TestAStreamRefusesWhatItCannotDo(t *testing.T) {
	kernel, err := Open(where, "where")
	if errors.Is(err, ErrNoDevice) {
		t.Skip("no gpu here")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer kernel.Close()
	if _, err := kernel.OpenGrid(0, 2, 2, 4, nil); err == nil {
		t.Error("a stream with no width was not refused")
	}
	if _, err := kernel.OpenGrid(2, 2, 2, 0, nil); err == nil {
		t.Error("a stream with no numbers a pixel was not refused")
	}
	stream, err := kernel.OpenGrid(3, 2, 2, 4, [][]byte{floats(7)})
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Update(1, floats(1)); err == nil {
		t.Error("updating an input that is not there was not refused")
	}
	stream.Close()
	stream.Close() // twice is harmless
	if err := stream.Update(0, floats(1)); err == nil {
		t.Error("updating a closed stream was not refused")
	}
	if _, err := stream.Draw(); err == nil {
		t.Error("drawing a closed stream was not refused")
	}
}
