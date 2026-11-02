package manufacture

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

func mustWhorls(t *testing.T, count, lowest int, set habit) whorls {
	t.Helper()
	up, err := newWhorls(count, lowest, set)
	if err != nil {
		t.Fatal(err)
	}
	return up
}

// habitOf is the spruce's habit with the two numbers ninja's
// parameters carry, the crown's start and the reach, taken from them.
func habitOf(params ProperConiferParams) habit {
	set := spruceHabit()
	set.start, set.reach = params.CrownStart, params.BranchLength
	return set
}

// the truth for whorls is ninja's BranchOf and BranchCount: every
// branch of every whorl, its foot, its way, its length and its girth,
// for the default conifer, a tall thin one, a squat one and a sparse
// one, each from a seed of its own.
func TestWhorlsAreNinjasBranches(t *testing.T) {
	tall := ProperConiferParams{Height: 31, TrunkRadius: 0.6, Whorls: 48,
		PerWhorl: 9, CrownStart: 0.25, BranchLength: 0.18, Seed: 77}
	squat := ProperConiferParams{Height: 2.5, TrunkRadius: 0.2, Whorls: 5,
		PerWhorl: 12, CrownStart: 0.02, BranchLength: 0.6,
		Seed: 1 << 40}
	// so few in its lowest whorl that the higher ones would fall below
	// the fewest a whorl may have
	sparse := ProperConiferParams{Height: 8, TrunkRadius: 0.25,
		Whorls: 12, PerWhorl: 4, CrownStart: 0.1, BranchLength: 0.3,
		Seed: 9}
	all := []ProperConiferParams{DefaultProperConiferParams(), tall,
		squat, sparse}
	for _, params := range all {
		up := mustWhorls(t, params.Whorls, params.PerWhorl,
			habitOf(params))
		if up.rows() != params.Whorls {
			t.Fatalf("%d rows, %d whorls", up.rows(), params.Whorls)
		}
		for whorl := range params.Whorls {
			flaw := whorlUnlike(up, params, whorl)
			if flaw != "" {
				t.Fatalf("height %v, whorl %d: %s",
					params.Height, whorl, flaw)
			}
		}
	}
}

// whorlUnlike is how one whorl differs from ninja's, or nothing.
func whorlUnlike(up whorls, params ProperConiferParams,
	whorl int) string {
	if got, want := up.across(whorl), BranchCount(params,
		whorl); got != want {
		return fmt.Sprintf("%d branches, ninja's has %d", got, want)
	}
	stem := size{length: measure.Metres(params.Height),
		girth: measure.Metres(params.TrunkRadius)}
	for index := range up.across(whorl) {
		got, err := up.at(stem, whorl, index, sow(params.Seed))
		if err != nil {
			return err.Error()
		}
		flaw := branchUnlike(got, BranchOf(params, whorl, index))
		if flaw != "" {
			return fmt.Sprintf("branch %d: %s", index, flaw)
		}
	}
	return ""
}

// branchUnlike is how a place differs from ninja's branch, or nothing.
func branchUnlike(got place, want Branch) string {
	checks := []struct {
		what string
		same bool
	}{
		{"foot", got.space.Origin.Near(turned(want.Base), close)},
		{"way", got.space.Orientation.Y.Near(turned(want.Dir), close)},
		{"length", math.Abs(inScene(got.size.length)-want.Length) <
			close},
		{"girth", math.Abs(inScene(got.size.girth)-want.Radius) <
			close},
	}
	for _, check := range checks {
		if !check.same {
			return "its " + check.what + " is not ninja's"
		}
	}
	return ""
}

// the spruce turns each whorl from the last by the golden angle, which
// is worked out and not written: the turn that never lines two whorls
// up, a full turn over the square of the golden ratio.
func TestTheSpruceTurnsByTheGoldenAngle(t *testing.T) {
	golden := (1 + math.Sqrt(5)) / 2
	want := 2 * math.Pi / (golden * golden)
	got := spruceHabit().turn.InRadians()
	if math.Abs(got-want) > close {
		t.Errorf("the spruce turns %v a whorl, the golden angle is %v",
			got, want)
	}
}

