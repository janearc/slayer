//go:build draw && darwin && cgo

package movie

import (
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// meanError is the mean difference of every red, green and blue sample
// of two pictures of one size, in 0-255 steps.
func meanError(one, other *image.RGBA) float64 {
	total, count := 0.0, 0
	for i := range one.Pix {
		if i%bytesPerPixel == bytesPerPixel-1 {
			continue
		}
		total += math.Abs(float64(one.Pix[i]) - float64(other.Pix[i]))
		count++
	}
	return total / float64(count)
}

// smallFilm writes the small movie, every frame a gradient that moves
// with its index, and is its path and size.
func smallFilm(t *testing.T) (string, image.Point) {
	t.Helper()
	request := smallRequest(t)
	written(t, request)
	return request.Path, image.Pt(request.Width, request.Height)
}

// frameTime is the middle of frame i at the small rate.
func frameTime(i int) time.Duration {
	rate := picked("small-rate")
	return time.Duration((float64(i) + 0.5) * float64(time.Second) /
		float64(rate))
}

// a film the package wrote reads back frame for frame, in order, each
// within film-error of what was written, and then ends.
func TestAWrittenFilmReadsBackFrameForFrame(t *testing.T) {
	path, size := smallFilm(t)
	film, err := OpenFilm(path, 0, size)
	if err != nil {
		t.Fatal(err)
	}
	defer film.Close()
	if film.Native() != size || film.Rate() != float64(picked("small-rate")) {
		t.Errorf("the film says it is %v at %v", film.Native(), film.Rate())
	}
	worst := 0.0
	for i := range picked("small-frames") {
		picture, err := film.Next()
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		worst = math.Max(worst, meanError(picture, gradient(size, i)))
	}
	if worst > pickedValue("film-error") {
		t.Errorf("a frame read back %.2f steps from what was written", worst)
	}
	t.Logf("the worst frame read back %.2f steps from what was written",
		worst)
	if _, err := film.Next(); !errors.Is(err, ErrPastEnd) {
		t.Errorf("after the last frame: %v", err)
	}
}

// the frame at a time is the frame showing then: nearer what was written
// for it than for the frames either side.
func TestTheFrameAtATimeIsTheRightFrame(t *testing.T) {
	path, size := smallFilm(t)
	for _, i := range []int{0, 1, 11, 23} {
		picture, err := FrameAt(path, frameTime(i), size)
		if err != nil {
			t.Fatal(err)
		}
		own := meanError(picture, gradient(size, i))
		for _, other := range []int{i - 1, i + 1} {
			if other < 0 {
				continue
			}
			if beside := meanError(picture, gradient(size, other)); beside <= own {
				t.Errorf("at frame %d's time the picture is as near frame "+
					"%d's (%.2f) as its own (%.2f)", i, other, beside, own)
			}
		}
	}
}

// read from a moment, the first picture is the one showing then, and
// the rest follow in order.
func TestAFilmReadFromAMomentStartsThere(t *testing.T) {
	path, size := smallFilm(t)
	film, err := OpenFilm(path, frameTime(10), size)
	if err != nil {
		t.Fatal(err)
	}
	defer film.Close()
	for i := 10; i < 13; i++ {
		picture, err := film.Next()
		if err != nil {
			t.Fatal(err)
		}
		if own := meanError(picture, gradient(size, i)); own > pickedValue("film-error") {
			t.Errorf("frame %d read %.2f steps off", i, own)
		}
	}
	if shown := film.Shown(); shown < frameTime(11) || shown > frameTime(12) {
		t.Errorf("the third picture read was shown at %v", shown)
	}
	if film.Length() < frameTime(picked("small-frames")-1) {
		t.Errorf("the film is %v long", film.Length())
	}
}

// pictures read at another size are scaled by the decoder to it.
func TestAFilmIsReadAtTheSizeAsked(t *testing.T) {
	path, size := smallFilm(t)
	half := size.Div(2)
	picture, err := FrameAt(path, frameTime(0), half)
	if err != nil {
		t.Fatal(err)
	}
	if picture.Bounds().Size() != half {
		t.Errorf("asked for %v, read %v", half, picture.Bounds().Size())
	}
}

// the anti-test: reading at or past the end is refused by name, and so
// are a size of nothing, a moment before the start, a closed film, and
// what is not a film.
func TestReadingPastTheEndIsRefused(t *testing.T) {
	path, size := smallFilm(t)
	end := time.Duration(picked("small-frames")) * time.Second /
		time.Duration(picked("small-rate"))
	for _, at := range []time.Duration{end, end + time.Minute} {
		_, err := FrameAt(path, at, size)
		if !errors.Is(err, ErrPastEnd) || !strings.Contains(err.Error(), "ends at") {
			t.Errorf("at %v: %v", at, err)
		}
	}
	for _, bad := range []image.Point{{0, 10}, {10, 0}, {mostSide + 1, 10}} {
		if _, err := OpenFilm(path, 0, bad); err == nil {
			t.Errorf("a size of %v was taken", bad)
		}
	}
	if _, err := OpenFilm(path, -time.Second, size); err == nil {
		t.Error("a moment before the start was taken")
	}
	film, err := OpenFilm(path, 0, size)
	if err != nil {
		t.Fatal(err)
	}
	film.Close()
	film.Close()
	if _, err := film.Next(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("a closed film read: %v", err)
	}
	notes := filepath.Join(t.TempDir(), notMovieName)
	if err := os.WriteFile(notes, []byte("not a film"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenFilm(notes, 0, size); err == nil {
		t.Error("a text file was read as a film")
	}
	if _, err := OpenFilm(filepath.Join(t.TempDir(), "gone.mp4"), 0, size); err == nil {
		t.Error("a missing film was read")
	}
}
