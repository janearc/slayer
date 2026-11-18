package hillside

import (
	"testing"
	"time"
)

// generationNow and generationLater are two frame times for the
// generation rules.
const (
	generationNow   = 277 * time.Second
	generationLater = generationNow + time.Second/6
)

// TestAGenerationIsNewWhenTheWorldCanBe: the first frame, timed or not,
// and a timed scene's frame at a new time take a new generation.
func TestAGenerationIsNewWhenTheWorldCanBe(t *testing.T) {
	if !worldChanged(true, false, 0, generationNow) {
		t.Error("the first frame kept a generation it never had")
	}
	if !worldChanged(true, true, 0, generationNow) {
		t.Error("a timed first frame kept a generation it never had")
	}
	if !worldChanged(false, true, generationNow, generationLater) {
		t.Error("a timed scene at a new time kept its generation")
	}
}

// TestAGenerationIsKeptWhenOnlyTheCameraMoved is the anti-test: a timed
// scene drawn again at its last time, and a scene that is not timed at
// any time, keep their generation, so the drawer can draw a camera move.
func TestAGenerationIsKeptWhenOnlyTheCameraMoved(t *testing.T) {
	if worldChanged(false, true, generationNow, generationNow) {
		t.Error("a timed scene at its last time took a new generation")
	}
	if worldChanged(false, false, generationNow, generationLater) {
		t.Error("a scene that is not timed took a new generation")
	}
}
