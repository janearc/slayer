package cheeptune

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Roles are the parts a score may hold. Four, because the first machine
// has four channels and a role nothing can play is a role nobody should
// write. The studio keeps the four, so every score plays on both.
var Roles = []string{"lead", "bass", "pad", "perc"}

// Score is what is played: roles, each with bars of notes, and no sound.
type Score struct {
	Name  string  `json:"name"`
	Tempo float64 `json:"tempo"`
	Meter int     `json:"meter"`
	Loop  bool    `json:"loop"`
	Parts []Part  `json:"parts"`

	// Tempos are the places the tempo changes, each at the start of a
	// bar. Without any, the piece keeps its tempo; tempo.go has the map.
	Tempos []TempoChange `json:"tempos"`

	spans []tempoSpan
}

// Part is one role's bars. Each bar is a string of tokens; DESIGN.md has the
// grammar.
type Part struct {
	Role string   `json:"role"`
	Bars []string `json:"bars"`

	events []event
}

// event is one token that sounds. Rests are not events: they are the gaps.
type event struct {
	at, len frac  // beats from the start of the piece
	pitches []int // midi numbers. one for a note, several for a chord
	hit     string
	stack   []string // the hits of a chord of hits, before it is split
	bar     int      // one-based, for errors and for the roll
	loud    float64  // the gain its loudness mark gives; one with none
	marked  bool
	slide   bool // it slides into the next note, which does not restart
}

// BeatSeconds is how long one beat lasts at the piece's first tempo. A
// piece whose tempo changes asks SecondsAt and BeatsAt instead.
func (s *Score) BeatSeconds() float64 { return 60 / s.Tempo }

// Bars is how many bars the piece holds.
func (s *Score) Bars() int {
	if len(s.Parts) == 0 {
		return 0
	}
	return len(s.Parts[0].Bars)
}

// Seconds is the piece's length, without any tail a release or echo adds.
func (s *Score) Seconds() float64 {
	return s.SecondsAt(float64(s.Bars() * s.Meter))
}

// ReadScore reads and checks a score, from a path or a built-in named with
// "@". Every problem is returned, joined, not just the first: a document with
// four mistakes should take one round to fix.
func ReadScore(path string) (*Score, error) {
	b, err := readDoc(path)
	if err != nil {
		return nil, err
	}
	return ParseScore(b)
}

// ParseScore is ReadScore on bytes already in hand.
func ParseScore(b []byte) (*Score, error) {
	var s Score
	if err := decodeStrict(b, &s); err != nil {
		return nil, fmt.Errorf("score: %w", err)
	}
	if err := s.check(); err != nil {
		return nil, err
	}
	return &s, nil
}

// check is every rule a score keeps on its own, all problems joined.
func (s *Score) check() error {
	var errs []error
	fail := func(f string, a ...any) {
		errs = append(errs, fmt.Errorf(f, a...))
	}
	if s.Name == "" {
		fail("score has no name")
	}
	// the tempo range is the register, not a rule about taste: below forty
	// a beat is longer than the driver can usefully count, above three
	// hundred a sixteenth is shorter than two driver ticks.
	if s.Tempo < 40 || s.Tempo > 300 {
		fail("tempo %v is outside 40 to 300", s.Tempo)
	}
	if s.Meter < 1 || s.Meter > 16 {
		fail("meter %d is outside 1 to 16", s.Meter)
	}
	if len(s.Parts) == 0 {
		fail("score has no parts")
	}
	errs = append(errs, s.checkTempos()...)
	seen := map[string]bool{}
	for i := range s.Parts {
		p := &s.Parts[i]
		if !known(p.Role, Roles) {
			fail("part %d: unknown role %q; roles are %s",
				i+1, p.Role, strings.Join(Roles, ", "))
			continue
		}
		if seen[p.Role] {
			fail("%s: the role appears twice", p.Role)
		}
		seen[p.Role] = true
		if len(p.Bars) != len(s.Parts[0].Bars) {
			first := s.Parts[0]
			fail("%s has %d bars and %s has %d; every part has "+
				"the same", p.Role, len(p.Bars),
				first.Role, len(first.Bars))
		}
		if len(p.Bars) == 0 {
			fail("%s has no bars", p.Role)
		}
		if s.Meter >= 1 {
			ev, perr := parseBars(p.Role, p.Bars, s.Meter)
			errs = append(errs, perr...)
			p.events = ev
		}
	}
	return errors.Join(errs...)
}

