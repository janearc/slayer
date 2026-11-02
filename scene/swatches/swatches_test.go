package swatches

import (
	"math"
	"testing"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/slayer/maths"
)

// noise is libtheme's tolerance for arithmetic noise: a round trip through
// a matrix and its inverse.
const noise = ok.Exact

func near(a, b maths.Light) bool {
	return math.Abs(a.R-b.R) <= noise && math.Abs(a.G-b.G) <= noise &&
		math.Abs(a.B-b.B) <= noise
}

// Every paint and lamp comes back as the light it was made from.
func TestSwatchesAreTheLightTheyWereMadeFrom(t *testing.T) {
	for name, c := range map[string]struct {
		got, want maths.Light
	}{
		"chalk":    {Light(Chalk), maths.RGB(0.80, 0.80, 0.78)},
		"clay":     {Light(Clay), maths.RGB(0.70, 0.30, 0.20)},
		"paint":    {Light(Paint), maths.Grey(0.90)},
		"daylight": {Light(Daylight), maths.Grey(1)},
		"red":      {Light(Red), maths.RGB(1, 0, 0)},
		"green":    {Light(Green), maths.RGB(0, 1, 0)},
		"blue":     {Light(Blue), maths.RGB(0, 0, 1)},
	} {
		if !near(c.got, c.want) {
			t.Errorf("%s is %v, made from %v", name, c.got, c.want)
		}
	}
}
