package session

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/job"
)

// TestFileMusicUsesItsSuppliedGrid checks the real job-to-clock path.
// The bytes are authored, deliberately not decodable audio: a supplied
// grid must avoid decoding and must bind to the exact source file.
func TestFileMusicUsesItsSuppliedGrid(t *testing.T) {
	directory := t.TempDir()
	write := func(name, text string) string {
		t.Helper()
		path := filepath.Join(directory, name)
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	write("song.opus", "authored audio bytes")
	sum := sha256.Sum256([]byte("authored audio bytes"))
	write("grid.yaml", fmt.Sprintf("score:\n  audio: song.opus\n"+
		"  sha256: %x\n  duration: 10\n  meter: 4\n  status: estimated\n"+
		"  method: fixture\n  beats:\n    times: [1, 1.5, 2.25, 3]\n"+
		"    measured: [1, 1, 0, 1]\n", sum))
	write("cues.yaml", "cues: []\n")
	path := write("job.yaml", "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\n"+
		"scene: {named: zappy, play: cues.yaml}\n"+
		"fps: 24\nduration: {value: 30, unit: s}\n"+
		"scene-duration: {value: 3600, unit: s}\n"+
		"sound: {file: song.opus, score: grid.yaml, "+
		"from: {value: 1.5, unit: s}}\n")
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	sound, _ := wanted.Sound()
	clock, err := soundtrackClock(sound)
	if err != nil {
		t.Fatal(err)
	}
	if clock.Moment(2) != 750*time.Millisecond {
		t.Fatal("audio trim was not applied on the original grid")
	}
	// anti-test of TestFileMusicUsesItsSuppliedGrid: same path, new audio.
	write("song.opus", "a different recording")
	if _, err := soundtrackClock(sound); err == nil {
		t.Fatal("accepted a grid for a different source recording")
	}
}
