package show

import (
	"math"
	"strings"
	"testing"
)

// near is how close two numbers must be to be the same here.
const near = 1e-9

// steady is a grid of beats half a second apart from nought: 120 bpm.
func steady(count int) []float64 {
	beats := make([]float64, count)
	for i := range beats {
		beats[i] = 0.5 * float64(i)
	}
	return beats
}

// a ramp holds its first value until it starts, eases to its last over
// its seconds, half way at the half, and holds that after; a ramp of no
// seconds is a cut.
func TestARampEasesBetweenItsEnds(t *testing.T) {
	if got := held(0.4).at(123); !(math.Abs(got-0.4) <= near) {
		t.Errorf("held at 0.4 is %v", got)
	}
	r := ramp{from: 0, to: 1, start: 10, seconds: 2}
	for _, each := range []struct{ at, want float64 }{
		{9, 0}, {10, 0}, {11, 0.5}, {12, 1}, {13, 1},
	} {
		if got := r.at(each.at); !(math.Abs(got-each.want) <= near) {
			t.Errorf("at %v the ramp is %v, not %v", each.at, got, each.want)
		}
	}
	if got := r.at(10.5); got <= 0 || got >= 0.25 {
		t.Errorf("a quarter in, an eased ramp is %v, under a quarter", got)
	}
	cut := ramp{from: 0, to: 1, start: 5}
	if cut.at(4.9) != 0 || cut.at(5) != 1 {
		t.Errorf("a cut is %v before and %v at its moment", cut.at(4.9),
			cut.at(5))
	}
}

// a glide starts from wherever the ramp is at its moment, so a cue that
// interrupts a fade does not jump.
func TestAGlideStartsWhereTheRampIs(t *testing.T) {
	r := held(0)
	r.glide(1, 0, 2)
	r.glide(0, 1, 2)
	if got := r.at(1); !(math.Abs(got-0.5) <= near) {
		t.Errorf("interrupted half way, the ramp is %v, not 0.5", got)
	}
	if got := r.at(3); got != 0 {
		t.Errorf("after the second glide the ramp is %v, not 0", got)
	}
}

// a moment's place on the grid is beats from its start, fractional, and
// none before it; past the last beat the last spacing carries on. A
// place back to seconds is the same moment.
func TestTheGridPlacesAMoment(t *testing.T) {
	beats := steady(5)
	for _, each := range []struct{ seconds, place float64 }{
		{-1, -1}, {0, 0}, {0.25, 0.5}, {1.75, 3.5}, {3, 6},
	} {
		if got := gridAt(each.seconds, beats); !(math.Abs(got-each.place) <= near) {
			t.Errorf("%v s is %v on the grid, not %v", each.seconds, got,
				each.place)
		}
		if each.place >= 0 {
			if back := beatTime(each.place, beats); !(math.Abs(back-each.seconds) <= near) {
				t.Errorf("%v on the grid is %v s, not %v", each.place, back,
					each.seconds)
			}
		}
	}
	if beatTime(-1, beats) != -1 {
		t.Error("a place before the grid has a time")
	}
}

// a cue at bar.beat is that measured beat; at_sec is as it is.
func TestACueAtBarAndBeatIsOnTheGrid(t *testing.T) {
	beats := steady(12)
	score := &Score{beats: beats, measured: make([]bool, len(beats))}
	if got, err := score.Moment("2.3", nil); err != nil || got != beats[6] {
		t.Errorf("2.3 is %v s (%v), not beat 6's %v", got, err, beats[6])
	}
	if got, err := score.Moment("3", nil); err != nil || got != beats[8] {
		t.Errorf("bar 3 is %v s (%v), not its first beat's %v", got, err,
			beats[8])
	}
	at := 1.25
	if got, err := score.Moment("1.1", &at); err != nil || got != at {
		t.Errorf("at_sec 1.25 is %v s (%v)", got, err)
	}
}

// a cue at a bar the score lacks, or that is not bar.beat, is refused
// and names what it was given.
func TestACueOffTheGridIsRefused(t *testing.T) {
	beats := steady(12)
	score := &Score{beats: beats, measured: make([]bool, len(beats))}
	for _, at := range []string{"4.1", "0.1", "1.5", "x", "1.y"} {
		if _, err := score.Moment(at, nil); err == nil ||
			!strings.Contains(err.Error(), at) {
			t.Errorf("%q gave %v, not a refusal naming it", at, err)
		}
	}
}

