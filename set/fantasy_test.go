package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// fantasyScene is a set with a fantasy block of the given lines, each
// indented under it.
func fantasyScene(lines string) string {
	return "scene:\n  name: a departure\n  light: moon\n" + lines
}

// readScene is the set a fantasy block of the given lines makes, or why it
// does not, read under a name.
func readScene(name, lines string) (set.Set, error) {
	raw := fantasyScene(lines)
	return set.Read(name, []byte(raw))
}

// The places of the keys of the full block of TestFantasyKeysAreRead, in
// the order it writes them.
const (
	lakeAt      = iota // defined: the first key written is lake
	lakeGlowAt         // defined: the second is lakeglow
	moonAt             // defined: the third is moon
	moonGlowAt         // defined: the fourth is moonglow
	giantsAt           // defined: the fifth is giants
	closeAt            // defined: the sixth is close
	closeGlowAt        // defined: the seventh is closeglow
)

// lakeMetres is the elevation the full block writes for the lake.
const lakeMetres = 1076 // defined: the number the block's text says

// nought is no amount.
const nought = 0 // defined: no amount

// TestFantasyKeysAreRead reads every key of the block and checks its
// value and its place in the order written.
func TestFantasyKeysAreRead(t *testing.T) {
	raw := "  fantasy:\n    lake: 1076\n    lakeglow: true\n" +
		"    moon: 6\n    moonglow: 0.042\n    giants: 5\n    close: 2.5\n" +
		"    closeglow: false\n"
	read, err := readScene("departure.yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		key, value string
	}{{"lake", "1076"}, {"lakeglow", "true"}, {"moon", "6"},
		{"moonglow", "0.042"}, {"giants", "5"}, {"close", "2.5"},
		{"closeglow", "false"}}
	got := read.Fantasy()
	if len(got) != len(want) {
		t.Fatalf("%d departures, want %d", len(got), len(want))
	}
	for i, each := range want {
		if got[i].Key() != each.key || got[i].Value() != each.value {
			t.Errorf("departure %d is %s %s, want %s %s", i,
				got[i].Key(), got[i].Value(), each.key, each.value)
		}
	}
	if got[lakeAt].Number() != lakeMetres || !got[lakeGlowAt].On() || got[closeGlowAt].On() {
		t.Errorf("lake %v, lakeglow %v, closeglow %v", got[lakeAt].Number(),
			got[lakeGlowAt].On(), got[closeGlowAt].On())
	}
}

// TestNoFantasyBlockIsNoDeparture reads a set without the block.
func TestNoFantasyBlockIsNoDeparture(t *testing.T) {
	read, err := readScene("plain.yaml", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := read.Fantasy(); len(got) != nought {
		t.Errorf("%d departures in a set with none", len(got))
	}
}

// TestFantasyRefusesWhatItDoesNotTake refuses an unknown key, saying
// the known ones, and a value outside a key's range.
func TestFantasyRefusesWhatItDoesNotTake(t *testing.T) {
	for _, each := range []refusal{
		{"unknown", "    moonbeam: 2\n",
			"[close closeglow giants lake lakeglow moon moonglow]"},
		{"zero moon", "    moon: 0\n", "moon takes a number"},
		{"negative giants", "    giants: -1\n", "giants takes a number"},
		{"zero close", "    close: 0\n", "close takes a number"},
		{"negative lake", "    lake: -3\n", "lake takes a number"},
		{"negative glow", "    moonglow: -0.5\n", "moonglow takes a number"},
		{"not a number", "    moon: big\n", "moon takes a number"},
		{"nan", "    giants: .nan\n", "giants takes a number"},
		{"not a flag", "    lakeglow: yes please\n",
			"lakeglow takes true or false"},
	} {
		t.Run(each.name, each.check)
	}
}

// refusal is a line of a fantasy block and what a set that has it must be
// refused with.
type refusal struct{ name, line, want string }

// check checks the set with the line is refused with what it should be.
func (each refusal) check(t *testing.T) {
	_, err := readScene("bad.yaml", "  fantasy:\n"+each.line)
	if err == nil || !strings.Contains(err.Error(), each.want) {
		t.Errorf("got %v, want a refusal with %q", err, each.want)
	}
}

// TestFantasyTakesKeysUnderIt refuses a block that is not keys.
func TestFantasyTakesKeysUnderIt(t *testing.T) {
	_, err := readScene("bad.yaml", "  fantasy: 3\n")
	if err == nil || !strings.Contains(err.Error(), "keys under it") {
		t.Errorf("got %v", err)
	}
}
