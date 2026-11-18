package research

import (
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// TestObserveSeesEachReadUntilStopped checks that an observer is told the
// number Value read, that a second observer takes its place, and that
// stopping each puts back the one before it.
func TestObserveSeesEachReadUntilStopped(t *testing.T) {
	const name = "hubble-primary-diameter-metres"
	want := Value(name)
	var first, second []numbers.Number
	stopFirst := Observe(func(number numbers.Number) { first = append(first, number) })
	Value(name)
	stopSecond := Observe(func(number numbers.Number) { second = append(second, number) })
	Value(name)
	stopSecond()
	Value(name)
	stopFirst()
	if got := Value(name); got != want {
		t.Errorf("observing changed the value: %v, want %v", got, want)
	}
	if len(first) != 2 || len(second) != 1 {
		t.Fatalf("the first observer saw %d reads and the second %d, want 2 and 1", len(first), len(second))
	}
	seen := second[0]
	if seen.Name != name || seen.Value != want || seen.Kind != numbers.Measured || seen.From == nil {
		t.Errorf("the observer was told %+v", seen)
	}
	if observer.Load() != nil {
		t.Error("an observer is still set after both were stopped")
	}
}