func TestWhorlsThatCannotBeAreRefused(t *testing.T) {
	if _, err := newWhorls(0, 7, spruceHabit()); !errors.Is(err,
		errWhorls) {
		t.Errorf("no whorls: %v, not %v", err, errWhorls)
	}
	if _, err := newWhorls(4, 0, spruceHabit()); !errors.Is(err,
		errWhorls) {
		t.Errorf("none in a whorl: %v, not %v", err, errWhorls)
	}
	none := measure.Radians(math.NaN())
	endless := measure.Degrees(math.Inf(1))
	spoils := map[string]func(set *habit){
		"turn":       func(set *habit) { set.turn = none },
		"start":      func(set *habit) { set.start = -0.1 },
		"top":        func(set *habit) { set.top = set.start },
		"top ":       func(set *habit) { set.top = 1.01 },
		"thinning":   func(set *habit) { set.thinning = 1.2 },
		"fewest":     func(set *habit) { set.fewest = 0 },
		"stray":      func(set *habit) { set.stray = math.NaN() },
		"low":        func(set *habit) { set.low = none },
		"lift":       func(set *habit) { set.lift = endless },
		"sway":       func(set *habit) { set.sway = none },
		"reach":      func(set *habit) { set.reach = 0 },
		"shortening": func(set *habit) { set.shortening = -1 },
		"uneven":     func(set *habit) { set.uneven = 1 },
		"girth":      func(set *habit) { set.girth = 0 },
		"least":      func(set *habit) { set.least = -0.5 },
	}
	for name, spoil := range spoils {
		set := spruceHabit()
		spoil(&set)
		_, err := newWhorls(4, 7, set)
		if !errors.Is(err, errHabit) {
			t.Errorf("a habit with a bad %s: %v, not %v", name, err,
				errHabit)
			continue
		}
		if !strings.Contains(err.Error(), strings.TrimSpace(name)) {
			t.Errorf("a bad %s is refused without naming it: %v",
				name, err)
		}
	}
}

// mustRing is a ring whose parts are twisted any way round, as
// stones are.
func mustRing(t *testing.T, count int, radius, lean, stray,
	uneven float64) ring {
	t.Helper()
	round, err := newRing(count, measure.Metres(radius),
		measure.Degrees(lean), stray, uneven, 1)
	if err != nil {
		t.Fatal(err)
	}
	return round
}

// turnOf is how far round the up axis a point is, from x toward minus z, as a
// share of a turn from zero to one.
func turnOf(point maths.Vec) float64 {
	share := math.Atan2(-point.Z, point.X) / (2 * math.Pi)
	return share - math.Floor(share)
}

// a ring's truth is what the word means: so many places, each the
// ring's radius from the middle and on the ground, a full turn shared
// among them, each straying no more than its share of the gap.
func TestARingIsEvenRoundItsMiddle(t *testing.T) {
	const count, radius, stray = 9, 0.7, 0.4
	round := mustRing(t, count, radius, 0, stray, 0)
	if round.rows() != 1 || round.across(0) != count {
		t.Fatalf("%d rows of %d, not one of %d", round.rows(),
			round.across(0), count)
	}
	strayed := 0
	for index := range count {
		got, err := round.at(size{}, 0, index, sow(5))
		if err != nil {
			t.Fatal(err)
		}
		foot := got.space.Origin
		if foot.Y != 0 || math.Abs(foot.Len()-radius) > close {
			t.Errorf("place %d is at %v, not %v out on the ground",
				index, foot, radius)
		}
		off := turnOf(foot)*count - float64(index)
		off -= math.Round(off/count) * count
		if math.Abs(off) > stray {
			t.Errorf("place %d strays %v of the gap, past %v",
				index, off, stray)
		}
		if math.Abs(off) > close {
			strayed++
		}
	}
	if strayed == 0 {
		t.Error("no place strays, with a stray of 0.4")
	}
}

// with no stray and nothing uneven a ring is exact: evenly round, each
// part its own size.
func TestARingWithNoStrayIsExact(t *testing.T) {
	const count = 6
	round := mustRing(t, count, 2, 0, 0, 0)
	for index := range count {
		got, err := round.at(size{}, 0, index, sow(1))
		if err != nil {
			t.Fatal(err)
		}
		want := float64(index) / count
		if math.Abs(turnOf(got.space.Origin)-want) > close {
			t.Errorf("place %d is %v of a turn round, not %v",
				index, turnOf(got.space.Origin), want)
		}
		if scale := got.space.Scale; scale != 1 {
			t.Errorf("place %d is scaled %v", index, scale)
		}
	}
}

