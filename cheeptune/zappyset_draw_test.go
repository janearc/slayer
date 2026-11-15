//go:build draw

package cheeptune

import "testing"

// zappyLoud is, for each piece, a part its plan calls loud and the parts
// it must measure louder than, by breakOver.
var zappyLoud = map[string]struct {
	loud  string
	quiet []string
}{
	"zappy-ident":         {"behind", []string{"hello"}},
	"zappy-ident-falling": {"behind", []string{"hello"}},
	"zappy-bed-ominous":   {"the-fact", []string{"low", "held-breath"}},
	"zappy-bed-awe":       {"arrival", []string{"looking-up", "wider"}},
	"zappy-big-minor":     {"return", []string{"plain", "silence"}},
	"zappy-big-phrygian":  {"return", []string{"plain", "silence"}},
	"zappy-engine":        {"home", []string{"thin", "silence"}},
}

func TestEveryZappyPieceIsLoudWhereItsPlanSays(t *testing.T) {
	for name, want := range zappyLoud {
		levels := spanLevels(t, name)
		for _, quiet := range want.quiet {
			if levels[want.loud] < breakOver*levels[quiet] {
				t.Errorf("%s: %q measures %.3f and %q %.3f; the plan "+
					"says the first is the louder", name, want.loud,
					levels[want.loud], quiet, levels[quiet])
			}
		}
	}
}

func TestTheBigPiecesBreakIsNearSilence(t *testing.T) {
	for _, name := range []string{"zappy-big-minor", "zappy-big-phrygian"} {
		levels := spanLevels(t, name)
		if levels["silence"]*silenceUnder > levels["destruction"] {
			t.Errorf("%s: the break measures %.3f against %.3f before "+
				"it; the plan says near silence", name, levels["silence"],
				levels["destruction"])
		}
		if levels["destruction"] <= levels["silence"] {
			t.Errorf("%s: the part before the break is no louder than "+
				"the break", name)
		}
	}
}

// silenceUnder is how many times quieter than the part before it the big
// piece's break must measure. Picked by the composer, 2026-11-12: a
// tenth is twenty decibels down.
const silenceUnder = 10.0
