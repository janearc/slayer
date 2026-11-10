package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// hudded is the kilauea iki set file with a hud of every kind on it.
func hudded(t *testing.T) string {
	t.Helper()
	const at = "  light: moon\n"
	return strings.Replace(kilaueaIki(t), at, at+
		"  hud:\n"+
		"    theme: themes/dusk.css\n"+
		"    rows: 30\n"+
		"    elements:\n"+
		"      edge:\n"+
		"        kind: frame\n"+
		"        left: 0.01\n        top: 0.02\n"+
		"        right: 0.98\n        bottom: 0.97\n"+
		"        ink: border\n"+
		"      title:\n"+
		"        kind: text\n"+
		"        words: kīlauea iki\n"+
		"        across: 0.5\n        down: 0\n        align: centre\n"+
		"        ink: ink\n        ground: ground\n"+
		"      burning:\n"+
		"        kind: bar\n"+
		"        left: 0.1\n        top: 0.9\n        right: 0.6\n"+
		"        bottom: 0.95\n"+
		"        value:\n          from: 0\n          to: 1\n"+
		"          start: 2\n          over: 8\n"+
		"        fill: warn\n        empty: surface-1\n"+
		"      clock:\n"+
		"        kind: text\n"+
		"        count:\n          from: 0\n          to: 10\n"+
		"          over: 10\n"+
		"        format: \"%04.1f s\"\n"+
		"        across: 0.62\n        down: 0.9\n"+
		"        ink: ink\n"+
		"      tree:\n"+
		"        kind: label\n"+
		"        of: the first tree\n        above: 12\n"+
		"        words: the first tree\n        occluded: true\n"+
		"        ink: ink\n        ground: ground\n", 1)
}

// a set's hud is what it says: its theme, its rows, and each element in
// the order written with every part it says.
func TestASetsHudIsWhatItSays(t *testing.T) {
	made, err := set.Read("kilauea-iki.yaml", []byte(hudded(t)))
	if err != nil {
		t.Fatal(err)
	}
	read, has := made.HudOf()
	if !has || read.Theme() != "themes/dusk.css" || read.Rows() != 30 {
		t.Fatalf("hud %v: %+v", has, read)
	}
	elements := read.Elements()
	var names []string
	for _, each := range elements {
		names = append(names, each.Name()+" "+each.Kind())
	}
	if strings.Join(names, ", ") != "edge frame, title text, burning bar, "+
		"clock text, tree label" {
		t.Errorf("elements %v", names)
	}
	left, top, right, bottom := elements[0].Box()
	ink, _, _, _ := elements[0].Roles()
	if left != 0.01 || top != 0.02 || right != 0.98 || bottom != 0.97 ||
		ink != "border" {
		t.Errorf("the edge %+v", elements[0])
	}
	words, _, counted, _ := elements[1].Words()
	across, down, align := elements[1].At()
	if words != "kīlauea iki" || counted || across != 0.5 || down != 0 ||
		align != "centre" {
		t.Errorf("the title %+v", elements[1])
	}
	_, _, fill, empty := elements[2].Roles()
	if v := elements[2].Value(); v != (set.Value{From: 0, To: 1, Start: 2,
		Over: 8}) || fill != "warn" || empty != "surface-1" {
		t.Errorf("the bar %+v", elements[2])
	}
	_, format, counted, count := elements[3].Words()
	if !counted || format != "%04.1f s" || count.To != 10 ||
		count.Over != 10 {
		t.Errorf("the clock %+v", elements[3])
	}
	of, above, occluded := elements[4].Pin()
	if of != "the first tree" || above != 12 || !occluded {
		t.Errorf("the label %+v", elements[4])
	}
}

