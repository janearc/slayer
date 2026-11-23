package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
)

// The files the shell and fizz tests read: the shells and the weather they
// are given, the ways a shell is refused, the densities across a shell and
// the spans along rays.
const (
	shellWeatherShellsFile    = "testdata/shell-weather-shells.csv"
	shellWeatherDescribedFile = "testdata/shell-weather-described.csv"
	shellWeatherRefusalsFile  = "testdata/shell-weather-refusals.csv"
	shellWeatherParabolaFile  = "testdata/shell-weather-parabola.csv"
	shellWeatherSpansFile     = "testdata/shell-weather-spans.csv"
)

// shellWeatherShell is the size of a shell and the weather it is given.
type shellWeatherShell struct {
	radius, low, high float64
	weather           GroveWeather
}

// shellWeatherWind is the wind a row names in three columns.
func shellWeatherWind(t *testing.T, row map[string]string) maths.Vec {
	t.Helper()
	x := cell(t, row, "windx")
	y := cell(t, row, "windy")
	z := cell(t, row, "windz")
	return maths.V(x, y, z)
}

// shellWeatherShape is the shell of a row of the shells file, with the weather
// described by the row.
func shellWeatherShape(t *testing.T, name string) shellWeatherShell {
	t.Helper()
	row := readSheet(t, shellWeatherShellsFile).row(t, shellWeatherShellsFile,
		"shell-weather-"+name)
	seed := uint64(cellCount(t, row, "seed"))
	weather := NewWeather(cell(t, row, "extinction"), cell(t, row, "depth"),
		cell(t, row, "coverage"), seed, shellWeatherWind(t, row))
	weather.PatchScale = cell(t, row, "patch")
	return shellWeatherShell{radius: cell(t, row, "radius"),
		low: cell(t, row, "low"), high: cell(t, row, "high"),
		weather: weather}
}

// shellWeatherBuild is the shell of a size and weather. A shell that is
// refused stops the test.
func shellWeatherBuild(t *testing.T, shape shellWeatherShell) *ShellWeather {
	t.Helper()
	shell, err := NewShellWeather(shape.radius, shape.low, shape.high,
		shape.weather)
	if err != nil {
		t.Fatal(err)
	}
	return shell
}

// shellOf is a shell of radius 100 with its weather between heights 10
// and 30 above it, fully covered so density has no patches in it.
func shellOf(t *testing.T) *ShellWeather {
	t.Helper()
	return shellWeatherBuild(t, shellWeatherShape(t, "clear"))
}

// TestNewWeatherKeepsWhatItIsGiven checks every field of the description.
func TestNewWeatherKeepsWhatItIsGiven(t *testing.T) {
	row := readSheet(t, shellWeatherDescribedFile).row(t,
		shellWeatherDescribedFile, "shell-weather-described")
	extinction, depth := cell(t, row, "extinction"), cell(t, row, "depth")
	coverage := cell(t, row, "coverage")
	seed := uint64(cellCount(t, row, "seed"))
	wind := shellWeatherWind(t, row)
	weather := NewWeather(extinction, depth, coverage, seed, wind)
	if weather.Extinction != extinction || weather.Depth != depth ||
		weather.Coverage != coverage || weather.Seed != seed ||
		weather.Wind != wind {
		t.Errorf("weather = %+v", weather)
	}
}

// TestShellTakesItsDepthFromItsHeights checks the shell's depth is the
// gap between its heights, and that the parameters are kept.
func TestShellTakesItsDepthFromItsHeights(t *testing.T) {
	shape := shellWeatherShape(t, "clear")
	got := shellWeatherBuild(t, shape).Parameters()
	if got.Radius != shape.radius || got.Low != shape.low ||
		got.High != shape.high || got.Weather.Depth != shape.high-shape.low {
		t.Errorf("parameters = %+v", got)
	}
	if got.NoiseOctaves != mistTuning.octaves ||
		got.ThresholdShoulder != mistTuning.shoulder {
		t.Errorf("noise choices not carried: %+v", got)
	}
}

// shellWeatherRefused is whether a shell is refused: the clear weather with
// the patch size, extinction, coverage and asymmetry a row gives it, and the
// radius and heights the row gives.
func shellWeatherRefused(t *testing.T, row map[string]string) bool {
	t.Helper()
	shape := shellWeatherShape(t, "clear")
	shape.weather.PatchScale = cell(t, row, "patch")
	shape.weather.Extinction = cell(t, row, "extinction")
	shape.weather.Coverage = cell(t, row, "coverage")
	shape.weather.Asymmetry = cell(t, row, "asymmetry")
	_, err := NewShellWeather(cell(t, row, "radius"), cell(t, row, "low"),
		cell(t, row, "high"), shape.weather)
	return err != nil
}

