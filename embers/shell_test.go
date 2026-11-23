package embers

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// near is whether got is within tolerance of want; a NaN is never near.
func near(got, want, tolerance float64) bool {
	return math.Abs(got-want) <= tolerance
}

// aShell is a small shell with round numbers, for the tests that work
// its arithmetic by hand.
func aShell() Shell {
	return Shell{Lift: 100, Terminal: 40, Throw: 30, Gravity: 10,
		Fuse: 3 * time.Second, StarDrag: time.Second, Burn: 2 * time.Second,
		Stars: 64, Radius: measure.Metres(0.005), Hot: measure.Kelvin(2000),
		Cool: measure.Kelvin(1500), Seed: 7}
}

// With the air's drag all but gone, the shell rises as a stone thrown
// up in a vacuum: 100 t - 5 t^2, up to 500 m at ten seconds, and back
// down after.
func TestAShellWithoutAirRisesAsAThrownStone(t *testing.T) {
	shell := aShell()
	shell.Terminal = 1e7
	for _, at := range []float64{1, 3, 9.5, 10, 12} {
		height, speed := shell.rise(at)
		if !near(height, 100*at-5*at*at, 1e-4) || !near(speed, 100-10*at, 1e-4) {
			t.Errorf("at %v s the shell is %v m up at %v m/s, want %v and %v",
				at, height, speed, 100*at-5*at*at, 100-10*at)
		}
	}
}

// Worked by hand, and checked with awk, for a lift of 100 m/s, a
// terminal speed of 40 and gravity of 10: theta = atan(2.5) = 1.190290
// rad, the apex comes at 40 theta / 10 = 4.761160 s, at
// 160 ln(1/cos theta) = 80 ln 7.25 = 158.480118 m. One second after the
// apex the shell is falling at 40 tanh(0.25) = 9.796746 m/s, and
// 160 ln cosh(0.25) = 4.948769 m lower, at 153.531349 m.
func TestAShellRisesAndFallsAgainstTheAirAsWorkedByHand(t *testing.T) {
	shell := aShell()
	apex := 4 * math.Atan(2.5)
	if !near(apex, 4.761160, 1e-6) {
		t.Errorf("the apex comes at %v s", apex)
	}
	if height, speed := shell.rise(apex); !near(height, 158.480118, 1e-6) ||
		!near(speed, 0, 1e-9) {
		t.Errorf("at the apex the shell is %v m up at %v m/s", height, speed)
	}
	height, speed := shell.rise(apex + 1)
	if !near(height, 153.531349, 1e-6) || !near(speed, -9.796746, 1e-6) {
		t.Errorf("a second past the apex the shell is %v m up at %v m/s",
			height, speed)
	}
	// on either side of the apex, the two halves of rise agree
	before, _ := shell.rise(apex - 1e-9)
	after, _ := shell.rise(apex + 1e-9)
	if !near(before, after, 1e-6) {
		t.Errorf("the rise jumps at the apex: %v then %v", before, after)
	}
}

// ln cosh, both ways it is written. Small, against its series
// s^2/2 - s^4/12 + s^6/45, since the library's ln(cosh(0.001)) is
// itself wrong in its tenth digit, the loss logCosh is written to
// avoid; middling, against the library; and a thousand, past where
// cosh overflows, as s - ln 2. A shell falling a thousand seconds past
// its apex, at its terminal 40 m/s for nearly all of it, has fallen
// 160 (250 - ln 2) = 39,889.0965 m from 158.480118 m, to -39,730.6163 m
// (awk).
func TestLogCoshKeepsItsDigitsSmallAndLarge(t *testing.T) {
	small := 1e-3
	series := small*small/2 - math.Pow(small, 4)/12 + math.Pow(small, 6)/45
	if got := logCosh(small); !near(got, series, 1e-15*series) {
		t.Errorf("ln cosh %v is %v, want %v", small, got, series)
	}
	for _, s := range []float64{0.5, 1, 1 + 1e-9, 2, 20} {
		want := math.Log(math.Cosh(s))
		if got := logCosh(s); !near(got, want, 1e-12*want) {
			t.Errorf("ln cosh %v is %v, want %v", s, got, want)
		}
	}
	if got := logCosh(1000); !near(got, 1000-math.Ln2, 1e-9) {
		t.Errorf("ln cosh 1000 is %v", got)
	}
	shell := aShell()
	height, speed := shell.rise(4*math.Atan(2.5) + 1000)
	if !near(height, -39730.6164, 1e-3) || !near(speed, -40, 1e-9) {
		t.Errorf("a long fall ends %v m up at %v m/s", height, speed)
	}
}

