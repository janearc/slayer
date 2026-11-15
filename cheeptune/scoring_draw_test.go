//go:build draw

package cheeptune

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exampleSeconds is how long each example is meant to be, as the page says.
var exampleSeconds = map[string]float64{"ember": 29, "sunrun": 20, "wire": 15}

func TestTheWorkedExamplesRenderWithTheVoicingsThePageNamesForThem(t *testing.T) {
	for name, set := range exampleVoicings {
		path := filepath.Join("examples", name+".json")
		for _, v := range set.works {
			p, err := Open(path, v)
			if err != nil {
				t.Errorf("%s with %s: %v", name, v, err)
				continue
			}
			if p.Peak >= 1 || p.Peak <= 0.05 {
				t.Errorf("%s with %s: peak %v", name, v, p.Peak)
			}
			got := p.Duration().Seconds()
			if math.Abs(got-exampleSeconds[name]) > 1 {
				t.Errorf("%s: %v s, the page says about %v",
					name, got, exampleSeconds[name])
			}
		}
	}
}

func TestTheClippingPairsThePageNamesAreRefused(t *testing.T) {
	for name, set := range exampleVoicings {
		path := filepath.Join("examples", name+".json")
		for _, v := range set.clips {
			_, err := Open(path, v)
			if err == nil || !strings.Contains(err.Error(), "clips") {
				t.Errorf("%s with %s: want a clip, got %v", name, v, err)
			}
		}
	}
	// the built-in pairs the page lists as refused.
	for _, pair := range [][2]string{{"@crypt", "@dmg-echo"},
		{"@crypt", "@overdrive-dmg"}, {"@leap", "@overdrive-dmg"}} {
		if _, err := Open(pair[0], pair[1]); err == nil {
			t.Errorf("%v: the page says refused", pair)
		}
	}
	if _, err := Open("@overdrive", "@crypt-dmg"); err == nil ||
		!strings.Contains(err.Error(), `hit "t" is not one of`) {
		t.Errorf("@overdrive needs @overdrive-dmg: %v", err)
	}
}

func TestAnExampleWithAPartMissingIsRefusedByEveryBuiltInVoicing(t *testing.T) {
	b, err := os.ReadFile("examples/wire.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseScore(b)
	if err != nil {
		t.Fatal(err)
	}
	s.Parts = s.Parts[:2] // lead and pad only
	_, voicings := Builtin()
	for _, v := range voicings {
		vo, err := ReadVoicing(v)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Render(s, vo); err == nil {
			t.Errorf("%s accepted a score with no bass or perc", v)
		}
	}
}

func TestThePairErrorsThePageExplainAreTheOnesReturned(t *testing.T) {
	text := page(t)
	score := func(roles ...string) *Score {
		doc := `{"name":"x","tempo":90,"meter":4,"parts":[`
		for i, r := range roles {
			if i > 0 {
				doc += ","
			}
			bar := `"a3:4"`
			if r == "perc" {
				bar = `"k:4"`
			}
			doc += `{"role":"` + r + `","bars":[` + bar + `]}`
		}
		s, err := ParseScore([]byte(doc + `]}`))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	cases := []struct {
		what     string
		s        *Score
		v        string
		fragment string
	}{
		{"a drop of a missing part", score("lead", "bass", "perc"),
			"@dmg-echo", "drop: the score has no pad part"},
		{"a missing part", score("pad", "bass", "perc"), "@crypt-dmg",
			"the score has no lead part"},
	}
	for _, c := range cases {
		vo, err := ReadVoicing(c.v)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Render(c.s, vo)
		if err == nil || !strings.Contains(err.Error(), c.fragment) {
			t.Errorf("%s: want %q, got %v", c.what, c.fragment, err)
		}
		// the page says it without the leading words of the first one.
		said := strings.TrimPrefix(c.fragment, "drop: ")
		if !strings.Contains(text, said) {
			t.Errorf("%s: SCORING.md does not say %q", c.what, said)
		}
	}
}

func TestAnExampleLoopsAndOneThatDoesNotLoopHasATail(t *testing.T) {
	p, err := Open("examples/ember.json", "@tape-chords")
	if err != nil {
		t.Fatal(err)
	}
	if p.LoopPeriod() != p.Duration() || p.LoopPeriod() == 0 {
		t.Errorf("a looping piece's period is its length: %v vs %v",
			p.LoopPeriod(), p.Duration())
	}
	s, err := ReadScore("examples/ember.json")
	if err != nil {
		t.Fatal(err)
	}
	s.Loop = false
	v, err := ReadVoicing("@tape-chords")
	if err != nil {
		t.Fatal(err)
	}
	q, err := Render(s, v)
	if err != nil {
		t.Fatal(err)
	}
	if q.LoopPeriod() != 0 || q.Duration() <= p.Duration() {
		t.Errorf("a piece that does not loop has period 0 and a tail: "+
			"%v, %v", q.LoopPeriod(), q.Duration())
	}
}
