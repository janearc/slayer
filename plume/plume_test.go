package plume_test

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/plume"
	"github.com/janearc/slayer/research"
)

// megawatt is a worked fire: a megawatt, seven tenths of it carried
// by the air, a base 1.13 m across, in air at 293 k; soot at nist's
// measured yield for douglas fir, 0.0024 (tn 2314), burning at 21 mj a
// kg, and 8.7 m2 a g. its numbers below were worked by hand from the
// correlations, apart from the code.
var megawatt = plume.Fire{Heat: 1000, Convective: 0.7, Base: 1.13,
	Yield: 0.0024, Combustion: 21, Extinction: 8.7, Air: 293,
	Density: 1.204, Top: 40}

// mustColumn is a fire's plume, or the test fails.
func mustColumn(t *testing.T, fire plume.Fire) *plume.Column {
	t.Helper()
	column, err := plume.New(fire)
	if err != nil {
		t.Fatal(err)
	}
	return column
}

// at 5 m over a megawatt, mccaffrey's plume rises 1.12 times the heat
// to the third over the height to the third, 6.55 m a second; and
// heskestad's middle is about 143 k warmer than the air, as the
// research worked it by hand from tn 1984's equations.
func TestThePlumeRisesAsMeasured(t *testing.T) {
	column := mustColumn(t, megawatt)
	speed := research.Value("plume-speed") * math.Cbrt(1000.0/5)
	if got := column.Speed(5); !(math.Abs(got-speed) <= 1e-9*speed) ||
		!(math.Abs(got-6.55) <= 0.005) {
		t.Errorf("at 5 m the plume rises %g m/s, want %g", got, speed)
	}
	if got := column.Rise(5); !(math.Abs(got-143) <= 0.02*143) {
		t.Errorf("at 5 m the middle is %g k warmer, want about 143", got)
	}
}

// the soot is carried, not made or lost, as it rises: through any
// height a second passes the soot the fire makes in a second, the
// yield times what burns, at every height.
func TestTheSootIsCarriedUp(t *testing.T) {
	column := mustColumn(t, megawatt)
	made := 0.0024 * 1000 / 21 // g a second: yield, kg a second, grams
	for _, height := range []float64{5, 10, 25} {
		radius := column.Radius(height)
		step := radius / 200
		passed := 0.0
		for out := step / 2; out < 6*radius; out += step {
			soot := column.Mean(maths.V(out, height, 0)) / 8.7
			speed := column.Speed(height) * math.Exp2(-out*out/
				(radius*radius))
			passed += soot * speed * 2 * math.Pi * out * step
		}
		if !(math.Abs(passed-made) <= 0.01*made) {
			t.Errorf("at %g m %g g a second pass, the fire makes %g",
				height, passed, made)
		}
	}
}

// the smoke grows out of the flame's tongues: none below mccaffrey's
// intermittent region, where his tongues begin over the heat to the two
// fifths, and all of it where his plume region starts; on the way, no
// edge, which jumps the whole: no two heights a centimetre apart differ
// by more than two percent of the soot where the plume starts.
func TestTheSmokeGrowsOutOfTheTongues(t *testing.T) {
	column := mustColumn(t, megawatt)
	fifths := math.Pow(1000, 0.4)
	tongues := research.Value("tongues-from") * fifths
	start := research.Value("plume-from") * fifths
	full := column.Mean(maths.V(0, start, 0))
	if below := column.Mean(maths.V(0, tongues-0.01, 0)); below != 0 ||
		column.Bounds().Min.Y != tongues {
		t.Errorf("below the tongues %g, the box from %g, want %g", below,
			column.Bounds().Min.Y, tongues)
	}
	before := column.Mean(maths.V(0, tongues-0.01, 0))
	for height := tongues; height <= start+1; height += 0.01 {
		now := column.Mean(maths.V(0, height, 0))
		if !(math.Abs(now-before) <= 0.02*full) {
			t.Fatalf("at %.2f m the soot jumps from %g to %g", height,
				before, now)
		}
		before = now
	}
}

// a fire that cannot make a plume is refused.
func TestAPlumeRefusesWhatCannotBe(t *testing.T) {
	for _, change := range []func(fire *plume.Fire){
		func(fire *plume.Fire) { fire.Heat = 0 },
		func(fire *plume.Fire) { fire.Convective = 1.5 },
		func(fire *plume.Fire) { fire.Yield = math.NaN() },
		func(fire *plume.Fire) { fire.Top = 1 },
		func(fire *plume.Fire) { fire.Extinction = -1 },
		func(fire *plume.Fire) { fire.Base = 0 },
		func(fire *plume.Fire) { fire.Combustion = 0 },
		func(fire *plume.Fire) { fire.Air = 0 },
		func(fire *plume.Fire) { fire.Density = math.Inf(1) },
		func(fire *plume.Fire) { fire.Convective = 0 },
		func(fire *plume.Fire) { fire.Heat = math.Inf(1) },
	} {
		fire := megawatt
		change(&fire)
		if _, err := plume.New(fire); err == nil {
			t.Errorf("a plume was made of %+v", fire)
		}
	}
}

// the eddies roll the soot about its mean and rise with it: the same
// time is the same smoke, another time is other smoke, and over many
// times a slice carries the soot the fire makes.
func TestTheEddiesRollAboutTheMean(t *testing.T) {
	column := mustColumn(t, megawatt)
	at := maths.V(0.4, 12, -0.3)
	once := column.Tick(time.Second).Density(at)
	if again := column.Tick(time.Second).Density(at); again != once {
		t.Errorf("one time, %g then %g", once, again)
	}
	if later := column.Tick(3 * time.Second).Density(at); later == once {
		t.Errorf("a later time is the same smoke, %g", later)
	}
	made := 0.0024 * 1000 / 21
	const height, times = 12.0, 60
	radius := column.Radius(height)
	step := radius / 40
	passed := 0.0
	for moment := range times {
		ticked := column.Tick(time.Duration(moment) * 700 *
			time.Millisecond)
		for x := -4 * radius; x < 4*radius; x += step {
			for z := -4 * radius; z < 4*radius; z += step {
				soot := ticked.Density(maths.V(x, height, z)) / 8.7
				speed := column.Speed(height) *
					math.Exp2(-(x*x+z*z)/(radius*radius))
				passed += soot * speed * step * step
			}
		}
	}
	passed /= times
	t.Logf("over %d times %g g a second pass, the fire makes %g", times,
		passed, made)
	if !(math.Abs(passed-made) <= 0.02*made) {
		t.Errorf("over time %g g a second pass, the fire makes %g",
			passed, made)
	}
}

// heskestad's width over a megawatt, worked by hand from 0.12 times the
// root of how much warmer the middle is, times the height above the
// virtual origin: 0.50, 0.71 and 1.27 m at 3, 5 and 10 m, within a
// centimetre. the constant itself is recalled, not read.
func TestThePlumeWidensAsWorked(t *testing.T) {
	column := mustColumn(t, megawatt)
	for _, want := range [][2]float64{{3, 0.50}, {5, 0.71}, {10, 1.27}} {
		if got := column.Radius(want[0]); !(math.Abs(got-want[1]) <= 0.01) {
			t.Errorf("at %g m the plume is %g m to the half, worked %g",
				want[0], got, want[1])
		}
	}
}
