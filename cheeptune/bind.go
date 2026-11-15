package cheeptune

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Plan is a score bound to a voicing: for each channel, what it sounds and
// when, in seconds. It is everything the machine needs and nothing it does
// not, so the render never looks back at either document.
type Plan struct {
	score   *Score
	voicing *Voicing
	voices  []voice
}

type voice struct {
	ch    *Channel
	label string // "lead" or "echo of lead", for errors and the roll
	notes []sounding
}

// sounding is one event on one channel.
type sounding struct {
	on, off float64 // seconds; off is where the release starts
	pitches []int
	hit     *Hit
	hitName string
	gain    float64
	loud    float64 // the gain the note's loudness mark gives
	onBeat  float64 // beats from the start, for the arp
	beats   float64 // how long it is held, in beats
	slide   bool    // it slides into the next note
}

// Bind checks that a score and a voicing fit each other and puts them
// together. Every role is played or dropped, every echo has something to
// echo, and every note is one the channel can make.
func Bind(s *Score, v *Voicing) (*Plan, error) {
	var errs []error
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(f, a...))
	}
	parts := map[string]*Part{}
	for i := range s.Parts {
		parts[s.Parts[i].Role] = &s.Parts[i]
	}
	played := map[string]bool{}
	for _, c := range v.Channels {
		if c.Role != "" {
			played[c.Role] = true
		}
	}
	dropped := map[string]bool{}
	for _, d := range v.Drop {
		dropped[d] = true
		if parts[d] == nil {
			fail("drop: the score has no %s part", d)
		}
	}
	for _, p := range s.Parts {
		if !played[p.Role] && !dropped[p.Role] {
			fail("%s: no channel plays it and it is not dropped",
				p.Role)
		}
	}

	m := machines[v.Machine]
	pl := &Plan{score: s, voicing: v}
	for i := range v.Channels {
		c := &v.Channels[i]
		src, label := c.Role, c.Role
		shift, gain := 0.0, 1.0
		if c.Echo != "" {
			src, label = c.Echo, "echo of "+c.Echo
			shift, gain = c.delay.float(), *c.Gain
			if dropped[c.Echo] {
				fail("channel %d echoes %s, which is dropped",
					i+1, c.Echo)
			}
		}
		p := parts[src]
		if p == nil {
			fail("channel %d (%s) plays %s; the score has no %s "+
				"part", i+1, c.Kind, label, src)
			continue
		}
		vc := voice{ch: c, label: label}
		for _, e := range p.events {
			if err := m.fits(c, e); err != nil {
				fail("%s bar %d on channel %d (%s): %v",
					src, e.bar, i+1, c.Kind, err)
				continue
			}
			from := e.at.float() + shift
			sd := sounding{
				on:      s.SecondsAt(from),
				off:     s.SecondsAt(from + e.len.float()),
				pitches: e.pitches,
				gain:    gain,
				loud:    e.loud,
				onBeat:  from,
				beats:   e.len.float(),
				slide:   e.slide,
			}
			if e.hit != "" {
				h := c.Hits[e.hit]
				sd.hit, sd.hitName = &h, e.hit
			}
			vc.notes = append(vc.notes, sd)
		}
		sort.SliceStable(vc.notes, func(a, b int) bool {
			return vc.notes[a].on < vc.notes[b].on
		})
		pl.voices = append(pl.voices, vc)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return pl, nil
}

// fits is whether a Game Boy channel can sound an event at all.
func (gameboy) fits(c *Channel, e event) error {
	if c.Kind == "noise" {
		if e.hit == "" {
			return fmt.Errorf(
				"a note on the noise channel; it plays hits")
		}
		if _, ok := c.Hits[e.hit]; !ok {
			names := make([]string, 0, len(c.Hits))
			for n := range c.Hits {
				names = append(names, n)
			}
			sort.Strings(names)
			return fmt.Errorf("hit %q is not one of %s", e.hit,
				strings.Join(names, ", "))
		}
		return nil
	}
	if e.hit != "" {
		return fmt.Errorf("hit %q on a pitched channel", e.hit)
	}
	lo, hi := rangeOf(c.Kind)
	for _, m := range e.pitches {
		if m < lo || m > hi {
			return fmt.Errorf("%s is outside the %s range %s to %s",
				NoteName(m), c.Kind, NoteName(lo), NoteName(hi))
		}
	}
	return nil
}
