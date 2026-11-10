package research_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/research"
	"github.com/janearc/slayer/sources"
)

func TestTheResearchDataIsReadWholeAndNamesItsSources(t *testing.T) {
	read, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	list, err := sources.Read("../sources.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, number := range read {
		if got := research.Value(number.Name); got != number.Value {
			t.Fatalf("cached %s: %g, parsed %g", number.Name, got, number.Value)
		}
		if number.Kind != numbers.Measured || number.From == nil {
			t.Fatalf("%s has no measured source", number.Name)
		}
		source, ok := list.Find(number.From.Key)
		if !ok || !source.Pinned() || source.Never() != "" {
			t.Fatalf("%s source is not pinned", number.Name)
		}
		from, err := url.Parse(source.From())
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(from.Path, ".pdf") {
			if number.From.Page <= 0 {
				t.Errorf("%s has no PDF page", number.Name)
			}
		} else if number.From.Page != 0 || number.From.Excerpt == "" {
			t.Errorf("%s must locate its unpaged source with an excerpt", number.Name)
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

func TestCallerChangesToReadCannotAlterTheRetainedValues(t *testing.T) {
	read, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	name, value := read[0].Name, read[0].Value
	if got := research.Value(name); got != value {
		t.Fatal("wrong retained value")
	}
	read[0].Value = -999
	read[0].From.Text = "caller's annotation"
	later, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	if research.Value(name) != value || later[0].Value != value || later[0].From.Text == read[0].From.Text {
		t.Fatal("a caller changed the private catalogue or a later read")
	}
}
