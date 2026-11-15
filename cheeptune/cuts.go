package cheeptune

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// A cut sheet says where the parts of a tune fall, so a video can be cut
// to them.
//
// The design, written before the code. A tune made for a video is a score
// plus a small JSON file beside it, NAME.cuts.json, holding the mood, the
// voicing it was written for, the tempo and meter, the length in seconds
// and the parts: each with a name, the bar it starts on (counted from one,
// as the checker counts), the second it starts at, and one line on what
// changes there.
//
// ParseCuts reads a sheet against its score and refuses it by name when
// they disagree: the tempo or meter differ, a bar is not in the score,
// parts are out of order, or a second is not what the bar makes at the
// score's tempo. Spans then gives each part's start and end. A job's sound
// does not read sheets yet; the shorts picker will.

// secondsTolerance is how far a written second may be from the one the bar
// makes, in seconds. Picked by the writer of this package, 2026-11-12:
// sheets are written to two decimals, so a correct one is out by at most
// half a hundredth, and a little over that is room for the sum.
const secondsTolerance = 0.006

// minimumParts is the fewest parts a sheet holds. Picked, 2026-11-12: a
// tune for a video has at least one change in it, and one part has none.
const minimumParts = 2

// moodWordsFewest and moodWordsMost bound the mood, in words. Picked
// 2026-11-12: two to five words.
const (
	moodWordsFewest = 2
	moodWordsMost   = 5
)

// feelsWordsMost is the most words a part's feels phrase holds. Picked by
// the writer of this package, 2026-11-12: a phrase a viewer reads at a
// glance beside a timeline, such as "everything arrives: release".
const feelsWordsMost = 10

// groupWordsMost is the most words a group holds. Picked, 2026-11-12: the
// group is a heading a page collects tunes under, so it is short.
const groupWordsMost = 5

// CutSheet is the parts of one tune and what the sheet says about it. The
// group is the family a page lists it under. The mood, feeling, intent and listen_for are plain words for a person who
// cannot read a score: what the tune is, what a viewer feels, what video
// it is for and the one thing to notice.
type CutSheet struct {
	Group     string  `json:"group"`
	Mood      string  `json:"mood"`
	Feeling   string  `json:"feeling"`
	Intent    string  `json:"intent"`
	ListenFor string  `json:"listen_for"`
	Voicing   string  `json:"voicing"`
	Tempo     float64 `json:"tempo"`
	Meter     int     `json:"meter"`
	Seconds   float64 `json:"seconds"`
	Parts     []Cut   `json:"parts"`
}

// Cut is one part: where it starts and what changes there.
type Cut struct {
	Name   string  `json:"name"`
	Bar    int     `json:"bar"`
	Second float64 `json:"second"`
	Change string  `json:"change"`
	Feels  string  `json:"feels"`
}

// Span is a part with its start and end as times in the rendered piece.
type Span struct {
	Name       string
	Bar        int
	Start, End time.Duration
	Change     string
	Feels      string
}

