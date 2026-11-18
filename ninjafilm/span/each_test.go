package span

import "testing"

// A span's every whole nanometre runs from its start to its end, both
// in it, in order, and not one more: each of 400 to 403 is four.
func TestEachIsEveryWholeNanometreOfASpanBothEndsIncluded(t *testing.T) {
	stretch := Span{from: 400, to: 403}
	got := stretch.each()
	want := []int{400, 401, 402, 403}
	if len(got) != len(want) {
		t.Fatalf("each of %v is %v, not %v", stretch, got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("each of %v is %v, not %v", stretch, got, want)
			break
		}
	}
	one := Span{from: 550, to: 550}
	if got := one.each(); len(got) != 1 || got[0] != 550 {
		t.Errorf("each of a span of one nanometre is %v", got)
	}
}

// The whole of what is seen is each of it, and its last nanometre is the
// last that is seen.
func TestEachOfWhatIsSeenEndsWhereTheSeeingEnds(t *testing.T) {
	all := seen()
	got := all.each()
	if len(got) != all.to-all.from+1 || got[0] != all.from ||
		got[len(got)-1] != all.to {
		t.Errorf("each of what is seen runs %d to %d in %d places, "+
			"not %d to %d", got[0], got[len(got)-1], len(got), all.from,
			all.to)
	}
}
