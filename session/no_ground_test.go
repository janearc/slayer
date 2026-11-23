package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// the band a refused stand asked for in these tests, in metres, and the
// slope one of them says: picked, a band and a slope no test fixture has
const (
	refusedAbove, refusedBelow = 9000.0, 9001.0
	refusedSteepest            = 0.8
)

// oneStand is the one stand of a set's text, or the test's end.
func oneStand(t *testing.T, text string) set.Stand {
	t.Helper()
	read, err := readSetText(text)
	if err != nil {
		t.Fatal(err)
	}
	stands := read.Stands()
	if len(stands) != 1 {
		t.Fatalf("%d stands, want the one", len(stands))
	}
	return stands[0]
}

// A stand that says no steepest slope allows any, and its refusal names
// the heights it asked for and no slope.
func TestAStandThatSaysNoSlopeIsRefusedByItsHeightsAlone(t *testing.T) {
	stand := oneStand(t, standSetOf("yurt", true, ""))
	got := noGround(stand, refusedAbove, refusedBelow).Error()
	want := fmt.Sprintf("no ground between 9000 and 9001 m in %d tries",
		tries)
	if got != want {
		t.Errorf("the refusal says %q, want %q", got, want)
	}
}

// Anti-test: a stand that says a steepest slope is refused by its heights
// and that slope, so the words without a slope are not the only ones.
func TestAStandThatSaysASlopeIsRefusedByItsHeightsAndTheSlope(t *testing.T) {
	stand := oneStand(t, standSetOf("yurt", true,
		fmt.Sprintf("        steepest: %v\n", refusedSteepest)))
	got := noGround(stand, refusedAbove, refusedBelow).Error()
	if !strings.Contains(got, "between 9000 and 9001 m and no steeper "+
		"than 0.8 in ") {
		t.Errorf("the refusal says %q, with no word of the slope", got)
	}
}
