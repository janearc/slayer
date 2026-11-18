package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
)

// shellOf is a shell of radius 100 with its weather between heights 10
// and 30 above it, fully covered so density has no patches in it.
func shellOf(t *testing.T) *ShellWeather {
	t.Helper()
	weather := NewWeather(2, 0, 1, 7, maths.Vec{})
	weather.PatchScale = 5
	shell, err := NewShellWeather(100, 10, 30, weather)
	if err != nil {
		t.Fatal(err)
	}
	return shell
}

// TestNewWeatherKeepsWhatItIsGiven checks every field of the description.
func TestNewWeatherKeepsWhatItIsGiven(t *testing.T) {
	wind := maths.V(1, 2, 3)
	weather := NewWeather(2, 3, 0.5, 9, wind)
	if weather.Extinction != 2 || weather.Depth != 3 || weather.Coverage != 0.5 ||
		weather.Seed != 9 || weather.Wind != wind {
		t.Errorf("weather = %+v", weather)
	}
}

// TestShellTakesItsDepthFromItsHeights checks the shell's depth is the
// gap between its heights, and that the parameters are kept.
func TestShellTakesItsDepthFromItsHeights(t *testing.T) {
	weather := NewWeather(2, 0, 1, 7, maths.Vec{})
	weather.PatchScale = 5
	shell, err := NewShellWeather(100, 10, 30, weather)
	if err != nil {
		t.Fatal(err)
	}
	got := shell.Parameters()
	if got.Radius != 100 || got.Low != 10 || got.High != 30 || got.Weather.Depth != 20 {
		t.Errorf("parameters = %+v", got)
	}
	if got.NoiseOctaves != mistTuning.octaves || got.ThresholdShoulder != mistTuning.shoulder {
		t.Errorf("noise choices not carried: %+v", got)
	}
}

// TestShellRefusesAnInvalidShell checks each way a shell can be wrong.
func TestShellRefusesAnInvalidShell(t *testing.T) {
	good := NewWeather(2, 0, 1, 7, maths.Vec{})
	good.PatchScale = 5
	bad := func(mutate func(*GroveWeather)) GroveWeather {
		weather := good
		mutate(&weather)
		return weather
	}
	nan, inf := math.NaN(), math.Inf(1)
	cases := map[string]struct {
		radius, low, high float64
		weather           GroveWeather
	}{
		"no radius":        {0, 10, 30, good},
		"nan radius":       {nan, 10, 30, good},
		"negative low":     {100, -1, 30, good},
		"nan low":          {100, nan, 30, good},
		"high at low":      {100, 10, 10, good},
		"infinite high":    {100, 10, inf, good},
		"no patch size":    {100, 10, 30, bad(func(w *GroveWeather) { w.PatchScale = 0 })},
		"negative mist":    {100, 10, 30, bad(func(w *GroveWeather) { w.Extinction = -1 })},
		"too much cover":   {100, 10, 30, bad(func(w *GroveWeather) { w.Coverage = 1.5 })},
		"asymmetry of one": {100, 10, 30, bad(func(w *GroveWeather) { w.Asymmetry = 1 })},
	}
	for name, each := range cases {
		if _, err := NewShellWeather(each.radius, each.low, each.high, each.weather); err == nil {
			t.Errorf("%s: shell accepted", name)
		}
	}
}

// TestShellBoundsEncloseTheOuterSphere checks the cube is the outer
// radius, 130, on every side.
func TestShellBoundsEncloseTheOuterSphere(t *testing.T) {
	box := shellOf(t).Bounds()
	if box.Min != maths.V(-130, -130, -130) || box.Max != maths.V(130, 130, 130) {
		t.Errorf("bounds = %+v", box)
	}
}

// TestShellTickSetsTheClockOnACopy checks the new shell reads the time
// and the old one does not.
func TestShellTickSetsTheClockOnACopy(t *testing.T) {
	shell := shellOf(t)
	later, ok := shell.Tick(1500 * time.Millisecond).(*ShellWeather)
	if !ok {
		t.Fatal("Tick did not give a shell")
	}
	if later.Parameters().Seconds != 1.5 {
		t.Errorf("seconds = %v", later.Parameters().Seconds)
	}
	if shell.Parameters().Seconds != 0 {
		t.Errorf("Tick changed the original to %v", shell.Parameters().Seconds)
	}
}

