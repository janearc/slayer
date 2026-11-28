package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// none is a Lookup that knows nothing.
func none(string) string { return "" }

// only is a Lookup that knows the one name it is given.
func only(name, value string) Lookup {
	return func(asked string) string {
		if asked == name {
			return value
		}
		return ""
	}
}

// fileIn writes text as a settings file in a new folder and returns its path.
func fileIn(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(text), 0o640); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPathIsUnderTheHomeItIsGiven(t *testing.T) {
	if got := Path("base"); got != "base/.config/ferruccio/config" {
		t.Fatalf("got %s", got)
	}
}

func TestAMissingFileIsNoFileAndEverythingIsUnset(t *testing.T) {
	resolved := Resolve(filepath.Join(t.TempDir(), "nowhere", "config"), none, none)
	if resolved.Problem != "" {
		t.Fatalf("a missing file was a problem: %s", resolved.Problem)
	}
	for _, entry := range resolved.Entries {
		if entry.Source != Unset || entry.Value != "" || entry.Origin != "" {
			t.Fatalf("%s is %+v, not unset", entry.Key.Name, entry)
		}
	}
}

func TestEveryNameHasAKeyAndAnEntryInOrder(t *testing.T) {
	resolved := Resolve(filepath.Join(t.TempDir(), "config"), none, none)
	if len(resolved.Entries) != len(Keys) {
		t.Fatalf("%d entries for %d keys", len(resolved.Entries), len(Keys))
	}
	for index, key := range Keys {
		if resolved.Entries[index].Key.Name != key.Name {
			t.Fatalf("entry %d is %s, not %s", index, resolved.Entries[index].Key.Name, key.Name)
		}
		if got, found := KeyOf(key.Name); !found || got.Label != key.Label {
			t.Fatalf("KeyOf(%s) lost it", key.Name)
		}
	}
}

func TestAFlagBeatsTheEnvironmentBeatsTheFileBeatsNothing(t *testing.T) {
	path := fileIn(t, "cache :: /from/the/file\n")
	file := Resolve(path, none, none).Get(Cache)
	if file.Value != "/from/the/file" || file.Source != FromFile || file.Origin != path {
		t.Fatalf("the file's value was %+v", file)
	}
	environment := Resolve(path, none, only("SLAYER_CACHE", "/from/the/environment")).Get(Cache)
	if environment.Value != "/from/the/environment" || environment.Source != FromEnvironment || environment.Origin != "SLAYER_CACHE" {
		t.Fatalf("the environment's value was %+v", environment)
	}
	flag := Resolve(path, only("cache", "/from/the/flag"), only("SLAYER_CACHE", "/from/the/environment")).Get(Cache)
	if flag.Value != "/from/the/flag" || flag.Source != FromFlag || flag.Origin != "--cache" {
		t.Fatalf("the flag's value was %+v", flag)
	}
}

func TestSceneDataCanBeSavedAndOverridden(t *testing.T) {
	path := fileIn(t, "processors 4\n")
	if err := Save(path, Data, "/from/file"); err != nil {
		t.Fatal(err)
	}
	if got := Resolve(path, none, none); got.Problem != "" || got.Value(Data) != "/from/file" {
		t.Fatalf("saved data: %+v", got)
	}
	if got := Resolve(path, only("data", "/from/flag"), only("SLAYER_DATA", "/from/environment")).Get(Data); got.Value != "/from/flag" || got.Source != FromFlag {
		t.Fatalf("data override: %+v", got)
	}
}

func TestWordStyleProcessorsAndSavedQueueAreBothRead(t *testing.T) {
	path := fileIn(t, "processors 4\n")
	if err := Save(path, Queue, "/shared/queue"); err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(path, none, none)
	if resolved.Problem != "" || resolved.Value(Processors) != "4" || resolved.Value(Queue) != "/shared/queue" {
		t.Fatalf("mixed settings: %+v", resolved)
	}
}

func TestOnlyAFlagOrTheEnvironmentLocksAValue(t *testing.T) {
	path := fileIn(t, "cache :: /c\n")
	if Resolve(path, none, none).Get(Cache).Locked() {
		t.Fatal("a value from the file was locked")
	}
	if !Resolve(path, only("cache", "/f"), none).Get(Cache).Locked() {
		t.Fatal("a value from a flag was not locked")
	}
	if !Resolve(path, none, only("SLAYER_CACHE", "/e")).Get(Cache).Locked() {
		t.Fatal("a value from the environment was not locked")
	}
	if Resolve(path, none, none).Get(Shelf).Locked() {
		t.Fatal("an unset value was locked")
	}
}