// parseBars turns a part's bars into events, checking each bar adds up to the
// meter exactly and that every tie lands on the same pitch.
func parseBars(role string, bars []string, meter int) ([]event, []error) {
	b := barReader{role: role}
	whole := frac{int64(meter), 1}
	at := frac{0, 1}
	for i, bar := range bars {
		b.bar = i + 1
		if sum, ok := b.read(bar, at); ok && sum != whole {
			b.fail("holds %s beats and the meter is %d", sum, meter)
		}
		at = at.add(whole)
	}
	if b.tied {
		b.fail("ties into a bar that is not there")
	}
	if b.slid {
		b.fail("slides into a bar that is not there")
	}
	return b.out, b.errs
}

// barReader is the state of reading one part, bar by bar.
type barReader struct {
	role string
	bar  int
	out  []event
	errs []error
	tied bool // the last event is waiting for its tie to land
	slid bool // the last event is waiting for its slide to land
}

// fail records a problem in the bar being read.
func (b *barReader) fail(f string, a ...any) {
	prefix := fmt.Sprintf("%s bar %d: ", b.role, b.bar)
	b.errs = append(b.errs, fmt.Errorf(prefix+f, a...))
}

// read reads one bar starting at beat at, and says what its tokens add up
// to. ok is false when a token did not parse: its length is unknown, so the
// sum is wrong for a reason already reported and saying so twice is noise.
func (b *barReader) read(bar string, at frac) (sum frac, ok bool) {
	toks, err := tokens(bar)
	if err != nil {
		b.fail("%v", err)
		b.tied = false
		return sum, false
	}
	sum, ok = frac{0, 1}, true
	for i, t := range toks {
		ev, tie, rest, err := parseToken(t)
		if err != nil {
			b.fail("token %d %q: %v", i+1, t, err)
			b.tied, b.slid, ok = false, false, false
			continue
		}
		pos := at.add(sum)
		sum = sum.add(ev.len)
		if b.tied && b.land(ev, tie, rest, t) {
			b.slid = ev.slide
			continue
		}
		if b.slid && (rest || len(ev.pitches) != 1) {
			b.fail("token %d %q: the slide before it lands on a "+
				"single note, not a rest, a chord or a hit", i+1, t)
		}
		b.slid = ev.slide
		if rest {
			continue
		}
		ev.at, ev.bar = pos, b.bar
		if len(ev.stack) > 0 {
			// a chord of hits is one event a hit, all at one moment.
			for _, name := range ev.stack {
				one := ev
				one.hit, one.stack = name, nil
				b.out = append(b.out, one)
			}
			continue
		}
		b.out = append(b.out, ev)
		switch {
		case !tie:
		case len(ev.pitches) != 1:
			b.fail("token %d %q: only a single note ties", i+1, t)
		case i != len(toks)-1:
			b.fail("token %d %q: a tie is only on a bar's "+
				"last token", i+1, t)
		default:
			b.tied = true
		}
	}
	return sum, ok
}

