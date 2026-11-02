package world

import "testing"

// the biome rules at their edges, each case a tile placed by hand on one
// side or the other of a threshold. e is height, cold, rain, and sea the
// tiles from the sea. below the water line: ice above a cold of 1, deep
// below a height of -0.35, else sea.
//
// above it: snow above a cold of 1.08 or a height of 0.82, rock above a
// height of 0.62, tundra above a cold of 0.88, beach on the first tile
// from the sea if lower than 0.03.
//
// then desert under a rain of 0.28, grass under 0.5, jungle if the cold
// is under 0.35 and the rain over 0.75, else forest.
//
// each edge is tried exactly on it, where the rule is strict, and a step
// either side, so a rule that moves, flips or is left out is seen.
func TestPickAtTheEdgesOfEachRule(t *testing.T) {
	for _, c := range []struct {
		name          string
		e, cold, rain float64
		sea           int
		want          Biome
	}{
		{"sea at the edge of the land", -0.001, 0.5, 0.6, 0, Sea},
		{"land at the water line", 0, 0.5, 0.6, 9, Forest},
		{"ice just over 1", -0.1, 1.001, 0.6, 0, Ice},
		{"not ice at 1", -0.1, 1.0, 0.6, 0, Sea},
		{"ice over deep", -0.5, 1.5, 0.6, 0, Ice},
		{"deep under -0.35", -0.36, 0.5, 0.6, 0, Deep},
		{"deep well under", -0.9, 0.5, 0.6, 0, Deep},
		{"sea at -0.35", -0.35, 0.5, 0.6, 0, Sea},
		{"sea over -0.35", -0.34, 0.5, 0.6, 0, Sea},
		{"sea just under the line", -0.1, 0.5, 0.6, 0, Sea},

		{"snow in the cold", 0.4, 1.09, 0.6, 5, Snow},
		{"not snow at 1.08", 0.4, 1.08, 0.6, 5, Tundra},
		{"snow high", 0.83, 0.1, 0.6, 5, Snow},
		{"snow high and cold", 0.9, 1.5, 0.6, 5, Snow},
		{"not snow at 0.82", 0.82, 0.1, 0.6, 5, Rock},
		{"rock over 0.62", 0.63, 0.1, 0.6, 5, Rock},
		{"rock at 0.7", 0.7, 0.1, 0.6, 5, Rock},
		{"not rock at 0.62", 0.62, 0.1, 0.6, 5, Forest},

		{"tundra over 0.88", 0.4, 0.89, 0.6, 5, Tundra},
		{"tundra at 0.9", 0.4, 0.9, 0.6, 5, Tundra},
		{"not tundra at 0.88", 0.4, 0.88, 0.6, 5, Forest},

		{"beach by the sea", 0.02, 0.3, 0.6, 1, Beach},
		{"not beach at 0.03", 0.03, 0.3, 0.6, 1, Forest},
		{"not beach high by the sea", 0.5, 0.3, 0.6, 1, Forest},
		{"not beach low and two in", 0.02, 0.3, 0.6, 2, Forest},
		{"not beach low and far in", 0.02, 0.3, 0.6, 5, Forest},

		{"desert under 0.28", 0.4, 0.3, 0.27, 5, Desert},
		{"grass at 0.28", 0.4, 0.3, 0.28, 5, Grass},
		{"grass under 0.5", 0.4, 0.3, 0.49, 5, Grass},
		{"forest at 0.5", 0.4, 0.3, 0.5, 5, Forest},
		{"forest at 0.6", 0.4, 0.5, 0.6, 5, Forest},

		{"jungle wet and cool", 0.4, 0.34, 0.76, 5, Jungle},
		{"forest at a cold of 0.35", 0.4, 0.35, 0.76, 5, Forest},
		{"forest at a rain of 0.75", 0.4, 0.34, 0.75, 5, Forest},
		{"forest wet and mild", 0.4, 0.5, 0.76, 5, Forest},
		{"forest cool and moist", 0.4, 0.3, 0.6, 5, Forest},
	} {
		if got := pick(c.e, c.cold, c.rain, c.sea); got != c.want {
			t.Errorf("%s: pick(%v, %v, %v, %d) is %d, want %d", c.name, c.e,
				c.cold, c.rain, c.sea, got, c.want)
		}
	}
}
