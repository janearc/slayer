//go:build draw

package main

import (
	"bytes"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

// TestSoundForSilenceAndTune checks the silent case and a tune cut under
// the short, with the level of each bar said.
func TestSoundForSilenceAndTune(t *testing.T) {
	var out bytes.Buffer
	silent, err := soundFor(asked{bar: 1}, 5, &out)
	if err != nil || silent.Length() != 0 || out.Len() != 0 {
		t.Fatalf("silence: %d samples, %q, %v", silent.Length(), out.String(), err)
	}
	song, err := soundFor(asked{score: "@crypt", voicing: "@crypt-dmg", bar: 2}, 5, &out)
	if err != nil {
		t.Fatal(err)
	}
	if got := song.Duration().Seconds(); math.Abs(got-5) > 0.001 {
		t.Errorf("span is %.3f s, want 5", got)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// A bar of this tune is 1.846 s, so bars 2, 3 and 4 start inside the
	// span that begins at bar 2 and runs five seconds; bar 5 does not.
	if len(lines) != 3 {
		t.Fatalf("got %d bar lines: %q", len(lines), out.String())
	}
	for i, prefix := range []string{"bar 2 at 1.846 s: level ", "bar 3 at 3.692 s: level ", "bar 4 at 5.538 s: level "} {
		if !strings.HasPrefix(lines[i], prefix) {
			t.Errorf("line %d = %q, want prefix %q", i, lines[i], prefix)
		}
	}
}

// TestSoundForRefusals covers a missing score, and a span longer than the
// song.
func TestSoundForRefusals(t *testing.T) {
	var out bytes.Buffer
	missing := filepath.Join(t.TempDir(), "none.json")
	if _, err := soundFor(asked{score: missing, voicing: "@crypt-dmg", bar: 1}, 5, &out); err == nil {
		t.Error("a score that is not there was taken")
	}
	if _, err := soundFor(asked{score: "@crypt", voicing: "@crypt-dmg", bar: 1}, 1e6, &out); err == nil ||
		!strings.Contains(err.Error(), "start it earlier") {
		t.Errorf("a span of a million seconds: err = %v", err)
	}
}