// land joins a tied note to the one before it, and says whether it did. A
// tie is only ever set on a bar's last token, so the token landing it is the
// first of the next bar.
func (b *barReader) land(ev event, tie, rest bool, t string) bool {
	b.tied = false
	last := &b.out[len(b.out)-1]
	if rest || len(ev.pitches) != 1 || ev.pitches[0] != last.pitches[0] {
		b.fail("the tie from bar %d lands on %q, not the same pitch",
			b.bar-1, t)
		return false
	}
	if ev.marked {
		b.fail("the tie from bar %d lands on %q, which has a loudness "+
			"mark; a tied note keeps the loudness it began with",
			b.bar-1, t)
	}
	last.len = last.len.add(ev.len)
	last.slide = ev.slide
	b.tied = tie
	return true
}

// tokens splits a bar on spaces, keeping a chord's parentheses whole.
func tokens(bar string) ([]string, error) {
	var out []string
	var cur strings.Builder
	depth := 0
	for _, r := range bar {
		switch {
		case r == '(':
			if depth > 0 {
				return nil, fmt.Errorf("a chord inside a chord")
			}
			depth++
		case r == ')':
			if depth == 0 {
				return nil, fmt.Errorf("a ) with no (")
			}
			depth--
		case r == ' ' || r == '\t':
			if depth == 0 {
				if cur.Len() > 0 {
					out = append(out, cur.String())
					cur.Reset()
				}
				continue
			}
		}
		cur.WriteRune(r)
	}
	if depth != 0 {
		return nil, fmt.Errorf("a ( with no )")
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	if len(out) == 0 {
		return nil, fmt.Errorf(
			"the bar is empty; a silent bar is \"-:n\"")
	}
	return out, nil
}

// parseToken reads one token: a note, a chord, a rest or a hit, with its
// length and tie.
func parseToken(t string) (ev event, tie, rest bool, err error) {
	t, tie = strings.CutSuffix(t, "~")
	t, ev.slide = strings.CutSuffix(t, ">")
	if ev.slide && (tie || strings.HasSuffix(t, "~")) {
		return ev, tie, rest, fmt.Errorf("a note slides (>) or ties " +
			"(~), not both")
	}
	ev.len, ev.loud = frac{1, 1}, 1
	if i := strings.LastIndex(t, "@"); i >= 0 {
		gain, ok := Marks[t[i+1:]]
		if !ok {
			return ev, tie, rest, fmt.Errorf("loudness mark %q; marks "+
				"are %s", t[i+1:], strings.Join(markNames, ", "))
		}
		t, ev.loud, ev.marked = t[:i], gain, true
	}
	body := t
	// the length is after the last colon outside a chord.
	if i := strings.LastIndex(t, ":"); i > strings.LastIndex(t, ")") {
		body = t[:i]
		if ev.len, err = parseFrac(t[i+1:]); err != nil {
			return ev, tie, rest, err
		}
	}
	switch {
	case body == "-":
		if tie {
			return ev, tie, rest, fmt.Errorf("a rest does not tie")
		}
		if ev.slide {
			return ev, tie, rest, fmt.Errorf("only a single note slides")
		}
		if ev.marked {
			return ev, tie, rest, fmt.Errorf(
				"a rest has no loudness to mark")
		}
		return ev, false, true, nil
	case strings.HasPrefix(body, "("):
		inner := strings.Fields(strings.Trim(body, "()"))
		if len(inner) < 2 {
			return ev, tie, rest,
				fmt.Errorf("a chord has two notes or more")
		}
		if stack, err := hitStack(inner); err != nil || stack != nil {
			if err == nil && tie {
				err = fmt.Errorf("a hit does not tie")
			}
			if err == nil && ev.slide {
				err = fmt.Errorf("only a single note slides")
			}
			ev.stack = stack
			return ev, tie, false, err
		}
		for _, n := range inner {
			m, perr := pitch(n)
			if perr != nil {
				return ev, tie, rest, perr
			}
			ev.pitches = append(ev.pitches, m)
		}
	case isPitch(body):
		m, perr := pitch(body)
		if perr != nil {
			return ev, tie, rest, perr
		}
		ev.pitches = []int{m}
	case isWord(body):
		if tie {
			return ev, tie, rest, fmt.Errorf("a hit does not tie")
		}
		ev.hit = body
	default:
		return ev, tie, rest, fmt.Errorf(
			"not a note (a4, c#5, bb3), chord, rest (-) or hit (k)")
	}
	if ev.slide && len(ev.pitches) != 1 {
		return ev, tie, rest, fmt.Errorf("only a single note slides")
	}
	return ev, tie, false, nil
}

// hitStack reads the inside of a chord as hits struck together, (k h).
// It is nil when the chord is of notes. A chord is all notes or all
// hits, and names a hit once.
func hitStack(inner []string) ([]string, error) {
	hits := 0
	seen := map[string]bool{}
	for _, name := range inner {
		if isWord(name) && !isPitch(name) {
			hits++
			if seen[name] {
				return nil, fmt.Errorf("the hit %s is in the chord "+
					"twice", name)
			}
			seen[name] = true
		}
	}
	switch hits {
	case 0:
		return nil, nil
	case len(inner):
		return inner, nil
	}
	return nil, fmt.Errorf("a chord is notes or hits, not both")
}

// isPitch is the shape of a note: a letter, an optional accidental, digits.
func isPitch(s string) bool {
	if len(s) < 2 || s[0] < 'a' || s[0] > 'g' {
		return false
	}
	r := s[1:]
	if r[0] == '#' || r[0] == 'b' {
		r = r[1:]
	}
	if r == "" {
		return false
	}
	for _, c := range r {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isWord is the shape of a hit: lowercase letters and nothing else.
func isWord(s string) bool {
	for _, c := range s {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return s != ""
}

// pitch is a note name as a midi number, where c4 is 60 and a4 is 440 Hz.
func pitch(s string) (int, error) {
	if !isPitch(s) {
		return 0, fmt.Errorf("%q is not a note (a4, c#5, bb3)", s)
	}
	semi := map[byte]int{'c': 0, 'd': 2, 'e': 4, 'f': 5, 'g': 7,
		'a': 9, 'b': 11}[s[0]]
	r := s[1:]
	switch r[0] {
	case '#':
		semi++
		r = r[1:]
	case 'b':
		semi--
		r = r[1:]
	}
	oct := 0
	for _, c := range r {
		oct = oct*10 + int(c-'0')
	}
	if oct > 9 {
		return 0, fmt.Errorf("%q: octave %d is past 9", s, oct)
	}
	return (oct+1)*12 + semi, nil
}

// NoteName is a midi number as the score would write it.
func NoteName(m int) string {
	names := []string{"c", "c#", "d", "d#", "e", "f", "f#", "g", "g#",
		"a", "a#", "b"}
	return fmt.Sprintf("%s%d", names[((m%12)+12)%12], m/12-1)
}

// known is whether s is one of set.
func known(s string, set []string) bool { return slices.Contains(set, s) }

// Event is one thing a part sounds: a note, a chord or a hit, and where it
// falls.
//
// The document's own side of a piece, as Piece.Ticks is the machine's. A
// tool that wants to show or edit what was written reads this; a tool that
// wants to know what was played reads the ticks.
type Event struct {
	Beats  float64 // from the start of the piece
	Length float64 // in beats
	Notes  []int   // midi numbers; more than one is a chord
	Hit    string  // the hit's name, on a percussion part
	Bar    int     // one-based, as the document writes it
	Loud   float64 // the gain its loudness mark gives; one with none
	Slide  bool    // it slides into the next note
}

// Events is everything a part sounds, in order. Rests are the gaps.
func (p *Part) Events() []Event {
	out := make([]Event, 0, len(p.events))
	for _, e := range p.events {
		out = append(out, Event{
			Beats: e.at.float(), Length: e.len.float(),
			Notes: append([]int(nil), e.pitches...),
			Hit:   e.hit, Bar: e.bar, Loud: e.loud, Slide: e.slide,
		})
	}
	return out
}
