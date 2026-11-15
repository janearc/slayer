package finish

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/frame"
)

// private is defined as a file only its owner reads and writes.
const private = 0o600

// heading is every job's first line, and still the least scene.
const heading = "$schema: https://github.com/janearc/slayer/schemas/" +
	"job-v1.json\nscene: {set: nowhere.yaml}\n"

// jobOf is the job that text says after the heading, read from a file in
// a directory of its own.
func jobOf(t *testing.T, text string) job.Job {
	t.Helper()
	return jobIn(t, t.TempDir(), heading+text)
}

// jobIn is the job a whole document says, written in a directory.
func jobIn(t *testing.T, directory, document string) job.Job {
	t.Helper()
	path := filepath.Join(directory, "job.yaml")
	if err := os.WriteFile(path, []byte(document), private); err != nil {
		t.Fatal(err)
	}
	made, err := job.Read(path)
	if err != nil {
		t.Fatalf("the job was refused: %v\n%s", err, document)
	}
	return made
}

// gradient is a frame of w by h whose light runs from dim at the top left
// to well past white at the bottom right, in every channel a little
// different, so a picture of it has shadows, middle and highlights.
func gradient(w, h int) *frame.Frame {
	made := frame.New(w, h)
	for y := range h {
		for x := range w {
			along := float64(x+y) / float64(w+h)
			pixel := made.At(x, y)
			pixel.R = float32(0.01 + 6*along)
			pixel.G = float32(0.02 + 4*along)
			pixel.B = float32(0.03 + 2*along)
			made.Set(x, y, pixel)
		}
	}
	return made
}

// scaled is a copy of the frame with all its light multiplied.
func scaled(from *frame.Frame, by float32) *frame.Frame {
	made := frame.New(from.W, from.H)
	copy(made.Pix, from.Pix)
	for i := range made.Pix {
		made.Pix[i].R *= by
		made.Pix[i].G *= by
		made.Pix[i].B *= by
	}
	return made
}
