package cheeptune

import (
	"fmt"
	"strings"
)

// gameboy is the first machine: two pulses, a wave and a noise channel,
// simulated at the register. machine.go is the simulation and render.go's
// sample-level half is its output stage; this file is its rules.
type gameboy struct{}

// kinds are the Game Boy's channels.
func (gameboy) kinds() []string { return kinds }

// percussive is the noise channel.
func (gameboy) percussive(kind string) bool { return kind == "noise" }

// check is the Game Boy's rules on a voicing's channels: how many of each
// kind, each channel's fields, and one channel a role.
func (g gameboy) check(v *Voicing) ([]error, map[string]int) {
	var errs []error
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(f, a...))
	}
	count := map[string]int{}
	plays := map[string]int{}
	for i := range v.Channels {
		c := &v.Channels[i]
		at := fmt.Sprintf("channel %d (%s)", i+1, c.Kind)
		if !known(c.Kind, kinds) {
			fail("channel %d: unknown kind %q; kinds are %s",
				i+1, c.Kind, strings.Join(kinds, ", "))
			continue
		}
		count[c.Kind]++
		if count[c.Kind] == kindLimit[c.Kind]+1 {
			n := kindLimit[c.Kind]
			fail("%s: the gameboy has %d %s channel%s and this is "+
				"one more", at, n, c.Kind, plural(n))
		}
		errs = append(errs, c.check(at)...)
		if c.Role != "" {
			plays[c.Role]++
			if plays[c.Role] == 2 {
				fail("%s: %s is already played by another "+
					"channel", at, c.Role)
			}
		}
	}
	errs = append(errs, v.noStudioMix()...)
	return errs, plays
}

// noStudioMix refuses the studio's mix fields on a Game Boy voicing.
func (v *Voicing) noStudioMix() []error {
	var errs []error
	for _, f := range []struct {
		name string
		set  bool
	}{
		{"master", v.Master != nil}, {"reverb", v.Reverb != nil},
		{"delay", v.Delay != nil},
	} {
		if f.set {
			errs = append(errs, fmt.Errorf("%s is only for the studio "+
				"machine; the gameboy has no mix to set", f.name))
		}
	}
	return errs
}

// studioField names the first studio field a channel carries, or nothing.
func (c *channel) studioField() string {
	for _, f := range []struct {
		name string
		set  bool
	}{
		{"place", c.Place != nil}, {"send", c.Send != nil},
		{"voices", c.Voices != 0}, {"glide", c.Glide != nil},
		{"fm", c.FM != nil},
		{"analog", c.Analog != nil}, {"pluck", c.Pluck != nil},
		{"modal", c.Modal != nil}, {"drums", c.Drums != nil},
	} {
		if f.set {
			return f.name
		}
	}
	return ""
}

// check is one channel's fields against its kind, on the Game Boy.
func (c *channel) check(at string) []error {
	errs := c.does(at)
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(at+": "+f, a...))
	}
	unit := func(name string, p *float64) {
		if p != nil && (*p < 0 || *p > 1) {
			fail("%s %v is outside 0 to 1", name, *p)
		}
	}
	if name := c.studioField(); name != "" {
		fail("%s is only for the studio machine", name)
	}
	switch c.Pan {
	case "", "both", "left", "right":
	default:
		fail("pan %q; pans are left, right, both", c.Pan)
	}
	if c.Detune < -100 || c.Detune > 100 {
		fail("detune %v cents is past a semitone", c.Detune)
	}

	// what belongs to which kind.
	pitched := c.Kind == "pulse" || c.Kind == "wave"
	if c.Duty != nil && c.Kind != "pulse" {
		fail("duty is only for a pulse")
	}
	if c.Table != "" && c.Kind != "wave" {
		fail("table is only for a wave")
	}
	if c.Hits != nil && c.Kind != "noise" {
		fail("hits are only for noise")
	}
	if !pitched {
		if c.Envelope != nil || c.Vibrato != nil || c.Arp.set ||
			c.Detune != 0 {
			fail("envelope, vibrato, arp and detune are for " +
				"pitched channels; a hit carries its own decay")
		}
	}
	switch c.Kind {
	case "pulse":
		if c.Duty == nil {
			fail("a pulse needs a duty: 0.125, 0.25, 0.5 or 0.75")
		} else if !knownFloat(*c.Duty, duties) {
			fail("duty %v; the hardware has 0.125, 0.25, 0.5, 0.75",
				*c.Duty)
		}
	case "wave":
		errs = append(errs, c.readTable(at)...)
	case "noise":
		if len(c.Hits) == 0 {
			fail("noise with no hits plays nothing")
		}
		for name, h := range c.Hits {
			if !isWord(name) || isPitch(name) {
				fail("hit %q: a hit is a lowercase word", name)
			}
			if h.Shift < 0 || h.Shift > 13 {
				fail("hit %s: shift %d is outside 0 to 13",
					name, h.Shift)
			}
			if h.Divisor < 0 || h.Divisor > 7 {
				fail("hit %s: divisor %d is outside 0 to 7",
					name, h.Divisor)
			}
			if h.Decay <= 0 || h.Decay > 4 {
				fail("hit %s: decay %v s is outside 0 to 4",
					name, h.Decay)
			}
			unit("hit "+name+" volume", h.Volume)
		}
	}
	if e := c.Envelope; e != nil {
		if e.Attack < 0 || e.Decay < 0 || e.Release < 0 {
			fail("envelope times are not below nought")
		}
		if e.Sustain < 0 || e.Sustain > 1 {
			fail("sustain %v is outside 0 to 1", e.Sustain)
		}
	}
	if vb := c.Vibrato; vb != nil {
		if vb.Rate <= 0 || vb.Rate > 20 {
			fail("vibrato rate %v hertz is outside 0 to 20",
				vb.Rate)
		}
		if vb.Depth < 0 || vb.Depth > 100 {
			fail("vibrato depth %v cents is outside 0 to 100",
				vb.Depth)
		}
	}
	return errs
}