// a pulse strikes full on each of its hits and falls by its decay, in
// sixteenths, until the next; its hits are counted from the grid's start;
// a pulse of no steps holds the fan full, one of no hits holds it dark.
func TestAPulseStrikesOnItsHitsAndFalls(t *testing.T) {
	const pulse, decay = "x...x...x...x...", 1.5
	for _, each := range []struct {
		position, level float64
		hit             int
	}{
		{0, 1, 0},
		{0.125, math.Exp(-0.5 / decay), 0},
		{0.25, math.Exp(-1 / decay), 0},
		{1, 1, 1},
		{4, 1, 4},
	} {
		level, hit := gated(pulse, decay, each.position)
		if !(math.Abs(level-each.level) <= near) || hit != each.hit {
			t.Errorf("at %v the pulse is %v, hit %d; not %v, hit %d",
				each.position, level, hit, each.level, each.hit)
		}
	}
	if level, _ := gated("", decay, 3); level != 1 {
		t.Errorf("no pulse holds the fan at %v", level)
	}
	if level, _ := gated("................", decay, 3); level != 0 {
		t.Errorf("a pulse of no hits holds the fan at %v", level)
	}
}

// the fires burst on their cue's beat and every four beats after, and a
// burst shows for burstSpan: each moment says how long ago each burst
// still showing began, newest first, and none before the cue.
func TestTheFiresBurstEveryFourBeatsFromTheirCue(t *testing.T) {
	beats := steady(40)
	const from = 1.0
	for _, each := range []struct {
		position float64
		want     []float64
	}{
		{0.5, nil},
		{1, []float64{0}},
		{5.5, []float64{0.25, 2.25}},
		{12, []float64{1.5}},
	} {
		got := bursts(beatTime(each.position, beats), each.position, from, beats)
		if len(got) != len(each.want) {
			t.Errorf("at %v the bursts are %v, not %v", each.position, got,
				each.want)
			continue
		}
		for i := range got {
			if !(math.Abs(got[i]-each.want[i]) <= near) {
				t.Errorf("at %v the bursts are %v, not %v", each.position,
					got, each.want)
			}
		}
	}
	if got := bursts(3, 6, -1, beats); got != nil {
		t.Errorf("unlit, the fires burst %v", got)
	}
}

// the fires are lit at their cue's beat: before it, and without a cue,
// they are not; after it, they have been lit since its beat, however
// many bursts have come and gone.
func TestTheFiresAreLitFromTheirCuesBeat(t *testing.T) {
	beats := steady(40)
	const from = 1.0
	for _, each := range []struct{ position, want float64 }{
		{0.5, -1},
		{1, 0},
		{12, beatTime(12, beats) - beatTime(from, beats)},
	} {
		got := lit(beatTime(each.position, beats), each.position, from, beats)
		if !(math.Abs(got-each.want) <= near) {
			t.Errorf("at %v the fires were lit %v ago, not %v", each.position,
				got, each.want)
		}
	}
	if got := lit(3, 6, -1, beats); got != -1 {
		t.Errorf("without a cue, the fires were lit %v ago", got)
	}
}

// a list taken in turn is the first before its cue or the grid, then
// each in turn so many bars apiece from the cue, with how far through
// its bars; nothing of an empty list.
func TestAListIsTakenInTurn(t *testing.T) {
	looks := []string{"fan", "scan", "beam"}
	for _, each := range []struct {
		position float64
		want     string
		through  float64
	}{
		{-1, "fan", 0}, {2, "fan", 0}, {4, "fan", 0}, {8, "fan", 0.5},
		{12, "scan", 0}, {20, "beam", 0}, {28, "fan", 0},
	} {
		got, through := inTurn(looks, 2, 4, each.position)
		if got != each.want || !(math.Abs(through-each.through) <= near) {
			t.Errorf("at %v: %s %.3f, not %s %.3f", each.position, got,
				through, each.want, each.through)
		}
	}
	if got, _ := inTurn(nil, 2, 0, 5); got != "" {
		t.Errorf("an empty list gives %q", got)
	}
}

// the floods fire on the ones a side at a time, left then right, each
// flash falling by floodFlash.
func TestTheFloodsAlternateSides(t *testing.T) {
	beats := steady(40)
	left, right := flooded("ones", beatTime(0, beats), 0, beats, nil, 0)
	if !(math.Abs(left-1) <= near) || right != 0 {
		t.Errorf("on the first one, left %v and right %v", left, right)
	}
	left, right = flooded("ones", beatTime(4, beats), 4, beats, nil, 0)
	if !(math.Abs(right-1) <= near) || !(left <= 1e-6) {
		t.Errorf("on the second one, left %v and right %v", left, right)
	}
	later := beatTime(4, beats) + floodFlash
	_, right = flooded("ones", later, 4+floodFlash/0.5, beats, nil, 0)
	if !(math.Abs(right-math.Exp(-1)) <= 1e-6) {
		t.Errorf("a flash floodFlash on is %v, not a third", right)
	}
	if left, right := flooded("off", 1, 2, beats, nil, 0); left != 0 || right != 0 {
		t.Errorf("off, the floods are %v and %v", left, right)
	}
}
