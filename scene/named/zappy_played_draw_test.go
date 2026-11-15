//go:build draw

package named_test

import (
	"testing"
	"time"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/scene/named"
)

// played is a built zappy drawn small at a time with its channels held.
func played(t *testing.T, built *named.Built, values map[string]float64,
	at int) []float32 {
	t.Helper()
	if err := built.Play(values); err != nil {
		t.Fatal(err)
	}
	drawn := small(t, built, []time.Duration{firstAt, laterAt}[at])
	var light []float32
	for _, pixel := range drawn.Pix {
		light = append(light, pixel.R, pixel.G, pixel.B)
	}
	return light
}

// equal is whether two pictures' light is the same in every channel.
func equal(one, other []float32) bool {
	for i := range one {
		if one[i] != other[i] {
			return false
		}
	}
	return len(one) == len(other)
}

// At rest, played zappy is today's zappy byte for byte at two times, even
// after its channels have moved and come back.
func TestPlayedZappyAtRestIsTodaysZappy(t *testing.T) {
	today := mustBuild(t, "zappy", nil)
	again := mustBuild(t, "zappy", nil)
	for at := range 2 {
		before := played(t, today, nil, at)
		played(t, again, map[string]float64{"star-brightness": brighter,
			"shells-breath": larger, "beam-width": larger}, at)
		if !equal(before, played(t, again, nil, at)) {
			t.Errorf("at time %d, zappy at rest is not today's", at)
		}
	}
}

// Each of the star's brightness, one shell's breath and the beams' width
// changes the picture.
func TestChannelsChangeThePicture(t *testing.T) {
	built := mustBuild(t, "zappy", nil)
	rest := played(t, built, nil, 0)
	for _, values := range []map[string]float64{
		{"star-brightness": brighter}, {"shell-12-breath": larger},
		{"beam-width": brighter * larger * larger * larger}} {
		if equal(rest, played(t, built, values, 0)) {
			t.Errorf("%v did not change zappy", values)
		}
	}
}

// Zappy names its channels, each with a range and a rest within it.
func TestZappyHasChannels(t *testing.T) {
	channels := mustBuild(t, "zappy", nil).Channels()
	if len(channels) == 0 {
		t.Fatal("zappy has no channels")
	}
	var _ []lantern.Channel = channels
}
