package cheeptune

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The page SCORING.md is for a model that has never seen the format. These
// tests keep it true: the examples it quotes parse and render, the built-in
// names it lists exist, the errors it explains are the errors returned, and
// the arithmetic in its length table is right.

// exampleVoicings is, for each worked example, the built-in voicings the page
// says work with it, and the ones it says clip.
var exampleVoicings = map[string]struct{ works, clips []string }{
	"ember": {
		works: []string{"@crypt-dmg", "@dmg-echo", "@dmg-pad",
			"@leap-dmg", "@tape-chords", "@tape-echo"},
		clips: []string{"@overdrive-dmg"},
	},
	"sunrun": {
		works: []string{"@crypt-dmg", "@dmg-pad", "@leap-dmg",
			"@overdrive-dmg", "@tape-chords", "@tape-echo"},
		clips: []string{"@dmg-echo"},
	},
	"wire": {
		works: []string{"@crypt-dmg", "@dmg-echo", "@dmg-pad",
			"@leap-dmg", "@overdrive-dmg", "@tape-chords",
			"@tape-echo"},
	},
}

// page is the text of SCORING.md.
func page(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("SCORING.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestThePageQuotesEachExampleFileWhole(t *testing.T) {
	text := page(t)
	for name := range exampleVoicings {
		b, err := os.ReadFile(filepath.Join("examples", name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(text, strings.TrimRight(string(b), "\n")) {
			t.Errorf("SCORING.md does not quote examples/%s.json", name)
		}
	}
}

func TestThePageListsEveryBuiltInAndNothingElse(t *testing.T) {
	text := page(t)
	scores, voicings := Builtin()
	for _, name := range append(scores, voicings...) {
		if !strings.Contains(text, "| `"+name+"` |") {
			t.Errorf("SCORING.md has no row for %s", name)
		}
	}
	// every name the page puts at the start of a table row is real.
	known := map[string]bool{}
	for _, name := range append(scores, voicings...) {
		known["| `"+name+"` |"] = true
	}
	for line := range strings.SplitSeq(text, "\n") {
		cell, _, _ := strings.Cut(line, "` |")
		if strings.HasPrefix(cell, "| `@") && !known[cell+"` |"] {
			t.Errorf("SCORING.md lists a built-in that is not there: %s",
				cell)
		}
	}
}

// badDocuments are mistakes a model makes, the error each one gets, and the
// words the page uses for that error. The fragment has to be in both.
var badDocuments = []struct {
	what, doc, fragment string
	voicing             bool
}{
	{"no name", `{"tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:4"]}]}`,
		"score has no name", false},
	{"tempo", `{"name":"x","tempo":20,"meter":4,"parts":[{"role":"lead","bars":["a4:4"]}]}`,
		"is outside 40 to 300", false},
	{"meter", `{"name":"x","tempo":90,"meter":0,"parts":[{"role":"lead","bars":["a4:4"]}]}`,
		"is outside 1 to 16", false},
	{"no parts", `{"name":"x","tempo":90,"meter":4,"parts":[]}`,
		"score has no parts", false},
	{"role", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"vocal","bars":["a4:4"]}]}`,
		"unknown role", false},
	{"twice", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:4"]},{"role":"lead","bars":["a4:4"]}]}`,
		"the role appears twice", false},
	{"bar count", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:4","a4:4"]},{"role":"bass","bars":["a2:4"]}]}`,
		"every part has the same", false},
	{"sum", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4 a4 a4"]}]}`,
		"holds 3 beats and the meter is 4", false},
	{"empty bar", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":[""]}]}`,
		"the bar is empty", false},
	{"token", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["x1:4"]}]}`,
		"not a note (a4, c#5, bb3), chord, rest (-) or hit (k)", false},
	{"octave", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a10:4"]}]}`,
		"is past 9", false},
	{"one note chord", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["(a4):4"]}]}`,
		"a chord has two notes or more", false},
	{"zero length", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:0 a4:4"]}]}`,
		"is not above nought", false},
	{"rest tie", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:3 -~"]}]}`,
		"a rest does not tie", false},
	{"tie not last", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:2~ a4:2"]}]}`,
		"a tie is only on a bar's last token", false},
	{"tie lands wrong", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:4~","b4:4"]}]}`,
		"not the same pitch", false},
	{"tie off the end", `{"name":"x","tempo":90,"meter":4,"parts":[{"role":"lead","bars":["a4:4~"]}]}`,
		"ties into a bar that is not there", false},
	{"unknown field", `{"name":"x","tempo":90,"meter":4,"volume":3,"parts":[{"role":"lead","bars":["a4:4"]}]}`,
		"unknown field", false},
	{"no name voicing", `{"machine":"gameboy","channels":[{"kind":"wave","role":"bass","table":"saw"}]}`,
		"voicing has no name", true},
	{"machine", `{"name":"v","machine":"sid","channels":[{"kind":"wave","role":"bass","table":"saw"}]}`,
		"unknown machine", true},
	{"third pulse", `{"name":"v","machine":"gameboy","channels":[{"kind":"pulse","role":"lead","duty":0.5},{"kind":"pulse","role":"pad","duty":0.5},{"kind":"pulse","role":"bass","duty":0.5}]}`,
		"the gameboy has 2 pulse channels and this is one more", true},
	{"duty", `{"name":"v","machine":"gameboy","channels":[{"kind":"pulse","role":"lead","duty":0.3}]}`,
		"the hardware has 0.125, 0.25, 0.5, 0.75", true},
	{"echo no delay", `{"name":"v","machine":"gameboy","channels":[{"kind":"pulse","echo":"lead","duty":0.5}]}`,
		"an echo needs a delay in beats", true},
	{"noise no hits", `{"name":"v","machine":"gameboy","channels":[{"kind":"noise","role":"perc"}]}`,
		"noise with no hits plays nothing", true},
	{"envelope on noise", `{"name":"v","machine":"gameboy","channels":[{"kind":"noise","role":"perc","envelope":{"decay":1},"hits":{"k":{"decay":0.1}}}]}`,
		"envelope, vibrato, arp and detune are for pitched channels", true},
	{"two plays", `{"name":"v","machine":"gameboy","channels":[{"kind":"pulse","role":"lead","duty":0.5},{"kind":"pulse","role":"lead","duty":0.5}]}`,
		"is already played by another channel", true},
	{"volume", `{"name":"v","machine":"gameboy","channels":[{"kind":"pulse","role":"lead","duty":0.5,"volume":2}]}`,
		"is outside 0 to 1", true},
}

