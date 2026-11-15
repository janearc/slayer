//go:build draw

package cheeptune

import (
	"testing"
	"time"
)

// levelStep is how often a part's level is sampled, and breakOver is how
// many times louder than its thin parts a member's break must measure.
// Picked by the composer, 2026-11-12: a twentieth of a second is the
// level's own window, and twice is a difference no reading of the roll
// would argue with.
const (
	levelStep = 50 * time.Millisecond
	breakOver = 2.0
)

// meanLevel is the mean of a piece's level between two moments.
func meanLevel(p *Piece, from, to time.Duration) float64 {
	sum, count := 0.0, 0
	for at := from; at < to; at += levelStep {
		if _, _, _, level, ok := p.At(at); ok {
			sum += level
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// spanLevels is the mean level of each part of a tune, by the part's name.
func spanLevels(t *testing.T, name string) map[string]float64 {
	t.Helper()
	p, sheet, err := OpenTune(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	out := map[string]float64{}
	for _, span := range sheet.Spans() {
		out[span.Name] = meanLevel(p, span.Start, span.End)
	}
	return out
}

// glassBreaks is, for each member, the part its plan calls the break and
// the thin parts it must be louder than.
var glassBreaks = map[string]struct {
	loud string
	thin []string
}{
	"glass-acid":  {"breaks", []string{"thin", "ice", "after"}},
	"glass-chase": {"breaks", []string{"thin", "ice", "after"}},
}

func TestEveryGlassMemberBreaksLouderThanItsThinParts(t *testing.T) {
	for name, want := range glassBreaks {
		levels := spanLevels(t, name)
		for _, thin := range want.thin {
			if levels[want.loud] < breakOver*levels[thin] {
				t.Errorf("%s: %q measures %.3f and %q %.3f; the plan "+
					"says the first is the break", name, want.loud,
					levels[want.loud], thin, levels[thin])
			}
		}
	}
}

func TestAPartThatIsNotInTheSheetMeasuresNothing(t *testing.T) {
	levels := spanLevels(t, "glass-acid")
	if got := levels["no-such-part"]; got != 0 {
		t.Errorf("a part the sheet does not have measured %v", got)
	}
	p, _, err := OpenTune("glass-acid")
	if err != nil {
		t.Fatal(err)
	}
	if got := meanLevel(p, time.Hour, time.Hour+time.Second); got != 0 {
		t.Errorf("past the end of the piece measured %v", got)
	}
}
