package job

import (
	"os"
	"path/filepath"
	"testing"
)

// framesJob is a film of the frames in a folder beside it.
const framesJob = heading + "scene: {frames: plates}\nfps: 24\n" +
	"duration: {value: 1, unit: s}\n"

// a frame that is a link is followed: the bytes it points at are in the
// key, so a frame changed where it is kept is not taken for the old one
func TestAFrameThatIsALinkIsKeyedByWhatItPointsAt(t *testing.T) {
	directory := t.TempDir()
	plates := filepath.Join(directory, "plates")
	if err := os.MkdirAll(plates, 0o700); err != nil {
		t.Fatal(err)
	}
	target := writtenIn(t, directory, "real.png", "one")
	if err := os.Symlink(target, filepath.Join(plates, "a.png")); err != nil {
		t.Fatal(err)
	}
	made, err := Read(writtenIn(t, directory, "job.yaml", framesJob))
	if err != nil {
		t.Fatal(err)
	}
	before := made.Key()
	writtenIn(t, directory, "real.png", "two")
	if reread(t, made).Key() == before {
		t.Error("a linked frame changed where it is kept left the key alone")
	}
}

// anti-test of TestAFrameThatIsALinkIsKeyedByWhatItPointsAt
func TestALinkToTheSameBytesKeepsTheKey(t *testing.T) {
	directory := t.TempDir()
	plates := filepath.Join(directory, "plates")
	if err := os.MkdirAll(plates, 0o700); err != nil {
		t.Fatal(err)
	}
	target := writtenIn(t, directory, "real.png", "one")
	if err := os.Symlink(target, filepath.Join(plates, "a.png")); err != nil {
		t.Fatal(err)
	}
	made, err := Read(writtenIn(t, directory, "job.yaml", framesJob))
	if err != nil {
		t.Fatal(err)
	}
	before := made.Key()
	writtenIn(t, directory, "real.png", "one")
	if reread(t, made).Key() != before {
		t.Error("the same bytes behind a link gave another key")
	}
}
