package job

import (
	"strings"
	"testing"
)

// helpLine is the most characters a help line has: Puzzlemouth's
// verb.Help, which the door hands each switch's Line to. Defined there.
const helpLine = 70

// TestAShortcutIsDecidedByTheFirstThatSaysIt reads the lamp store
// under every way of saying it: its own word, fastlight, vibes, and
// nothing, which is off.
func TestAShortcutIsDecidedByTheFirstThatSaysIt(t *testing.T) {
	for said, want := range map[string]bool{
		"":                            false,
		"vibes: on\n":                 true,
		"vibes: off\n":                false,
		"fastlight: on\n":             true,
		"vibes: on\nfastlight: off\n": false,
		"vibes: off\nfastlight: on\n": true,
		"fastlight: on\nshortcuts: {lamp-store: off}\n": false,
		"vibes: on\nshortcuts: {lamp-store: off}\n":     false,
		"shortcuts: {lamp-store: on}\n":                 true,
		"fastlight: off\nshortcuts: {lamp-store: on}\n": true,
	} {
		job := read(t, still+said)
		if job.Shortcut(ShortcutLampStore) != want {
			t.Errorf("%q: lamp-store %v", said, !want)
		}
	}
	apart := read(t, still+"fastlight: on\nshortcuts: {sky-looks: off}\n")
	if !apart.Shortcut(ShortcutLampStore) ||
		apart.Shortcut(ShortcutSkyLooks) {
		t.Error("one light shortcut's own word decided the other")
	}
}

// TestSaidIsTheJobsOwnWordAndNothingElse reads what a job says of each
// name, without working anything out, and a name the schema has not.
func TestSaidIsTheJobsOwnWordAndNothingElse(t *testing.T) {
	job := read(t, still+"vibes: on\nshortcuts: {lamp-store: off}\n")
	for name, want := range map[string][2]bool{
		FlagVibes: {true, true}, FlagFastlight: {false, false},
		ShortcutLampStore: {false, true}, "cascades": {}} {
		on, said := job.Said(name)
		if got := [2]bool{on, said}; got != want {
			t.Errorf("%s: on and said %v, want %v", name, got, want)
		}
	}
}

// TestTheFlagsKeysChangeOnlyWithThePicture holds the keys: a shortcut
// said as the job's own fastlight says it is the job without it; every
// other word is another job, off among them, since a set may say on.
func TestTheFlagsKeysChangeOnlyWithThePicture(t *testing.T) {
	for _, pair := range [][2]string{
		{still + "fastlight: on\n",
			still + "fastlight: on\nshortcuts: {lamp-store: on}\n"},
		{still + "fastlight: off\n",
			still + "fastlight: off\nshortcuts: {lamp-store: off}\n"},
	} {
		if keyOf(t, pair[nothing]) != keyOf(t, pair[one]) {
			t.Errorf("%q and %q have different keys", pair[nothing],
				pair[one])
		}
	}
	// anti-test of TestTheFlagsKeysChangeOnlyWithThePicture
	keys := map[string]string{}
	for _, said := range []string{"", "vibes: on\n", "vibes: off\n",
		"fastlight: on\n", "fastlight: off\n",
		"shortcuts: {lamp-store: on}\n", "shortcuts: {lamp-store: off}\n",
		"fastlight: on\nshortcuts: {lamp-store: off}\n"} {
		key := keyOf(t, still+said)
		if other, taken := keys[key]; taken {
			t.Errorf("%q and %q have one key", said, other)
		}
		keys[key] = said
	}
}

// TestSwitchesAreTheSchemasAndEachSaysOneLine reads the flags and the
// shortcuts out of the schema: the two flags first, every shortcut
// after, each with one line that says what it is refused with.
func TestSwitchesAreTheSchemasAndEachSaysOneLine(t *testing.T) {
	made := Switches()
	var names []string
	for _, each := range made {
		names = append(names, each.Name())
		if each.Says() == "" || strings.Contains(each.Says(), "\n") ||
			!strings.Contains(each.Says(), "Refused") {
			t.Errorf("%s says %q: one line, with what it is refused with",
				each.Name(), each.Says())
		}
		if line := each.Line(); len(line) > helpLine || len(line) < 2 ||
			!strings.HasPrefix(each.Says(), line) {
			t.Errorf("%s's help line is %q: the first sentence, of %d "+
				"characters or fewer", each.Name(), line, helpLine)
		}
		if each.Shortcut() == (each.Name() == FlagVibes ||
			each.Name() == FlagFastlight) {
			t.Errorf("%s is a shortcut: %v", each.Name(), each.Shortcut())
		}
	}
	want := []string{FlagVibes, FlagFastlight, ShortcutLampStore,
		ShortcutSkyLooks}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("the switches are %v, want %v", names, want)
	}
}

// TestFlaggedStandsOverTheJob lays each kind of switch over a job,
// leaves the job it was laid over alone, and refuses a name the schema
// has not.
func TestFlaggedStandsOverTheJob(t *testing.T) {
	job := read(t, still+"vibes: on\n")
	over, err := job.With(Flagged(FlagVibes, false),
		Flagged(FlagFastlight, true), Flagged(ShortcutLampStore, false))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][2]bool{
		FlagVibes: {false, true}, FlagFastlight: {true, true},
		ShortcutLampStore: {false, true}} {
		on, said := over.Said(name)
		if got := [2]bool{on, said}; got != want {
			t.Errorf("%s over the job: on and said %v, want %v", name, got,
				want)
		}
	}
	if _, said := job.Said(ShortcutLampStore); said {
		t.Error("an override changed the job it was laid over")
	}
	// anti-test of TestFlaggedStandsOverTheJob
	for words, override := range map[string]Override{
		"cascades":  Flagged("cascades", true),
		" is empty": Flagged("", true),
	} {
		if _, err := job.With(override); err == nil {
			t.Errorf("an override that should say %q was taken", words)
		} else {
			saying(t, err, words)
		}
	}
}
