package cells_test

import (
	"image"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// Frame zero is at the start. A second of frames on is a second later, for any
// start, rate and frame.
func TestAnyPaceRunsInStepWithTheSecond(t *testing.T) {
	rapid.Check(t, paceRunsInStep)
}

func paceRunsInStep(test *rapid.T) {
	start := rapid.IntRange(nothing, picked("start-most-ms")).Draw(test,
		"start")
	rate := rapid.IntRange(once, picked("rate-most")).Draw(test, "rate")
	number := rapid.IntRange(nothing, picked("number-most")).Draw(test,
		"number")
	pace := paceOf(test, start, rate)
	if first := pace.At(nothing); first != time.Duration(start)*time.Millisecond {
		test.Fatalf("frame nought is at %v, not the start %dms", first,
			start)
	}
	if step := pace.At(number+rate) - pace.At(number); step != time.Second {
		test.Fatalf("%d frames at %d a second are %v, not a second", rate,
			rate, step)
	}
}

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
