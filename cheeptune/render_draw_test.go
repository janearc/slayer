//go:build draw

package cheeptune

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func render(t *testing.T, sc, vc []byte) *Piece {
	t.Helper()
	s, err := ParseScore(sc)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVoicing(vc)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Render(s, v)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// frequency counts upward zero crossings in a stretch of samples.
func frequency(x []float32, rate int) float64 {
	first, last, n := -1, -1, 0
	for i := 1; i < len(x); i++ {
		if x[i-1] < 0 && x[i] >= 0 {
			if first < 0 {
				first = i
			} else {
				n++
			}
			last = i
		}
	}
	return float64(n) / (float64(last-first) / float64(rate))
}

func rms(x []float32) float64 {
	var s float64
	for _, v := range x {
		s += float64(v) * float64(v)
	}
	return math.Sqrt(s / float64(len(x)))
}

func seconds(p *Piece, a, b float64) []float32 {
	return p.Left[int(a*float64(p.Rate)):int(b*float64(p.Rate))]
}

func TestPitchIsWhereTheRegisterPutsIt(t *testing.T) {
	// a4 is 440 Hz on paper. the pulse register holds 2048 less 131072
	// over 440, rounded: 1750, and that plays 131072 / 298.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["a4:4"]}`),
		voicing(lead, ""))
	want := 131072.0 / 298
	if got := frequency(seconds(p, 0.2, 1.8), p.Rate); math.Abs(got-want) > 0.05 {
		t.Errorf("a4 on the pulse plays %.2f Hz, want %.2f", got, want)
	}
	// the wave register counts a whole 32-step cycle, so its top is
	// half the pulse's: a3 is 65536 / (2048 - 1750).
	p = render(t,
		score(4, false, `{"role": "bass", "bars": ["a3:4"]}`),
		voicing(`{"kind": "wave", "role": "bass", "table": "triangle"}`,
			""))
	want = 65536.0 / 298
	if got := frequency(seconds(p, 0.2, 1.8), p.Rate); math.Abs(got-want) > 0.05 {
		t.Errorf("a3 on the wave plays %.2f Hz, want %.2f", got, want)
	}
}

func TestAnEchoIsLaterQuieterAndWhereItIsPanned(t *testing.T) {
	// the lead left, its echo right, half a beat late at half the
	// volume. at 120 a beat is half a second.
	p := render(t,
		score(4, false, `{"role": "lead", "bars": ["c5:1/4 -:15/4"]}`),
		voicing(`{"kind": "pulse", "role": "lead", "duty": 0.5,
			"pan": "left"},
			{"kind": "pulse", "echo": "lead", "delay": "1/2",
			"gain": 0.5, "duty": 0.5, "pan": "right"}`, ""))
	onset := func(x []float32) float64 {
		for i, v := range x {
			if math.Abs(float64(v)) > 0.05 {
				return float64(i) / float64(p.Rate)
			}
		}
		return -1
	}
	l, r := onset(p.Left), onset(p.Right)
	tick := 1 / tickRate
	if math.Abs(l) > tick {
		t.Errorf("the lead starts at %.4f s, want within a tick of 0", l)
	}
	if math.Abs(r-0.25) > tick {
		t.Errorf("the echo starts at %.4f s, want within a tick of 0.25",
			r)
	}
	lead := rms(p.Left[int(l*44100)+500 : int(l*44100)+4000])
	echo := rms(p.Right[int(r*44100)+500 : int(r*44100)+4000])
	// half of fifteen is seven and a half, which the four-bit volume
	// rounds to eight.
	if got := echo / lead; math.Abs(got-8.0/15) > 0.03 {
		t.Errorf("echo is %.3f of the lead, want 8/15", got)
	}
}

func TestALoopCarriesItsTailIntoItsStart(t *testing.T) {
	// the only note is in the last bar and rings for a second past it.
	parts := `{"role": "lead", "bars": ["-:4", "c5:4"]}`
	vc := voicing(`{"kind": "pulse", "role": "lead", "duty": 0.5,
		"envelope": {"attack": 0, "decay": 0, "sustain": 1,
		"release": 1}}`, "")
	once := render(t, score(4, false, parts), vc)
	loop := render(t, score(4, true, parts), vc)
	if got := rms(seconds(once, 0.1, 0.5)); got > 0.001 {
		t.Errorf("a piece that does not loop is not silent at its start: "+
			"rms %.4f", got)
	}
	if got := rms(seconds(loop, 0.1, 0.5)); got < 0.05 {
		t.Errorf("a loop's start does not carry the release from its "+
			"end: rms %.4f", got)
	}
	if once.Duration() <= loop.Duration() {
		t.Errorf("the piece that does not loop (%v) should outlast the "+
			"loop (%v) by its release", once.Duration(), loop.Duration())
	}
}

