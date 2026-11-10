package set_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/slayer/set"
)

func TestLegacyEffectsEnforceSceneWideLimitsAndSingleTargetRecipes(t *testing.T) {
	front := legacyScene(frontFixture(t))
	bed := legacyScene(bedFixture(t))
	repeated := func(raw, header, next, name string, count int) string {
		start := strings.Index(raw, header) + len(header)
		finish := start + strings.Index(raw[start:], next)
		if start < len(header) || finish < start || !strings.Contains(raw[start:finish], name) {
			t.Fatal("missing fixture section", header)
		}
		var blocks strings.Builder
		for i := range count {
			blocks.WriteString(strings.Replace(raw[start:finish], name,
				fmt.Sprintf("    item %d:", i), 1))
		}
		return raw[:start] + blocks.String() + raw[finish:]
	}
	fronts := repeated(front, "  fire-fronts:\n", "  film:", "    through the grove:", 9)
	beds := repeated(bed, "  fuel-beds:\n", "  set:", "    litter:", 2)
	layout := "      layout: {pattern: cluster, count: 3, radius: {value: 1, unit: m}}\n"
	for _, test := range []struct{ raw, want string }{
		{fronts, "at most eight fire fronts"},
		{beds, "at most one experimental fuel bed"},
		{strings.Replace(front, "      flame:", layout+"      flame:", 1), "omit recipe layout"},
		{strings.Replace(front, "kind: coast-redwood,", "recipe: wood fire, alight: 1,", 1), "fire stand the grove is already alight"},
		{strings.Replace(front, "kind: coast-redwood,", "kind: coast-redwood, recipe: wood fire,", 1), "exactly one of kind or recipe"},
		{strings.Replace(front, "kind: coast-redwood,", "kind: coast-redwood, above: 5, below: 4,", 1), "which is nowhere"},
		{strings.Replace(front, "recipe: wood fire", "recipe: []", 1), "recipe takes a word"},
		{strings.Replace(front, "  recipes:\n", "  recipes: []\n  unused-recipes:\n", 1), "takes keys under it"},
		{strings.Replace(front, "      flame:", "      layout: {pattern: cluster, typo: 3}\n      flame:", 1), "typo"},
		{strings.Replace(front, "      flame:", "      layout: {pattern: cluster}\n      flame:", 1), "this has no count"},
	} {
		_, err := set.Read("effect-limits.yaml", []byte(test.raw))
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("want %q, got %v", test.want, err)
		}
	}
}
