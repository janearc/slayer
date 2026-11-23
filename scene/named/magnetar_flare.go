package named

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/research"
)

// The 2004 giant flare of SGR 1806-20 as light on the star: how many
// times its quiet glow the star shines at a moment. Its times are
// measured (Hurley and others, 2005): a spike that rises within a
// millisecond and decays in 65 ms, over in about 0.2 s, then a tail
// that pulses at the star's spin for 380 s. The tail is shown sped up,
// by speed; the spike runs at its true pace.
//
// How bright the spike and
// the pulses are, and the shapes of the tail's fade and of one pulse,
// are picked for the picture: the true spike is about a hundred billion
// times the quiet star, which no picture holds.

// the flare's picked brightnesses, as multiples of the star's quiet
// glow: the spike's peak, the tail's first pulse, and how many times
// the tail's light falls by e over its length. Picked by the author of
// this scene, 2026-11-12, by looking: the spike overwhelms the picture
// and the pulses read against the quiet star.
const (
	flarePeak  = 2e4
	flarePulse = 40
	flareFade  = 3
)

// flareSpeed is how many times faster the tail is shown than it ran,
// by default: 380 s in 19, a pulse every 0.38 s, nine frames at 24 a
// second. Picked by the author of this scene, 2026-11-12.
const flareSpeed = 20

// flareStarts is when the spike begins, by default, in the scene's
// time: a few turns of the field before it. Picked by the author of
// this scene, 2026-11-12.
const flareStarts = 5 * time.Second

// pulseSharpness is the power one pulse's raised cosine is taken to,
// so a pulse is a peak and not a sway; picked by the author of this
// scene, 2026-11-12.
const pulseSharpness = 2

// flare is the giant flare's light: when it starts, its measured times,
// and how much faster than true its tail is shown.
type flare struct {
	start, rise, decay, spike, tail, period time.Duration
	speed                                   float64
}

// newFlare is the 2004 flare starting at start, its tail shown speed
// times faster than it ran. A tail shown slower than true, or not at
// all, is refused by name.
func newFlare(start time.Duration, speed float64) (flare, error) {
	if !(speed >= single) {
		return flare{}, fmt.Errorf("%w: flare-speed %g; the tail is "+
			"shown at its true pace or faster, one or more", ErrBounds,
			speed)
	}
	return flare{start: start, speed: speed,
		rise:   measured("sgr1806-spike-rise-ms", time.Millisecond),
		decay:  measured("sgr1806-spike-decay-ms", time.Millisecond),
		spike:  measured("sgr1806-spike-seconds", time.Second),
		tail:   measured("sgr1806-tail-seconds", time.Second),
		period: measured("sgr1806-tail-period-seconds", time.Second)}, nil
}

// measured is a registered time, read in the unit it is held in.
func measured(name string, unit time.Duration) time.Duration {
	return time.Duration(research.Value(name) * float64(unit))
}

// brightness is how many times its quiet glow the star shines at a
// moment of the scene: one before the flare and after its tail.
func (lit flare) brightness(at time.Duration) float64 {
	since := at - lit.start
	if since < 0 {
		return single
	}
	return single + lit.spiked(since) + lit.tailed(since)
}

// spiked is the spike's light a time after it begins: a straight rise
// to its peak, then a fall by e every decay, until the spike is over.
func (lit flare) spiked(since time.Duration) float64 {
	if since >= lit.spike {
		return 0
	}
	if since < lit.rise {
		return flarePeak * float64(since) / float64(lit.rise)
	}
	return flarePeak * math.Exp(-float64(since-lit.rise)/
		float64(lit.decay))
}

// tailed is the tail's light a scene time after the spike begins: none
// during the spike; then, in the tail's own time, sped up, pulses at the
// spin period under a fall that reaches nothing at the tail's end.
func (lit flare) tailed(since time.Duration) float64 {
	if since < lit.spike {
		return 0
	}
	into := float64(since-lit.spike) * lit.speed / float64(lit.tail)
	if into >= single {
		return 0
	}
	turns := into * float64(lit.tail) / float64(lit.period)
	pulse := math.Pow((single+math.Cos(2*math.Pi*turns))/diameter,
		pulseSharpness)
	return flarePulse * math.Exp(-flareFade*into) * (single - into) * pulse
}