// The air takes more of a fast shell's speed than a slow one's: a shell
// with drag never climbs as high as one without.
func TestTheAirAlwaysCostsAShellHeight(t *testing.T) {
	shell := aShell()
	free := aShell()
	free.Terminal = 1e7
	for _, at := range []float64{0.5, 1, 2, 3, 4} {
		drag, _ := shell.rise(at)
		vacuum, _ := free.rise(at)
		if !(drag < vacuum) {
			t.Errorf("at %v s the shell is %v m up against the air, %v without",
				at, drag, vacuum)
		}
	}
}

// Before its fuse burns out the mortar is one particle, the shell, on
// its rise; after, it is its stars; once they have burnt, nothing.
func TestAMortarIsItsShellThenItsStarsThenNothing(t *testing.T) {
	mortar, err := NewMortar(aShell())
	if err != nil {
		t.Fatal(err)
	}
	if got := mortar.At(-time.Second); got != nil {
		t.Errorf("before it is fired the mortar is %d particles", len(got))
	}
	climbing := mortar.At(time.Second)
	height, _ := aShell().rise(1)
	if len(climbing) != 1 || !near(climbing[0].Position().Y, height, 1e-9) {
		t.Errorf("a second after firing the mortar is %v", climbing)
	}
	if got := mortar.At(3500 * time.Millisecond); len(got) != 64 {
		t.Errorf("after the burst the mortar is %d particles, not its 64 stars",
			len(got))
	}
	if got := mortar.At(5 * time.Second); len(got) != 0 {
		t.Errorf("after the stars burn out the mortar is %d particles", len(got))
	}
	when, high := mortar.Burst()
	burst, _ := aShell().rise(3)
	if when != 3*time.Second || !near(high.InMetres(), burst, 1e-9) {
		t.Errorf("the shell bursts at %v, %v m up; want 3 s and %v m",
			when, high.InMetres(), burst)
	}
}

// The burst is a sphere: a moment after it, every star has left the
// shell's own path at nine tenths to all of the throw, and between them
// they go every way, so their mean way is near none.
func TestTheStarsLeaveTheBurstOnASphere(t *testing.T) {
	shell := aShell()
	shell.Stars = 4096
	mortar, err := NewMortar(shell)
	if err != nil {
		t.Fatal(err)
	}
	still := shell
	still.Throw = 0
	quiet, err := NewMortar(still)
	if err != nil {
		t.Fatal(err)
	}
	moment := time.Millisecond
	at := shell.Fuse + moment
	centre := quiet.At(at)[0].Position()
	var mean maths.Vec
	for _, star := range mortar.At(at) {
		away := star.Position().Sub(centre)
		speed := away.Len() / moment.Seconds()
		if !(speed >= 0.9*shell.Throw*0.998 && speed <= shell.Throw*1.001) {
			t.Fatalf("star %d left the burst at %v m/s, throw %v", star.ID(),
				speed, shell.Throw)
		}
		reach := 1 / away.Len()
		mean = mean.Add(away.Scale(reach))
	}
	if spread := mean.Len() / 4096; !(spread <= 0.05) {
		t.Errorf("the stars' mean way is %v of a whole: lopsided", spread)
	}
}

