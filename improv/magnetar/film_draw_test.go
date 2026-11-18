//go:build draw

package main

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// A short written to a movie says how many frames it wrote: five seconds
// at 24 a second.
func TestRunWritesAShortAsAMovie(t *testing.T) {
	out := filepath.Join(t.TempDir(), "field.mp4")
	var said strings.Builder
	err := run([]string{"magnetar", "magnetar-walk.yaml",
		stills(t, color.RGBA{R: 30, G: 60, B: 90, A: 255}), "field", out},
		&said)
	if err != nil {
		t.Fatal(err)
	}
	if want := out + ": 120 frames\n"; said.String() != want {
		t.Errorf("the command said %q, want %q", said.String(), want)
	}
	facts, err := movie.Probe(out)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Frames() != 120 {
		t.Errorf("the movie holds %d frames, want 120", facts.Frames())
	}
}

// The film command reads the plan, writes the short as a movie of the
// film's frame with the tune's sound, and says how many frames it wrote.
func TestFilmWritesTheShortAndSaysHowManyFrames(t *testing.T) {
	fixture := newFilmFixture(t, "[spreads, breaks]", "")
	var said strings.Builder
	err := run([]string{"magnetar", "film", fixture.plan, fixture.frames,
		fixture.out}, &said)
	if err != nil {
		t.Fatal(err)
	}
	if want := fixture.out + ": 3 frames\n"; said.String() != want {
		t.Errorf("the command said %q, want %q", said.String(), want)
	}
	facts, err := movie.Probe(fixture.out)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Frames() != 3 || facts.Width() != filmSize.X {
		t.Errorf("the movie holds %d frames %d wide, want 3 and %d",
			facts.Frames(), facts.Width(), filmSize.X)
	}
	if facts.AudioDuration() <= 0 {
		t.Error("the movie has no sound")
	}
}

// A plan that names a clips folder lays the host's clips in the film,
// and the named intro clip leads it for the intro's frames.
func TestFilmWithClipsPutsTheIntroClipBeforeTheShort(t *testing.T) {
	fixture := newFilmFixture(t, "[spreads]", writeClips(t))
	raw, err := os.ReadFile(fixture.plan)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(raw), "frames: 0", "frames: 2", 1)
	if err := os.WriteFile(fixture.plan, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	var said strings.Builder
	if err := film(fixture.plan, fixture.frames, fixture.out, &said); err != nil {
		t.Fatal(err)
	}
	if want := fixture.out + ": 5 frames\n"; said.String() != want {
		t.Errorf("the command said %q, want %q", said.String(), want)
	}
	facts, err := movie.Probe(fixture.out)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Frames() != 5 {
		t.Errorf("the movie holds %d frames, want the intro's 2 and the "+
			"short's 3", facts.Frames())
	}
}

// writeClips writes the five clips the host's folder holds, each of two
// frames the host's size, into a new folder, and gives its path.
func writeClips(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"resting", "speaking", "angry", "flash",
		"quiet"} {
		settings, err := movie.NewSettings(movie.Request{
			Path: filepath.Join(dir, name+".mp4"), Width: hostSize.X,
			Height: hostSize.Y, Rate: filmRate, Frames: 2, Pad: true})
		if err != nil {
			t.Fatal(err)
		}
		writer, err := movie.New(context.Background(), settings)
		if err != nil {
			t.Fatal(err)
		}
		for i := range 2 {
			picture := image.NewRGBA(image.Rectangle{Max: hostSize})
			for p := 0; p < len(picture.Pix); p += 4 {
				picture.Pix[p+3] = 255
				picture.Pix[p] = uint8(100 * i)
			}
			if err := writer.Frame(i, picture); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
