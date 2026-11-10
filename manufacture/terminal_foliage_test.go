package manufacture

import (
	"slices"
	"strings"
	"testing"
)

func TestTerminalFoliageSelectionDoesNotDependOnNeighbouringParts(t *testing.T) {
	for _, style := range []int{1, 2} {
		row := redwoodRow()
		if style == 2 {
			row = sequoiaRow()
		}
		scaffold, err := row.scaffold(Sow(5))
		if err != nil {
			t.Fatal(err)
		}
		terminal, major := 0, 0
		for _, part := range scaffold.parts {
			if part.of != leaf {
				continue
			}
			if part.terminal {
				terminal++
			} else {
				major++
			}
		}
		if terminal == 0 || major == 0 {
			t.Fatal("fixture does not distinguish foliage roles")
		}
		first, err := recursiveFoliage(scaffold, Sow(5), style, "quick")
		if err != nil {
			t.Fatal(err)
		}
		reordered := scaffold
		reordered.parts = slices.Clone(scaffold.parts)
		slices.Reverse(reordered.parts)
		second, err := recursiveFoliage(reordered, Sow(5), style, "quick")
		if err != nil {
			t.Fatal(err)
		}
		// Random rolls are indexed by construction order. Selection must
		// nevertheless retain every terminal cell and omit major sleeves.
		if len(first.groups[leaf].Members()) != len(second.groups[leaf].Members()) {
			t.Fatal("reordering changed which foliage was represented", style)
		}
	}
}

func TestPrototypeCacheRejectsUnboundedOrUnknownKeys(t *testing.T) {
	for _, key := range []shootKey{
		{0, 0, 0, "quick"}, {4, 0, 0, "quick"}, {1, -1, 0, "quick"},
		{1, 5, 0, "quick"}, {1, 0, -1, "quick"}, {1, 0, 2, "quick"},
		{1, 0, 0, "hull"}, {1, 0, 0, "unknown"},
	} {
		if _, err := prototype(key.style, key.bin, key.variant, key.detail); err == nil ||
			!strings.Contains(err.Error(), "unsupported shoot prototype key") {
			t.Fatalf("invalid prototype entered the cache: %+v, %v", key, err)
		}
	}
}
