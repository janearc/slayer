//go:build draw

package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// The command writes the intro and the short as one film with the
// tune's sound, and says how many frames it wrote.
func TestRunWritesTheHostedShort(t *testing.T) {
	f := newFixture(t)
	var out, problems bytes.Buffer
	if code := run(f.args, &out, &problems); code != ok {
		t.Fatalf("run exited %d: %s", code, problems.String())
	}
	film := filepath.Join(f.dir, "out.mp4")
	if want := film + ": 10 frames\n"; out.String() != want {
		t.Errorf("the command said %q, want %q", out.String(), want)
	}
	facts, err := movie.Probe(film)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Frames() != 10 || facts.Width() != frameWide ||
		facts.AudioDuration() <= 0 {
		t.Errorf("the film holds %d frames %d wide with %v of sound, want "+
			"10 frames, %d wide, with sound", facts.Frames(), facts.Width(),
			facts.AudioDuration(), frameWide)
	}
}