// a hud that says less takes 24 rows and aligns words left, and a value
// said as a number alone is from and to both; a set with no hud has none.
func TestAHudSaysWhatItLeavesOut(t *testing.T) {
	written := strings.Replace(hudded(t), "    rows: 30\n", "", 1)
	written = strings.Replace(written, "        align: centre\n", "", 1)
	ramp := "        value:\n          from: 0\n          to: 1\n" +
		"          start: 2\n          over: 8\n"
	written = strings.Replace(written, ramp,
		"        value: 0.25\n", 1)
	made, err := set.Read("kilauea-iki.yaml", []byte(written))
	if err != nil {
		t.Fatal(err)
	}
	read, _ := made.HudOf()
	_, _, align := read.Elements()[1].At()
	if read.Rows() != 24 || align != "left" ||
		read.Elements()[2].Value() != (set.Value{From: 0.25,
			To: 0.25}) {
		t.Errorf("hud %+v", read)
	}
	if _, has := kilaueaIkiRead(t).HudOf(); has {
		t.Error("a set with no hud has one")
	}
}

// a hud's mistakes are refused where they are written.
func TestAHudsMistakesAreRefused(t *testing.T) {
	for _, mistake := range []struct{ was, now, said string }{
		{"    theme: themes/dusk.css\n", "", "says its theme"},
		{"        kind: frame\n", "", "says its kind"},
		{"        kind: frame\n", "        kind: dial\n", "kind takes"},
		{"        ink: border\n", "", "a frame says its"},
		{"        fill: warn\n", "", "a bar says its"},
		{"        words: kīlauea iki\n", "", "its words or its count"},
		{"        format: \"%04.1f s\"\n", "",
			"a count says its format"},
		{"        words: the first tree\n", "",
			"its words or its count"},
		{"of: the first tree", "of: the second tree", "pinned to"},
		{"align: centre", "align: justified", "align takes"},
		{"          to: 1\n", "", "says its from and its to"},
		{"right: 0.6", "right: 1.6",
			"right takes a number from 0 to 1"},
		{"occluded: true", "occluded: maybe", "occluded takes true"},
		{"    rows: 30\n", "    rows: 1\n", "rows takes"},
	} {
		was, now := mistake.was, mistake.now
		written := strings.Replace(hudded(t), was, now, 1)
		_, err := set.Read("kilauea-iki.yaml", []byte(written))
		if err == nil || !strings.Contains(err.Error(), mistake.said) {
			t.Errorf("%q for %q: refused with %v, wanted %q",
				now, was, err, mistake.said)
		}
	}
}

// a hud's numbers are taken to their edges and refused past them: rows from 2
// to 500, a label from ten kilometres under its thing to ten over, a value from
// minus to plus a million million, and a start and a span from zero, or half a
// second, to a thousand million seconds.
func TestAHudsNumbersAreTakenToTheirEdges(t *testing.T) {
	ramp := "        value:\n          from: 0\n          to: 1\n" +
		"          start: 2\n          over: 8\n"
	const v = "        value:\n          from: "
	for _, edge := range []struct {
		was, now string
		taken    bool
	}{{"    rows: 30\n", "    rows: 2\n", true},
		{"    rows: 30\n", "    rows: 500\n", true},
		{"    rows: 30\n", "    rows: 501\n", false},
		{"above: 12", "above: 10000", true},
		{"above: 12", "above: -10000", true},
		{"above: 12", "above: 10001", false},
		{"above: 12", "above: -10001", false},
		{ramp, v + "-1e12\n          to: 1e12\n" +
			"          start: 0\n          over: 0.5\n", true},
		{ramp, v + "1\n          to: 1\n" +
			"          start: 1e9\n          over: 1e9\n", true},
		{ramp, v + "1.1e12\n          to: 1\n", false},
		{ramp, v + "0\n          to: -1.1e12\n", false},
		{ramp, v + "0\n          to: 1\n" +
			"          start: 1.1e9\n", false},
		{ramp, v + "0\n          to: 1\n" +
			"          over: 1.1e9\n", false},
		{ramp, "        value: 1.1e12\n", false},
		{ramp, "        value: -1.1e12\n", false},
		{ramp, v + "-1.1e12\n          to: 1\n", false},
		{ramp, v + "0\n          to: 1.1e12\n", false},
	} {
		written := strings.Replace(hudded(t), edge.was, edge.now, 1)
		_, err := set.Read("kilauea-iki.yaml", []byte(written))
		if (err == nil) != edge.taken {
			t.Errorf("%q: %v", edge.now, err)
		}
	}
}
