package job

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// keySet is the three keys of a job: the whole, the picture's and the
// light's.
type keySet struct{ whole, picture, light string }

// moved is which of the three keys differ between two jobs.
type moved struct{ whole, picture, light bool }

// movedFrom is which of the keys of got are not those of base.
func movedFrom(base, got keySet) moved {
	return moved{got.whole != base.whole, got.picture != base.picture,
		got.light != base.light}
}

// keysOf is the three keys of the job text says, written in the
// directory all of a test's jobs are, so relative paths are one file.
func keysOf(t *testing.T, text string) keySet {
	t.Helper()
	hashedName := hashed(t.Name())
	directory := filepath.Join(os.TempDir(), "job-keys-"+
		hashedName)
	if err := os.MkdirAll(directory, private|executable); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	job, err := Read(writtenIn(t, directory, "job.yaml", text))
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return keySet{job.Key(), job.PictureKey(), job.LightKey()}
}

// hashed is a name made safe for a directory.
func hashed(name string) string {
	safe := []byte(name)
	for i, each := range safe {
		if each == '/' || each == ' ' {
			safe[i] = '-'
		}
	}
	return string(safe)
}

// anti-test of TestTheKeyIgnoresWhatDoesNotChangeThePicture
// anti-test of TestTheKeyFillsInDefaults
// anti-test of TestTheLightKeyCoversTheSetAndNotTheDevelopment
// TestEveryKnobOfTheJobMovesTheKeysItShould changes one value of the
// whole job at a time and holds which of the three keys moves: a change
// to the light moves all three; a change to how the light is developed
// leaves the light's key; a change to the sound moves the whole key only.
func TestEveryKnobOfTheJobMovesTheKeysItShould(t *testing.T) {
	base := keysOf(t, whole)
	forLight := moved{true, true, true}
	forDevelopment := moved{true, true, false}
	forSound := moved{true, false, false}
	for name, each := range map[string]struct {
		before, after string
		want          moved
	}{
		"width":     {"width: 640", "width: 641", forLight},
		"height":    {"height: 360", "height: 361", forLight},
		"wide":      {"wide: 50", "wide: 51", forLight},
		"wide/tall": {"wide: 50", "tall: 50", forLight},
		"exposure":  {"exposure: 0.5", "exposure: 0.6", forLight},
		"duration": {"duration: {value: 2, unit: s}",
			"duration: {value: 3, unit: s}", forLight},
		"stretch": {"scene-duration: {value: 8", "scene-duration: {value: 9",
			forLight},
		"shutter angle": {"angle: {value: 180", "angle: {value: 90",
			forLight},
		"zoom duration": {"unit: deg}, duration: {value: 8",
			"unit: deg}, duration: {value: 9", forLight},
		"zoom ease": {"unit: deg}, duration: {value: 8, unit: s}}",
			"unit: deg}, duration: {value: 8, unit: s}, ease: smooth}",
			forLight},
		"renderer unsaid": {"renderer: gpu\n", "", forLight},
		"vibes on": {"renderer: gpu\n", "renderer: gpu\nvibes: on\n",
			forLight},
		"vibes off": {"renderer: gpu\n", "renderer: gpu\nvibes: off\n",
			forLight},
		"fastlight": {"renderer: gpu\n", "renderer: gpu\nfastlight: on\n",
			forLight},
		"a shortcut": {"renderer: gpu\n",
			"renderer: gpu\nshortcuts: {sky-looks: on, lamp-store: on}\n",
			forLight},
		"stock name":  {"name: dusk", "name: ash", forDevelopment},
		"stock rated": {"rated: 1", "rated: 2", forDevelopment},
		"stock gone": {"stock: {name: dusk, seed: 7, rated: 1}\n", "",
			forDevelopment},
		"print knee": {"stock: {name: dusk, seed: 7, rated: 1}",
			"print: {knee: 0.5, top: 8}", forDevelopment},
		"title rows":   {"rows: 24", "rows: 25", forDevelopment},
		"title across": {"across: 0.1", "across: 0.11", forDevelopment},
		"title down":   {"down: 0.2", "down: 0.3", forDevelopment},
		"title ink": {"words: A, ink: ink", "words: A, ink: dim",
			forDevelopment},
		"title theme": {"titles:\n  rows: 24",
			"titles:\n  theme: t.yaml\n  rows: 24", forDevelopment},
		"sound from": {"from: {value: 1, unit: s}}",
			"from: {value: 2, unit: s}}", forSound},
		"sound score": {"sound: {file: take.wav,",
			"sound: {file: take.wav, score: s.yaml,", forSound},
		"sound gone": {"sound: {file: take.wav, from: {value: 1, unit: s}}\n",
			"", forSound},
	} {
		edited := changed(t, each.before, each.after)
		got := movedFrom(base, keysOf(t, edited))
		if got != each.want {
			t.Errorf("%s: moved %+v, want %+v", name, got, each.want)
		}
	}
}

