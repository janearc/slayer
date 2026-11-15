package job

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inputsJob is a film that reads a set, a theme, a tune and a voicing,
// and a frames-free scene, every file written in directory.
const inputsJob = heading + `scene: {set: lake.yaml}
fps: 24
duration: {value: 1, unit: s}
titles:
  theme: theme.yaml
  elements:
    name: {kind: text, words: A, ink: ink, across: 0.1, down: 0.1}
sound: {tune: tune.json, voicing: voicing.json}
`

// inputsIn is the inputs job read from directory, with each file it
// reads written there first holding its own name.
func inputsIn(t *testing.T, directory string) Job {
	t.Helper()
	for _, name := range []string{"lake.yaml", "theme.yaml", "tune.json",
		"voicing.json"} {
		writtenIn(t, directory, name, name)
	}
	made, err := Read(writtenIn(t, directory, "job.yaml", inputsJob))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// reread is the job at the same file read again.
func reread(t *testing.T, made Job) Job {
	t.Helper()
	again, err := Read(made.File())
	if err != nil {
		t.Fatal(err)
	}
	return again
}

// the key covers what each file the job reads holds: changing the set
// or the theme changes the key and the picture key; changing the tune or
// the voicing changes the key and leaves the picture key alone.
func TestTheKeyCoversTheFilesTheJobReads(t *testing.T) {
	directory := t.TempDir()
	made := inputsIn(t, directory)
	key, picture := made.Key(), made.PictureKey()
	paths := made.Inputs()
	for _, name := range []string{inputSet, inputTheme, inputTune,
		inputVoicing} {
		if !filepath.IsAbs(paths[name]) {
			t.Errorf("the input %s is %q", name, paths[name])
		}
	}
	for _, each := range []struct {
		file    string
		picture bool
	}{{"lake.yaml", true}, {"theme.yaml", true}, {"tune.json", false},
		{"voicing.json", false}} {
		writtenIn(t, directory, each.file, "changed")
		changed := reread(t, made)
		if changed.Key() == key {
			t.Errorf("changing %s left the key alone", each.file)
		}
		moved := changed.PictureKey() != picture
		if moved != each.picture {
			t.Errorf("changing %s moved the picture key: %v", each.file,
				moved)
		}
		writtenIn(t, directory, each.file, each.file)
	}
	if !strings.Contains(string(made.Canonical()), `"inputs"`) {
		t.Error("the canonical form names no inputs")
	}
}

// the anti-test: writing a file again with the same bytes, or changing a
// file the job does not read, leaves both keys alone; and a built-in
// tune is not an input.
// anti-test of TestTheKeyCoversTheFilesTheJobReads
// anti-test of TestTheKeyCoversAFramesFolder
func TestTheKeyLeavesAloneWhatItDoesNotRead(t *testing.T) {
	directory := t.TempDir()
	made := inputsIn(t, directory)
	key, picture := made.Key(), made.PictureKey()
	writtenIn(t, directory, "lake.yaml", "lake.yaml")
	writtenIn(t, directory, "stranger.yaml", "anything")
	again := reread(t, made)
	if again.Key() != key || again.PictureKey() != picture {
		t.Error("a file unchanged, or one not read, changed the key")
	}
	builtIns := read(t, film+"sound: {tune: \"@crypt\", voicing: "+
		"\"@crypt-dmg\"}\n")
	if _, has := builtIns.Inputs()[inputTune]; has {
		t.Error("a built-in tune was taken for a file")
	}
}

// a folder of frames is an input by every file in it: adding a frame or
// changing one changes the key; a file not there is absent, not refused.
func TestTheKeyCoversAFramesFolder(t *testing.T) {
	directory := t.TempDir()
	plates := filepath.Join(directory, "plates")
	if err := os.MkdirAll(plates, private|executable); err != nil {
		t.Fatal(err)
	}
	writtenIn(t, plates, "a.png", "a")
	made, err := Read(writtenIn(t, directory, "job.yaml", heading+
		"scene: {frames: plates}\nfps: 24\nduration: {value: 1, unit: s}\n"))
	if err != nil {
		t.Fatal(err)
	}
	key := made.Key()
	writtenIn(t, plates, "b.png", "b")
	if reread(t, made).Key() == key {
		t.Error("a frame added left the key alone")
	}
	if digestOf(filepath.Join(directory, "gone")) != absentDigest {
		t.Error("a file not there is not absent")
	}
	var zero Job
	if len(zero.Inputs()) != nothing || zero.PictureKey() == "" {
		t.Error("the zero job has inputs, or no key")
	}
}

// the light's key covers the set and leaves alone what only develops
// the light: the theme a title reads, the tune and the voicing.
func TestTheLightKeyCoversTheSetAndNotTheDevelopment(t *testing.T) {
	directory := t.TempDir()
	made := inputsIn(t, directory)
	light := made.LightKey()
	for _, each := range []struct {
		file  string
		moves bool
	}{{"lake.yaml", true}, {"theme.yaml", false}, {"tune.json", false},
		{"voicing.json", false}} {
		writtenIn(t, directory, each.file, "changed")
		moved := reread(t, made).LightKey() != light
		if moved != each.moves {
			t.Errorf("changing %s moved the light key: %v", each.file,
				moved)
		}
		writtenIn(t, directory, each.file, each.file)
	}
}

// sideBySide is jobs read from one folder, so their sets are one path.
func sideBySide(t *testing.T, texts ...string) []Job {
	t.Helper()
	directory := t.TempDir()
	var made []Job
	for i, text := range texts {
		job, err := Read(writtenIn(t, directory, fmt.Sprintf("job%d.yaml",
			i), text))
		if err != nil {
			t.Fatal(err)
		}
		made = append(made, job)
	}
	return made
}

// the anti-test: a print or titles said leave the light's key alone and
// move the picture's, and a change of the film's size moves the light's.
// anti-test of TestTheLightKeyCoversTheSetAndNotTheDevelopment
func TestTheLightKeyLeavesTheDevelopmentAlone(t *testing.T) {
	jobs := sideBySide(t, film,
		film+"print: {stops: 1, knee: 0.7, top: 8}\n",
		film+"titles: {elements: {name: {kind: text, words: A, ink: ink, "+
			"across: 0.1, down: 0.1}}}\n",
		film+"film: {width: 64, height: 36}\n",
		film+"film: {width: 65, height: 36}\n")
	plain := jobs[0]
	for _, developed := range jobs[1:3] {
		if developed.LightKey() != plain.LightKey() {
			t.Errorf("%s moved the light key", developed.Canonical())
		}
		if developed.PictureKey() == plain.PictureKey() {
			t.Errorf("%s left the picture key alone", developed.Canonical())
		}
	}
	if jobs[3].LightKey() == jobs[4].LightKey() {
		t.Error("another width left the light key alone")
	}
}