// TestShellDensityIsAParabolaAcrossTheShell checks the density with the
// mist's extinction of 2: 2 at mid-shell, 1.5 a quarter in (4*h*(1-h) is
// 0.75), and none at or beyond either edge.
func TestShellDensityIsAParabolaAcrossTheShell(t *testing.T) {
	shell := shellOf(t)
	for _, each := range []struct{ height, want float64 }{
		{20, 2}, {15, 1.5}, {25, 1.5}, {10, 0}, {30, 0}, {5, 0}, {40, 0}, {0, 0},
	} {
		got := shell.Density(maths.V(0, 100+each.height, 0))
		if math.Abs(got-each.want) > 1e-12 {
			t.Errorf("at height %v density is %v, want %v", each.height, got, each.want)
		}
	}
	if got := shell.Density(maths.V(math.NaN(), 120, 0)); got != 0 {
		t.Errorf("a point that is not a number has density %v", got)
	}
}

// TestShellDensityWithPatchesStaysUnderTheClearShell checks that with
// partial cover the density is never above the fully covered value and
// the patches change with the clock.
func TestShellDensityWithPatchesStaysUnderTheClearShell(t *testing.T) {
	weather := NewWeather(2, 0, 0.5, 7, maths.V(3, 0, 0))
	weather.PatchScale = 5
	shell, err := NewShellWeather(100, 10, 30, weather)
	if err != nil {
		t.Fatal(err)
	}
	moved := false
	for x := -40.0; x < 40; x += 2.5 {
		at := maths.V(x, 120, 0)
		first, second := shell.Density(at), shell.Tick(2*time.Second).Density(at)
		if first < 0 || first > 2 || second < 0 || second > 2 {
			t.Fatalf("density %v and %v are outside 0 to 2", first, second)
		}
		moved = moved || first != second
	}
	if !moved {
		t.Error("the patches did not move with the wind")
	}
}

// TestShellSpansLeaveOutTheInteriorBall checks the intervals along rays
// worked by hand against a shell with outer radius 130 and inner 110.
func TestShellSpansLeaveOutTheInteriorBall(t *testing.T) {
	shell := shellOf(t)
	along := func(y float64) maths.Ray {
		return maths.Ray{At: maths.V(0, y, -300), Dir: maths.V(0, 0, 1)}
	}
	for _, each := range []struct {
		name string
		ray  maths.Ray
		want [2]Interval
	}{
		{"through the middle", along(0), [2]Interval{{170, 190}, {410, 430}}},
		{"clipping the shell only", along(120), [2]Interval{{250, 350}, {}}},
		{"missing the shell", along(500), [2]Interval{}},
		{"from inside the hollow", maths.Ray{At: maths.V(0, 0, 0), Dir: maths.V(0, 0, 1)},
			[2]Interval{{0, 0}, {110, 130}}},
	} {
		got := shell.Spans(each.ray, 0, 1000)
		for i := range got {
			if math.Abs(got[i].Enter-each.want[i].Enter) > 1e-9 ||
				math.Abs(got[i].Leave-each.want[i].Leave) > 1e-9 {
				t.Errorf("%s: spans = %v, want %v", each.name, got, each.want)
				break
			}
		}
	}
}

// TestFizzBoundsAreTheColumnAtItsWidest checks the box is twice the
// radius at the foot's width on each side and the height up.
func TestFizzBoundsAreTheColumnAtItsWidest(t *testing.T) {
	box := Fizz{Radius: 0.2, Height: 1, Thick: 2}.Bounds()
	if box.Min != maths.V(-0.4, 0, -0.4) || box.Max != maths.V(0.4, 1, 0.4) {
		t.Errorf("bounds = %+v", box)
	}
}

// TestFizzWithNoSizeHasNoDensity checks a fizz with no radius, height or
// thickness is empty at its own middle.
func TestFizzWithNoSizeHasNoDensity(t *testing.T) {
	middle := maths.V(0, 0.1, 0)
	for name, fizz := range map[string]Fizz{
		"no radius": {Height: 1, Thick: 2},
		"no height": {Radius: 0.2, Thick: 2},
		"no thick":  {Radius: 0.2, Height: 1},
	} {
		if got := fizz.Density(middle); got != 0 {
			t.Errorf("%s: density %v", name, got)
		}
	}
	if got := (Fizz{Radius: 0.2, Height: 1, Thick: 2}).Density(middle); got <= 0 {
		t.Errorf("a whole fizz is empty at its foot: %v", got)
	}
}

// TestMistChoicesFaultOnBadData checks that an unknown name and a
// broken embedded file are code faults, not quiet zeros.
func TestMistChoicesFaultOnBadData(t *testing.T) {
	panics := func(run func()) (did bool) {
		defer func() { did = recover() != nil }()
		run()
		return false
	}
	read, err := numbers.Read("mist-weather.yaml", mistWeatherRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !panics(func() { mistChoice(read, "no-such-choice") }) {
		t.Error("an unknown choice did not fault")
	}
	kept := mistWeatherRaw
	defer func() { mistWeatherRaw = kept }()
	mistWeatherRaw = []byte("numbers: [")
	if !panics(func() { readMistChoices() }) {
		t.Error("a broken file did not fault")
	}
}
