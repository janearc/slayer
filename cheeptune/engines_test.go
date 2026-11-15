package cheeptune

import (
	"math"
	"testing"
)

// The floors and tolerances the engines are held to. Each was set from
// what the engine measured on 2026-11-12, with a little room, by the
// writer of the studio; nobody chose them by ear.
const (
	// aliasFloor is the most a band-limited wave at c7 may leave under
	// its own fundamental, in decibels against the fundamental.
	aliasFloor = -50
	// naiveAlias is the least a wave with no correction leaves there.
	naiveAlias = -40
	// pluckCents is how far a plucked string may be from its note.
	pluckCents = 3
	// driveAliasFloor is the most the drive stage at full push may
	// leave under a saw at c6, and driveAliasGain how much better than
	// a plain tangent it must be, in decibels.
	driveAliasFloor = -45
	driveAliasGain  = 9
	// decayTolerance is how far a measured decay may be from the stated
	// one, as a share.
	decayTolerance = 0.15
)

// waveLength is how many samples of a wave the tests measure: a power of
// two, so a frequency can be put exactly on a bin of the spectrum.
const waveLength = 65536

// onBin is the frequency at the centre of a bin of a wave's spectrum.
func onBin(bin int) float64 {
	return float64(bin) * studioRate / waveLength
}

// wave is a stretch of one shape at a frequency, as samples.
func wave(sh shape, hertz float64) []float32 {
	out := make([]float32, waveLength)
	step := hertz / studioRate
	phase := 0.0
	for i := range out {
		out[i] = float32(sh(phase, step, 0.5))
		phase = wrapped(phase + step)
	}
	return out
}

// under is the strongest thing below a wave's fundamental, in decibels
// against the fundamental: all of it is overtones folded back.
func under(x []float32, hertz float64) float64 {
	s := newSpectrum(x, studioRate)
	own, _ := s.near(hertz, 10)
	return decibels(s.most(20, hertz*0.9) / own)
}

func TestAHighNoteLeavesAlmostNothingFoldedBackUnderIt(t *testing.T) {
	c7 := hz(96, 0)
	for name, sh := range shapes {
		if got := under(wave(sh, c7), c7); got > aliasFloor {
			t.Errorf("a %s at c7 has %.1f dB under it, over %v", name,
				got, aliasFloor)
		}
	}
	// the anti-test: the same saw and square with the correction left
	// out fold plenty back, so the measure can tell.
	naive := map[string]shape{
		"saw": func(p, _, _ float64) float64 { return 2*p - 1 },
		"square": func(p, _, w float64) float64 {
			if p < w {
				return 1
			}
			return -1
		},
	}
	for name, sh := range naive {
		if got := under(wave(sh, c7), c7); got < naiveAlias {
			t.Errorf("a naive %s at c7 has only %.1f dB under it", name,
				got)
		}
	}
}

func TestEachShapeIsTheWaveItSaysItIs(t *testing.T) {
	// at a low note the corrections are small and each wave is close to
	// its drawing: a saw's overtones fall as one over their number, a
	// square at a half has no even ones, a triangle's fall as the square.
	low := onBin(163) // 109.7 Hz
	strength := func(sh shape, multiple float64) float64 {
		s := newSpectrum(wave(sh, low), studioRate)
		got, _ := s.near(low*multiple, 5)
		own, _ := s.near(low, 5)
		return got / own
	}
	if got := strength(sawWave, 2); math.Abs(got-0.5) > 0.02 {
		t.Errorf("a saw's second overtone is %.3f of its first", got)
	}
	if got := strength(squareWave, 2); got > 0.01 {
		t.Errorf("a square at a half has a second overtone, %.3f", got)
	}
	if got := strength(squareWave, 3); math.Abs(got-1.0/3) > 0.02 {
		t.Errorf("a square's third overtone is %.3f of its first", got)
	}
	if got := strength(triangleWave, 3); math.Abs(got-1.0/9) > 0.01 {
		t.Errorf("a triangle's third overtone is %.3f of its first", got)
	}
}

func TestTheDriveStageFoldsBackLessThanAPlainTangent(t *testing.T) {
	// a saw at c6, pushed nine times over into each clipper. what is
	// under the fundamental is all folded back.
	c6 := hz(84, 0)
	saw := wave(sawWave, c6)
	clip := &softClip{curve: tangent}
	averaged := make([]float32, len(saw))
	plain := make([]float32, len(saw))
	for i, v := range saw {
		pushed := float64(v) * (1 + driveMost)
		averaged[i] = float32(clip.step(pushed))
		plain[i] = float32(math.Tanh(pushed))
	}
	got, was := under(averaged, c6), under(plain, c6)
	if got > driveAliasFloor {
		t.Errorf("the drive stage leaves %.1f dB under c6, over %v", got,
			driveAliasFloor)
	}
	// the anti-test: the plain tangent is worse by a clear margin.
	if was-got < driveAliasGain {
		t.Errorf("the plain tangent leaves %.1f dB and the averaged "+
			"one %.1f; no better", was, got)
	}
	// and a quiet signal comes through as itself: the stage is a
	// tangent, and a small tangent is its input.
	clip = &softClip{curve: tangent}
	for i := range 100 {
		x := 0.01 * math.Sin(float64(i)/7)
		y := clip.step(x)
		if i > 0 && math.Abs(y-x) > 0.002 {
			t.Fatalf("a quiet %v came out as %v", x, y)
		}
	}
}