func TestASettingWithNoVariableIgnoresTheEnvironment(t *testing.T) {
	path := fileIn(t, "renderer :: gpu\n")
	loud := func(string) string { return "ignored" }
	entry := Resolve(path, none, loud).Get(Renderer)
	if entry.Value != "gpu" || entry.Source != FromFile {
		t.Fatalf("the renderer was %+v", entry)
	}
}

func TestEverySettingHasAFlagAndRendererAndProcessorsHaveNoVariable(t *testing.T) {
	for _, key := range Keys {
		if key.Flag == "" {
			t.Fatalf("%s has no flag; every setting is given by flag", key.Name)
		}
	}
	for _, name := range []Name{Renderer, Processors, Slayer, Examples, Theme} {
		if key, _ := KeyOf(name); key.Variable != "" {
			t.Fatalf("%s has the variable %s, which nothing reads", name, key.Variable)
		}
	}
}

func TestAFlagGivesRendererAndProcessorsAndLocksThem(t *testing.T) {
	path := fileIn(t, "renderer :: cpu\nprocessors :: 2\n")
	flag := func(name string) string {
		return map[string]string{"renderer": "gpu", "processors": "all"}[name]
	}
	resolved := Resolve(path, flag, none)
	for name, want := range map[Name]string{Renderer: "gpu", Processors: "all"} {
		entry := resolved.Get(name)
		if entry.Value != want || entry.Source != FromFlag || !entry.Locked() {
			t.Fatalf("%s is %+v", name, entry)
		}
	}
}

func TestDefaultsAreWhatFerruccioDidBeforeAndNoPathOfAnyonesIsOne(t *testing.T) {
	resolved := Resolve(filepath.Join(t.TempDir(), "config"), none, none)
	if resolved.Get(Renderer).Effective() != "cpu" || resolved.Get(Processors).Effective() != "2" {
		t.Fatalf("the defaults are %q and %q", resolved.Get(Renderer).Effective(), resolved.Get(Processors).Effective())
	}
	for _, name := range []Name{Cache, Shelf, Slayer, Examples, Theme} {
		if resolved.Get(name).Effective() != "" {
			t.Fatalf("%s has a default, %q, so it is not unset", name, resolved.Get(name).Effective())
		}
	}
}

func TestAPathWithSpacesAndColonsSurvivesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	odd := "/Volumes/My Drive/a  b :: c/cache"
	if err := Save(path, Cache, odd); err != nil {
		t.Fatal(err)
	}
	if got := Resolve(path, none, none).Value(Cache); got != odd {
		t.Fatalf("got %q, not %q", got, odd)
	}
}

func TestAFileWithAnUnknownKeyIsRefusedWholeAndGivesNoValues(t *testing.T) {
	path := fileIn(t, "cache :: /c\nsurprise :: yes\n")
	resolved := Resolve(path, none, none)
	if !strings.Contains(resolved.Problem, "surprise") {
		t.Fatalf("the problem does not name the key: %q", resolved.Problem)
	}
	if resolved.Value(Cache) != "" {
		t.Fatal("a file that was refused still gave a value")
	}
}

func TestAFlagStillWinsWhenTheFileIsRefused(t *testing.T) {
	path := fileIn(t, "nonsense line here\n")
	resolved := Resolve(path, only("cache", "/f"), none)
	if resolved.Problem == "" || resolved.Value(Cache) != "/f" {
		t.Fatalf("problem %q, cache %q", resolved.Problem, resolved.Value(Cache))
	}
}

func TestSaveRefusesAValueOutsideItsListAndAnUnknownName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := Save(path, Renderer, "quantum"); err == nil {
		t.Fatal("a renderer that does not exist was saved")
	}
	if err := Save(path, Processors, "3"); err == nil {
		t.Fatal("a processor count that is not offered was saved")
	}
	if err := Save(path, Name("nonsense"), "x"); err == nil {
		t.Fatal("a setting that does not exist was saved")
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("a refused save made the file")
	}
}

func TestSaveAcceptsEachOfferedChoiceAndEmptyRemoves(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	for _, value := range []string{"2", "4", "8", "all"} {
		if err := Save(path, Processors, value); err != nil {
			t.Fatalf("%s: %v", value, err)
		}
		if got := Resolve(path, none, none).Value(Processors); got != value {
			t.Fatalf("saved %s, read %s", value, got)
		}
	}
	if err := Save(path, Processors, ""); err != nil {
		t.Fatal(err)
	}
	if got := Resolve(path, none, none).Get(Processors); got.Source != Unset {
		t.Fatalf("a removed value is %+v", got)
	}
}

