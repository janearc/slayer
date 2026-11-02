package planet

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/occultation"
)

// profileOf is a profile of 1 km bins from 1 km out, each of a depth,
// read from a table as the pds writes one, or the test stops.
func profileOf(t *testing.T, depths ...float64) occultation.Profile {
	t.Helper()
	var table strings.Builder
	for k, depth := range depths {
		fmt.Fprintf(&table, "%9.2f, 10.000, 1.000,%7.4f, 0.0000, 1.0000\r\n",
			float64(k+1), depth)
	}
	profile, err := occultation.Read(strings.NewReader(table.String()))
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

// a ring laid over a stretch of a profile has, at each reach, the depth
// the profile has where that reach is laid: here a ring from 100 to 104
// m over the profile's 0.5 to 4.5 km, so each metre is a km of it, its
// ringlets the profile's in place of the made up.
func TestAMeasuredRingIsItsProfile(t *testing.T) {
	profile := profileOf(t, 0.2, 1.5, 0.05, 0.9)
	ring := mustRing(t, 100, 104, 0.3, 0.8, 0)
	ring, err := ring.Ringleted(9, 0.5, 1)
	if err != nil {
		t.Fatal(err)
	}
	ring, err = ring.Measured(profile, 500, 4500)
	if err != nil {
		t.Fatal(err)
	}
	if ring.ringlets.strength != 0 {
		t.Errorf("the made-up ringlets were kept")
	}
	for k, want := range []float64{0.2, 1.5, 0.05, 0.9} {
		reach := 100 + float64(k) + 0.5
		got := ring.depth * ring.share(reach, 0)
		if math.Abs(got-want) > 1e-5 {
			t.Errorf("at %g m, depth %g, want %g", reach, got, want)
		}
	}
	whole := profile.Seen(2500, 4000)
	if math.Abs(ring.depth-whole) > 1e-12 {
		t.Errorf("the ring's depth %g, want the stretch's %g", ring.depth,
			whole)
	}
}

// a stretch outside the profile, backward, or empty is refused.
func TestAMeasuredRingNeedsItsStretch(t *testing.T) {
	profile := profileOf(t, 0, 0, 1)
	ring := mustRing(t, 100, 104, 0.3, 0.8, 0)
	for _, x := range []struct{ from, to float64 }{
		{0, 3500}, {500, 4000}, {2500, 1500}, {500, 2500},
	} {
		if _, err := ring.Measured(profile, x.from, x.to); err == nil {
			t.Errorf("measured over %g to %g m", x.from, x.to)
		}
	}
}
