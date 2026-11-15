package job

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// anti-test of TestAStretchedFilmSpansItsSceneDuration
// TestAFilmWithNoStretchRunsAtItsOwnPace holds that the same film with
// no scene-duration is at its own seconds, and that a stretch shorter than
// the film runs the scene slower than the film.
func TestAFilmWithNoStretchRunsAtItsOwnPace(t *testing.T) {
	base := still + "fps: 6\nduration: {value: 30, unit: s}\n" +
		"start: {value: 45, unit: s}\n"
	times := read(t, base).Times()
	if got := fmt.Sprint(len(times), times[len(times)/halves]); got != "180 1m0s" {
		t.Errorf("an unstretched film is %s, want 180 1m0s", got)
	}
	times = read(t, base+"scene-duration: {value: 15, unit: s}\n").Times()
	if got := fmt.Sprint(len(times), times[len(times)/halves]); got != "180 52.5s" {
		t.Errorf("a film stretched to 15s is %s, want 180 52.5s", got)
	}
}

// anti-test of TestSchemaIsACopy
// TestSchemaStaysTheSchemaWhateverACallerDoes holds that after a caller
// writes over what Schema gave, the next call still decodes and still
// names a schema.
func TestSchemaStaysTheSchemaWhateverACallerDoes(t *testing.T) {
	clear(Schema())
	var decoded map[string]any
	if err := json.Unmarshal(Schema(), &decoded); err != nil {
		t.Fatalf("the schema after a caller cleared a copy: %v", err)
	}
	if decoded["title"] != "slayer job" {
		t.Errorf("the schema is titled %v", decoded["title"])
	}
}

// anti-test of TestSwitchesAreTheSchemasAndEachSaysOneLine
// TestAnythingElseIsNotASwitch holds that a name the schema has no
// switch for is not said, and not taken as an override.
func TestAnythingElseIsNotASwitch(t *testing.T) {
	job := read(t, still+"vibes: on\n")
	for _, name := range []string{"cascades", "", "Vibes", "lamp_store"} {
		if on, said := job.Said(name); on || said {
			t.Errorf("%q is said: %v %v", name, on, said)
		}
		if _, err := job.With(Flagged(name, true)); err == nil {
			t.Errorf("an override of the switch %q was taken", name)
		}
		if slices.ContainsFunc(Switches(), func(each Switch) bool {
			return each.Name() == name
		}) {
			t.Errorf("%q is listed among the switches", name)
		}
	}
}

// anti-test of TestAnUnsaidPartIsEmpty
// TestASaidPartIsNotEmpty holds that a title and a sound a job says are
// said, by the same accessors that give nothing for an empty one.
func TestASaidPartIsNotEmpty(t *testing.T) {
	job := read(t, titled)
	words, _ := job.Titles()
	element := words.Elements()[nothing]
	if _, has := element.From(); !has {
		t.Error("a title that says from does not")
	}
	if _, has := element.Until(); !has {
		t.Error("a title that says until does not")
	}
	tuned := read(t, film+"sound: {tune: '@moon', voicing: '@plain'}\n")
	track, _ := tuned.Sound()
	if _, _, has := track.Tune(); !has {
		t.Error("a sound that says a tune does not")
	}
	filed := read(t, film+"sound: {file: a.wav, from: {value: 3, unit: s}}\n")
	track, _ = filed.Sound()
	if _, has := track.File(); !has {
		t.Error("a sound that says a file does not")
	}
	if from, has := track.From(); !has || from == nothing {
		t.Error("a sound that says from does not")
	}
}

// anti-test of TestCanonicalIsSortedJSONWithoutOutputs
// TestCanonicalHoldsWhatChangesThePicture holds that the canonical form
// says the values that change the picture and the sound, and that two
// jobs that differ in the order of their titles are two forms.
func TestCanonicalHoldsWhatChangesThePicture(t *testing.T) {
	text := string(read(t, whole).Canonical())
	for _, present := range []string{`"glimpses":2`, `"seed":7`,
		`"rays":2`, `"quality":"preview"`, `"renderer":"gpu"`,
		`"elements"`, `"rows":24`, "moonlake.yaml", "take.wav"} {
		if !strings.Contains(text, present) {
			t.Errorf("the canonical form does not say %s", present)
		}
	}
	other := strings.Replace(whole, "first: {kind: text, words: A",
		"zzz: {kind: text, words: A", one)
	if string(read(t, other).Canonical()) == text {
		t.Error("two jobs whose titles are named alike differ not")
	}
}

// anti-test of TestThePresetsAreOneTable
// TestAReadmeTableThatDriftsIsNotTheTable holds the README's row pattern
// to counting only a preset's row: a row dropped, a heading and a row
// with no size are left out, and the names then differ from the shelf.
func TestAReadmeTableThatDriftsIsNotTheTable(t *testing.T) {
	text := "| Preset | Size | Rate |\n|---|---|---|\n" +
		"| `portrait` | 1080 by 1920 | |\n" +
		"| `broken` | no size | |\n" +
		"| `4k` | 3840 by 2160 | |\n"
	tabled := tabledPresets(text)
	if !slices.Equal(tabled, []string{"portrait", "4k"}) {
		t.Errorf("the pattern took %v", tabled)
	}
	var shelved []string
	for _, each := range Presets() {
		shelved = append(shelved, each.Name())
	}
	if slices.Equal(shelved, tabled) {
		t.Error("a table of two rows is taken for the whole shelf")
	}
}
