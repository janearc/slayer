//go:build draw

package cheeptune

import (
	"math"
	"strings"
	"testing"
)

func TestAStudioMixThatClipsIsRefusedWithItsPeakAndMaster(t *testing.T) {
	loud := []byte(`{"name": "v", "machine": "studio", "master": 3,
		"channels": [` + fmPair("2") + `]}`)
	s, _ := ParseScore(slow(`{"role": "lead", "bars": ["a4:4"]}`))
	v, err := ParseVoicing(loud)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Render(s, v)
	wantErr(t, err, "and clips; lower the voicing's master, which is 3")
	// the near miss: with no master the same note is under full scale.
	quiet := []byte(`{"name": "v", "machine": "studio",
		"channels": [` + fmPair("2") + `]}`)
	v, _ = ParseVoicing(quiet)
	p, err := Render(s, v)
	if err != nil || p.Peak > masterDefault*1.05 {
		t.Errorf("at the default master: peak %v, %v", p, err)
	}
}

func TestATakenVoiceIsHeardToStop(t *testing.T) {
	// a ringing bowl with one voice: the second note takes the first.
	one := studioChannelOf("modal", "lead", `"voices": 1, "modal":
		{"set": "bowl", "hardness": 0.5, "decay": 10}`)
	many := studioChannelOf("modal", "lead", `"modal":
		{"set": "bowl", "hardness": 0.5, "decay": 10}`)
	sc := slow(`{"role": "lead", "bars": ["a3:1 a5:1 -:2"]}`)
	low := func(channel string) float64 {
		p := render(t, sc, studioVoicing(channel, ""))
		s := newSpectrum(seconds(p, 2, 3), p.Rate)
		strength, _ := s.near(220, 10)
		return strength
	}
	if taken, kept := low(one), low(many); taken > kept/100 {
		t.Errorf("the taken voice still rings: %v against %v", taken,
			kept)
	}
}

func TestAStudioLoopCarriesItsTailIntoItsStart(t *testing.T) {
	room := `, "reverb": {"size": 0.8, "decay": 3, "damping": 0,
		"level": 1}`
	bars := `{"role": "perc", "bars": ["-:3 k", "-:4"]}`
	once := render(t, slow(bars), studioVoicing(click(`"reverb": 1`),
		room))
	loop := render(t, []byte(strings.Replace(string(slow(bars)),
		`"loop": false`, `"loop": true`, 1)),
		studioVoicing(click(`"reverb": 1`), room))
	// the loop is exactly its bars long, and its first second holds the
	// room still ringing from the hit at the end of bar one... which
	// has died by then; the tail that matters is past bar two.
	if got := loop.Duration().Seconds(); math.Abs(got-8) > 0.001 {
		t.Errorf("the loop is %v s, want 8", got)
	}
	// what played once rings past eight seconds; folded, that ring is
	// at the loop's start, where played once there is silence.
	if rms(seconds(once, 0, 0.5)) != 0 {
		t.Errorf("played once, the start is not silent")
	}
	tail := rms(once.Left[8*once.Rate:])
	if start := rms(seconds(loop, 0, 0.5)); start == 0 || tail == 0 {
		t.Errorf("the loop's start holds %v of a tail of %v", start, tail)
	}
	// and a loop's marks stay inside it.
	for _, tick := range loop.Ticks[0] {
		if tick.At < 0 || tick.At >= 8 {
			t.Fatalf("a mark at %v s in an 8 s loop", tick.At)
		}
	}
}

func TestAnArpInTicksAndADefaultArpBreakTheChordToo(t *testing.T) {
	sc := slow(`{"role": "pad", "bars": ["(c4 e4 g4):4"]}`)
	for _, arp := range []string{`"arp": 6`, `"arp": "1/4"`} {
		ch := `{"kind": "pluck", "role": "pad", ` + arp + `, ` +
			goodPluck + `}`
		p := render(t, sc, studioVoicing(ch, ""))
		pitches := map[float64]bool{}
		starts := 0
		last := -1.0
		for _, tick := range p.Ticks[0] {
			pitches[tick.Pitch] = true
			if tick.At-last > 0.05 {
				starts++
			}
			last = tick.At
		}
		if len(pitches) != 3 {
			t.Errorf("%s: the arp played %d pitches", arp, len(pitches))
		}
	}
}

func TestTheInstrumentSheetDrawsEveryInstrumentOfAStudioVoicing(
	t *testing.T) {
	img, err := InstrumentSheet([]string{"@glass-bells", "@iron-works"})
	if err != nil {
		t.Fatal(err)
	}
	// iron-works has five playing channels, so the sheet is five wide
	// and two rows tall.
	wantWide := 2*sheetMargin + 5*(sheetPanelWide+sheetGap)
	wantTall := sheetTitle + sheetMargin +
		2*(sheetPanelTall+sheetLabel+sheetGap)
	if b := img.Bounds(); b.Dx() != wantWide || b.Dy() != wantTall {
		t.Errorf("the sheet is %v, want %d by %d", b, wantWide, wantTall)
	}
	// the first panel has something drawn in it brighter than its
	// ground, and the empty fifth place of the first row has nothing.
	bright := func(column, row int) int {
		x := sheetMargin + column*(sheetPanelWide+sheetGap)
		y := sheetTitle + row*(sheetPanelTall+sheetLabel+sheetGap) +
			sheetLabel
		count := 0
		for dy := range sheetPanelTall {
			for dx := range sheetPanelWide {
				if img.RGBAAt(x+dx, y+dy).R > 0x80 {
					count++
				}
			}
		}
		return count
	}
	if bright(0, 0) == 0 || bright(4, 0) != 0 {
		t.Errorf("panels: first %d, empty %d", bright(0, 0), bright(4, 0))
	}
	_, err = InstrumentSheet([]string{"@dmg-pad"})
	wantErr(t, err, "@dmg-pad is on the gameboy machine")
	_, err = InstrumentSheet(nil)
	wantErr(t, err, "no studio voicings to draw")
	_, err = InstrumentSheet([]string{"@nope"})
	wantErr(t, err, `no built-in "@nope"`)
	_, err = InstrumentSheet([]string{"@crypt"})
	wantErr(t, err, "@crypt:")
}
