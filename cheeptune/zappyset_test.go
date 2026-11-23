package cheeptune

import "testing"

// Zappy's theme is written by hand around one motif of five notes. These
// tests hold what each piece's plan says about it that can be read from
// the notes: that the motif is where the plan puts it, in the form the
// plan names, and that the loud part measures louder than the quiet one.

// The motif's notes, as midi numbers: a4 d5 e5 a5, then the last note,
// which is f#5 in the major form and f5 in the minor. The bent form
// lowers the third note to e flat. Defined by zappy-ident's plan.
const (
	motifA4      = 69
	motifD5      = 74
	motifE5      = 76
	motifEFlat5  = 75
	motifA5      = 81
	motifFSharp5 = 78
	motifF5      = 77
)

// motifForms are the motif's five pitches in each form a plan names.
var motifForms = map[string][]int{
	"major": {motifA4, motifD5, motifE5, motifA5, motifFSharp5},
	"minor": {motifA4, motifD5, motifE5, motifA5, motifF5},
	"bent":  {motifA4, motifD5, motifEFlat5, motifA5, motifF5},
	"bent-major": {motifA4, motifD5, motifEFlat5, motifA5,
		motifFSharp5},
	"falling": {motifA5, motifE5, motifD5, motifA4, motifD5},
}

// motifBeats is when each of the motif's notes starts, in beats from its
// first: short, short, long, long, and the last on the next bar line.
var motifBeats = []float64{0, 0.5, 1, 2, 3}

// slowBeats is the motif at half speed as the plans write it: a beat
// each for the first two notes, two for the next two.
var slowBeats = []float64{0, 1, 2, 4, 6}

// motifAt is one place a plan puts the motif: the form, the bar and the
// beat in it that the first note starts on (both counted from one), how
// far it is moved in semitones, and whether it is at half speed.
type motifAt struct {
	form  string
	bar   int
	beat  float64
	shift int
	slow  bool
}

// octaveUp is the motif moved an octave higher, in semitones. Defined.
const octaveUp = 12

// zappyMotifs is every statement of the motif each plan claims. The
// motif's first note is on the second beat of its bar at speed, and on
// the third at half speed.
var zappyMotifs = map[string][]motifAt{
	"zappy-ident":         {{form: "major", bar: 1, beat: 2}},
	"zappy-ident-falling": {{form: "falling", bar: 1, beat: 2}},
	"zappy-bed-curious": {{form: "major", bar: 3, beat: 2},
		{form: "major", bar: 19, beat: 2}},
	"zappy-bed-ominous": {{form: "minor", bar: 3, beat: 2},
		{form: "major", bar: 10, beat: 2, shift: octaveUp},
		{form: "minor", bar: 13, beat: 2}},
	"zappy-bed-awe": {{form: "major", bar: 2, beat: 3, slow: true},
		{form: "major", bar: 10, beat: 3, slow: true}},
	"zappy-big-minor": {{form: "major", bar: 1, beat: 2},
		{form: "minor", bar: 3, beat: 2}, {form: "minor", bar: 5, beat: 2},
		{form: "major", bar: 21, beat: 2},
		{form: "major", bar: 23, beat: 3, slow: true}},
	"zappy-engine": {
		{form: "major", bar: 3, beat: 2, shift: octaveDown},
		{form: "minor", bar: 5, beat: 2, shift: octaveDown},
		{form: "minor", bar: 7, beat: 2, shift: octaveDown},
		{form: "major", bar: 23, beat: 2},
		{form: "major", bar: 25, beat: 3, slow: true}},
	"zappy-big-phrygian": {{form: "major", bar: 1, beat: 2},
		{form: "minor", bar: 3, beat: 2}, {form: "bent", bar: 5, beat: 2},
		{form: "major", bar: 21, beat: 2},
		{form: "bent-major", bar: 23, beat: 3, slow: true}},
}

// motifMisses lists the notes of a statement a score's lead does not
// start where the statement says.
func motifMisses(t *testing.T, s *Score, at motifAt) []string {
	t.Helper()
	lead := partOf(t, s, "lead")
	beats := motifBeats
	if at.slow {
		beats = slowBeats
	}
	first := float64((at.bar-1)*s.Meter) + at.beat - 1
	var out []string
	for i, midi := range motifForms[at.form] {
		if _, ok := startsNote(lead, first+beats[i], midi+at.shift); !ok {
			out = append(out, noteName(midi+at.shift))
		}
	}
	return out
}

func TestEveryZappyPieceStatesTheMotifWhereItsPlanSays(t *testing.T) {
	for name, places := range zappyMotifs {
		s, err := ReadScore("@" + name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		for _, at := range places {
			if miss := motifMisses(t, s, at); len(miss) > 0 {
				t.Errorf("%s: the %s motif at bar %d lacks %v", name,
					at.form, at.bar, miss)
			}
		}
	}
}

func TestAMotifInTheWrongFormOrPlaceIsMissed(t *testing.T) {
	s, err := ReadScore("@zappy-ident")
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []motifAt{
		{form: "minor", bar: 1, beat: 2},
		{form: "major", bar: 1, beat: 1},
		{form: "major", bar: 1, beat: 2, shift: octaveUp},
		{form: "major", bar: 1, beat: 2, slow: true},
	} {
		if len(motifMisses(t, s, at)) == 0 {
			t.Errorf("%+v was found in the ident, and it is not there",
				at)
		}
	}
}
