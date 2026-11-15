package cheeptune

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSaysWhatThePairHolds(t *testing.T) {
	var out strings.Builder
	if err := Check(&out, "@dusk", "@dmg-echo"); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	for _, want := range []string{
		`score "dusk": 92 bpm, 4 beats a bar, 8 bars, 20.87 s, loops`,
		"pad   8 chords, c4 to e5",
		"2 pulse echoes lead  3/4 beats late at 0.35",
		"drops pad",
		"together: every role is played or dropped",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("check does not say %q:\n%s", want, out.String())
		}
	}
}

func TestCheckListsEveryProblemAndFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	doc := `{"name": "x", "tempo": 92, "meter": 4, "parts": [
		{"role": "lead", "bars": ["c4 d4", "c4:4"]},
		{"role": "pad", "bars": ["(c4 e4):4", "q9"]}]}`
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := Check(&out, path); err == nil {
		t.Fatalf("a bad score checked clean:\n%s", out.String())
	}
	for _, want := range []string{
		"problem: lead bar 1: holds 2 beats",
		`problem: pad bar 2: token 1 "q9"`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("check does not say %q:\n%s", want, out.String())
		}
	}
}

func TestCheckRefusesAPairThatDoesNotFit(t *testing.T) {
	var out strings.Builder
	dir := t.TempDir()
	path := filepath.Join(dir, "lead.json")
	if err := os.WriteFile(path, voicing(lead, ""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Check(&out, "@dusk", path); err == nil {
		t.Fatalf("dusk through a lead alone checked clean:\n%s", out.String())
	}
	if !strings.Contains(out.String(),
		"problem: pad: no channel plays it and it is not dropped") {
		t.Errorf("the unplayed pad is not named:\n%s", out.String())
	}
}