func TestUpdateKeepsCommentsOrderAndKeysItDoesNotKnow(t *testing.T) {
	original := "# my notes\n\nshelf :: /s\nfuture-key a b c\n# between\ncache :: /old\nlast :: x\n"
	path := fileIn(t, original)
	if err := Update(path, []Change{{Key: "cache", Value: "/new"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	want := "# my notes\n\nshelf :: /s\nfuture-key a b c\n# between\ncache :: /new\nlast :: x\n"
	if string(raw) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", raw, want)
	}
}

func TestUpdateAppendsAKeyItDoesNotHaveAndRemovesOneOnEmpty(t *testing.T) {
	path := fileIn(t, "a :: 1\nb :: 2\n")
	if err := Update(path, []Change{{Key: "c", Value: "3"}, {Key: "a", Value: ""}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "b :: 2\nc :: 3\n" {
		t.Fatalf("got %q", raw)
	}
}

func TestUpdateDropsRepeatsOfAKeyItOwnsAndTheirIndentedLines(t *testing.T) {
	path := fileIn(t, "cache :: /one\n  indented under it\nkeep :: k\ncache :: /two\n")
	if err := Update(path, []Change{{Key: "cache", Value: "/three"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "cache :: /three\nkeep :: k\n" {
		t.Fatalf("got %q", raw)
	}
}

func TestUpdateLeavesAnotherKeysIndentedLinesAlone(t *testing.T) {
	path := fileIn(t, "block one\n  inside :: x\ncache :: /a\n")
	if err := Update(path, []Change{{Key: "cache", Value: "/b"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "block one\n  inside :: x\ncache :: /b\n" {
		t.Fatalf("got %q", raw)
	}
}

func TestUpdateMakesTheFileAndItsFolderWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".config", "ferruccio", "config")
	if err := Update(path, []Change{{Key: "theme", Value: "x"}}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("the new file is %v, %v", info, err)
	}
}

func TestUpdateReplacesByRenameAndLeavesNoStagedFileBehind(t *testing.T) {
	path := fileIn(t, "a :: 1\n")
	before, _ := os.Stat(path)
	if err := Update(path, []Change{{Key: "a", Value: "2"}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("the folder holds %d entries, %v", len(entries), err)
	}
	after, _ := os.Stat(path)
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("the mode went from %v to %v", before.Mode().Perm(), after.Mode().Perm())
	}
	if os.SameFile(before, after) {
		t.Fatal("the file was written in place and not replaced")
	}
}

func TestUpdateRefusesAKeyOrValueThatCannotBeOneLineBeforeTouchingTheFile(t *testing.T) {
	path := fileIn(t, "a :: 1\n")
	for _, bad := range []Change{{Key: "two words", Value: "x"}, {Key: "", Value: "x"}, {Key: "k", Value: "line\nbreak"}} {
		if err := Update(path, []Change{{Key: "a", Value: "2"}, bad}); err == nil {
			t.Fatalf("%+v was accepted", bad)
		}
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "a :: 1\n" {
		t.Fatalf("a refused update changed the file: %q", raw)
	}
}

func TestUpdateTouchesNoOtherFileInTheFolder(t *testing.T) {
	directory := t.TempDir()
	other := filepath.Join(directory, "other")
	if err := os.WriteFile(other, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(filepath.Join(directory, "config"), []Change{{Key: "a", Value: "1"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(other)
	if string(raw) != "untouched" {
		t.Fatal("another file was changed")
	}
}

func TestRenderQueueUsesTheExplicitFlagBeforeTheLauncherEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := Save(path, Queue, "/saved queue"); err != nil {
		t.Fatal(err)
	}
	if got := Resolve(path, none, none).Get(Queue); got.Value != "/saved queue" || got.Source != FromFile {
		t.Fatalf("saved queue: %+v", got)
	}
	env := only("FERRUCCIO_AZAZEL_QUEUE", "/environment queue")
	if got := Resolve(path, none, env).Get(Queue); got.Value != "/environment queue" || got.Source != FromEnvironment {
		t.Fatalf("environment queue: %+v", got)
	}
	flag := only("azazel-queue", "/flag queue")
	if got := Resolve(path, flag, env).Get(Queue); got.Value != "/flag queue" || got.Source != FromFlag {
		t.Fatalf("explicit queue: %+v", got)
	}
}
