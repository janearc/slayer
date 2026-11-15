//go:build draw

package cells_test

import (
	"image"
	"testing"

	"pgregory.net/rapid"
)

// Any frame at any size is the size asked for, and is what the tracer
// draws developed.
func TestAnyFrameIsTheSizeAskedAndWhatTheTracerDraws(t *testing.T) {
	rapid.Check(t, frameIsAsAsked)
}

func frameIsAsAsked(test *rapid.T) {
	number := rapid.IntRange(nothing, picked("number-most")).Draw(test,
		"number")
	wide := rapid.IntRange(once, picked("wide-most")).Draw(test, "wide")
	high := rapid.IntRange(once, picked("high-most")).Draw(test, "high")
	drawn, err := placed(test, nil).Frame(number, wide, high)
	if err != nil {
		test.Fatal(err)
	}
	got, ok := drawn.(*image.RGBA)
	if !ok || got.Bounds().Dx() != wide || got.Bounds().Dy() != high {
		test.Fatalf("frame %d at %dx%d came back as %T %v", number, wide,
			high, drawn, drawn.Bounds())
	}
	want := directly(test, number, wide, high)
	if string(got.Pix) != string(want.Pix) {
		test.Fatalf("frame %d at %dx%d is not what the tracer draws",
			number, wide, high)
	}
}