// soundJob is a film of a set with an audio file and a score for its
// beats, both beside the job.
const soundJob = heading + "scene: {set: lake.yaml}\nfps: 24\n" +
	"duration: {value: 1, unit: s}\n" +
	"sound: {file: take.wav, score: beats.yaml}\n"

// anti-test of TestTheKeyCoversTheFilesTheJobReads
// TestTheKeyCoversTheSoundFileAndItsBeats holds that changing the bytes
// of the audio file or of its score moves the whole key, never the
// picture's or the light's, and that a file that was not there and is
// is another key.
func TestTheKeyCoversTheSoundFileAndItsBeats(t *testing.T) {
	directory := t.TempDir()
	writtenIn(t, directory, "lake.yaml", "lake")
	made, err := Read(writtenIn(t, directory, "job.yaml", soundJob))
	if err != nil {
		t.Fatal(err)
	}
	absent := keySet{made.Key(), made.PictureKey(), made.LightKey()}
	for _, name := range []string{"take.wav", "beats.yaml"} {
		writtenIn(t, directory, name, name)
	}
	present := keySet{reread(t, made).Key(), reread(t, made).PictureKey(),
		reread(t, made).LightKey()}
	if got := movedFrom(absent, present); got != (moved{true, false, false}) {
		t.Errorf("sound files appearing moved %+v", got)
	}
	for _, name := range []string{"take.wav", "beats.yaml"} {
		writtenIn(t, directory, name, "changed")
		again := reread(t, made)
		got := movedFrom(present, keySet{again.Key(), again.PictureKey(),
			again.LightKey()})
		if got != (moved{true, false, false}) {
			t.Errorf("changing %s moved %+v, want the whole key only", name,
				got)
		}
		writtenIn(t, directory, name, name)
	}
}

// playedWithFile is a played scene whose voicing is a file.
const playedWithFile = heading + "scene: {named: zappy, play: cues.yaml}\n" +
	"fps: 24\nduration: {value: 2, unit: s}\n" +
	"sound: {tune: tune.json, voicing: voicing.json}\n"

// anti-test of TestAPlayedScenesPictureKeyCoversItsCuesAndItsTune
// TestAPlayedScenesPictureKeyCoversItsVoicing holds that the voicing, which
// plays the scene's channels, moves the picture's key as the tune does,
// and that a file the job does not read does not.
func TestAPlayedScenesPictureKeyCoversItsVoicing(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"cues.yaml", "tune.json", "voicing.json"} {
		writtenIn(t, directory, name, name)
	}
	made, err := Read(writtenIn(t, directory, "job.yaml", playedWithFile))
	if err != nil {
		t.Fatal(err)
	}
	picture := made.PictureKey()
	writtenIn(t, directory, "stranger.json", "anything")
	if reread(t, made).PictureKey() != picture {
		t.Error("a file the job does not read moved the picture key")
	}
	writtenIn(t, directory, "voicing.json", "changed")
	if reread(t, made).PictureKey() == picture {
		t.Error("changing the voicing left the picture key alone")
	}
}