func TestEveryErrorThePageExplainsIsTheErrorReturned(t *testing.T) {
	text := page(t)
	for _, c := range badDocuments {
		var err error
		if c.voicing {
			_, err = ParseVoicing([]byte(c.doc))
		} else {
			_, err = ParseScore([]byte(c.doc))
		}
		if err == nil || !strings.Contains(err.Error(), c.fragment) {
			t.Errorf("%s: want %q, got %v", c.what, c.fragment, err)
		}
		if !strings.Contains(text, c.fragment) {
			t.Errorf("%s: SCORING.md does not say %q", c.what, c.fragment)
		}
	}
}

func TestAScoreWithManyMistakesIsToldAllOfThemAtOnce(t *testing.T) {
	doc := `{"name":"","tempo":20,"meter":0,"parts":[{"role":"vocal","bars":["a4"]}]}`
	_, err := ParseScore([]byte(doc))
	if err == nil {
		t.Fatal("want errors")
	}
	for _, want := range []string{"score has no name", "tempo 20",
		"meter 0", "unknown role"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not hold %q: %v", want, err)
		}
	}
}

// lengthRows are the page's table of bars for a length at a tempo: tempo,
// then bars for 10, 20, 30 and 60 seconds, and the seconds they make.
var lengthRows = []struct {
	tempo float64
	bars  [4]int
}{
	{70, [4]int{3, 6, 9, 18}}, {90, [4]int{4, 8, 11, 22}},
	{120, [4]int{5, 10, 15, 30}}, {140, [4]int{6, 12, 17, 35}},
	{160, [4]int{7, 13, 20, 40}},
}

func TestTheLengthTableIsRightAndEachRowIsNearItsTarget(t *testing.T) {
	text := page(t)
	targets := [4]float64{10, 20, 30, 60}
	for _, row := range lengthRows {
		for i, bars := range row.bars {
			secs := float64(bars) * 4 * 60 / row.tempo
			cell := fmt.Sprintf("%d bars, %.1f s", bars, secs)
			if !strings.Contains(text, cell) {
				t.Errorf("tempo %v: the page has no %q", row.tempo, cell)
			}
			if math.Abs(secs-targets[i]) > 2 {
				t.Errorf("tempo %v: %d bars is %.1f s, aim %v",
					row.tempo, bars, secs, targets[i])
			}
		}
	}
}
