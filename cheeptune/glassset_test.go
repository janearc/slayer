package cheeptune

import (
	"fmt"
	"testing"
)

// The glass-hour set is written by hand, each member to a plan in its
// score's comment. Nobody on the team can hear it, so these tests hold
// the two things a plan claims that can be measured: that glass-hour's
// own melody is in the member where the plan says, note for note, and
// that the part the plan calls the break is louder than the thin parts
// either side of it.

// statement is one place a member states bars of glass-hour's lead.
type statement struct {
	role  string // the member's part that carries it
	from  int    // the first glass-hour bar stated, counted from one
	at    int    // the member's bar it lands on
	bars  int    // how many bars
	shift int    // semitones the member moves it by
	quiet bool   // the plan states it quietly: the cold part after
}

// octaveDown is the melody moved an octave lower, in semitones. Defined.
const octaveDown = -12

// glassStatements is every statement each member's plan claims.
var glassStatements = map[string][]statement{
	"glass-acid":  acidStatements,
	"glass-chase": chaseStatements,
}

// chaseStatements is where glass-chase's plan puts glass-hour, all of it
// at its own height: bars 1 to 4, 5 to 7 twice, the crack twice, and the
// last three bars.
var chaseStatements = []statement{
	{role: "lead", from: 1, at: 3, bars: 4},
	{role: "lead", from: 5, at: 7, bars: 3},
	{role: "lead", from: 5, at: 11, bars: 3},
	{role: "lead", from: 9, at: 15, bars: 2},
	{role: "lead", from: 9, at: 19, bars: 2},
	{role: "lead", from: 11, at: 23, bars: 3, quiet: true},
}

// acidStatements is where glass-acid's plan puts glass-hour: its bars 1
// to 4 and 5 to 7 an octave down, 5 to 7 again at their own height, and
// the crack, bars 9 and 10, twice, an octave down.
var acidStatements = []statement{
	{role: "lead", from: 1, at: 3, bars: 4, shift: octaveDown},
	{role: "lead", from: 5, at: 7, bars: 3, shift: octaveDown},
	{role: "lead", from: 5, at: 11, bars: 3},
	{role: "lead", from: 9, at: 15, bars: 2, shift: octaveDown},
	{role: "lead", from: 9, at: 19, bars: 2, shift: octaveDown},
}

// partOf is a score's part by role.
func partOf(t *testing.T, s *Score, role string) *Part {
	t.Helper()
	for i := range s.Parts {
		if s.Parts[i].Role == role {
			return &s.Parts[i]
		}
	}
	t.Fatalf("%s has no %s part", s.Name, role)
	return nil
}

// startsNote is whether a part starts a note of a pitch on a beat, and how
// loud its mark makes it.
func startsNote(p *Part, beats float64, midi int) (loud float64, ok bool) {
	for _, e := range p.Events() {
		if e.Beats != beats {
			continue
		}
		for _, n := range e.Notes {
			if n == midi {
				return e.Loud, true
			}
		}
	}
	return 0, false
}

// misses lists the notes of a statement the member does not have: each
// glass-hour note in the stated bars wants a note of the moved pitch in
// the member's part at the same place in its bar.
func misses(theme, member *Score, st statement) []string {
	var out []string
	var lead, part *Part
	for i := range theme.Parts {
		if theme.Parts[i].Role == "lead" {
			lead = &theme.Parts[i]
		}
	}
	for i := range member.Parts {
		if member.Parts[i].Role == st.role {
			part = &member.Parts[i]
		}
	}
	if lead == nil || part == nil {
		return []string{"a part is missing"}
	}
	meter := float64(theme.Meter)
	for _, e := range lead.Events() {
		if e.Bar < st.from || e.Bar >= st.from+st.bars {
			continue
		}
		beats := e.Beats + float64(st.at-st.from)*meter
		for _, n := range e.Notes {
			if _, ok := startsNote(part, beats, n+st.shift); !ok {
				out = append(out, fmt.Sprintf("bar %d: no %s at beat %v",
					e.Bar-st.from+st.at, NoteName(n+st.shift), beats))
			}
		}
	}
	return out
}

func TestEveryGlassMemberStatesGlassHourWhereItsPlanSays(t *testing.T) {
	theme, err := ReadScore("@glass-hour")
	if err != nil {
		t.Fatal(err)
	}
	for name, sts := range glassStatements {
		member, err := ReadScore("@" + name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		for _, st := range sts {
			for _, miss := range misses(theme, member, st) {
				t.Errorf("%s, glass-hour bar %d on: %s", name, st.from,
					miss)
			}
		}
	}
}

func TestAStatementInTheWrongPlaceIsMissed(t *testing.T) {
	theme, err := ReadScore("@glass-hour")
	if err != nil {
		t.Fatal(err)
	}
	member, err := ReadScore("@glass-acid")
	if err != nil {
		t.Fatal(err)
	}
	// the near misses: the right bars a bar late, and at the wrong height.
	late := statement{role: "lead", from: 1, at: 4, bars: 4,
		shift: octaveDown}
	high := statement{role: "lead", from: 1, at: 3, bars: 4}
	for _, st := range []statement{late, high} {
		if len(misses(theme, member, st)) == 0 {
			t.Errorf("%+v was found, and it is not there", st)
		}
	}
}

func TestTheFastLineMarksTheMelodyLouderThanItsFilling(t *testing.T) {
	theme, err := ReadScore("@glass-hour")
	if err != nil {
		t.Fatal(err)
	}
	for name, sts := range glassStatements {
		member, err := ReadScore("@" + name)
		if err != nil {
			t.Fatal(err)
		}
		lead := partOf(t, theme, "lead")
		for _, st := range sts {
			if st.role != "lead" || st.quiet {
				continue
			}
			part := partOf(t, member, st.role)
			for _, e := range lead.Events() {
				if e.Bar < st.from || e.Bar >= st.from+st.bars {
					continue
				}
				beats := e.Beats +
					float64((st.at-st.from)*theme.Meter)
				loud, ok := startsNote(part, beats, e.Notes[0]+st.shift)
				if ok && loud < Marks["f"] {
					t.Errorf("%s bar %d: the melody's %s is marked %v, "+
						"under f", name, e.Bar-st.from+st.at,
						NoteName(e.Notes[0]+st.shift), loud)
				}
			}
		}
	}
}
