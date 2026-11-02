package research_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/research"
)

func TestTheResearchDataIsReadWholeAndAllMeasured(t *testing.T) {
	read, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, number := range read {
		if number.Kind != numbers.Measured || number.From == nil ||
			number.From.Page == 0 {
			t.Errorf("%s is not measured from a page of a source",
				number.Name)
		}
	}
}

func TestANameTheDataDoesNotHoldStopsTheProgram(t *testing.T) {
	defer stopped(t, "guessed-speed")
	research.Value("guessed-speed")
}

// stopped is a test held to a stop that says what it was asked.
func stopped(t *testing.T, saying string) {
	said := recover()
	err, ok := said.(error)
	if !ok || !strings.Contains(err.Error(), saying) {
		t.Errorf("stopped with %v, want %q", said, saying)
	}
}
