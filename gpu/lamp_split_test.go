package gpu

import (
	"fmt"
	"strings"
	"testing"
)

// TestTheKernelIsToldTheSplit says that the numbers the kernel is given
// carry the split's two: the most nodes of the lamp tree a step is cut
// into, and the square of how wide across a node may be for its distance.
func TestTheKernelIsToldTheSplit(t *testing.T) {
	given := numbers(1, 1)
	for _, line := range []string{
		fmt.Sprintf("lamp_split_most = %d;", lampSplitMost),
		fmt.Sprintf("lamp_split_square = %v;",
			lampSplitSpread*lampSplitSpread),
	} {
		if !strings.Contains(given, line) {
			t.Errorf("the numbers do not say %q", line)
		}
	}
}

// TestEveryStepThroughAMediumAsksBySplitting is the anti-test of the walk
// from the root at a step: both places the kernel lights a bit of a medium,
// a mist's and a gas's, ask through asked_marched, and the one place that
// still asks lamps_marched of asked is asked_marched's own, for a world
// with no tree.
func TestEveryStepThroughAMediumAsksBySplitting(t *testing.T) {
	const marched = "= asked_marched(world, point, asks);"
	if got := strings.Count(traceSource, marched); got != 2 {
		t.Errorf("%d steps ask by splitting, want a mist's and a gas's", got)
	}
	const walked = "lamps_marched, asks)"
	if got := strings.Count(traceSource, walked); got != 1 {
		t.Errorf("%d asks of lamps_marched by walking, want the one in "+
			"asked_marched", got)
	}
}
