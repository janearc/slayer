package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// fourFourFrames is the sha256 of the frames draw writes for the test
// score, in four with its meter unsaid, at two frames a second and 320
// by 180. Measured at b9e5ba20, before the meter was the score's.
const fourFourFrames = "8b55f2a1565157844a6660715ad66210" +
	"02ca1bcb8f61a54eee89a91d8208e446"

// a score in four draws the very frames it drew before the meter was
// read from the score.
func TestFourFourFramesAreUnchanged(t *testing.T) {
	scorePath, cuesPath := writeFiles(t, scoreText, cuesText)
	status, out, errs := runDraw(t, "--score", scorePath, "--cues",
		cuesPath, "--fps", "2", "--width", "320", "--height", "180")
	if status != 0 {
		t.Fatalf("status %d: %s", status, errs)
	}
	sum := sha256.Sum256(out)
	if got := hex.EncodeToString(sum[:]); got != fourFourFrames {
		t.Errorf("four-four frames hash %s, not %s", got, fourFourFrames)
	}
}
