package session

import (
	"testing"

	"github.com/janearc/slayer/maths/strategies/scrooge"
)

// TestAnyTestStartsWithScroogeUnasked checks what TestMain does, whatever
// the environment the run was started in: scrooge is not asked until a test
// asks for it.
func TestAnyTestStartsWithScroogeUnasked(t *testing.T) {
	if scrooge.Asked() {
		t.Error("scrooge is asked at the start of a test")
	}
}

// TestATestThatAsksForScroogeHasItAndGivesItBack is the anti-test: asking
// with t.Setenv turns it on for that test, and the test after it finds it
// off again.
func TestATestThatAsksForScroogeHasItAndGivesItBack(t *testing.T) {
	t.Run("asked", func(t *testing.T) {
		t.Setenv(scrooge.Variable, "1")
		if !scrooge.Asked() {
			t.Error("scrooge is not asked after the test asked for it")
		}
	})
	if scrooge.Asked() {
		t.Error("scrooge is still asked after the test that asked for it")
	}
}
