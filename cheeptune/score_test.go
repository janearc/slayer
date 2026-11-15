package cheeptune

import (
	"strconv"
	"strings"
	"testing"
)

// score is a small score around the given parts, as a document.
func score(meter int, loop bool, parts string) []byte {
	l := "false"
	if loop {
		l = "true"
	}
	return []byte(`{"name": "t", "tempo": 120, "meter": ` +
		strconv.Itoa(meter) + `, "loop": ` + l + `, "parts": [` + parts + `]}`)
}

func wantErr(t *testing.T, err error, parts ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("no error; wanted one saying %q", parts)
	}
	for _, p := range parts {
		if !strings.Contains(err.Error(), p) {
			t.Errorf("error does not say %q:\n%v", p, err)
		}
	}
}

func TestABarThatDoesNotAddUpIsRefusedByName(t *testing.T) {
	_, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4 d4 e4 f4", "e5 d5 c5"]}`))
	wantErr(t, err, "lead bar 2: holds 3 beats and the meter is 4")
}

func TestTripletsAddUpExactly(t *testing.T) {
	s, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4:1/3 d4:1/3 e4:1/3 f4:3"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Parts[0].events[3].at; got != (frac{1, 1}) {
		t.Errorf("f4 starts at %s, want 1", got)
	}
}

func TestEveryProblemIsReportedNotJustTheFirst(t *testing.T) {
	_, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4", "x4:0 c4:3"]},
		 {"role": "lute", "bars": ["c4:4"]}`))
	wantErr(t, err, "lead bar 1: holds 1 beats", "lead bar 2: token 1",
		`unknown role "lute"`)
}

func TestATieExtendsTheNoteAcrossTheBar(t *testing.T) {
	s, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4:3 c4~", "c4 d4:3"]}`))
	if err != nil {
		t.Fatal(err)
	}
	ev := s.Parts[0].events
	if len(ev) != 3 {
		t.Fatalf("%d events, want 3: c4, c4 tied, d4", len(ev))
	}
	if ev[1].at != (frac{3, 1}) || ev[1].len != (frac{2, 1}) {
		t.Errorf("tied c4 at %s for %s, want at 3 for 2",
			ev[1].at, ev[1].len)
	}
	if ev[2].at != (frac{5, 1}) {
		t.Errorf("d4 at %s, want 5", ev[2].at)
	}
}

func TestATieToAnotherPitchIsRefused(t *testing.T) {
	_, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4:3 c4~", "d4:4"]}`))
	wantErr(t, err, "the tie from bar 1 lands on \"d4:4\"")
}

func TestAnUnknownFieldIsRefused(t *testing.T) {
	_, err := ParseScore([]byte(`{"name": "t", "tempoo": 120}`))
	wantErr(t, err, "tempoo")
}

func TestPartsHaveTheSameNumberOfBars(t *testing.T) {
	_, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4:4", "c4:4"]},
		 {"role": "bass", "bars": ["c3:4"]}`))
	wantErr(t, err, "bass has 1 bars and lead has 2")
}

func TestPitchIsScientific(t *testing.T) {
	for name, want := range map[string]int{
		"a4": 69, "c4": 60, "bb3": 58, "c#5": 73, "c-1": -1,
	} {
		got, err := pitch(name)
		if name == "c-1" {
			if err == nil {
				t.Errorf("c-1 parsed as %d; negative octaves are not notes",
					got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("%s = %d (%v), want %d", name, got, err, want)
		}
	}
}

func TestCommentsGoAndStringsStay(t *testing.T) {
	b := []byte("{\"name\": \"a // b\", // gone\n\"x\": 1}")
	got := string(stripComments(b))
	want := "{\"name\": \"a // b\", \n\"x\": 1}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKindReadsKeysNotText(t *testing.T) {
	sc := []byte("{\n// no channels here\n\"parts\": []}")
	if k, err := Kind(sc); k != "score" {
		t.Errorf("a score mentioning channels in a comment is %q (%v)",
			k, err)
	}
	if k, _ := Kind([]byte(`{"channels": []}`)); k != "voicing" {
		t.Errorf("voicing read as %q", k)
	}
	_, err := Kind([]byte(`{"layers": []}`))
	wantErr(t, err, "neither a score", "layers")
}

func TestAPartSaysWhatItSounds(t *testing.T) {
	s, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c4:2 -:1 (e4 g4):1", "d4:4"]}`))
	if err != nil {
		t.Fatal(err)
	}
	ev := s.Parts[0].Events()
	if len(ev) != 3 {
		t.Fatalf("%d events, want three: a note, a chord, a note", len(ev))
	}
	if ev[0].Beats != 0 || ev[0].Length != 2 || len(ev[0].Notes) != 1 {
		t.Errorf("first event %+v", ev[0])
	}
	if ev[1].Beats != 3 || len(ev[1].Notes) != 2 || ev[1].Bar != 1 {
		t.Errorf("the chord is %+v", ev[1])
	}
	if ev[2].Beats != 4 || ev[2].Bar != 2 {
		t.Errorf("the second bar's note is %+v", ev[2])
	}
	// the copy is a copy: a caller cannot reach into the score
	ev[0].Notes[0] = 0
	if s.Parts[0].events[0].pitches[0] == 0 {
		t.Error("Events handed out the score's own pitches")
	}
}
