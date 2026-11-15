//go:build draw

package cheeptune

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// lengthFewest and lengthMost are the length a tune for a video has, in
// seconds. Defined 2026-11-12: 30 to 90 seconds.
const (
	lengthFewest = 30.0
	lengthMost   = 90.0
)

// identFewest and identMost are the length of an ident, the few seconds
// at the head of every short, in seconds. Defined by the composer's
// brief, 2026-11-12: "two to four seconds".
const (
	identFewest = 2.0
	identMost   = 4.0
)

// shortForms are the library's tunes that are idents and not tunes for
// the length of a video. A tune is one or the other by name here, not by
// how long it happens to be, so a tune that came out short is still
// refused.
var shortForms = map[string]bool{
	"zappy-ident":         true,
	"zappy-ident-falling": true,
}

func TestAShortTuneThatIsNotAnIdentIsNotExcused(t *testing.T) {
	for _, name := range Library() {
		if shortForms[name] {
			continue
		}
		_, sheet, err := OpenTune(name)
		if err == nil && sheet.Seconds < lengthFewest {
			t.Errorf("%s is %.1f s and is not listed as an ident", name,
				sheet.Seconds)
		}
	}
	for name := range shortForms {
		if !slices.Contains(Library(), name) {
			t.Errorf("%s is listed as an ident and is not in the library",
				name)
		}
	}
}

func TestEveryLibraryTuneLoadsRendersAndIsTheRightLength(t *testing.T) {
	names := Library()
	if len(names) == 0 {
		t.Fatal("the library is empty")
	}
	for _, name := range names {
		p, sheet, err := OpenTune(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		fewest, most := lengthFewest, lengthMost
		if shortForms[name] {
			fewest, most = identFewest, identMost
		}
		if sheet.Seconds < fewest || sheet.Seconds > most {
			t.Errorf("%s is %.1f s; it should be %v to %v", name,
				sheet.Seconds, fewest, most)
		}
		if p.Peak >= 1 {
			t.Errorf("%s clips: peak %v", name, p.Peak)
		}
		if p.Score.Name != name {
			t.Errorf("%s is named %q inside", name, p.Score.Name)
		}
	}
}

func TestALibraryTuneIsNamedWithAnAtLikeABuiltIn(t *testing.T) {
	name := Library()[0]
	_, sheet, err := OpenTune(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Open("@"+name, sheet.Voicing); err != nil {
		t.Errorf("@%s does not open: %v", name, err)
	}
	if _, err := Open("@no-such-tune", sheet.Voicing); err == nil ||
		!strings.Contains(err.Error(), "no built-in") {
		t.Errorf("an unknown name gave %v", err)
	}
}

func TestASheetWhoseVoicingIsNotThereNamesTheTuneAndTheVoicing(t *testing.T) {
	name := Library()[0]
	read := func(doc string) ([]byte, error) {
		b, err := libraryDoc(doc)
		if err == nil && strings.HasSuffix(doc, cutsSuffix) {
			var sheet CutSheet
			if err := decodeStrict(b, &sheet); err != nil {
				return nil, err
			}
			sheet.Voicing = "@no-such-voicing"
			return json.Marshal(sheet)
		}
		return b, err
	}
	_, _, err := openTune(name, read)
	if err == nil || !strings.Contains(err.Error(), name) ||
		!strings.Contains(err.Error(), "@no-such-voicing") {
		t.Errorf("a missing voicing gave %v, want the tune and the voicing",
			err)
	}
	// anti: the same tune read as it is opens.
	if _, _, err := openTune(name, libraryDoc); err != nil {
		t.Errorf("%s: %v", name, err)
	}
}

func TestALibraryVoicingIsNamedWithAnAtAndIsNotATune(t *testing.T) {
	voicings := LibraryVoicings()
	if !slices.Contains(voicings, "@acid-draft") {
		t.Fatalf("the library's voicings are %v, want @acid-draft", voicings)
	}
	for _, name := range voicings {
		if slices.Contains(Library(), strings.TrimPrefix(name, "@")) {
			t.Errorf("%s is a voicing and is listed as a tune", name)
		}
		if _, err := Open("@glass-acid", name); err != nil &&
			!strings.Contains(err.Error(), "clips") {
			t.Errorf("%s does not open as a voicing: %v", name, err)
		}
	}
}
