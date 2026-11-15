package cheeptune

import (
	"bytes"
	"os"
	"testing"
)

// Scoring is SCORING.md as it is on disk.
func TestScoringBytesAreTheFile(t *testing.T) {
	raw, err := os.ReadFile("SCORING.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(Scoring(), raw) {
		t.Error("Scoring is not SCORING.md")
	}
}

// anti-test: a caller that changes what it was given does not change
// what the next caller gets.
func TestScoringBytesAreACopy(t *testing.T) {
	given := Scoring()
	given[0] = 'X'
	if Scoring()[0] == 'X' {
		t.Error("Scoring gave its own bytes to be changed")
	}
}
