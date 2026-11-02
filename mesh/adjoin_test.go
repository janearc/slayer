package mesh

import (
	"testing"
	"time"
)

// within fails the test if f has not finished in so long, and leaves
// it running: a root that never ends must fail a test, not hold the rest.
func within(t *testing.T, wait time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(wait):
		t.Fatalf("still going after %v", wait)
	}
}

// faces joined are one piece with one root, whichever way round they
// were joined and through whom, and faces not joined each have their
// own. this is early so that a root which never ends is caught first.
func TestJoinedFacesShareARoot(t *testing.T) {
	within(t, 2*time.Second, func() {
		joined := newJoins(6)
		joined.join(0, 1)
		joined.join(3, 2)
		joined.join(4, 3)
		for _, same := range [][2]int{{0, 1}, {2, 3}, {3, 4}, {2, 4}} {
			if joined.root(same[0]) != joined.root(same[1]) {
				t.Errorf("faces %d and %d are not joined", same[0], same[1])
			}
		}
		for _, apart := range [][2]int{{0, 2}, {1, 4}, {0, 5}, {5, 3}} {
			if joined.root(apart[0]) == joined.root(apart[1]) {
				t.Errorf("faces %d and %d are joined", apart[0], apart[1])
			}
		}
		joined.join(1, 4)
		if joined.root(0) != joined.root(2) {
			t.Error("joining two pieces left them two")
		}
	})
}
