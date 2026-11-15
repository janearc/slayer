//go:build draw

package gpu

import "testing"

// the looks are shared only where they are dealt evenly: a pixel's
// samples divide sixteen, or sixteen divides them; at three or five along
// a side each sample takes all sixteen
func TestTheSkysLooksAreSharedOnlyWhereTheyDivideEvenly(t *testing.T) {
	for side, want := range map[int]bool{1: true, 2: true, 4: true,
		3: false, 5: false, 8: true, 16: true, 6: false, 0: false} {
		if got := sharesEvenly(side); got != want {
			t.Errorf("side %d shares evenly: %v, want %v", side, got, want)
		}
	}
}
