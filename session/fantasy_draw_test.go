//go:build draw

package session

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/scene/thing"
)

// fantasyFixture is the fixture scene with a moon, three named trees (one
// of the rim's, one close, one other) and the given fantasy lines under
// scene; and the set file's path.
func fantasyFixture(t *testing.T, lines string) string {
	t.Helper()
	path := onFixture(t, "  terrain:", "  moon:\n    azimuth: 90\n"+
		"    altitude: 10\n"+lines+"  terrain:")
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	trees := "    things:\n" +
		"      rim one: {recipe: fixture-spruce, seed: 5, east: 1040, north: 8950}\n" +
		"      close one: {recipe: fixture-spruce, seed: 6, east: 1060, north: 8950}\n"
	more := strings.Replace(string(written), "    things:\n", trees, 1)
	if err := os.WriteFile(path, []byte(more), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// fingerprint is every thing of a world as text: its extent, its
// material's name, colour and glow, and where it stands.
func fingerprint(world *thing.World) string {
	var lines strings.Builder
	for _, each := range world.Things {
		material := "none"
		if each.Material != nil {
			material = fmt.Sprintf("%s %v %v", each.Material.Name,
				each.Material.Colour, each.Material.Glow)
		}
		fmt.Fprintf(&lines, "%v|%s|%v\n", each.Shape.Bounds(), material,
			each.At(0))
	}
	return lines.String()
}

// stagedWith stages the fantasy fixture with the lines in the set and
// the arguments as flags, and gives the world's fingerprint and the
// tally.
func stagedWith(t *testing.T, lines string, arguments ...string) (string,
	[]string) {
	t.Helper()
	given := mustParse(t, arguments...)
	given.set = fantasyFixture(t, lines)
	world, _, ready, err := staged(given)
	if err != nil {
		t.Fatal(err)
	}
	if ready.live != nil {
		defer ready.live.closeBeds()
	}
	return fingerprint(world), ready.departures
}

// fantasyCases are the keys: the YAML line, the flag that says the same,
// and a second flag value for the override.
var fantasyCases = []struct {
	name, line string
	flags      []string
	other      []string
	overridden string
}{
	{"lake", "    lake: 1500\n", []string{"-lake", "1500"},
		[]string{"-lake", "1700"}, "lake"},
	{"lakeglow", "    lake: 1500\n    lakeglow: true\n",
		[]string{"-lake", "1500", "-lakeglow"},
		[]string{"-lake", "1500", "-lakeglow=false"}, "lakeglow"},
	{"moon", "    moon: 6\n", []string{"-moon", "6"},
		[]string{"-moon", "3"}, "moon"},
	{"moonglow", "    moonglow: 0.25\n", []string{"-moonglow", "0.25"},
		[]string{"-moonglow", "0.5"}, "moonglow"},
	{"giants", "    giants: 3\n", []string{"-giants", "3"},
		[]string{"-giants", "2"}, "giants"},
	{"close", "    close: 2\n", []string{"-close", "2"},
		[]string{"-close", "3"}, "close"},
	{"closeglow", "    closeglow: true\n", []string{"-closeglow"},
		[]string{"-closeglow=false"}, "closeglow"},
}

// TestFantasyKeyStagesTheWorldTheFlagStages stages each key as YAML and
// as its flag and wants the same world, and a world unlike the plain one.
func TestFantasyKeyStagesTheWorldTheFlagStages(t *testing.T) {
	plain, _ := stagedWith(t, "")
	for _, each := range fantasyCases {
		t.Run(each.name, func(t *testing.T) {
			byFlag, _ := stagedWith(t, "", each.flags...)
			byFile, _ := stagedWith(t, "  fantasy:\n"+each.line)
			if byFlag == plain {
				t.Fatal("the flag changes nothing in this fixture")
			}
			if byFile != byFlag {
				t.Errorf("the key stages a world unlike the flag's:\n%s\nflag:\n%s",
					byFile, byFlag)
			}
		})
	}
}

// TestFantasyFlagOverridesTheKey gives each key and a flag that says
// another value; the flag's world is drawn and the tally says so.
func TestFantasyFlagOverridesTheKey(t *testing.T) {
	for _, each := range fantasyCases {
		t.Run(each.name, func(t *testing.T) {
			byFlag, _ := stagedWith(t, "", each.other...)
			both, lines := stagedWith(t, "  fantasy:\n"+each.line,
				each.other...)
			if both != byFlag {
				t.Errorf("the key beat the flag:\n%s\nflag:\n%s", both,
					byFlag)
			}
			said := strings.Join(lines, "\n")
			if !strings.Contains(said, "the -"+each.overridden+" flag overrides") {
				t.Errorf("the tally does not say the flag won: %q", said)
			}
		})
	}
}

// TestFantasyLinesAreReported wants one line a departure, in the words
// the report uses, and none for a set with no block.
func TestFantasyLinesAreReported(t *testing.T) {
	_, lines := stagedWith(t, "  fantasy:\n    moon: 6\n    lakeglow: true\n"+
		"    lake: 1500\n")
	want := []string{
		"fantasy: moon = 6 departs from the measured or derived value",
		"fantasy: lakeglow = true departs from the measured or derived value",
		"fantasy: lake = 1500 departs from the measured or derived value",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("lines %q, want %q", lines, want)
	}
	_, lines = stagedWith(t, "")
	if len(lines) != 0 {
		t.Errorf("a set with no block says %q", lines)
	}
}

// TestFantasyOverrideLineNamesBothValues words the override line.
func TestFantasyOverrideLineNamesBothValues(t *testing.T) {
	_, lines := stagedWith(t, "  fantasy:\n    moon: 6\n", "-moon", "3")
	want := "fantasy: moon = 3 (the -moon flag overrides the set's 6) " +
		"departs from the measured or derived value"
	if strings.Join(lines, "\n") != want {
		t.Errorf("line %q, want %q", lines, want)
	}
}

// TestRunPrintsFantasyLines draws a set with a block and wants the lines
// in what the command says, after the tally.
func TestRunPrintsFantasyLines(t *testing.T) {
	path := fantasyFixture(t, "  fantasy:\n    moon: 6\n")
	var said, wrong strings.Builder
	out := t.TempDir() + "/f.png"
	code := run([]string{"-set", path, "-out", out}, &said, &wrong)
	if code != 0 {
		t.Fatalf("drawing gave %d: %s", code, wrong.String())
	}
	line := "fantasy: moon = 6 departs from the measured or derived value"
	tallied := strings.Index(said.String(), "lit by")
	if !strings.Contains(said.String()[tallied:], "\n"+line+"\n") {
		t.Errorf("the line is not after the tally: %q", said.String())
	}
}
