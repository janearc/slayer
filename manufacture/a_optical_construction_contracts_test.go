package manufacture

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// A small segmented enclosure catches a broken panel walk before detailed
// catalogue fixtures can spend the suite timeout expanding that walk.
func TestOpticalHullFinishesWithItsThreeMaterials(t *testing.T) {
	row, err := Observatory(opticalSpec())
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		count int
		err   error
	}
	done := make(chan result, 1)
	go func() {
		made, err := MakeWithComplexity(row, Sow(5), "hull")
		if err != nil {
			done <- result{err: err}
			return
		}
		things, err := made.Things(maths.Subspace{})
		done <- result{count: len(things), err: err}
	}()
	select {
	case got := <-done:
		if got.err != nil || got.count != 3 {
			t.Fatalf("enclosure materials %d: %v", got.count, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("small enclosure construction did not finish within one second")
	}
}