// TestShellRefusesAnInvalidShell checks each way a shell can be wrong.
func TestShellRefusesAnInvalidShell(t *testing.T) {
	for _, row := range readSheet(t, shellWeatherRefusalsFile) {
		if !shellWeatherRefused(t, row) {
			t.Errorf("%s: shell accepted", row["name"])
		}
	}
}

// TestShellBoundsEncloseTheOuterSphere checks the cube is the outer
// radius, 130, on every side.
func TestShellBoundsEncloseTheOuterSphere(t *testing.T) {
	shape := shellWeatherShape(t, "clear")
	outer := shape.radius + shape.high
	box := shellOf(t).Bounds()
	if box.Min != maths.V(-outer, -outer, -outer) ||
		box.Max != maths.V(outer, outer, outer) {
		t.Errorf("bounds = %+v", box)
	}
}

// TestShellTickSetsTheClockOnACopy checks the ticked shell reads the time
// and the shell it was ticked from does not.
func TestShellTickSetsTheClockOnACopy(t *testing.T) {
	shell := shellOf(t)
	wait := time.Duration(pickedCount(t, "shell-weather-tick-milliseconds")) *
		time.Millisecond
	later, ok := shell.Tick(wait).(*ShellWeather)
	if !ok {
		t.Fatal("Tick did not give a shell")
	}
	if later.Parameters().Seconds != picked(t, "shell-weather-tick-seconds") {
		t.Errorf("seconds = %v", later.Parameters().Seconds)
	}
	if shell.Parameters().Seconds != nought {
		t.Errorf("Tick changed the original to %v", shell.Parameters().Seconds)
	}
}

// TestShellDensityIsAParabolaAcrossTheShell checks the density with the
// mist's extinction of 2: 2 at mid-shell, 1.5 a quarter in (4*h*(1-h) is
// 0.75), and none at or beyond either edge.
func TestShellDensityIsAParabolaAcrossTheShell(t *testing.T) {
	shell := shellOf(t)
	radius := shellWeatherShape(t, "clear").radius
	tolerance := picked(t, "shell-weather-tolerance-density")
	for _, each := range readSheet(t, shellWeatherParabolaFile) {
		height, want := cell(t, each, "height"), cell(t, each, "want")
		got := shell.Density(maths.V(nought, radius+height, nought))
		if math.Abs(got-want) > tolerance {
			t.Errorf("at height %v density is %v, want %v", height, got, want)
		}
	}
	middle := picked(t, "shell-weather-mid-shell")
	nowhere := maths.V(math.NaN(), middle, nought)
	if got := shell.Density(nowhere); got != nought {
		t.Errorf("a point that is not a number has density %v", got)
	}
}

// shellWeatherPatchesSeen is whether the patches of a shell moved between
// the clock at nought and the clock a tick on, along a line of points at the
// middle of the shell. A density outside nought to the extinction stops the
// test.
func shellWeatherPatchesSeen(t *testing.T, shell *ShellWeather,
	extinction float64) bool {
	t.Helper()
	moved := false
	middle := picked(t, "shell-weather-mid-shell")
	seconds := pickedCount(t, "shell-weather-patches-tick-seconds")
	wait := time.Duration(seconds) * time.Second
	start := picked(t, "shell-weather-patches-sweep-start")
	end := picked(t, "shell-weather-patches-sweep-end")
	step := picked(t, "shell-weather-patches-sweep-step")
	for x := start; x < end; x += step {
		at := maths.V(x, middle, nought)
		first, second := shell.Density(at), shell.Tick(wait).Density(at)
		if first < nought || first > extinction || second < nought ||
			second > extinction {
			t.Fatalf("density %v and %v are outside 0 to %v", first, second,
				extinction)
		}
		moved = moved || first != second
	}
	return moved
}

// TestShellDensityWithPatchesStaysUnderTheClearShell checks that with
// partial cover the density is never above the fully covered value and
// the patches change with the clock.
func TestShellDensityWithPatchesStaysUnderTheClearShell(t *testing.T) {
	shape := shellWeatherShape(t, "patchy")
	shell := shellWeatherBuild(t, shape)
	if !shellWeatherPatchesSeen(t, shell, shape.weather.Extinction) {
		t.Error("the patches did not move with the wind")
	}
}

// shellWeatherInterval is the interval of a row in two columns.
func shellWeatherInterval(t *testing.T, row map[string]string,
	enter, leave string) Interval {
	t.Helper()
	return Interval{Enter: cell(t, row, enter), Leave: cell(t, row, leave)}
}

