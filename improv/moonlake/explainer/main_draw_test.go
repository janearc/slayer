//go:build draw

package main

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// filmSize keeps the film shape at the frame width: 64 by 48 is
// 4 to 3, which is 1080 by 810.
func TestFilmSizeKeepsTheShape(t *testing.T) {
	size, err := filmSize(smallFilm(t, 24))
	if err != nil {
		t.Fatal(err)
	}
	if size != image.Pt(1080, 810) {
		t.Errorf("size = %v, want 1080 by 810", size)
	}
	if _, err := filmSize(filepath.Join(t.TempDir(), "none.mp4")); err == nil {
		t.Error("filmSize accepted a missing film")
	}
}

// still writes a frame of the short's size, and refuses a moment the
// film does not reach, a missing film and an unwritable png.
func TestStillWritesAFrameOfTheShortsSize(t *testing.T) {
	asked := room(t)
	made, err := explainerFor(asked)
	if err != nil {
		t.Fatal(err)
	}
	asked.film = smallFilm(t, 48)
	asked.png = filepath.Join(t.TempDir(), "still.png")
	asked.still = 0.5
	if err := still(made, asked); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(asked.png)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	if config.Width != frameWidth || config.Height != frameHigh {
		t.Errorf("still is %d by %d", config.Width, config.Height)
	}

	late := asked
	late.still = 30
	if err := still(made, late); err == nil {
		t.Error("still accepted a moment past the film's end")
	}
	gone := asked
	gone.film += ".gone"
	if err := still(made, gone); err == nil {
		t.Error("still accepted a missing film")
	}
	blocked := asked
	blocked.png = filepath.Join(t.TempDir(), "no", "such", "dir.png")
	if err := still(made, blocked); err == nil {
		t.Error("still wrote into a directory that is not there")
	}
	off := asked
	off.still = -5
	if err := still(made, off); err == nil {
		t.Error("still composed a moment before the film")
	}
}

// short writes the script's second as a vertical movie with the song
// under it, 24 frames of 1080 by 1920.
func TestShortWritesTheScriptsSpan(t *testing.T) {
	asked := room(t)
	made, err := explainerFor(asked)
	if err != nil {
		t.Fatal(err)
	}
	asked.film = smallFilm(t, 96)
	asked.song = song(t, 4)
	asked.name = "tiny"
	asked.mp4 = filepath.Join(t.TempDir(), "short.mp4")
	if err := short(made, asked); err != nil {
		t.Fatal(err)
	}
	facts, err := movie.Probe(asked.mp4)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Frames() != 24 || facts.Width() != frameWidth || facts.Height() != frameHigh ||
		facts.AudioChannels() != 2 {
		t.Errorf("short is %d frames of %dx%d with %d channels", facts.Frames(),
			facts.Width(), facts.Height(), facts.AudioChannels())
	}
}

// short refuses a name the script lacks, naming the ones it has, and
// each input it cannot use.
func TestShortRefusesWhatItCannotUse(t *testing.T) {
	asked := room(t)
	made, err := explainerFor(asked)
	if err != nil {
		t.Fatal(err)
	}
	asked.film = smallFilm(t, 96)
	asked.song = song(t, 4)
	asked.name = "tiny"
	asked.mp4 = filepath.Join(t.TempDir(), "short.mp4")

	unknown := asked
	unknown.name = "huge"
	if err := short(made, unknown); err == nil || !strings.Contains(err.Error(), `"huge"`) ||
		!strings.Contains(err.Error(), "tiny") {
		t.Errorf("unknown short gave %v", err)
	}
	noFilm := asked
	noFilm.film += ".gone"
	if err := short(made, noFilm); err == nil {
		t.Error("short accepted a missing film")
	}
	noSong := asked
	noSong.song += ".gone"
	if err := short(made, noSong); err == nil {
		t.Error("short accepted a missing song")
	}
	badOut := asked
	badOut.mp4 = filepath.Join(t.TempDir(), "no", "such", "short.mp4")
	if err := short(made, badOut); err == nil {
		t.Error("short wrote into a directory that is not there")
	}
	early := asked
	early.film = smallFilm(t, 30)
	if err := short(made, early); err == nil {
		t.Error("short accepted a film that ends before the short starts")
	}
	if _, err := os.Stat(asked.mp4); err == nil {
		t.Error("a refused short left a movie behind")
	}
}

// short stops, and leaves no movie, when the film ends inside the span
// it was asked to cover.
func TestShortStopsWhenTheFilmEndsEarly(t *testing.T) {
	asked := room(t)
	made, err := explainerFor(asked)
	if err != nil {
		t.Fatal(err)
	}
	asked.film = smallFilm(t, 54) // ends at 2.25 s, inside the 2 s to 3 s short
	asked.song = song(t, 4)
	asked.name = "tiny"
	asked.mp4 = filepath.Join(t.TempDir(), "short.mp4")
	if err := short(made, asked); err == nil {
		t.Fatal("short finished with too few film frames")
	}
	if _, err := os.Stat(asked.mp4); err == nil {
		t.Error("a short that failed left its movie behind")
	}
}

// a film so tall that its picture at the frame's width leaves no room
// for the panel is refused by still and by short, and short leaves no
// movie behind.
func TestATallFilmLeavesNoRoomForThePanel(t *testing.T) {
	asked := room(t)
	made, err := explainerFor(asked)
	if err != nil {
		t.Fatal(err)
	}
	asked.film = filmOf(t, 32, 96, 96)
	asked.song = song(t, 4)
	asked.name = "tiny"
	asked.mp4 = filepath.Join(t.TempDir(), "short.mp4")
	asked.png = filepath.Join(t.TempDir(), "still.png")
	asked.still = 0.5
	if err := still(made, asked); err == nil || !strings.Contains(err.Error(), "under") {
		t.Errorf("still gave %v", err)
	}
	if err := short(made, asked); err == nil || !strings.Contains(err.Error(), "under") {
		t.Errorf("short gave %v", err)
	}
	if _, err := os.Stat(asked.mp4); err == nil {
		t.Error("a refused short left a movie behind")
	}
}

// smallFilm writes a film of that many frames at 24 a second, 64 by 48,
// and is its path.
func smallFilm(t *testing.T, frames int) string {
	t.Helper()
	return filmOf(t, 64, 48, frames)
}

// filmOf writes a film of that size and many frames at 24 a second and is
// its path.
func filmOf(t *testing.T, width, height, frames int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "film.mp4")
	settings, err := movie.NewSettings(movie.Request{Path: path, Width: width,
		Height: height, Rate: shortRate, Frames: frames, Pad: true})
	if err != nil {
		t.Fatal(err)
	}
	writer, err := movie.New(context.Background(), settings)
	if err != nil {
		t.Fatal(err)
	}
	for i := range frames {
		picture := image.NewRGBA(image.Rect(0, 0, width, height))
		for x := range width {
			picture.Set(x, 10, color.RGBA{R: uint8(i * 3), G: 200, B: 50, A: 255})
		}
		if err := writer.Frame(i, picture); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
