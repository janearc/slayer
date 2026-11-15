package job

import "testing"

// playedJob is a named scene played from a cue sheet by a tune: both are
// files beside the job.
const playedJob = heading + "scene: {named: zappy, play: cues.yaml}\n" +
	"fps: 24\nduration: {value: 2, unit: s}\n" +
	"sound: {tune: tune.json, voicing: '@plain'}\n"

// played is the played job read from a folder holding its cue sheet and
// its tune.
func played(t *testing.T, directory string) Job {
	t.Helper()
	writtenIn(t, directory, "cues.yaml", "cues")
	writtenIn(t, directory, "tune.json", "tune")
	made, err := Read(writtenIn(t, directory, "job.yaml", playedJob))
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// a scene played from music moves with its cue sheet and its tune: a
// change to either changes the picture's key, so frames drawn to the old
// music are never taken for the new (review of 2026-11-12)
func TestAPlayedScenesPictureKeyCoversItsCuesAndItsTune(t *testing.T) {
	directory := t.TempDir()
	made := played(t, directory)
	if made.Inputs()[inputPlay] == "" {
		t.Fatal("the cue sheet is not an input")
	}
	picture := made.PictureKey()
	for _, file := range []string{"cues.yaml", "tune.json"} {
		writtenIn(t, directory, file, "changed")
		if reread(t, made).PictureKey() == picture {
			t.Errorf("changing %s left the picture key alone", file)
		}
		writtenIn(t, directory, file, map[string]string{
			"cues.yaml": "cues", "tune.json": "tune"}[file])
	}
	// anti-test of TestAPlayedScenesPictureKeyCoversItsCuesAndItsTune
	if reread(t, made).PictureKey() != picture {
		t.Error("the same files gave another picture key")
	}
}
