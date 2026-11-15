//go:build draw

package main

import (
	"os"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// Run draws the frames of a placed scene to the writer it is given.
func TestRunWritesTheFrames(t *testing.T) {
	mode, world, eye := smokeMode(t)
	out, err := os.CreateTemp(t.TempDir(), "frames")
	if err != nil {
		t.Fatal(err)
	}
	if err := mode.run(out, world, eye); err != nil {
		t.Fatalf("run: %v", err)
	}
	if info, _ := out.Stat(); info.Size() == noCount {
		t.Error("run wrote nothing")
	}
}

// anti-tests: a writer that will take nothing stops the run with an error.
func TestRunStopsWhenTheWriterRefuses(t *testing.T) {
	mode, world, eye := smokeMode(t)
	readOnly, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	if err := mode.run(readOnly, world, eye); err == nil {
		t.Error("run went on with a writer that takes nothing")
	}
}

// smokeMode is a good ask's cells mode, with the picked smoke scene's
// world and eye to place in it.
func smokeMode(t *testing.T) (cellsMode, *thing.World, maths.Subspace) {
	t.Helper()
	mode, err := askCells(goodAsk())
	if err != nil {
		t.Fatal(err)
	}
	world, eye := smokeScene(t)
	return mode, world, eye
}
