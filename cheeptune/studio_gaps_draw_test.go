//go:build draw

package cheeptune

import (
	"image"
	"math"
	"testing"
)

func TestAModeAboveTheCeilingIsLeftOutNotFoldedBack(t *testing.T) {
	// the bar's sixth mode on c8 would be at 78 kHz. it is left out:
	// nothing sounds under the note, where a folded mode would land.
	p := render(t, slow(`{"role": "lead", "bars": ["c8:4"]}`),
		studioVoicing(struck(`{"set": "bar", "hardness": 1,
			"decay": 2}`), ""))
	s := newSpectrum(seconds(p, 0.01, 0.5), p.Rate)
	own, _ := s.near(hz(108, 0), 20)
	if got := decibels(s.most(100, 3500) / own); got > -60 {
		t.Errorf("under c8 there is %.1f dB of something", got)
	}
	e := buildModal(&channel{Modal: &modal{Set: "bar", Decay: 2}})
	if got := len(e.(*modalEngine).bank(hz(108, 0))); got != 2 {
		t.Errorf("c8 on a bar rings %d modes, want the 2 under the "+
			"ceiling", got)
	}
}

func TestAPieceNeverRingsPastItsLongestTail(t *testing.T) {
	// an echo fifty beats late starts after the cap on a tail: it is
	// left out and the piece ends at the cap.
	late := `{"kind": "fm", "echo": "lead", "delay": "50", "gain": 0.5,
		` + goodFM + `}`
	p := render(t, slow(`{"role": "lead", "bars": ["a4:4"]}`),
		studioVoicing(fmPair("2")+","+late, ""))
	if got := p.Duration().Seconds(); math.Abs(got-(4+tailMost)) > 0.01 {
		t.Errorf("the piece is %v s, want %v", got, 4+tailMost)
	}
}

func TestTheSheetShowsAKitOfAnySize(t *testing.T) {
	// a kit of more drums than a panel has seconds shows the first of
	// them; one that cannot sound is refused with its channel.
	many := channel{Kind: "drum", Role: "perc", Drums: map[string]drum{}}
	for _, name := range drumLetters {
		many.Drums[name] = drum{Tone: 100, Decay: 0.2}
	}
	sound, err := alone(many)
	if err != nil || len(sound) != sheetSeconds*studioRate {
		t.Errorf("a kit of six: %d samples, %v", len(sound), err)
	}
	if rms(sound[3*studioRate:3*studioRate+2000]) == 0 {
		t.Errorf("the fourth second of a kit of six is silent")
	}
	bad := channel{Kind: "fm", Role: "lead", FM: &fmSound{
		Arrangement: "sum", Operators: []operator{{Ratio: 1, Level: 1},
			{Ratio: 1, Level: 1}, {Ratio: 1, Level: 1},
			{Ratio: 1, Level: 1}}}}
	loud := 1.0
	bad.Volume = &loud
	if _, err := alone(bad); err != nil {
		t.Errorf("four full carriers alone: %v", err)
	}
	if _, err := alone(channel{Kind: "fm", Role: "perc", FM: bad.FM}); err == nil {
		t.Errorf("an fm channel on perc drew a panel")
	}
}

func TestTheSheetPadsASmallKitAndRefusesWhatCannotSound(t *testing.T) {
	two := channel{Kind: "drum", Role: "perc", Drums: map[string]drum{
		"k": {Tone: 60, Decay: 0.2}, "s": {Tone: 200, Decay: 0.1}}}
	sound, err := alone(two)
	if err != nil {
		t.Fatal(err)
	}
	// two hits, then two seconds of nothing.
	if rms(sound[1*studioRate:1*studioRate+2000]) == 0 ||
		rms(sound[2*studioRate:]) != 0 {
		t.Errorf("a kit of two is not two hits and silence")
	}
	// a channel on a role that is not one draws nothing.
	two.Role = "vocal"
	if _, err := alone(two); err == nil {
		t.Errorf("a channel on a role that does not exist drew a panel")
	}
	// a voicing with an echo draws only the channels that play.
	img, err := InstrumentSheet([]string{"@night-pad"})
	if err != nil {
		t.Fatal(err)
	}
	if want := 2*sheetMargin + 4*(sheetPanelWide+sheetGap); img.
		Bounds().Dx() != want {
		t.Errorf("night-pad's sheet is %d wide, want %d",
			img.Bounds().Dx(), want)
	}
	// a silent instrument is an empty panel, not a division by nought.
	blank := image.NewRGBA(image.Rect(0, 0, 40, 30))
	drawPanel(blank, blank.Bounds(), panel{role: "lead",
		sound: make([]float32, sheetSeconds*studioRate)})
	if blank.RGBAAt(5, 5) != rollGrid {
		t.Errorf("a silent panel is not its ground")
	}
}
