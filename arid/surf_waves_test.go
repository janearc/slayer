package arid

import (
	"math"
	"testing"
)

func TestWaveNumberMeetsTheDispersionRelation(t *testing.T) {
	for _, omega := range []float64{0.3, 0.8, 1.2, 4} {
		for _, depth := range []float64{0.2, 1, 5, 40, 400} {
			number := waveNumber(omega, depth)
			got := gravity * number * math.Tanh(number*depth)
			if math.Abs(got-omega*omega) > 1e-9*omega*omega {
				t.Fatalf("omega %g, depth %g: g k tanh(k h) is %g, want %g",
					omega, depth, got, omega*omega)
			}
		}
	}
}

// deepWaterTolerance is how far, as a share of the answer, the deep-water
// wave number may be from omega squared over gravity worked in one piece:
// the test's constants are worked exactly and the function's are rounded,
// omega once and again in its square, then the square, gravity and the
// quotient, five roundings of at most 2^-53 (1.1e-16) each, so within
// 5.6e-16 of the answer; picked as 1e-15.
const deepWaterTolerance = 1e-15

func TestWaveNumberIsDeepAndShallowAtItsLimits(t *testing.T) {
	const omega = 1.1
	got, want := waveNumber(omega, 0), omega*omega/gravity
	if math.Abs(got-want) > deepWaterTolerance*want {
		t.Errorf("deep water: %g, want %g", got, want)
	}
	shallow := waveNumber(omega, 0.01)
	want = omega / math.Sqrt(gravity*0.01)
	if math.Abs(shallow/want-1) > 1e-3 {
		t.Errorf("shallow water: %g, want about %g", shallow, want)
	}
	if frequency(waveNumber(omega, 3), 3)-omega > 1e-12 {
		t.Error("frequency does not undo waveNumber")
	}
}

func TestGroupSpeedIsHalfInDeepAndWholeInShallow(t *testing.T) {
	const omega = 0.9
	got, want := groupSpeed(omega, 0), gravity/omega/2
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("deep: %g, want %g", got, want)
	}
	got, want = groupSpeed(omega, 0.02), math.Sqrt(gravity*0.02)
	if math.Abs(got/want-1) > 1e-3 {
		t.Errorf("shallow: %g, want about %g", got, want)
	}
}

func TestShoalingDipsBeforeItGrows(t *testing.T) {
	const omega = 1
	if got := shoaling(omega, 500, 0); math.Abs(got-1) > 1e-9 {
		t.Errorf("deep water shoals the waves by %g", got)
	}
	least := math.Inf(1)
	for depth := 0.5; depth < 50; depth += 0.05 {
		least = math.Min(least, shoaling(omega, depth, 0))
	}
	// linear theory's least shoaling coefficient is about 0.913
	if least > 0.92 || least < 0.90 {
		t.Errorf("the least shoaling is %g, want about 0.913", least)
	}
	if shoaling(omega, 0.1, 0) <= 1 {
		t.Error("the waves do not grow in shallow water")
	}
}

func TestBreakingFractionMeetsBattjesAndJanssen(t *testing.T) {
	for _, ratio := range []float64{0.3, 0.5, 0.7, 0.9, 0.99} {
		share := breakingFraction(ratio)
		if !(share > 0 && share < 1) {
			t.Fatalf("ratio %g: share %g is not between none and all",
				ratio, share)
		}
		if got := (1 - share) / math.Log(share); math.Abs(got+ratio*ratio) > 1e-9 {
			t.Fatalf("ratio %g: (1 - Q) / ln Q is %g, want %g", ratio, got,
				-ratio*ratio)
		}
	}
	if breakingFraction(1) != 1 || breakingFraction(3) != 1 {
		t.Error("waves at the limit are not all breaking")
	}
	if breakingFraction(0) != 0 || breakingFraction(math.NaN()) != 0 {
		t.Error("no waves, or no number, are breaking")
	}
}

func TestBreakingFractionGrowsWithTheRatio(t *testing.T) {
	last := 0.0
	for ratio := 0.05; ratio < 1; ratio += 0.05 {
		share := breakingFraction(ratio)
		if share < last {
			t.Fatalf("at %g the share %g fell from %g", ratio, share, last)
		}
		last = share
	}
}

func TestRunupIsStockdonsTwoInAHundred(t *testing.T) {
	const slope, height, length = 0.1, 1.5, 80
	mean, deviation := runup(slope, height, length)
	scale := math.Sqrt(height * length)
	want := 1.1 * (0.35*slope*scale +
		0.5*math.Sqrt(math.Pow(0.75*slope*scale, 2)+math.Pow(0.06*scale, 2)))
	if got := mean + 2*deviation; math.Abs(got-want) > 1e-12 {
		t.Errorf("R2 is %g, want %g", got, want)
	}
	flat, spread := runup(0.005, height, length)
	if flat != 0 || math.Abs(2*spread-0.043*scale) > 1e-12 {
		t.Errorf("a dissipative beach runs up %g and %g, want 0 and %g",
			flat, spread, 0.043*scale/2)
	}
}

func TestBreakerKindsFollowBattjes(t *testing.T) {
	const height, length = 1.0, 64.0
	steepness := math.Sqrt(height / length)
	for similar, want := range map[float64]breaker{0.3: spilling,
		1: plunging, 5: surging} {
		if got := breakerKind(similar*steepness, height, length); got != want {
			t.Errorf("surf similarity %g: %d, want %d", similar, got, want)
		}
	}
}

func TestWhitecapsFollowMonahan(t *testing.T) {
	got, want := whitecapCover(10), 3.84e-6*math.Pow(10, 3.41)
	if math.Abs(got-want) > 1e-15 {
		t.Errorf("at 10 m/s whitecaps cover %g, want %g", got, want)
	}
	if whitecapCover(0) != 0 || whitecapCover(-3) != 0 {
		t.Error("no wind makes whitecaps")
	}
	if whitecapCover(1000) != 1 {
		t.Error("whitecaps cover more than all the sea")
	}
}
