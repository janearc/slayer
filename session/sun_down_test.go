package session

import (
	"testing"

	"github.com/janearc/slayer/set"
)

// downText is a small set lit by the sun, with the lines added under
// scene.
func downText(add string) []byte {
	return []byte(undrawnHead[:len(undrawnHead)-len("  light: ambient\n")] +
		"  light: sun\n" + undrawnFilm + undrawnCameras + add)
}

// A set whose sun its place, date and time put below the horizon is lit by
// a sun that is down; the same place and date at noon is not, and a sun
// said by azimuth and altitude below the horizon is not down either: a
// stage refuses that one.
func TestASunWorkedBelowTheHorizonIsDown(t *testing.T) {
	place := "  location: {lat: 40, lon: 0}\n  date: \"2026-06-21\"\n" +
		"  utc-offset: {value: 0, unit: h}\n"
	for why, test := range map[string]struct {
		add  string
		down bool
	}{
		"midnight":          {place + "  timeofday: 0\n", true},
		"noon":              {place + "  timeofday: 1200\n", false},
		"a typed sun below": {"  sun: {azimuth: 90, altitude: -5}\n", false},
	} {
		read, err := set.Read("t.yaml", downText(test.add))
		if err != nil {
			t.Fatalf("%s: %v", why, err)
		}
		if got := sunDown(read); got != test.down {
			t.Errorf("%s: the sun is down %v, want %v", why, got, test.down)
		}
	}
}
