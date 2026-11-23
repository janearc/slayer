//go:build draw

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/improv/walk"
)

// readShort reads the script and returns its short named where.
func readShort(t *testing.T, path string) walk.Short {
	t.Helper()
	script, err := walk.ReadScript(path, "")
	if err != nil {
		t.Fatal(err)
	}
	short, found := script.Short("where")
	if !found {
		t.Fatal("the script has no short where")
	}
	return short
}

// TestSoundFor checks the span under a short: none for a silent short, the
// score's span for a sounded one, and each way it is refused.
func TestSoundFor(t *testing.T) {
	silent := readShort(t, writeScript(t, ""))
	if sound, err := soundFor(silent, asked{}); err != nil || sound.Rate() != 0 {
		t.Errorf("silent short: %d, %v", sound.Rate(), err)
	}
	sounded := readShort(t, writeScript(t, "    sound: {bar: 2}"))
	score := "@glass-hour"
	got, err := soundFor(sounded, asked{score: score, voicing: "@night-pad"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Duration() != 2*time.Second {
		t.Errorf("span is %v, want 2s", got.Duration())
	}
	cases := map[string]asked{
		"no score":    {voicing: "@night-pad"},
		"no voicing":  {score: score},
		"bad score":   {score: filepath.Join(t.TempDir(), "gone"), voicing: "@night-pad"},
		"bad voicing": {score: score, voicing: "@no-such-voicing"},
	}
	for name, given := range cases {
		_, err := soundFor(sounded, given)
		if err == nil {
			t.Errorf("%s: no error", name)
		}
		if name == "no score" && !errors.Is(err, errAsked) {
			t.Errorf("%s: not an asked error: %v", name, err)
		}
	}
	late := readShort(t, writeScript(t, "    sound: {bar: 9999}"))
	if _, err := soundFor(late, asked{score: score, voicing: "@night-pad"}); !errors.Is(err, errAsked) {
		t.Errorf("a bar the score lacks: %v", err)
	}
}

// TestRunWritesShort writes a two second short with the score under it and
// checks the film is there.
func TestRunWritesShort(t *testing.T) {
	script := writeScript(t, "    sound: {bar: 2}")
	film := filepath.Join(t.TempDir(), "short.mp4")
	given := asked{script: script, short: "where", frame: -1, mp4: film,
		score: "@glass-hour", voicing: "@night-pad"}
	if err := run(given); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(film); err != nil || info.Size() == 0 {
		t.Fatalf("film: %v", err)
	}
	silent := asked{script: writeScript(t, ""), short: "where", frame: -1,
		mp4: filepath.Join(t.TempDir(), "gone", "short.mp4")}
	if err := run(silent); err == nil {
		t.Error("wrote a film into a folder that is not there")
	}
}
