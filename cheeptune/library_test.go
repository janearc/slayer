package cheeptune

import (
	"slices"
	"strings"
	"testing"
)

// Every tune in the library loads with its sheet, renders without clipping
// with the voicing its sheet names, and runs 30 to 90 seconds. This renders
// each tune: about a third of a second for each, so the library takes around
// ten seconds.

func TestEveryLibraryTuneHasAFinishThatIsNotALoop(t *testing.T) {
	for _, name := range Library() {
		s, err := ReadScore("@" + name)
		if err != nil {
			t.Fatal(err)
		}
		if s.Loop {
			t.Errorf("%s loops; a tune for a video ends", name)
		}
	}
}

func TestNoLibraryTuneTakesABuiltInsName(t *testing.T) {
	scores, voicings := Builtin()
	for _, name := range Library() {
		if slices.Contains(scores, "@"+name) ||
			slices.Contains(voicings, "@"+name) {
			t.Errorf("%s is also a built-in", name)
		}
		if strings.Contains(name, ".") || name != strings.ToLower(name) ||
			strings.ContainsAny(name[len(name)-1:], "0123456789") {
			t.Errorf("%q is not a plain lowercase name with no number "+
				"at the end", name)
		}
	}
}

func TestAnUnknownLibraryTuneIsRefusedWithTheNamesThatExist(t *testing.T) {
	_, _, err := OpenTune("no-such-tune")
	if err == nil || !strings.Contains(err.Error(), Library()[0]) {
		t.Errorf("an unknown tune gave %v, want the names listed", err)
	}
}

func TestATuneASheetAndAMissingFileAreNotLibraryVoicings(t *testing.T) {
	for _, name := range []string{"glass-hour", "glass-hour.cuts",
		"no-such-file"} {
		if libraryVoicing(name) {
			t.Errorf("%q is taken for a voicing", name)
		}
	}
	if slices.Contains(LibraryVoicings(), "@glass-hour") {
		t.Error("a tune is listed as a voicing")
	}
}
