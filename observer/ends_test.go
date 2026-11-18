package observer_test

import (
	"testing"
	"time"

	"github.com/janearc/celestial/bands"
	theme "github.com/janearc/libtheme-css/primitives/bands"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/slayer/observer"
)

// patience is how long a swatch may take before the test says its loop
// does not end. Picked by the test author, 2026-11-17: a swatch of a few
// lines takes microseconds.
const patience = 10 * time.Second

// A swatch of lines in several bands comes back, and is the colour
// libtheme gives the same lines: one band, three bands side by side, and
// the visible's two ends.
func TestASwatchComesBackAsLibthemesColour(t *testing.T) {
	lo, hi := swatch.Observed()
	for name, spectrum := range map[string]map[float64]float64{
		"one band":        {550: unity},
		"three bands":     {450: 0.5, 550: unity, 650: 0.25},
		"the two ends":    {float64(lo): unity, float64(hi) - 1: unity},
		"a band is empty": {450: unity, 650: unity},
	} {
		done := make(chan swatch.Swatch, 1)
		go func() { done <- observer.Swatch(bands.Lines(spectrum)) }()
		select {
		case got := <-done:
			if want := theme.Lines(spectrum).Swatch(); got != want {
				t.Errorf("%s is %v, not %v", name, got, want)
			}
		case <-time.After(patience):
			t.Fatalf("%s did not come back in %v: a loop does not end",
				name, patience)
		}
	}
}
