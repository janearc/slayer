package cheeptune

import (
	"math"
	"testing"
)

func TestASpectrumFindsAToneWhereItIsAndIsHonestAtItsEdges(t *testing.T) {
	// a sine between two bins is found to well under a bin, which
	// here is five hertz.
	x := make([]float32, 8192)
	for i := range x {
		phase := 2 * math.Pi * 1000.3 * float64(i) /
			studioRate
		x[i] = float32(math.Sin(phase))
	}
	s := newSpectrum(x, studioRate)
	strength, at := s.near(1000, 20)
	if math.Abs(at-1000.3) > 0.15 || math.Abs(strength-1) > 0.15 {
		t.Errorf("a 1000.3 Hz sine reads %.2f Hz at %.2f", at, strength)
	}
	// at the first and last bins there are no two neighbours to fit,
	// and the bin's own frequency is the answer.
	if got := s.peak(0); got != 0 {
		t.Errorf("the first bin is at %v Hz", got)
	}
	last := len(s.strength) - 1
	if got := s.peak(last); got != s.hertz(last) {
		t.Errorf("the last bin is at %v Hz", got)
	}
	// silence has no peak to refine: three equal bins are the middle.
	quiet := newSpectrum(make([]float32, 64), studioRate)
	if got := quiet.peak(5); got != quiet.hertz(5) {
		t.Errorf("a flat spectrum peaks at %v Hz", got)
	}
}

func TestAGameBoyHitsVolumeKeepsItsRange(t *testing.T) {
	_, err := ParseVoicing(voicing(`{"kind": "noise", "role": "perc",
		"hits": {"k": {"shift": 1, "divisor": 1, "decay": 0.1,
		"volume": 2}}}`, ""))
	wantErr(t, err, "hit k volume 2 is outside 0 to 1")
}