// anti-test of TestTheKeyCoversAFramesFolder
// TestTheKeyCoversWhatIsInEachFrame holds that changing the bytes of a
// frame, renaming one, and swapping the bytes of two each change the key,
// and that a folder with nothing in it is not a folder that is not there.
func TestTheKeyCoversWhatIsInEachFrame(t *testing.T) {
	directory := t.TempDir()
	plates := filepath.Join(directory, "plates")
	if err := os.MkdirAll(plates, private|executable); err != nil {
		t.Fatal(err)
	}
	job := heading + "scene: {frames: plates}\nfps: 24\n" +
		"duration: {value: 1, unit: s}\n"
	keyNow := func() string {
		made, err := Read(writtenIn(t, directory, "job.yaml", job))
		if err != nil {
			t.Fatal(err)
		}
		return made.Key()
	}
	empty := keyNow()
	if err := os.RemoveAll(plates); err != nil {
		t.Fatal(err)
	}
	if keyNow() == empty {
		t.Error("a folder with nothing in it has the key of one not there")
	}
	if err := os.MkdirAll(plates, private|executable); err != nil {
		t.Fatal(err)
	}
	writtenIn(t, plates, "a.png", "one")
	writtenIn(t, plates, "b.png", "two")
	start := keyNow()
	writtenIn(t, plates, "a.png", "changed")
	changedBytes := keyNow()
	if changedBytes == start {
		t.Error("changing a frame's bytes left the key alone")
	}
	writtenIn(t, plates, "a.png", "two")
	writtenIn(t, plates, "b.png", "one")
	if swapped := keyNow(); swapped == start {
		t.Error("swapping the bytes of two frames left the key alone")
	}
	writtenIn(t, plates, "a.png", "one")
	writtenIn(t, plates, "b.png", "two")
	if keyNow() != start {
		t.Error("the same frames put back have another key")
	}
	if err := os.Rename(filepath.Join(plates, "b.png"),
		filepath.Join(plates, "c.png")); err != nil {
		t.Fatal(err)
	}
	if keyNow() == start {
		t.Error("renaming a frame left the key alone")
	}
}

// anti-test of TestTheKeyCoversTheFilesTheJobReads
// TestASetThatCannotBeReadIsNotOneThatCan holds that a set that is not
// there, one that is there and one that cannot be read are three keys, so
// a picture is never reused for a set that has gone.
func TestASetThatCannotBeReadIsNotOneThatCan(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a file nobody can read is read by root")
	}
	directory := t.TempDir()
	keyNow := func() string {
		made, err := Read(writtenIn(t, directory, "job.yaml", still))
		if err != nil {
			t.Fatal(err)
		}
		return made.Key()
	}
	gone := keyNow()
	set := writtenIn(t, directory, "moonlake.yaml", "lake")
	there := keyNow()
	if err := os.Chmod(set, nothing); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(set, private) })
	shut := keyNow()
	if gone == there || there == shut || gone == shut {
		t.Errorf("a set gone, there and unreadable have keys %.8s %.8s %.8s",
			gone, there, shut)
	}
}

// anti-test of TestTheKeyIsStableUnderReorderingAndOutputs
// TestTheSameJobInAnotherFolderIsAnotherKey holds that a job's relative
// paths are the files they name where it lies: the same text beside
// another set is another key.
func TestTheSameJobInAnotherFolderIsAnotherKey(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	one, err := Read(writtenIn(t, first, "job.yaml", still))
	if err != nil {
		t.Fatal(err)
	}
	two, err := Read(writtenIn(t, second, "job.yaml", still))
	if err != nil {
		t.Fatal(err)
	}
	if one.Key() == two.Key() || one.PictureKey() == two.PictureKey() ||
		one.LightKey() == two.LightKey() {
		t.Error("two jobs naming two different sets share a key")
	}
}

