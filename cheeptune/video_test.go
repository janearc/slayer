package cheeptune

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// VIDEO.md teaches a model to write a tune for a video. These tests keep it
// true: the two worked examples it quotes render with the voicing their
// sheets name and pass their sheets, and the page names every field a sheet
// holds.

// workedExamples are the two tunes the page quotes whole, as the score's
// path and its sheet's.
var workedExamples = [][2]string{
	{"examples/ember-rising.json", "examples/ember-rising.cuts.json"},
	{"library/glass-hour.json", "library/glass-hour.cuts.json"},
}

func TestVideoMDQuotesEachWorkedExampleWhole(t *testing.T) {
	b, err := os.ReadFile("VIDEO.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, ex := range workedExamples {
		for _, path := range ex {
			file, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b),
				strings.TrimRight(string(file), "\n")) {
				t.Errorf("VIDEO.md does not quote %s whole", path)
			}
		}
	}
}

func TestVideoMDNamesEveryFieldOfASheet(t *testing.T) {
	b, err := os.ReadFile("VIDEO.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(CutSheet{}),
		reflect.TypeOf(Cut{})} {
		for i := range typ.NumField() {
			tag, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
			if !strings.Contains(string(b), "`"+tag+"`") {
				t.Errorf("VIDEO.md does not name the field %q", tag)
			}
		}
	}
}

func TestTheDraftOfEmberRisingHasItsRestBeatBeforeTheDrop(t *testing.T) {
	s, err := ReadScore("examples/ember-rising.json")
	if err != nil {
		t.Fatal(err)
	}
	lead := s.Parts[0].Bars
	if !strings.HasSuffix(lead[9], "-:1") {
		t.Errorf("bar 10 of the lead is %q, want it to end in a beat of "+
			"rest", lead[9])
	}
	if strings.HasSuffix(lead[8], "-:1") {
		t.Errorf("bar 9 of the lead rests, and only bar 10 should")
	}
}
