package flame

import (
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

func TestLegacyAndScheduledFieldsKeepTheirDistinctBurnWindows(t *testing.T) {
	for _, example := range []struct {
		name            string
		start, duration time.Duration
		active          []bool
	}{
		{"indefinite", 0, 0, []bool{true, true, true, true, true}},
		{"scheduled at zero", 0, 2 * time.Second, []bool{false, true, true, false, false}},
		{"delayed indefinite", time.Second, 0, []bool{false, false, true, true, true}},
	} {
		t.Run(example.name, func(t *testing.T) {
			field, err := NewField([]Ignition{{Fire: campfire, Start: example.start, Duration: example.duration}}, 1)
			if err != nil {
				t.Fatal(err)
			}
			for i, at := range []time.Duration{-time.Second, 0, time.Second, 2 * time.Second, 3 * time.Second} {
				sampled := field.Tick(at).(*Field)
				pieces := len(sampled.Pieces())
				lamps := len(sampled.Sources(6))
				density := sampled.Density(maths.V(0, .05, 0))
				light := field.Shine().Tick(at).At(maths.V(0, .05, 0))
				if example.active[i] {
					if pieces != 1 || lamps == 0 || !(density > 0) || !(light.R+light.G+light.B > 0) {
						t.Fatalf("at %s lost occupied emitting field: pieces %d lamps %d density %g", at, pieces, lamps, density)
					}
				} else if pieces != 0 || lamps != 0 || density != 0 || light != (maths.Light{}) {
					t.Fatalf("at %s retained an expired/unstarted field: pieces %d lamps %d density %g", at, pieces, lamps, density)
				}
			}
		})
	}
}
