//go:build draw

package door

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// anti-test of TestAssessSaysStaleFramesAndLeavesThem
func TestAssessDoesNotCallCurrentFramesStale(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml", filmJob(t,
		"outputs: {frames: film}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	got := running(Tool(env), "assess", path)
	if got.code != nothing || strings.Contains(got.said,
		"earlier version") || !strings.Contains(got.said,
		"film.yaml will draw") {
		t.Errorf("frames of this version gave %d: %q %q", got.code,
			got.said, got.wrongSaid)
	}
}

// anti-test of TestDrawStockChangesThePictureTheSameEachTime
func TestDrawStockOfAnotherSeedIsAnotherPicture(t *testing.T) {
	directory := t.TempDir()
	env := envIn(t, context.Background(), directory)
	for _, seed := range []string{"7", "8"} {
		drew(t, env, written(t, directory, "s"+seed+".yaml", stillJob(t,
			"stock: {name: lantern, seed: "+seed+"}\n"+
				"outputs: {picture: s"+seed+".png}\n")))
	}
	if slices.Equal(pictureOf(t, filepath.Join(directory, "s7.png")).Pix,
		pictureOf(t, filepath.Join(directory, "s8.png")).Pix) {
		t.Error("two seeds of one stock drew one picture")
	}
}

// anti-test of TestDrawProgressIsPlain
func TestDrawProgressSaysAKeptFrameKeptNotDrawn(t *testing.T) {
	directory := t.TempDir()
	path := written(t, directory, "film.yaml",
		filmJob(t, "outputs: {frames: film}\n"))
	env := envIn(t, context.Background(), directory)
	drew(t, env, path)
	if err := os.Remove(filepath.Join(directory, "film",
		fmt.Sprintf(frameName, nothing))); err != nil {
		t.Fatal(err)
	}
	said := drew(t, env, path)
	drawn := strings.Count(said, fmt.Sprintf(" of %d: draw ", filmFrames(t)))
	kept := strings.Count(said, ": kept from the last run")
	if drawn != one || kept != filmFrames(t)-one {
		t.Errorf("%d frames said drawn, %d kept: %q", drawn, kept, said)
	}
}
