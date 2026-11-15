//go:build draw

package door

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/file"
)

// a bare-named film begun under a renders home before midnight resumes
// its folder after midnight, keeping its frames, and says so; it does
// not start a dated folder of the new day.
func TestDrawResumesAcrossMidnight(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	path := written(t, directory, "night.yaml",
		filmJob(t, "outputs: {frames: frames}\n"))
	late := envAt(t, Given{Context: context.Background(),
		Directory: directory, Now: func() time.Time {
			return noon().Add(lateBy)
		}}, rendersVariable, home)
	drew(t, late, path)
	early := envAt(t, Given{Context: context.Background(),
		Directory: directory, Now: func() time.Time {
			return noon().Add(earlyBy)
		}}, rendersVariable, home)
	said := drew(t, early, path)
	begun := filepath.Join(filmsFolder, today+"-night")
	if !strings.Contains(said, "resuming "+begun) ||
		!strings.Contains(said, "nothing to draw") {
		t.Errorf("the film after midnight said %q", said)
	}
	next := filepath.Join(home, filmsFolder,
		noon().Add(earlyBy).Format(dateLayout)+"-night")
	if _, err := os.Stat(next); err == nil {
		t.Error("a folder of the new day was made")
	}
}

// a folder of an earlier day holding frames of another
// job of the same name is left alone, and the film starts the new day's
// folder.
// anti-test of TestDrawResumesAcrossMidnight
func TestDrawLeavesAnotherJobsEarlierFolder(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	path := written(t, directory, "night.yaml",
		filmJob(t, "outputs: {frames: frames}\n"))
	late := envAt(t, Given{Context: context.Background(),
		Directory: directory, Now: func() time.Time {
			return noon().Add(lateBy)
		}}, rendersVariable, home)
	drew(t, late, path)
	early := envAt(t, Given{Context: context.Background(),
		Directory: directory, Now: func() time.Time {
			return noon().Add(earlyBy)
		}}, rendersVariable, home)
	said := drew(t, early, path, "--rays", "2")
	if strings.Contains(said, "resuming") {
		t.Errorf("another job resumed the earlier folder: %q", said)
	}
	next := filepath.Join(home, filmsFolder,
		noon().Add(earlyBy).Format(dateLayout)+"-night")
	if _, err := os.Stat(filepath.Join(next, manifestName)); err != nil {
		t.Errorf("the new day's folder was not made: %v", err)
	}
}

// frames drawn by one build are not resumed by another but drawn again,
// the key holding the build; the anti-test: the same build twice draws
// nothing the second time.
func TestDrawRedrawsFramesOfAnotherBuild(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n"))
	built := func(build string) Env {
		return envAt(t, Given{Context: context.Background(), Now: noon,
			Directory: directory, Build: build})
	}
	drew(t, built("one"), path)
	other := drew(t, built("two"), path)
	if !strings.Contains(other, "from an earlier version") ||
		!strings.Contains(other, fmt.Sprintf("drew %d frames and kept 0",
			filmFrames(t))) {
		t.Errorf("another build resumed the frames: %q", other)
	}
	// anti-test of TestDrawRedrawsFramesOfAnotherBuild
	again := drew(t, built("two"), path)
	if !strings.Contains(again, "nothing to draw") {
		t.Errorf("the same build did not resume: %q", again)
	}
}

// frames drawn from a folder of frames already rendered are drawn again
// once a file in that folder changes, the key holding what the files
// say, and the old frames are gone.
func TestDrawRedrawsWhenAnInputChanges(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	drew(t, env, written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n")))
	again := written(t, directory, "again.yaml", jobLine+
		"scene: {frames: film}\n"+fmt.Sprintf("fps: %d\n",
		number(t, "fps"))+"duration: {value: 1, unit: s}\n"+
		"outputs: {frames: again}\n")
	drew(t, env, again)
	plate := filepath.Join(directory, "film", fmt.Sprintf(frameName, one))
	before := fileSum(filepath.Join(directory, "again",
		fmt.Sprintf(frameName, one)))
	changed := pictureOf(t, plate)
	changed.Pix[nothing] ^= byte(one)
	if err := file.SequencePNG(plate, changed); err != nil {
		t.Fatal(err)
	}
	said := drew(t, env, again)
	if !strings.Contains(said, "from an earlier version") ||
		!strings.Contains(said, fmt.Sprintf("drew %d frames and kept 0",
			filmFrames(t))) {
		t.Errorf("a changed input resumed the frames: %q", said)
	}
	if fileSum(filepath.Join(directory, "again",
		fmt.Sprintf(frameName, one))) == before {
		t.Error("the old frame of the changed plate is still there")
	}
	// anti-test of TestDrawRedrawsWhenAnInputChanges
	if still := drew(t, env, again); !strings.Contains(still,
		"nothing to draw") {
		t.Errorf("an input that did not change again redrew: %q", still)
	}
}

// a frame whose picture was written and whose record was not, as a kill
// between the two leaves it, is drawn again on the next run.
func TestDrawRedrawsAFrameWrittenAndNotRecorded(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	folder := filepath.Join(directory, "film")
	book, _, err := openLedger(folder, keyOfFolder(folder))
	if err != nil {
		t.Fatal(err)
	}
	delete(book.record.Frames, fmt.Sprintf(frameName, one))
	if err := book.write(); err != nil {
		t.Fatal(err)
	}
	said := drew(t, env, path)
	if !strings.Contains(said, fmt.Sprintf("drew 1 frames and kept %d",
		filmFrames(t)-one)) {
		t.Errorf("an unrecorded frame was not drawn again: %q", said)
	}
	// anti-test of TestDrawRedrawsAFrameWrittenAndNotRecorded
	if again := drew(t, env, path); !strings.Contains(again,
		"nothing to draw") {
		t.Errorf("a frame recorded after it was drawn was drawn again: %q",
			again)
	}
}
