package texture

import (
	"testing"
	"time"
)

// the black body table is worked out once, on the first call, a step of fifty
// kelvin from zero to sixty thousand: 1201 entries, built in a tenth of a
// second.
//
// a table of the wrong size would take minutes, and every test that asks for a
// colour would wait on it. so this one comes first (the file name sorts before
// the tests that draw a disc) and fails in five seconds.
func TestTheThermalTableIsBuiltInNoTime(t *testing.T) {
	done := make(chan struct{})
	go func() {
		thermal(5000)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the black body table was not built in five seconds")
	}
	if got := len(thermals); got != 1201 {
		t.Errorf("the table is %d entries, not 1201", got)
	}
}