// a part of a ring leans in toward the middle by the ring's lean, or out by a
// lean less than zero, and is turned about its own length by the seed, so a
// ring of stones does not all face one way.
func TestARingsPartsLeanInOrOut(t *testing.T) {
	for _, lean := range []float64{0, 25, 90, -40} {
		round := mustRing(t, 5, 1.5, lean, 0.2, 0)
		faces := map[float64]bool{}
		for index := range 5 {
			got, err := round.at(size{}, 0, index, sow(8))
			if err != nil {
				t.Fatal(err)
			}
			if flaw := leanUnlike(got.space, lean); flaw != "" {
				t.Errorf("lean %v, place %d: %s", lean, index,
					flaw)
			}
			faces[math.Round(got.space.Orientation.X.X*1e6)] = true
		}
		if len(faces) < 4 {
			t.Errorf("lean %v: five parts face only %d ways", lean,
				len(faces))
		}
	}
}

// leanUnlike is how a part's space fails to lean as asked, or nothing.
func leanUnlike(space maths.Subspace, lean float64) string {
	axes, foot := space.Orientation, space.Origin
	tilt := math.Acos(axes.Y.Dot(maths.Up)) / maths.Degree
	if math.Abs(tilt-math.Abs(lean)) > 1e-9 {
		return fmt.Sprintf("it leans %v", tilt)
	}
	inward := -axes.Y.Dot(foot.Unit())
	if inward*lean < 0 || (lean != 0 && inward == 0) {
		return "it leans the wrong way"
	}
	if !axes.X.Cross(axes.Y).Near(axes.Z, close) {
		return "it is not right handed"
	}
	return ""
}

func TestARingsPartsAreUnevenByTheSeed(t *testing.T) {
	const uneven = 0.3
	round := mustRing(t, 40, 1, 0, 0, uneven)
	low, high := math.Inf(1), math.Inf(-1)
	for index := range 40 {
		got, _ := round.at(size{}, 0, index, sow(2))
		same, _ := round.at(size{}, 0, index, sow(2))
		other, _ := round.at(size{}, 0, index, sow(3))
		if got != same {
			t.Fatalf("place %d changed for the same seed", index)
		}
		if got == other {
			t.Fatalf("place %d is the same for another seed", index)
		}
		low = math.Min(low, got.space.Scale)
		high = math.Max(high, got.space.Scale)
	}
	if low < 1-uneven || high > 1+uneven {
		t.Errorf("sizes from %v to %v, past %v either way", low, high,
			uneven)
	}
	if high-low < uneven {
		t.Errorf("sizes from %v to %v: forty parts barely differ", low,
			high)
	}
}

func TestARingThatCannotBeIsRefused(t *testing.T) {
	nan := math.NaN()
	cases := map[string][6]float64{
		"no parts":            {0, 1, 0, 0, 0, 0},
		"a radius below zero": {5, -1, 0, 0, 0, 0},
		"an endless radius":   {5, math.Inf(1), 0, 0, 0, 0},
		"a lean of no number": {5, 1, nan, 0, 0, 0},
		"a stray of one":      {5, 1, 0, 1, 0, 0},
		"a stray below zero":  {5, 1, 0, -0.1, 0, 0},
		"uneven by one":       {5, 1, 0, 0, 1, 0},
		"uneven of no number": {5, 1, 0, 0, nan, 0},
		"a twist past a turn": {5, 1, 0, 0, 0, 1.1},
		"a twist below zero":  {5, 1, 0, 0, 0, -0.1},
	}
	for name, with := range cases {
		_, err := newRing(int(with[0]), measure.Metres(with[1]),
			measure.Degrees(with[2]), with[3], with[4], with[5])
		if !errors.Is(err, errRing) {
			t.Errorf("a ring with %s: %v, not %v", name, err,
				errRing)
		}
	}
	if _, err := newRing(1, measure.Metres(0), measure.Degrees(0), 0,
		0, 0); err != nil {
		t.Errorf("one part at the middle was refused: %v", err)
	}
}

// with no twist every part of a ring faces its middle the same way:
// its x runs round the ring, level, and so a frond lies face up.
func TestARingUntwistedFacesItsMiddle(t *testing.T) {
	round, err := newRing(6, measure.Metres(1), measure.Degrees(-35),
		0.3, 0.2, 0)
	if err != nil {
		t.Fatal(err)
	}
	for index := range 6 {
		got, _ := round.at(size{}, 0, index, sow(4))
		axes, out := got.space.Orientation, got.space.Origin.Unit()
		if axes.X.Y != 0 || math.Abs(axes.X.Dot(out)) > close {
			t.Errorf("place %d: its x is %v, not level and round "+
				"the ring", index, axes.X)
		}
		if !axes.X.Cross(axes.Y).Near(axes.Z, close) {
			t.Errorf("place %d is not right handed", index)
		}
		if axes.Y.Dot(out) <= 0 {
			t.Errorf("place %d does not lean out", index)
		}
	}
}

