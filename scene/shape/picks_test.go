package shape

import (
	"math"
	"testing"
)

// chosenInPicks are the names the code asks picks.yaml for.
var chosenInPicks = []string{"normal-step", "orthonormal-tolerance",
	"leaf-members", "tube-most-sides", "tube-chord-share",
	"sphere-unit-slack", "sphere-cancellation-limit"}

// TestEveryNumberTheCodeChoosesIsInItsBook says that each name the code asks
// for is in picks.yaml, with a value that is a finite number above nought.
func TestEveryNumberTheCodeChoosesIsInItsBook(t *testing.T) {
	for _, name := range chosenInPicks {
		number, err := picks.Get(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !(number.Value > nought) || math.IsInf(number.Value, unity) {
			t.Errorf("%s: value %v, want a finite number above nought", name,
				number.Value)
		}
	}
}

// TestABookWithoutTheNameRefuses is the anti-test of the one above: a name
// that is not in the book is refused.
func TestABookWithoutTheNameRefuses(t *testing.T) {
	if _, err := picks.Get("no-such-number"); err == nil {
		t.Error("the book gave a number it does not hold")
	}
}

// TestTheTestBookIsRead says that numbers.yaml in testdata is read by
// picked, so the helpers the tests share work.
func TestTheTestBookIsRead(t *testing.T) {
	if got := picked(t, "fixtures-probe"); got != 1e8 {
		t.Errorf("probe %v, want the number in the book", got)
	}
}
