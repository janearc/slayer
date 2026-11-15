package set_test

import (
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

// TestASetSaysItsFlagsOrLeavesThemUnsaid reads vibes and fastlight
// said on, said off and unsaid, and a name that is no flag.
func TestASetSaysItsFlagsOrLeavesThemUnsaid(t *testing.T) {
	plain := fireRecipeFixture(t)
	for _, each := range []struct {
		said string
		want map[string][2]bool
	}{{"", map[string][2]bool{"vibes": {}, "fastlight": {}}},
		{"  vibes: on\n  fastlight: off\n", map[string][2]bool{
			"vibes": {true, true}, "fastlight": {false, true},
			"lamp-store": {}}},
		{"  fastlight: on\n", map[string][2]bool{"vibes": {},
			"fastlight": {true, true}}},
	} {
		raw := strings.Replace(plain, "  light: none\n",
			each.said+"  light: none\n", 1)
		read, err := set.Read("flags.yaml", []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		for name, want := range each.want {
			on, said := read.Flag(name)
			if got := [2]bool{on, said}; got != want {
				t.Errorf("%q: %s on and said %v, want %v", each.said, name,
					got, want)
			}
		}
	}
}

// TestASetsFlagTakesOnOrOffAndNothingElse refuses a flag given another
// word, and a shortcut, which a set does not say.
func TestASetsFlagTakesOnOrOffAndNothingElse(t *testing.T) {
	plain := fireRecipeFixture(t)
	for _, said := range []string{"  vibes: true\n",
		"  fastlight: cascades\n", "  shortcuts: {lamp-store: on}\n"} {
		raw := strings.Replace(plain, "  light: none\n",
			said+"  light: none\n", 1)
		if _, err := set.Read("flags.yaml", []byte(raw)); err == nil {
			t.Errorf("a set saying %q was read", said)
		}
	}
}
