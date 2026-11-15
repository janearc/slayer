package door

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The tests' times either side of midnight, from noon on the day the door
// was built. Picked by the door slice's author, 2026-11-12.
const (
	lateBy  = 11*time.Hour + 59*time.Minute // 23:59 that day
	earlyBy = 12*time.Hour + time.Minute    // 00:01 the next day
)

// a film whose working frames were begun for a movie on an earlier day
// is placed back in that day's folder after midnight, beside those
// frames, and says it resumes; the anti-test: frames of another key are
// left alone and the film takes the new day's folder.
func TestPlacesResumeWorkingFramesOfAnEarlierDay(t *testing.T) {
	directory, home := t.TempDir(), t.TempDir()
	path := written(t, directory, "short.yaml",
		filmJob(t, "outputs: {movie: short.mp4}\n"))
	wanted, err := jobOf(path)
	if err != nil {
		t.Fatal(err)
	}
	at := func(by time.Duration) Env {
		return envAt(t, Given{Context: context.Background(),
			Directory: directory, Now: func() time.Time {
				return noon().Add(by)
			}}, rendersVariable, home, teamVariable, "team")
	}
	const key = "the frames' key"
	begun := placesOf(placing{wanted: wanted, given: path,
		env: at(lateBy), key: key})
	book, _, err := openLedger(begun.frames, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := book.plan(begun.movie); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(begun.movie), folderMode); err != nil {
		t.Fatal(err)
	}
	after := placesOf(placing{wanted: wanted, given: path,
		env: at(earlyBy), key: key})
	if after.movie != begun.movie || after.frames != begun.frames ||
		after.resumed == "" {
		t.Errorf("after midnight the film is placed %+v", after)
	}
	// anti-test of TestPlacesResumeWorkingFramesOfAnEarlierDay
	other := placesOf(placing{wanted: wanted, given: path,
		env: at(earlyBy), key: "another key"})
	if other.movie == begun.movie || other.resumed != "" {
		t.Errorf("another key resumed the earlier day: %+v", other)
	}
}
