package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// titled is the kilauea iki set file with a text on it, every key said.
func titled(t *testing.T) string {
	t.Helper()
	return strings.Replace(kilaueaIki(t), "    stands:\n", "    texts:\n"+
		"      the title:\n"+
		"        words: kīlauea iki\n"+
		"        font: go-bold\n"+
		"        size: 6\n"+
		"        depth: 1.5\n"+
		"        bevel: 0.1\n"+
		"        smooth: 30\n"+
		"        tolerance: 1.5\n"+
		"        colour:\n"+
		"          red: 0.7\n"+
		"          green: 0.4\n"+
		"          blue: 0.2\n"+
		"        gloss: 0.3\n"+
		"        east: 264790\n"+
		"        north: 2147950\n"+
		"        up: 2\n"+
		"        facing: 135\n"+
		"        spin: 12\n"+
		"    stands:\n", 1)
}

// a set's text is what it says: its words, its font, its measures, its
// colour, where it stands, which way it faces and how fast it turns; and
// a camera can point at it by its name.
func TestASetsTextIsWhatItSays(t *testing.T) {
	written := strings.Replace(titled(t), "point_to: the first tree",
		"point_to: the title", 1)
	made, err := set.Read("kilauea-iki.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	texts := made.Texts()
	if len(texts) != 1 {
		t.Fatalf("%d texts", len(texts))
	}
	text := texts[0]
	size, depth, bevel, smooth, tolerance := text.Measures()
	red, green, blue, gloss := text.Colour()
	east, north, up := text.At()
	bearing, spin := text.Facing()
	if text.Name() != "the title" || text.Words() != "kīlauea iki" ||
		text.Font() != "go-bold" || size != 6 || depth != 1.5 ||
		bevel != 0.1 || smooth != 30 || tolerance != 1.5 ||
		red != 0.7 || green != 0.4 || blue != 0.2 || gloss != 0.3 ||
		east != 264790 || north != 2147950 || up != 2 ||
		bearing != 135 || spin != 12 {
		t.Errorf("text %+v", text)
	}
}

// a text that says only what letters need faces south, still, on the
// ground, a pale grey with no gloss; a set with no texts has none.
func TestATextSaysWhatItLeavesOut(t *testing.T) {
	written := strings.Replace(kilaueaIki(t), "    stands:\n",
		"    texts:\n"+
			"      plain:\n        words: hi\n        font: go\n"+
			"        size: 1\n        depth: 0.2\n    stands:\n", 1)
	made, err := set.Read("kilauea-iki.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	text := made.Texts()[0]
	bearing, spin := text.Facing()
	_, _, up := text.At()
	red, green, blue, gloss := text.Colour()
	if bearing != 180 || spin != 0 || up != 0 || red != 0.8 ||
		green != 0.8 || blue != 0.8 || gloss != 0 {
		t.Errorf("text %+v", text)
	}
	if got := kilaueaIkiRead(t).Texts(); len(got) != 0 {
		t.Errorf("a set with no texts has %d", len(got))
	}
}

// a text's mistakes are refused where they are written: no words, no
// font, no size or depth, a bevel of half its depth, a colour past one
// or with a part left out, a facing past a whole turn, a key it does
// not know.
func TestATextsMistakesAreRefused(t *testing.T) {
	const all = "words, font, size and depth"
	for _, mistake := range []struct{ was, now, said string }{
		{"        words: kīlauea iki\n", "", all},
		{"        font: go-bold\n", "", all},
		{"        size: 6\n", "", all},
		{"        depth: 1.5\n", "", all},
		{"bevel: 0.1", "bevel: 0.75", "half its depth"},
		{"red: 0.7", "red: 1.2", "red takes a number from 0 to 1"},
		{"          blue: 0.2\n", "", "says its red, green and blue"},
		{"facing: 135", "facing: 400", "facing takes a number"},
		{"spin: 12", "spinning: 12", "spinning is not a key here"},
		{"size: 6", "size: 0", "size takes a number"},
	} {
		was, now := mistake.was, mistake.now
		written := strings.Replace(titled(t), was, now, 1)
		_, err := set.Read("kilauea-iki.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q for %q: refused with %v, wanted %q",
				now, was, err, mistake.said)
		}
	}
}
