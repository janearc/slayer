//go:build draw && darwin && cgo

package pan

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// the film is written at its size, rate and count, with the sound.
func TestWriteMakesTheFilm(t *testing.T) {
	cut, err := scripted(t, twoShots, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "cut.mp4")
	if err := cut.Write(context.Background(), path, io.Discard); err != nil {
		t.Fatal(err)
	}
	facts, err := movie.Probe(path)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Width() != filmWidth || facts.Frames() != cut.Frames() ||
		int(facts.Rate()) != testRate || facts.AudioCodec() == "" {
		t.Errorf("the film is %dx%d, %d frames at %v, sound %q",
			facts.Width(), facts.Height(), facts.Frames(), facts.Rate(),
			facts.AudioCodec())
	}
}

// the anti-test: a cancelled context leaves no film.
func TestWriteCancelledLeavesNothing(t *testing.T) {
	cut, err := scripted(t, twoShots, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(t.TempDir(), "cut.mp4")
	if err := cut.Write(ctx, path, io.Discard); err == nil {
		t.Error("a cancelled film was written")
	}
	if _, err := movie.Probe(path); err == nil {
		t.Error("a cancelled film left a file")
	}
}
