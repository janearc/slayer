package manufacture

import (
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func TestTerminalFoliageDoesNotStartAnExtraCellAtItsExactEnd(t *testing.T) {
	for _, length := range []float64{.4, .8} {
		trunk, err := newTube(measure.Metres(1), measure.Metres(.1), measure.Metres(.08))
		if err != nil {
			t.Fatal(err)
		}
		axis, err := newTube(measure.Metres(length), measure.Metres(.01), measure.Metres(.001))
		if err != nil {
			t.Fatal(err)
		}
		scaffold := good{parts: []part{{of: body, shape: trunk, fine: cut{around: 6}},
			{of: leaf, shape: axis, terminal: true}}, dressed: looks{
			body: plain("wood", .2, .1, .05), leaf: plain("leaves", .1, .3, .1)}}
		made, err := recursiveFoliage(scaffold, Sow(11), 1, "quick")
		if err != nil {
			t.Fatal(err)
		}
		members := made.groups[leaf].Members()
		expected := 1
		if length == .8 {
			expected = 2
		}
		if len(members) != expected {
			t.Fatalf("%g m axis got %d cells, want %d", length, len(members), expected)
		}
		for _, member := range members {
			if member.Space.Origin.Y >= length {
				t.Fatalf("cell starts at or past axis end: %v", member.Space.Origin)
			}
		}
		if _, err := made.Things(maths.Subspace{}); err != nil {
			t.Fatal(err)
		}
	}
}
