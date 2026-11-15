package door

import (
	"context"
	"strings"
	"testing"
	"time"
)

// helpOn is the tool's help, and its --age, on the env's clock.
func helpOn(t *testing.T, at time.Time) (string, string) {
	t.Helper()
	env, err := NewEnv(Given{Context: context.Background(),
		Environment: func(string) (string, bool) { return "", false },
		Now:         func() time.Time { return at },
		Directory:   t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	return running(Tool(env), "--help").said,
		running(Tool(env), "--age").said
}

// namesItself is whether a help line's first word is its second: a verb
// whose line begins with its own name.
func namesItself(line string) bool {
	words := strings.Fields(line)
	return len(words) >= pair && words[nothing] == words[one]
}

// neither the tool's line nor any verb's begins with its own name: each
// line's first word, read from the help the tool prints.
func TestNoVerbsLineBeginsWithItsName(t *testing.T) {
	help, _ := helpOn(t, noon())
	if top := strings.Fields(strings.TrimPrefix(help, plainName+": ")); len(
		top) == nothing || strings.EqualFold(strings.Trim(top[nothing],
		"'s"), plainName) {
		t.Errorf("the tool's line begins with its name: %q", help)
	}
	verbs := nothing
	for _, line := range strings.Split(help, "\n") {
		words := strings.Fields(line)
		if !strings.HasPrefix(line, "  ") || len(words) < pair {
			continue
		}
		verbs++
		if namesItself(line) {
			t.Errorf("%q begins with its own name", line)
		}
	}
	if verbs < len([]string{"draw", "list", "tunes", "assess",
		"develop"}) {
		t.Errorf("the help listed %d verbs: %q", verbs, help)
	}
}

// the tool's name on the twenty-fifth of December, by the local date,
// and on the days either side of it.
func TestTheToolsNameOnTheTwentyFifthOfDecember(t *testing.T) {
	kept := time.Local
	t.Cleanup(func() { time.Local = kept })
	time.Local = time.FixedZone("five hours west", -5*60*60)
	for _, each := range []struct {
		at   time.Time
		name string
	}{
		// anti-test of TestTheToolsNameOnTheTwentyFifthOfDecember
		{time.Date(2026, time.December, 24, 12, 0, 0, 0, time.Local),
			"slayer"},
		{time.Date(2026, time.December, 25, 12, 0, 0, 0, time.Local),
			"sleigh"},
		{time.Date(2026, time.December, 26, 12, 0, 0, 0, time.Local),
			"slayer"},
		// the twenty-sixth in UTC, and the twenty-fifth here
		{time.Date(2026, time.December, 26, 2, 0, 0, 0, time.UTC),
			"sleigh"},
		// the twenty-fifth in UTC, and the twenty-fourth here
		{time.Date(2026, time.December, 25, 2, 0, 0, 0, time.UTC),
			"slayer"},
	} {
		help, age := helpOn(t, each.at)
		if !strings.HasPrefix(help, each.name+": ") ||
			!strings.Contains(help, each.name+" VERB [ARGS]") ||
			!strings.HasPrefix(age, each.name) {
			t.Errorf("%v: the help says %q and --age %q, not %s", each.at,
				help, age, each.name)
		}
	}
}
