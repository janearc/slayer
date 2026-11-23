package cheeptune

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Check says what the documents at the given paths hold and what is wrong
// with them: a score, a voicing, or a score and a voicing together. It
// reports every problem it finds, not just the first. The error it returns
// is non-nil when there was any.
func Check(w io.Writer, paths ...string) error {
	var s *Score
	var v *Voicing
	bad := false
	for _, path := range paths {
		b, err := readDoc(path)
		if err != nil {
			problems(w, err)
			bad = true
			continue
		}
		kind, err := kind(b)
		if err != nil {
			problems(w, fmt.Errorf("%s: %w", path, err))
			bad = true
			continue
		}
		switch kind {
		case "score":
			if s, err = ParseScore(b); err != nil {
				fmt.Fprintf(w, "score %s\n", path)
				problems(w, err)
				bad = true
				continue
			}
			describeScore(w, s)
		case "voicing":
			if v, err = ParseVoicing(b); err != nil {
				fmt.Fprintf(w, "voicing %s\n", path)
				problems(w, err)
				bad = true
				continue
			}
			describeVoicing(w, v)
		}
	}
	if s != nil && v != nil {
		if _, err := bindVoicing(s, v); err != nil {
			fmt.Fprintf(w, "together\n")
			problems(w, err)
			bad = true
		} else {
			fmt.Fprintf(w, "together: every role is played or "+
				"dropped, and every note fits its channel\n")
		}
	}
	if bad {
		return errors.New("problems found")
	}
	return nil
}

// problems writes an error's lines, one problem a line.
func problems(w io.Writer, err error) {
	for line := range strings.SplitSeq(err.Error(), "\n") {
		fmt.Fprintf(w, "  problem: %s\n", line)
	}
}

// describeScore is a score as check prints it: the piece, then each part.
func describeScore(w io.Writer, s *Score) {
	loop := "plays once"
	if s.Loop {
		loop = "loops"
	}
	fmt.Fprintf(w,
		"score %q: %v bpm, %d beats a bar, %d bars, %.2f s, %s\n",
		s.Name, s.Tempo, s.Meter, s.Bars(), s.Seconds(), loop)
	for _, c := range s.Tempos {
		fmt.Fprintf(w, "  from bar %d, %v bpm\n", c.Bar, c.Tempo)
	}
	for _, p := range s.Parts {
		notes, chords := 0, 0
		lo, hi := 1000, -1000
		hits := map[string]int{}
		for _, e := range p.events {
			switch {
			case e.hit != "":
				hits[e.hit]++
			case len(e.pitches) > 1:
				chords++
			default:
				notes++
			}
			for _, m := range e.pitches {
				lo, hi = min(lo, m), max(hi, m)
			}
		}
		var what []string
		if notes > 0 {
			what = append(what, fmt.Sprintf("%d notes", notes))
		}
		if chords > 0 {
			what = append(what, fmt.Sprintf("%d chords", chords))
		}
		if hi >= lo {
			span := noteName(lo) + " to " + noteName(hi)
			what = append(what, span)
		}
		if len(hits) > 0 {
			names := make([]string, 0, len(hits))
			for h, n := range hits {
				names = append(names, h+" "+strconv.Itoa(n))
			}
			sort.Strings(names)
			what = append(what, "hits "+strings.Join(names, ", "))
		}
		if len(what) == 0 {
			what = append(what, "silent")
		}
		fmt.Fprintf(w, "  %-5s %s\n", p.Role, strings.Join(what, ", "))
	}
}

// describeVoicing is a voicing as check prints it: a line a channel.
func describeVoicing(w io.Writer, v *Voicing) {
	fmt.Fprintf(w, "voicing %q on the %s\n", v.Name, v.Machine)
	for i, c := range v.Channels {
		does := "plays " + c.Role
		var what []string
		if c.Echo != "" {
			does = "echoes " + c.Echo
			what = append(what, fmt.Sprintf("%s beats late at %v",
				c.Delay, *c.Gain))
		}
		if c.Duty != nil {
			what = append(what, fmt.Sprintf("duty %v", *c.Duty))
		}
		if c.Table != "" {
			what = append(what, "table "+c.Table)
		}
		if c.Volume != nil {
			what = append(what, fmt.Sprintf("volume %v", *c.Volume))
		}
		if c.Detune != 0 {
			what = append(what,
				fmt.Sprintf("detune %v cents", c.Detune))
		}
		if c.Pan != "" && c.Pan != "both" {
			what = append(what, "pan "+c.Pan)
		}
		if e := c.Envelope; e != nil {
			what = append(what, fmt.Sprintf("envelope %v %v %v %v",
				e.Attack, e.Decay, e.Sustain, e.Release))
		}
		if vb := c.Vibrato; vb != nil {
			what = append(what, fmt.Sprintf(
				"vibrato %v Hz %v cents after %v s",
				vb.Rate, vb.Depth, vb.Delay))
		}
		if c.Kind != "noise" && c.Arp.set {
			what = append(what, "arp "+c.Arp.String())
		}
		if len(c.Hits) > 0 {
			names := make([]string, 0, len(c.Hits))
			for h := range c.Hits {
				names = append(names, h)
			}
			sort.Strings(names)
			what = append(what, "hits "+strings.Join(names, " "))
		}
		fmt.Fprintf(w, "  %d %-5s %-12s %s\n", i+1, c.Kind, does,
			strings.Join(what, ", "))
	}
	if len(v.Drop) > 0 {
		fmt.Fprintf(w, "  drops %s\n", strings.Join(v.Drop, ", "))
	}
}