// The same seed throws the same stars; another seed, others.
func TestAShellsStarsAreTheSameEveryTime(t *testing.T) {
	one, _ := NewMortar(aShell())
	two, _ := NewMortar(aShell())
	other := aShell()
	other.Seed = 8
	three, _ := NewMortar(other)
	at := 3500 * time.Millisecond
	a, b, c := one.At(at), two.At(at), three.At(at)
	if a[5].Position() != b[5].Position() {
		t.Error("one seed threw two different stars")
	}
	if a[5].Position() == c[5].Position() {
		t.Error("two seeds threw the same star")
	}
}

// A shell that cannot be fired is refused, by each of its numbers.
func TestAMortarRefusesAShellThatCannotFly(t *testing.T) {
	broken := []func(*Shell){
		func(s *Shell) { s.Lift = 0 },
		func(s *Shell) { s.Lift = math.NaN() },
		func(s *Shell) { s.Terminal = math.Inf(1) },
		func(s *Shell) { s.Gravity = 0 },
		func(s *Shell) { s.Fuse = 0 },
		func(s *Shell) { s.StarDrag = -time.Second },
		func(s *Shell) { s.Burn = 0 },
		func(s *Shell) { s.Radius = measure.Metres(0) },
		func(s *Shell) { s.Throw = -1 },
		func(s *Shell) { s.Throw = math.NaN() },
		func(s *Shell) { s.Air = maths.V(math.Inf(1), 0, 0) },
		func(s *Shell) { s.Cool = measure.Kelvin(2500) },
		func(s *Shell) { s.Stars = 0 },
		func(s *Shell) { s.Stars = mostLiving + 1 },
	}
	for i, breaks := range broken {
		shell := aShell()
		breaks(&shell)
		if _, err := NewMortar(shell); err == nil {
			t.Errorf("broken shell %d was fired: %+v", i, shell)
		}
	}
	if _, err := NewMortar(aShell()); err != nil {
		t.Errorf("a sound shell was refused: %v", err)
	}
}

// A display shell bursts where Poulton and Kosanke say: 120 feet up for
// each inch, when its fuse burns out, three seconds at three inches and
// six at six; it leaves the mortar at 250 miles an hour; and its stars
// spread 90 feet across for each inch, before gravity and the shell's
// own motion move them.
func TestADisplayShellBurstsWhereTheArticleSays(t *testing.T) {
	for _, inches := range []float64{2, 3, 4.5, 6, 8, 12} {
		shell, err := DisplayShell(inches, 1)
		if err != nil {
			t.Fatalf("%v inches: %v", inches, err)
		}
		mortar, err := NewMortar(shell)
		if err != nil {
			t.Fatalf("%v inches: %v", inches, err)
		}
		when, high := mortar.Burst()
		fuse := math.Min(6, math.Max(3, inches))
		if !near(when.Seconds(), fuse, 1e-9) ||
			!near(high.InMetres(), 120*0.3048*inches, 1e-6*inches) {
			t.Errorf("%v inches bursts at %v, %v m up", inches, when,
				high.InMetres())
		}
		if !near(shell.Lift, 250*1609.344/3600, 1e-9) {
			t.Errorf("%v inches leaves the mortar at %v m/s", inches, shell.Lift)
		}
		reach := shell.Throw * shell.StarDrag.Seconds() *
			-math.Expm1(-shell.Burn.Seconds()/shell.StarDrag.Seconds())
		if !near(2*reach, 90*0.3048*inches, 1e-9*inches) {
			t.Errorf("%v inches spreads %v m across", inches, 2*reach)
		}
	}
	// a six-inch shell, the article's own example, at about 700 feet
	six, _ := DisplayShell(6, 1)
	if mortar, _ := NewMortar(six); true {
		_, high := mortar.Burst()
		if feet := high.InMetres() / 0.3048; !near(feet, 720, 1e-6) {
			t.Errorf("a six-inch shell bursts %v feet up", feet)
		}
	}
	for _, inches := range []float64{1.9, 12.1, 0, -3, math.NaN(), math.Inf(1)} {
		if _, err := DisplayShell(inches, 1); err == nil {
			t.Errorf("a %v-inch display shell was made", inches)
		}
	}
}
