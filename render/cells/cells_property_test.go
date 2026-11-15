package cells_test

import (
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
