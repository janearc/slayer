package cheeptune

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// These tests cover what a caller reaches only on a bad day: a path that
// cannot be written, a file that is not there, a document that is not a
// document. Each has the near miss beside it.

// dusk is one rendered piece the tests below share, so that a render is
// paid for once.
var dusk = sync.OnceValues(func() (*Piece, error) {
	return Open("@dusk", "@dmg-pad")
})

func TestAnArpIsWrittenBackAsItWasRead(t *testing.T) {
	for _, in := range []string{`3`, `"1/8"`} {
		var a Arp
		if err := json.Unmarshal([]byte(in), &a); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		out, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != in {
			t.Errorf("%s came back as %s", in, out)
		}
	}
	for in, want := range map[string]string{
		`2`: "2 ticks", `"1/4"`: "1/4 beats"} {
		var a Arp
		if err := json.Unmarshal([]byte(in), &a); err != nil {
			t.Fatal(err)
		}
		if a.String() != want {
			t.Errorf("%s reads %q, want %q", in, a.String(), want)
		}
	}
}

func TestAnArpNotSetIsWrittenAsNullAndPrintsItsDefault(t *testing.T) {
	var a Arp
	out, err := json.Marshal(a)
	if err != nil || string(out) != "null" {
		t.Errorf("an unset arp is %s, %v", out, err)
	}
	if a.String() != "3 ticks (default)" {
		t.Errorf("an unset arp reads %q", a.String())
	}
}

func TestAnArpThatIsNeitherTicksNorBeatsIsRefused(t *testing.T) {
	for _, in := range []string{`0`, `61`, `2.5`, `true`, `"x"`, `"0"`} {
		var a Arp
		if err := json.Unmarshal([]byte(in), &a); err == nil {
			t.Errorf("%s was accepted", in)
		}
	}
}

