package job

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Above names the switches that decide a shortcut, first to last: a light
// shortcut is decided by itself, then fastlight, then vibes; any other by
// itself and vibes.
func TestAboveNamesTheSwitchesThatDecideAShortcut(t *testing.T) {
	for _, each := range []struct {
		name string
		want []string
	}{
		{ShortcutLampStore, []string{"lamp-store", "fastlight", "vibes"}},
		{ShortcutSkyLooks, []string{"sky-looks", "fastlight", "vibes"}},
		{"cascades", []string{"cascades", "vibes"}},
		{FlagFastlight, []string{"fastlight", "vibes"}},
	} {
		if got := Above(each.name); !reflect.DeepEqual(got, each.want) {
			t.Errorf("Above(%q) is %v, want %v", each.name, got, each.want)
		}
	}
}

// A shortcut turned on keeps the shortcuts the job already said, and
// turning one that was said off replaces only its own word.
func TestFlaggedKeepsTheOtherShortcutsTheJobSaid(t *testing.T) {
	job := read(t, still+"shortcuts: {lamp-store: on}\n")
	over, err := job.With(Flagged(ShortcutSkyLooks, true))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{ShortcutLampStore: true,
		ShortcutSkyLooks: true} {
		if on, said := over.Said(name); !said || on != want {
			t.Errorf("%s is on %v and said %v, want on and said", name, on,
				said)
		}
	}
	off, err := over.With(Flagged(ShortcutLampStore, false))
	if err != nil {
		t.Fatal(err)
	}
	if on, said := off.Said(ShortcutLampStore); !said || on {
		t.Errorf("lamp-store is on %v and said %v, want said off", on, said)
	}
	if on, said := off.Said(ShortcutSkyLooks); !said || !on {
		t.Errorf("sky-looks is on %v and said %v, want it left on", on, said)
	}
}

// A path that cannot be looked at, or a file or folder that cannot be
// read, is unreadable, and what is not there is absent.
func TestDigestOfSaysUnreadableForWhatCannotBeRead(t *testing.T) {
	directory := t.TempDir()
	plain := filepath.Join(directory, "plain.txt")
	if err := os.WriteFile(plain, []byte("hello"), private); err != nil {
		t.Fatal(err)
	}
	// a name under a file is not a missing file but a path that cannot be
	// looked at
	if got := digestOf(filepath.Join(plain, "under")); got != unreadableDigest {
		t.Errorf("a name under a file is %q, want %q", got, unreadableDigest)
	}
	shut := filepath.Join(directory, "shut.txt")
	if err := os.WriteFile(shut, []byte("hello"), private); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shut, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(shut, private) })
	if got := digestOf(shut); got != unreadableDigest {
		t.Errorf("a file nobody may read is %q, want %q", got, unreadableDigest)
	}
	closed := filepath.Join(directory, "closed")
	if err := os.Mkdir(closed, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(closed, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(closed, 0o700) })
	if got := digestOf(closed); got != unreadableDigest {
		t.Errorf("a folder nobody may list is %q, want %q", got,
			unreadableDigest)
	}
	// the sha256 of "hello", worked outside the code
	const hello = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if err := os.Chmod(shut, private); err != nil {
		t.Fatal(err)
	}
	if got := digestOf(shut); got != hello {
		t.Errorf("the digest of hello is %q, want %q", got, hello)
	}
}

// The light key leaves out the titles' theme, so a job whose only input
// is a theme has the key of the same job with no titles.
func TestLightKeyIgnoresAJobWhoseOnlyInputIsATheme(t *testing.T) {
	directory := t.TempDir()
	writtenIn(t, directory, "theme.yaml", "theme")
	titled := heading + "scene: {smoke: 1}\ntitles:\n  theme: theme.yaml\n" +
		"  elements:\n    name: {kind: text, words: A, ink: ink, " +
		"across: 0.1, down: 0.1}\n"
	plain := heading + "scene: {smoke: 1}\n"
	withTitles, err := Read(writtenIn(t, directory, "titled.yaml", titled))
	if err != nil {
		t.Fatal(err)
	}
	without, err := Read(writtenIn(t, directory, "plain.yaml", plain))
	if err != nil {
		t.Fatal(err)
	}
	if withTitles.LightKey() != without.LightKey() {
		t.Error("a theme changed the light key")
	}
	if withTitles.Key() == without.Key() {
		t.Error("the titles changed nothing in the full key")
	}
}

// A job holding a number that is not finite cannot be written in its
// canonical form, and is refused by name, as one built from it is.
func TestACanonicalFormRefusesANumberThatIsNotFinite(t *testing.T) {
	wire := document{Schema: schemaName, Scene: sceneWire{Smoke: 1},
		At: &quantity{Value: math.NaN(), Unit: seconds}}
	if raw, err := wire.canonical(); err == nil {
		t.Errorf("a NaN time was written as %s", raw)
	}
	_, err := built(source{name: "nan.yaml"}, &yaml.Node{}, wire)
	if err == nil || !strings.Contains(err.Error(),
		"cannot be written in its canonical form") {
		t.Errorf("built from a NaN gave %v, want a refusal naming the "+
			"canonical form", err)
	}
}

// A file digest of something that opens but cannot be read through, as a
// folder cannot, is unreadable.
func TestFileDigestOfAFolderIsUnreadable(t *testing.T) {
	if got := fileDigest(t.TempDir()); got != unreadableDigest {
		t.Errorf("the file digest of a folder is %q, want %q", got,
			unreadableDigest)
	}
}