// anti-test of TestTheKeyIgnoresWhatDoesNotChangeThePicture
// TestEveryWayOfNamingTheSceneChangesTheKey holds each form of scene to a
// key of its own, and a named scene's parameters, a played scene's sheet,
// a camera's path and a zoom's easing to the picture; the same
// parameters in another order are one key.
func TestEveryWayOfNamingTheSceneChangesTheKey(t *testing.T) {
	named := heading + "scene: {named: fire, with: {a: 1, b: 2}}\n"
	base := keysOf(t, named)
	if keysOf(t, heading+"scene: {named: fire, with: {b: 2, a: 1}}\n") != base {
		t.Error("the parameters in another order are another key")
	}
	for name, text := range map[string]string{
		"parameter":      heading + "scene: {named: fire, with: {a: 1, b: 3}}\n",
		"parameter key":  heading + "scene: {named: fire, with: {a: 1, c: 2}}\n",
		"parameter gone": heading + "scene: {named: fire, with: {a: 1}}\n",
		"no parameters":  heading + "scene: {named: fire}\n",
		"name":           heading + "scene: {named: ice, with: {a: 1, b: 2}}\n",
		"smoke":          heading + "scene: {smoke: 3}\n",
		"other smoke":    heading + "scene: {smoke: 4}\n",
		"frames":         heading + "scene: {frames: plates}\n",
		"other frames":   heading + "scene: {frames: other}\n",
		"set":            still,
	} {
		if keysOf(t, text) == base {
			t.Errorf("a scene with another %s has the key of the first", name)
		}
	}
	playing := func(sheet string) string {
		return heading + "scene: {named: zappy, play: " + sheet + "}\n" +
			"fps: 24\nduration: {value: 2, unit: s}\n" +
			"sound: {tune: '@moon', voicing: '@plain'}\n"
	}
	if keysOf(t, playing("a.yaml")) == keysOf(t, playing("b.yaml")) {
		t.Error("two cue sheets of other names are one key")
	}
	moving := func(change func(string) string) keySet {
		return keysOf(t, still+change(pathCamera))
	}
	path := moving(func(s string) string { return s })
	for name, change := range map[string]func(string) string{
		"to": func(s string) string {
			return strings.Replace(s, "z: 2}", "z: 3}", 1)
		},
		"from": func(s string) string {
			return strings.Replace(s, "z: 6}", "z: 7}", 1)
		},
		"look-at": func(s string) string {
			return strings.Replace(s, "look-at: {x: 0", "look-at: {x: 1", 1)
		},
		"speed": func(s string) string {
			return strings.Replace(s, "value: 2, unit: m/s", "value: 3, unit: m/s", 1)
		},
		"up": func(s string) string { return s + "  up: {x: 1, y: 0, z: 0}\n" },
		"unit": func(s string) string {
			return strings.Replace(s, "unit: m\n", "unit: a thousand kilometres\n", 1)
		},
	} {
		if moving(change) == path {
			t.Errorf("a camera path with another %s has the same key", name)
		}
	}
}

// anti-test of TestTheCommandLineStandsOverTheJob
// TestAnOverrideMovesTheKeyUnlessItIsAnOutput holds that each value the
// command line can lay over a job moves the key, and the output, which
// is not part of the picture, does not.
func TestAnOverrideMovesTheKeyUnlessItIsAnOutput(t *testing.T) {
	job := read(t, still+"camera: {name: rim}\n")
	for name, override := range map[string]Override{
		"width": Width(641), "height": Height(481), "rays": Rays(3),
		"at": At(time.Minute), "camera": CameraName("low"),
		"renderer": Renderer("cpu"), "quality": Quality("preview"),
		"vibes": Flagged(FlagVibes, true),
	} {
		over, err := job.With(override)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if over.Key() == job.Key() {
			t.Errorf("an override of the %s left the key alone", name)
		}
	}
	over, err := job.With(Output("elsewhere.png"))
	if err != nil {
		t.Fatal(err)
	}
	if over.Key() != job.Key() || over.PictureKey() != job.PictureKey() {
		t.Error("moving the output moved the key")
	}
}

// anti-test of TestAShortcutIsDecidedByTheFirstThatSaysIt
// TestAShortcutsOwnWordStandsOverEveryFlag holds that a shortcut said
// off stays off under vibes and fastlight on, said on stays on under both
// off, and that a flag that is not on turns nothing on.
func TestAShortcutsOwnWordStandsOverEveryFlag(t *testing.T) {
	for _, name := range []string{ShortcutLampStore, ShortcutSkyLooks} {
		off := read(t, still+"vibes: on\nfastlight: on\nshortcuts: {"+
			name+": off}\n")
		if off.Shortcut(name) {
			t.Errorf("%s said off is on under the flags", name)
		}
		on := read(t, still+"vibes: off\nfastlight: off\nshortcuts: {"+
			name+": on}\n")
		if !on.Shortcut(name) {
			t.Errorf("%s said on is off under the flags", name)
		}
		if read(t, still+"vibes: off\nfastlight: off\n").Shortcut(name) {
			t.Errorf("%s is on under flags that are off", name)
		}
	}
}