// shellWeatherSpanRay is the ray of a row, from its place in y and z along
// the z axis.
func shellWeatherSpanRay(t *testing.T, row map[string]string) maths.Ray {
	t.Helper()
	y, z := cell(t, row, "y"), cell(t, row, "z")
	return maths.Ray{At: maths.V(nought, y, z), Dir: maths.ZAxis}
}

// TestShellSpansLeaveOutTheInteriorBall checks the intervals along rays
// worked by hand against a shell with outer radius 130 and inner 110.
func TestShellSpansLeaveOutTheInteriorBall(t *testing.T) {
	shell := shellOf(t)
	tolerance := picked(t, "shell-weather-tolerance-spans")
	far := picked(t, "shell-weather-spans-far")
	for _, each := range readSheet(t, shellWeatherSpansFile) {
		want := [shellLegs]Interval{
			shellWeatherInterval(t, each, "nearEnter", "nearLeave"),
			shellWeatherInterval(t, each, "farEnter", "farLeave"),
		}
		got := shell.Spans(shellWeatherSpanRay(t, each), nought, far)
		for leg := range got {
			if math.Abs(got[leg].Enter-want[leg].Enter) > tolerance ||
				math.Abs(got[leg].Leave-want[leg].Leave) > tolerance {
				t.Errorf("%s: spans = %v, want %v", each["name"], got, want)
				break
			}
		}
	}
}

// shellWeatherFizz is the whole fizz the fizz tests start from.
func shellWeatherFizz(t *testing.T) Fizz {
	t.Helper()
	return Fizz{Radius: picked(t, "shell-weather-fizz-radius"),
		Height: picked(t, "shell-weather-fizz-height"),
		Thick:  picked(t, "shell-weather-fizz-thick")}
}

// TestFizzBoundsAreTheColumnAtItsWidest checks the box is twice the
// radius at the foot's width on each side and the height up.
func TestFizzBoundsAreTheColumnAtItsWidest(t *testing.T) {
	fizz := shellWeatherFizz(t)
	reach := picked(t, "shell-weather-fizz-reach")
	box := fizz.Bounds()
	if box.Min != maths.V(-reach, nought, -reach) ||
		box.Max != maths.V(reach, fizz.Height, reach) {
		t.Errorf("bounds = %+v", box)
	}
}

// TestFizzWithNoSizeHasNoDensity checks a fizz with no radius, height or
// thickness is empty at its own middle.
func TestFizzWithNoSizeHasNoDensity(t *testing.T) {
	middle := maths.V(nought, picked(t, "shell-weather-fizz-middle"), nought)
	noRadius, noHeight, noThick := shellWeatherFizz(t), shellWeatherFizz(t),
		shellWeatherFizz(t)
	noRadius.Radius = nought
	noHeight.Height = nought
	noThick.Thick = nought
	for name, fizz := range map[string]Fizz{
		"no radius": noRadius,
		"no height": noHeight,
		"no thick":  noThick,
	} {
		if got := fizz.Density(middle); got != nought {
			t.Errorf("%s: density %v", name, got)
		}
	}
	if got := shellWeatherFizz(t).Density(middle); got <= nought {
		t.Errorf("a whole fizz is empty at its foot: %v", got)
	}
}

// shellWeatherNote sets whether a panic was met, and so stops it. It is the
// deferred call of shellWeatherPanics, which recover asks for.
func shellWeatherNote(did *bool) { *did = recover() != nil }

// shellWeatherPanics is whether running the function panics.
func shellWeatherPanics(run func()) (did bool) {
	defer shellWeatherNote(&did)
	run()
	return false
}

// shellWeatherRestore puts back the embedded file a test replaced.
func shellWeatherRestore(kept []byte) { mistWeatherRaw = kept }

// shellWeatherUnknown is a choice the file does not hold, asked for by name.
type shellWeatherUnknown struct{ read numbers.Numbers }

// ask is the unknown choice asked for.
func (unknown shellWeatherUnknown) ask() {
	mistChoice(unknown.read, "no-such-choice")
}

// shellWeatherReadAll reads the choices from the embedded file.
func shellWeatherReadAll() { readMistChoices() }

// TestMistChoicesFaultOnBadData checks that an unknown name and a
// broken embedded file are code faults, not quiet zeros.
func TestMistChoicesFaultOnBadData(t *testing.T) {
	read, err := numbers.Read("mist-weather.yaml", mistWeatherRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !shellWeatherPanics(shellWeatherUnknown{read}.ask) {
		t.Error("an unknown choice did not fault")
	}
	kept := mistWeatherRaw
	defer shellWeatherRestore(kept)
	mistWeatherRaw = []byte("numbers: [")
	if !shellWeatherPanics(shellWeatherReadAll) {
		t.Error("a broken file did not fault")
	}
}
