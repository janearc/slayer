//go:build draw

package main

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// TestRunWritesShortAndFilmPicture writes a two second short with a song
// under it, then takes a picture of the film at its own size.
func TestRunWritesShortAndFilmPicture(t *testing.T) {
	script := writeScript(t, "    sound: {bar: 2}")
	film := filepath.Join(t.TempDir(), "short.mp4")
	given := asked{script: script, short: "where", frame: -1, mp4: film,
		score: scoreFile, song: writeWav(t, 8)}
	if err := run(given); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(film); err != nil || info.Size() == 0 {
		t.Fatalf("film: %v", err)
	}
	picture := filepath.Join(t.TempDir(), "still.png")
	if err := run(asked{film: film, at: 0.5, png: picture}); err != nil {
		t.Fatal(err)
	}
	if size := readPNG(t, picture).Bounds().Size(); size != image.Pt(1080, 1920) {
		t.Errorf("picture is %v", size)
	}
	if err := run(asked{film: film, at: 600, png: picture}); err == nil {
		t.Error("took a picture long after the film ends")
	}
	blocked := asked{film: film, at: 0.5, png: filepath.Join(t.TempDir(), "gone", "still.png")}
	if err := run(blocked); err == nil {
		t.Error("wrote a picture into a folder that is not there")
	}
	if _, err := movie.ReadSound(film); err != nil {
		t.Errorf("the film has no readable sound: %v", err)
	}
}
