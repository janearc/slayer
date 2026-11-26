package manufacture

import (
	"testing"

	"github.com/janearc/slayer/bioforms/skeleton"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals"
)

func TestMammalAxialHoldPreservesOrderAndEmptyResults(t *testing.T) {
	spans := []mammals.Span{{Bone: 4, Middle: 3, Run: 2}, {Bone: 7, Middle: 1, Run: 2}}
	for _, tc := range []struct {
		x       float64
		bones   []int
		weights []float64
	}{
		{4, []int{4}, []float64{1}},
		{2, []int{4, 7}, []float64{0.6666666666666667, 0.6666666666666667}},
		{0, []int{7}, []float64{1}},
	} {
		got := mammals.AxialHold(spans, tc.x, 1.5, nil)
		if len(got) != len(tc.bones) {
			t.Fatalf("x %g: %v", tc.x, got)
		}
		for i, influence := range got {
			if influence.Bone() != tc.bones[i] || influence.Weight() != tc.weights[i] {
				t.Fatalf("x %g influence %d: %v", tc.x, i, influence)
			}
		}
	}
	gap := []mammals.Span{{Bone: 1, Middle: 10, Run: 1}, {Bone: 2, Middle: 0, Run: 1}}
	if got := mammals.AxialHold(gap, 5, 1.5, nil); got != nil {
		t.Fatal("seal-style gap lost nil")
	}
	if got := mammals.AxialHold(gap, 5, 1.5, []skeleton.Influence{}); got == nil || len(got) != 0 {
		t.Fatal("orca-style gap lost non-nil empty result")
	}
}
