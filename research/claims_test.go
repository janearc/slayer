package research_test

import (
	"github.com/janearc/slayer/research"
	"testing"
)

func TestPublishedLeafRangeRolesAndUnitsStayTogether(t *testing.T) {
	data, err := research.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		name, unit, key, excerpt string
		value                    float64
	}{
		{"redwood-needle-minimum", "cm", "osu-coast-redwood-foliage", "1-2 cm long", 1},
		{"redwood-needle-maximum", "cm", "osu-coast-redwood-foliage", "1-2 cm long", 2},
		{"sequoia-scale-minimum", "mm", "osu-giant-sequoia-foliage", "3-6 mm long on leaders", 3},
		{"sequoia-scale-maximum", "mm", "osu-giant-sequoia-foliage", "3-6 mm long on leaders", 6},
	} {
		got, err := data.Get(want.name)
		if err != nil {
			t.Fatal(err)
		}
		if got.Value != want.value || got.Unit != want.unit || got.From == nil || got.From.Key != want.key || got.From.Excerpt != want.excerpt {
			t.Fatalf("range role/unit changed for %s: %+v", want.name, got)
		}
	}
}