func TestAMixThatClipsIsRefused(t *testing.T) {
	// narrow pulses in phase, and a wave that is nearly all low: the
	// output filter takes the offset away and the brief highs rise past
	// full scale.
	s, err := ParseScore(score(4, false,
		`{"role": "lead", "bars": ["c2:4"]},
		 {"role": "pad", "bars": ["c2:4"]},
		 {"role": "bass", "bars": ["c1:4"]}`))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVoicing(voicing(`
		{"kind": "pulse", "role": "lead", "duty": 0.125},
		{"kind": "pulse", "role": "pad", "duty": 0.125},
		{"kind": "wave", "role": "bass",
		 "table": "f0000000000000000000000000000000"}`, ""))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Render(s, v)
	wantErr(t, err, "clips")
}

func TestTheSameScoreThroughTwoVoicingsIsTheSameTune(t *testing.T) {
	s, err := ReadScore("@dusk")
	if err != nil {
		t.Fatal(err)
	}
	var leads [][]Tick
	for _, path := range []string{"@dmg-echo",
		"@dmg-pad"} {
		v, err := ReadVoicing(path)
		if err != nil {
			t.Fatal(err)
		}
		p, err := Render(s, v)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		// a loop is exactly its bars: eight of four at 92.
		want := 8 * 4 * 60 / 92.0
		if d := p.Duration().Seconds() - want; math.Abs(d) > 0.001 {
			t.Errorf("%s lasts %v s, want %v", path,
				p.Duration().Seconds(), want)
		}
		for i, l := range p.Labels {
			if l == "lead" {
				leads = append(leads, p.Ticks[i])
			}
		}
	}
	// the lead is voiced differently in each, and plays the same notes at
	// the same moments in both: which pitch sounds on every tick is the
	// score's, and nothing a voicing does can move it.
	a, b := leads[0], leads[1]
	if len(a) != len(b) {
		t.Fatalf("lead ticks %d and %d", len(a), len(b))
	}
	for i := range a {
		if (a[i].Vol > 0) != (b[i].Vol > 0) {
			continue // the envelopes differ, and so do the tails
		}
		if a[i].Vol > 0 && math.Round(a[i].Pitch) != math.Round(b[i].Pitch) {
			t.Fatalf("tick %d at %.3f s: %s in one voicing, %s in the "+
				"other", i, a[i].At, NoteName(int(math.Round(a[i].Pitch))),
				NoteName(int(math.Round(b[i].Pitch))))
		}
	}
}

func TestALoopsSeamIsNoWorseThanContinuousPlay(t *testing.T) {
	// the same score with every part's bars twice, rendered straight
	// through, is what the loop's seam would be if it were continuous.
	// the join between its two halves is the yardstick; the step from a
	// one-pass file's last sample to its first may not exceed it by more
	// than a hair. before the seam fade this was 0.086 against 0.0001.
	b, err := readDoc("@dusk")
	if err != nil {
		t.Fatal(err)
	}
	once, err := ParseScore(b)
	if err != nil {
		t.Fatal(err)
	}
	var doc Score
	if err := decodeStrict(b, &doc); err != nil {
		t.Fatal(err)
	}
	for i := range doc.Parts {
		bars := doc.Parts[i].Bars
		doc.Parts[i].Bars = append(append([]string{}, bars...), bars...)
	}
	doc.Loop = false
	j, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := ParseScore(j)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"@dmg-echo",
		"@dmg-pad"} {
		v, err := ReadVoicing(path)
		if err != nil {
			t.Fatal(err)
		}
		p1, err := Render(once, v)
		if err != nil {
			t.Fatal(err)
		}
		p2, err := Render(twice, v)
		if err != nil {
			t.Fatal(err)
		}
		n := len(p1.Left)
		for _, ch := range []struct {
			name      string
			one, both []float32
		}{{"left", p1.Left, p2.Left}, {"right", p1.Right, p2.Right}} {
			join := math.Abs(float64(ch.both[n] - ch.both[n-1]))
			file := math.Abs(float64(ch.one[0] - ch.one[n-1]))
			if file > join+0.002 {
				t.Errorf("%s %s: the loop seam steps %.5f; played "+
					"straight through the same place steps %.5f",
					path, ch.name, file, join)
			}
		}
	}
}

func TestOnlyALoopHasALoopPeriod(t *testing.T) {
	parts := `{"role": "lead", "bars": ["c4:4"]}`
	if d := render(t, score(4, false, parts), voicing(lead, "")).
		LoopPeriod(); d != 0 {
		t.Errorf("a piece that plays once has loop period %v", d)
	}
	p := render(t, score(4, true, parts), voicing(lead, ""))
	if d := p.LoopPeriod(); d != p.Duration() || d < 1999*time.Millisecond {
		t.Errorf("a two-second loop has period %v, duration %v", d,
			p.Duration())
	}
}
