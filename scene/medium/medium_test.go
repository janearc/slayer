package medium

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The csv files the tests of the media read their shapes from.
const (
	mediumPuffsFile    = "testdata/medium-puffs.csv"
	mediumPuffOutsFile = "testdata/medium-puff-outs.csv"
	mediumWispiesFile  = "testdata/medium-wispies.csv"
	mediumGiantsFile   = "testdata/medium-giants.csv"
	mediumJetsFile     = "testdata/medium-jets.csv"
)

// mediumOrigin is the point at the middle of space.
func mediumOrigin() maths.Vec { return maths.V(nought, nought, nought) }

// mediumPuff is the puff called name in the puffs file.
func mediumPuff(t *testing.T, name string) Puff {
	t.Helper()
	row := readSheet(t, mediumPuffsFile).row(t, mediumPuffsFile, name)
	return Puff{Radius: cell(t, row, "radius"), Thick: cell(t, row, "thick")}
}

// mediumWispy is a wispy medium of what it is given, with the noise called
// name in the wispies file.
func mediumWispy(t *testing.T, of Medium, name string) Wispy {
	t.Helper()
	row := readSheet(t, mediumWispiesFile).row(t, mediumWispiesFile, name)
	return Wispy{Of: of, Seed: uint64(cellCount(t, row, "seed")),
		Scale: cell(t, row, "scale"), Amount: cell(t, row, "amount")}
}

// mediumGiant is the giant called name in the giants file.
func mediumGiant(t *testing.T, name string) Giant {
	t.Helper()
	row := readSheet(t, mediumGiantsFile).row(t, mediumGiantsFile, name)
	return Giant{Radius: cell(t, row, "radius"), Depth: cell(t, row, "depth"),
		Thick: cell(t, row, "thick")}
}

// mediumJet is the jet called name in the jets file.
func mediumJet(t *testing.T, name string) Jet {
	t.Helper()
	row := readSheet(t, mediumJetsFile).row(t, mediumJetsFile, name)
	return Jet{Length: cell(t, row, "length"), Base: cell(t, row, "base"),
		Spread: cell(t, row, "spread"), Thick: cell(t, row, "thick"),
		Knots: cell(t, row, "knots"), Seed: uint64(cellCount(t, row, "seed")),
		Flow: cell(t, row, "flow")}
}

// mediumPuffFalls holds that a puff's density falls from the middle to the
// edge in steps no larger than the fall the test allows.
func mediumPuffFalls(t *testing.T, puff Puff) {
	t.Helper()
	start := picked(t, "medium-puff-walk-start")
	step := picked(t, "medium-puff-walk-step")
	fall := picked(t, "medium-puff-walk-fall")
	last := puff.Thick
	for radius := start; radius < puff.Radius; radius += step {
		density := puff.Density(maths.V(nought, radius, nought))
		if density > last || last-density > fall {
			t.Fatalf("at %v the density went %v then %v", radius, last,
				density)
		}
		last = density
	}
}

// A puff is thickest in the middle, zero at and beyond its edge, and falls
// between without a step.
func TestPuff(t *testing.T) {
	puff := mediumPuff(t, "walk")
	if density := puff.Density(mediumOrigin()); density != puff.Thick {
		t.Errorf("the middle is %v", density)
	}
	for _, out := range readSheet(t, mediumPuffOutsFile) {
		radius := cell(t, out, "radius")
		density := puff.Density(maths.V(radius, nought, nought))
		if density != nought {
			t.Errorf("%v out is %v", radius, density)
		}
	}
	mediumPuffFalls(t, puff)
	start := maths.V(nought, nought, picked(t, "medium-puff-ray-start"))
	through := maths.Ray{At: start, Dir: maths.V(nought, nought, -unity)}
	if _, _, ok := puff.Bounds().Hit(through, nought,
		math.Inf(unity)); !ok {
		t.Error("a ray through the middle misses the bounds")
	}
}

// A wispy medium is never thicker than the one it wraps, is zero where that one
// is, and differs from place to place.
func TestWispy(t *testing.T) {
	puff := mediumPuff(t, "wisp")
	wispy := mediumWispy(t, puff, "wisp")
	seen := map[float64]bool{}
	stride := picked(t, "medium-wispy-stride")
	height := picked(t, "medium-wispy-height")
	depth := picked(t, "medium-wispy-depth")
	tolerance := picked(t, "medium-wispy-tolerance")
	resolution := picked(t, "medium-wispy-resolution")
	for place := range pickedCount(t, "medium-wispy-samples") {
		at := maths.V(float64(place)*stride, height, depth)
		density, full := wispy.Density(at), puff.Density(at)
		if density > full+tolerance || density < nought {
			t.Fatalf("at %v wispy is %v, the puff %v", at, density, full)
		}
		seen[math.Round(density/full*resolution)] = true
	}
	if len(seen) < pickedCount(t, "medium-wispy-varied-least") {
		t.Errorf("the wisps hardly vary: %d values", len(seen))
	}
	outside := maths.V(picked(t, "medium-wispy-outside"), nought, nought)
	if wispy.Density(outside) != nought {
		t.Error("wisps outside the puff")
	}
	if (Wispy{}).Density(mediumOrigin()) != nought {
		t.Error("a wispy nothing has density")
	}
}

