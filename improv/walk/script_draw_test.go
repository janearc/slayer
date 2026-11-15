//go:build draw

package walk

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/janearc/slayer/interfaces/movie"
)

// TestWrite writes a short of the fixture with its sound to an mp4 and
// reads its facts back. On a machine without AVFoundation the writer
// refuses, and so does Write.
func TestWrite(t *testing.T) {
	script, err := scriptWith(t, "", "")
	if err != nil {
		t.Fatal(err)
	}
	short, _ := script.Short("where")
	short.shots = short.shots[:1]
	short.shots[0].seconds = 0.5
	path := filepath.Join(t.TempDir(), "short.mp4")
	written, err := short.Write(context.Background(), path, movie.Sound{})
	if runtime.GOOS != "darwin" {
		if err == nil {
			t.Error("a movie was written without AVFoundation")
		}
		return
	}
	if err != nil || written != short.Frames() {
		t.Fatalf("wrote %d frames: %v", written, err)
	}
	facts, err := movie.Probe(path)
	if err != nil || facts.Width() != frameSize.X ||
		facts.Frames() != short.Frames() {
		t.Errorf("the movie is %+v: %v", facts, err)
	}
	if _, err := short.Write(context.Background(),
		filepath.Join(t.TempDir(), "none", "x.mp4"),
		movie.Sound{}); err == nil {
		t.Error("a movie was written into a folder that is not there")
	}
}