func TestFractionsAreExactAndAboveNought(t *testing.T) {
	if got := mkfrac(2, -4); got != (frac{-1, 2}) {
		t.Errorf("2/-4 is %v", got)
	}
	if got := mkfrac(0, 0); got != (frac{0, 1}) {
		t.Errorf("0/0 is %v", got)
	}
	if abs64(-3) != 3 || abs64(3) != 3 {
		t.Error("abs64")
	}
	if s := (frac{3, 4}).String(); s != "3/4" {
		t.Errorf("string %q", s)
	}
	if s := (frac{3, 1}).String(); s != "3" {
		t.Errorf("string %q", s)
	}
	for _, bad := range []string{"", "x", "1/0", "1/x", "-1", "0", "0/3"} {
		if _, err := parseFrac(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestAMissingFileIsNamedNotGuessedAt(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone.json")
	if _, err := ReadScore(gone); err == nil {
		t.Error("a missing score was read")
	}
	if _, err := ReadVoicing(gone); err == nil {
		t.Error("a missing voicing was read")
	}
	if _, err := Open(gone, "@dmg-pad"); err == nil {
		t.Error("a missing score was opened")
	}
	if _, err := Open("@dusk", gone); err == nil {
		t.Error("a missing voicing was opened")
	}
	if _, err := ReadDoc(gone); err == nil {
		t.Error("a missing document was read")
	}
}

func TestAnythingThatIsNotADocumentIsRefusedByName(t *testing.T) {
	for _, in := range []string{`[]`, `nope`, `{"name":"x"}`,
		`{"parts":[],"channels":[]}`} {
		if k, err := Kind([]byte(in)); err == nil {
			t.Errorf("%s is %q", in, k)
		}
	}
	if _, err := ParseScore([]byte(`{"name":"x"} {}`)); err == nil ||
		!strings.Contains(err.Error(), "more after the document ends") {
		t.Errorf("trailing text: %v", err)
	}
	if _, err := ParseVoicing([]byte(`{`)); err == nil {
		t.Error("half a document was read")
	}
	if _, err := ParseScore([]byte(`{"name":"x"}`)); err == nil {
		t.Error("a score with no parts was read")
	}
}

func TestCheckReportsAFileThatIsNotThereAndAJunkOne(t *testing.T) {
	dir := t.TempDir()
	junk := filepath.Join(dir, "junk.json")
	if err := os.WriteFile(junk, []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := Check(&out, filepath.Join(dir, "gone.json"), junk)
	if err == nil {
		t.Fatal("want an error")
	}
	if n := strings.Count(out.String(), "problem:"); n != 2 {
		t.Errorf("want one problem for each, got %d:\n%s", n, out.String())
	}
	var clean bytes.Buffer
	if err := Check(&clean, "@dusk", "@dmg-pad"); err != nil {
		t.Errorf("a clean pair: %v\n%s", err, clean.String())
	}
	if strings.Contains(clean.String(), "problem:") {
		t.Errorf("a clean pair has problems:\n%s", clean.String())
	}
}

func TestTheMachineRefusesWhatItCannotSound(t *testing.T) {
	if lo, hi := rangeOf("noise"); lo != 0 || hi != 127 {
		t.Errorf("an unpitched kind has no range: %d %d", lo, hi)
	}
	if got := (ramp{}).at(1); got != 0 {
		t.Errorf("an empty ramp is %v", got)
	}
	r := ramp{{0, 0}, {1, 1}}
	if r.at(-1) != 0 || r.at(2) != 1 || r.at(0.5) != 0.5 {
		t.Error("a ramp is flat past its ends and straight between them")
	}
}

func TestTheBuiltInsAreListedEvenWhenNothingIsRead(t *testing.T) {
	scores, voicings := Builtin()
	if len(scores) != 5 || len(voicings) != 15 {
		t.Errorf("%d scores, %d voicings", len(scores), len(voicings))
	}
	if _, err := ReadDoc("@nope"); err == nil ||
		!strings.Contains(err.Error(), "@crypt") {
		t.Errorf("an unknown built-in should list the real ones: %v", err)
	}
}

// voicingWith is a one-channel voicing around the channel text given.
func voicingWith(channel string, extra string) []byte {
	return []byte(`{"name":"v","machine":"gameboy","channels":[` +
		channel + `]` + extra + `}`)
}

func TestEveryRuleAChannelKeepsRefusesItsNearMiss(t *testing.T) {
	cases := []struct{ doc, fragment string }{
		{`{"kind":"flute","role":"lead"}`, "unknown kind"},
		{`{"kind":"pulse","role":"vocal","duty":0.5}`, "unknown role"},
		{`{"kind":"pulse","echo":"vocal","delay":"1","gain":0.5,"duty":0.5}`,
			"echoes unknown role"},
		{`{"kind":"pulse","role":"lead","echo":"pad","duty":0.5}`,
			"a channel does one"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"delay":"1"}`,
			"delay is only for an echo"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"gain":0.5}`,
			"gain is only for an echo"},
		{`{"kind":"pulse","echo":"lead","delay":"x","gain":0.5,"duty":0.5}`,
			"delay:"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"pan":"up"}`,
			"pans are left, right, both"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"detune":150}`,
			"is past a semitone"},
		{`{"kind":"wave","role":"bass","table":"saw","duty":0.5}`,
			"duty is only for a pulse"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"table":"saw"}`,
			"table is only for a wave"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"hits":{}}`,
			"hits are only for noise"},
		{`{"kind":"pulse","role":"lead"}`, "a pulse needs a duty"},
		{`{"kind":"wave","role":"bass"}`, "a wave needs a table"},
		{`{"kind":"wave","role":"bass","table":"abc"}`,
			"is not triangle, saw, square or 32 hex digits"},
		{`{"kind":"wave","role":"bass","table":"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"}`,
			"is not hex"},
		{`{"kind":"noise","role":"perc","hits":{"k1":{"decay":0.1}}}`,
			"a hit is a lowercase word"},
		{`{"kind":"noise","role":"perc","hits":{"k":{"decay":0.1,"shift":14}}}`,
			"shift 14 is outside 0 to 13"},
		{`{"kind":"noise","role":"perc","hits":{"k":{"decay":0.1,"divisor":8}}}`,
			"divisor 8 is outside 0 to 7"},
		{`{"kind":"noise","role":"perc","hits":{"k":{"decay":9}}}`,
			"decay 9 s is outside 0 to 4"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"envelope":{"attack":-1}}`,
			"envelope times are not below nought"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"envelope":{"sustain":2}}`,
			"sustain 2 is outside 0 to 1"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"vibrato":{"rate":30}}`,
			"vibrato rate"},
		{`{"kind":"pulse","role":"lead","duty":0.5,"vibrato":{"rate":5,"depth":200}}`,
			"vibrato depth"},
	}
	for _, c := range cases {
		_, err := ParseVoicing(voicingWith(c.doc, ""))
		if err == nil || !strings.Contains(err.Error(), c.fragment) {
			t.Errorf("%s: want %q, got %v", c.doc, c.fragment, err)
		}
	}
	good := `{"kind":"wave","role":"bass","table":"0123456789abcdef0123456789abcdef"}`
	if _, err := ParseVoicing(voicingWith(good, "")); err != nil {
		t.Errorf("a table of 32 hex digits: %v", err)
	}
	pulse := `{"kind":"pulse","role":"lead","duty":0.5}`
	for extra, fragment := range map[string]string{
		`,"drop":["vocal"]`: "drop: unknown role",
		`,"drop":["lead"]`:  "is dropped and also played",
	} {
		_, err := ParseVoicing(voicingWith(pulse, extra))
		if err == nil || !strings.Contains(err.Error(), fragment) {
			t.Errorf("%s: want %q, got %v", extra, fragment, err)
		}
	}
	if _, err := ParseVoicing(voicingWith(pulse, `,"drop":["pad"]`)); err != nil {
		t.Errorf("dropping a role nothing plays: %v", err)
	}
}