// which draw feeds what is a contract: a kernel that works out the
// same ring must take the stray from the seed's first number for a
// part, the spin from its second and the size from its third.
func TestARingTakesEachNumberFromItsOwnDraw(t *testing.T) {
	const count, stray, uneven = 7, 0.3, 0.2
	round, from := mustRing(t, count, 1, 0, stray, uneven), sow(21)
	for index := range count {
		got, err := round.at(size{}, 0, index, from)
		if err != nil {
			t.Fatal(err)
		}
		off := turnOf(got.space.Origin)*count - float64(index)
		off -= math.Round(off/count) * count
		if want := stray * from.swing(0, index, 0); math.Abs(off-
			want) > 1e-9 {
			t.Errorf("place %d strays %v, its draw says %v", index,
				off, want)
		}
		// untwisted, a part's x runs round the ring, a quarter turn on
		// from where the part stands
		spun := turnOf(got.space.Orientation.X) -
			turnOf(got.space.Origin) - 0.25
		spun -= math.Floor(spun)
		if want := from.draw(0, index, 1); math.Abs(spun-want) > 1e-9 {
			t.Errorf("place %d is spun %v, its draw says %v",
				index, spun, want)
		}
		scale := got.space.Scale
		if want := 1 + uneven*from.swing(0, index, 2); scale != want {
			t.Errorf("place %d is scaled %v, its draw says %v",
				index, scale, want)
		}
	}
}

// the edges of what is allowed are allowed: a habit may sit exactly on
// each bound, and one whorl of one part is whorls.
func TestTheEdgesOfAHabitAreAllowed(t *testing.T) {
	edges := map[string]func(set *habit){
		"a top of one":      func(set *habit) { set.top = 1 },
		"a start of nought": func(set *habit) { set.start = 0 },
		"no thinning":       func(set *habit) { set.thinning = 0 },
		"all thinned":       func(set *habit) { set.thinning = 1 },
		"a fewest of one":   func(set *habit) { set.fewest = 1 },
		"nothing uneven":    func(set *habit) { set.uneven = 0 },
		"a least of nought": func(set *habit) { set.least = 0 },
		"no stray":          func(set *habit) { set.stray = 0 },
		"a turn the other way": func(set *habit) {
			set.turn = measure.Degrees(-180)
		},
	}
	for name, edge := range edges {
		set := spruceHabit()
		edge(&set)
		if _, err := newWhorls(1, 1, set); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
	lost := spruceHabit()
	lost.least = math.Inf(1)
	if _, err := newWhorls(1, 1, lost); !errors.Is(err, errHabit) {
		t.Errorf("an endless least girth: %v, not %v", err, errHabit)
	}
}

func TestFiniteAndShareAtTheirEdges(t *testing.T) {
	for _, number := range []float64{math.Inf(1), math.Inf(-1),
		math.NaN()} {
		if finite(number) {
			t.Errorf("%v is called finite", number)
		}
		if share(number) {
			t.Errorf("%v is called a share", number)
		}
	}
	for _, number := range []float64{0, 1, 0.5} {
		if !finite(number) || !share(number) {
			t.Errorf("%v is not called a finite share", number)
		}
	}
	for _, number := range []float64{-1e-9, 1 + 1e-9} {
		if share(number) {
			t.Errorf("%v is called a share", number)
		}
	}
}

// a stem with no size would put every part at no number and say
// nothing; whorls refuse it. this was found by asking, not by
// reading: the first version returned the places and no error.
func TestWhorlsOnAStemOfNoSizeSayWhy(t *testing.T) {
	up := mustWhorls(t, 4, 5, spruceHabit())
	stems := map[string][2]float64{
		"a length of no number": {math.NaN(), 0.3},
		"an endless length":     {math.Inf(1), 0.3},
		"no length":             {0, 0.3},
		"no girth":              {10, 0},
		"a girth of no number":  {10, math.NaN()},
		"a girth below nought":  {10, -0.3},
	}
	for name, of := range stems {
		stem := size{length: measure.Metres(of[0]),
			girth: measure.Metres(of[1])}
		got, err := up.at(stem, 0, 0, sow(1))
		if !errors.Is(err, errStem) || got != (place{}) {
			t.Errorf("a stem with %s: %v, not %v", name, err,
				errStem)
		}
	}
}