// A giant is clear above its radius and thick a depth below it, with
// nothing in between but a smooth rise.
func TestGiant(t *testing.T) {
	giant := mediumGiant(t, "rise")
	above := maths.V(picked(t, "medium-giant-above"), nought, nought)
	if density := giant.Density(above); density != nought {
		t.Errorf("above the top, %v", density)
	}
	well := maths.V(nought, picked(t, "medium-giant-below"), nought)
	if density := giant.Density(well); density != giant.Thick {
		t.Errorf("well below, %v", density)
	}
	rise := maths.V(nought, nought, picked(t, "medium-giant-rise"))
	if density := giant.Density(rise); !(density > nought &&
		density < giant.Thick) {
		t.Errorf("halfway down the rise, %v", density)
	}
	if density := giant.Density(mediumOrigin()); density != giant.Thick {
		t.Errorf("at the centre, %v", density)
	}
}

// mediumSpanners are a puff, a giant and a wispy puff of one radius, the
// media whose spans are their ball's.
func mediumSpanners(t *testing.T) []Spanner {
	t.Helper()
	puff := mediumPuff(t, "span")
	return []Spanner{puff, mediumGiant(t, "span"),
		mediumWispy(t, puff, "span")}
}

// A puff and a giant span a ray by their ball, not their box: a ray
// through the middle enters at the radius, a ray past the edge misses
// though it crosses the box, and a wispy puff spans as its puff does.
func TestSpanIsTheBall(t *testing.T) {
	var _ Spanner = Puff{}
	var _ Spanner = Giant{}
	var _ Spanner = Wispy{}
	height := picked(t, "medium-span-height")
	down := maths.Ray{At: maths.V(nought, height, nought),
		Dir: maths.V(nought, -unity, nought)}
	tolerance := picked(t, "medium-span-tolerance")
	wantEnter := picked(t, "medium-span-enter")
	wantLeave := picked(t, "medium-span-leave")
	skew := picked(t, "medium-span-corner")
	corner := maths.Ray{At: maths.V(skew, height, skew),
		Dir: maths.V(nought, -unity, nought)}
	for _, spanner := range mediumSpanners(t) {
		enter, leave, ok := spanner.Span(down, nought, math.Inf(unity))
		leaveAt := wantLeave
		if giant, is := spanner.(Giant); is {
			// a giant's span ends under its ramp, where nothing
			// gets through
			leaveAt = wantEnter + giant.Depth
		}
		if !ok || !(math.Abs(enter-wantEnter) <= tolerance) ||
			!(math.Abs(leave-leaveAt) <= tolerance) {
			t.Errorf("%T spans %v to %v, %v", spanner, enter, leave, ok)
		}
		if _, _, ok := spanner.Span(corner, nought, math.Inf(unity)); ok {
			t.Errorf("%T spans a ray through its box's corner", spanner)
		}
	}
}

// A jet with a flow carries its knots along: the density a way up the
// jet after a time is the density that far back before, and a jet
// without a flow is the same at every time.
func TestAJetsKnotsFlow(t *testing.T) {
	jet := mediumJet(t, "flowing")
	seconds := time.Duration(picked(t, "medium-jet-seconds"))
	later := jet.Tick(seconds * time.Second).(Jet)
	station := picked(t, "medium-jet-station")
	at := maths.V(nought, station, nought)
	if jet.Density(at) == later.Density(at) {
		t.Errorf("the knots did not move")
	}
	// on the axis the density is the knot pattern times the fall off
	// along the jet; the pattern three base widths back before is the
	// pattern here now
	back := picked(t, "medium-jet-widths-back") * jet.Base
	earlier := maths.V(nought, station-back, nought)
	here, there := later.Density(at), jet.Density(earlier)
	stationBack := picked(t, "medium-jet-station-back")
	tolerance := picked(t, "medium-jet-tolerance")
	fadeHere := unity - station/jet.Length
	fadeThere := unity - stationBack/jet.Length
	if !(math.Abs(here/fadeHere-there/fadeThere) <= tolerance) {
		t.Errorf("after three seconds the knot at 20 is %v; "+
			"at 14 before it was %v", here, there)
	}
	still := mediumJet(t, "still")
	if still.Density(at) != still.Tick(time.Hour).(Jet).Density(at) {
		t.Errorf("a jet without a flow moved")
	}
}