// ReadCuts reads a cut sheet from a path and checks it against the score.
func ReadCuts(path string, s *Score) (*CutSheet, error) {
	b, err := readDoc(path)
	if err != nil {
		return nil, err
	}
	c, err := ParseCuts(b, s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// ParseCuts is ReadCuts on bytes already in hand. Every problem is
// returned, one a line, not just the first.
func ParseCuts(b []byte, s *Score) (*CutSheet, error) {
	var c CutSheet
	if err := decodeStrict(b, &c); err != nil {
		return nil, fmt.Errorf("not a cut sheet: %w", err)
	}
	var errs []error
	errs = append(errs, c.checkHeader(s)...)
	errs = append(errs, c.checkParts(s)...)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return &c, nil
}

// sameSecond says two seconds agree to within the tolerance.
func sameSecond(a, b float64) bool { return math.Abs(a-b) <= secondsTolerance }

// checkHeader checks the fields that are about the whole tune.
func (c *CutSheet) checkHeader(s *Score) []error {
	var errs []error
	if n := len(strings.Fields(c.Mood)); n < moodWordsFewest ||
		n > moodWordsMost {
		errs = append(errs, fmt.Errorf("the mood %q is %d words; say it "+
			"in %d to %d", c.Mood, n, moodWordsFewest, moodWordsMost))
	}
	if n := len(strings.Fields(c.Group)); n < 1 || n > groupWordsMost {
		errs = append(errs, fmt.Errorf("the group %q is %d words; name "+
			"the family of tunes it belongs to in 1 to %d", c.Group, n,
			groupWordsMost))
	}
	for name, text := range map[string]string{"feeling": c.Feeling,
		"intent": c.Intent, "listen_for": c.ListenFor} {
		if strings.TrimSpace(text) == "" {
			errs = append(errs, fmt.Errorf("the sheet has no %s; write "+
				"one plain sentence", name))
		}
	}
	if !strings.HasPrefix(c.Voicing, "@") {
		errs = append(errs, fmt.Errorf("voicing %q is not a built-in; "+
			"name one with an @, such as @leap-dmg", c.Voicing))
	}
	if c.Tempo != s.Tempo || c.Meter != s.Meter {
		errs = append(errs, fmt.Errorf("the sheet says %v a minute and "+
			"%d to a bar and the score says %v and %d; make them "+
			"agree", c.Tempo, c.Meter, s.Tempo, s.Meter))
	}
	if !sameSecond(c.Seconds, s.Seconds()) {
		errs = append(errs, fmt.Errorf("the sheet says %.2f s and the "+
			"score's %d bars make %.2f s; write the score's length",
			c.Seconds, s.Bars(), s.Seconds()))
	}
	return errs
}

// checkParts checks each part against the score and the one before it.
func (c *CutSheet) checkParts(s *Score) []error {
	if len(c.Parts) < minimumParts {
		return []error{fmt.Errorf("the sheet has %d parts and a tune "+
			"for a video has at least %d", len(c.Parts), minimumParts)}
	}
	var errs []error
	seen := map[string]bool{}
	last := 0
	for i, p := range c.Parts {
		at := fmt.Sprintf("part %d (%q)", i+1, p.Name)
		switch {
		case strings.TrimSpace(p.Name) == "":
			errs = append(errs, fmt.Errorf("part %d has no name", i+1))
		case seen[p.Name]:
			errs = append(errs, fmt.Errorf("%s: the name is used "+
				"twice; names are how a cut asks for a part", at))
		}
		seen[p.Name] = true
		if strings.TrimSpace(p.Change) == "" {
			errs = append(errs, fmt.Errorf("%s: say in one line what "+
				"changes there", at))
		}
		errs = append(errs, checkFeels(at, p.Feels)...)
		errs = append(errs, c.checkBar(s, at, i, p, last)...)
		if p.Bar >= 1 && p.Bar <= s.Bars() {
			last = p.Bar
		}
	}
	return errs
}

// checkFeels checks a part's feels phrase is there and short.
func checkFeels(at, feels string) []error {
	n := len(strings.Fields(feels))
	switch {
	case n == 0:
		return []error{fmt.Errorf("%s: say how the part feels, in a "+
			"short phrase", at)}
	case n > feelsWordsMost:
		return []error{fmt.Errorf("%s: the feels phrase is %d words and "+
			"the most is %d; shorten it", at, n, feelsWordsMost)}
	}
	return nil
}

// checkBar checks where one part starts: that its bar exists, comes after
// the last, and that its second is what the bar makes.
func (c *CutSheet) checkBar(s *Score, at string, i int, p Cut, last int) []error {
	var errs []error
	switch {
	case p.Bar < 1 || p.Bar > s.Bars():
		errs = append(errs, fmt.Errorf("%s: bar %d is not in the score, "+
			"which has bars 1 to %d", at, p.Bar, s.Bars()))
		return errs
	case i == 0 && p.Bar != 1:
		errs = append(errs, fmt.Errorf("%s: the first part starts on "+
			"bar 1, not bar %d", at, p.Bar))
	case i > 0 && p.Bar <= last:
		errs = append(errs, fmt.Errorf("%s: bar %d is not after bar %d; "+
			"parts are in order", at, p.Bar, last))
	}
	want := s.SecondsAt(float64((p.Bar - 1) * s.Meter))
	if !sameSecond(p.Second, want) {
		errs = append(errs, fmt.Errorf("%s: bar %d starts at %.2f s at "+
			"%v a minute, and the sheet says %.2f s", at, p.Bar, want,
			s.Tempo, p.Second))
	}
	return errs
}

// Spans gives each part's start and end in the rendered piece. A part ends
// where the next begins and the last ends at the sheet's length.
func (c *CutSheet) Spans() []Span {
	out := make([]Span, len(c.Parts))
	for i, p := range c.Parts {
		end := c.Seconds
		if i+1 < len(c.Parts) {
			end = c.Parts[i+1].Second
		}
		out[i] = Span{Name: p.Name, Bar: p.Bar, Change: p.Change, Feels: p.Feels,
			Start: asDuration(p.Second), End: asDuration(end)}
	}
	return out
}

// asDuration is a number of seconds as a duration.
func asDuration(x float64) time.Duration {
	return time.Duration(x * float64(time.Second))
}
