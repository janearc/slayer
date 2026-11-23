package thing

import (
	"math"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
)

// The files the orbit tests read their tables from.
const (
	orbitPointsFile = "testdata/orbit_points.csv"
	orbitRoundFile  = "testdata/orbit_round.csv"
	orbitTippedFile = "testdata/orbit_tipped.csv"
)

// What the orbit tests work with, each true by definition.
const (
	orbitQuarterTurn = math.Pi / pair // defined: a right angle is half of pi
	orbitDayHours    = 24             // defined: hours in a day
)

// orbitRowPoint is the point a row names in its x, y and z columns.
func orbitRowPoint(t *testing.T, row map[string]string) maths.Vec {
	t.Helper()
	return maths.V(cell(t, row, "x"), cell(t, row, "y"), cell(t, row, "z"))
}

// orbitPoint is the point called name in orbitPointsFile.
func orbitPoint(t *testing.T, name string) maths.Vec {
	t.Helper()
	rows := readSheet(t, orbitPointsFile)
	return orbitRowPoint(t, rows.row(t, orbitPointsFile, name))
}

// orbitSeconds is the picked number of seconds called name, as a Duration.
func orbitSeconds(t *testing.T, name string) time.Duration {
	t.Helper()
	return time.Duration(picked(t, name)) * time.Second
}

// orbitMillis is the time a row names in its millis column.
func orbitMillis(t *testing.T, row map[string]string) time.Duration {
	t.Helper()
	return time.Duration(cellCount(t, row, "millis")) * time.Millisecond
}

// orbitTolerance is how far a worked point or length may be out.
func orbitTolerance(t *testing.T) float64 {
	t.Helper()
	return picked(t, "orbit-tolerance")
}

// orbitTight is how far out a point may be that was never moved.
func orbitTight(t *testing.T) float64 {
	t.Helper()
	return picked(t, "orbit-tight-tolerance")
}

// orbitMoon is the moon the locked, nested and growing tests go round with.
func orbitMoon(t *testing.T, locked bool) Orbit {
	t.Helper()
	return Orbit{Radius: picked(t, "orbit-moon-radius"),
		Period: orbitSeconds(t, "orbit-moon-period-seconds"), Locked: locked}
}

// orbitFrameAt is the unchanged frame moved to origin.
func orbitFrameAt(origin maths.Vec) maths.Subspace {
	frame := maths.Local()
	frame.Origin = origin
	return frame
}

// orbitRoundBeam is a round beam at the speed picked as speedName.
func orbitRoundBeam(t *testing.T, speedName string) Beam {
	t.Helper()
	return Beam{Speed: picked(t, speedName), Round: true,
		At: picked(t, "orbit-beam-at")}
}

// orbitDays is the days a moon of the radius takes round a body of the mass.
func orbitDays(mass, radius float64) float64 {
	return Period(mass, radius).Hours() / orbitDayHours
}

// orbitKeplerIo fails the test when Io does not take about a day and three
// quarters to go round its planet.
func orbitKeplerIo(t *testing.T) {
	t.Helper()
	jupiter, radius := picked(t, "orbit-jupiter-mass"),
		picked(t, "orbit-io-radius")
	days := orbitDays(jupiter, radius)
	want := picked(t, "orbit-io-days")
	if !nearly(days, want, picked(t, "orbit-io-days-tolerance")) {
		t.Errorf("io goes round in %.3f days", days)
	}
}

// orbitKeplerYear fails the test when the earth does not take about a year
// to go round its star.
func orbitKeplerYear(t *testing.T) {
	t.Helper()
	sun, radius := picked(t, "orbit-sun-mass"), picked(t, "orbit-earth-radius")
	days := orbitDays(sun, radius)
	want := picked(t, "orbit-year-days")
	if !nearly(days, want, picked(t, "orbit-year-days-tolerance")) {
		t.Errorf("the earth goes round in %.1f days", days)
	}
}

// orbitKeplerRatio fails the test when four times the radius does not take
// eight times as long.
func orbitKeplerRatio(t *testing.T) {
	t.Helper()
	jupiter := picked(t, "orbit-jupiter-mass")
	inner := Period(jupiter, picked(t, "orbit-inner-radius"))
	outer := Period(jupiter, picked(t, "orbit-outer-radius"))
	ratio := float64(outer) / float64(inner)
	want := picked(t, "orbit-radius-ratio")
	if !nearly(ratio, want, picked(t, "orbit-radius-ratio-tolerance")) {
		t.Errorf("four times the radius takes %v times as long", ratio)
	}
}

// Kepler: a moon's period comes from its parent's mass and its radius.
// Io round Jupiter takes about a day and three quarters; the earth round
// the sun about a year; a bigger radius takes longer by the three halves
// power.
func TestPeriodIsKeplers(t *testing.T) {
	orbitKeplerIo(t)
	orbitKeplerYear(t)
	orbitKeplerRatio(t)
	if Period(nought, unity) != nought || Period(unity, nought) != nought {
		t.Error("no mass or no radius has a period")
	}
}

// orbitRoundOrbit is the orbit the going-round test places a thing on.
func orbitRoundOrbit(t *testing.T) Orbit {
	t.Helper()
	return Orbit{Centre: orbitPoint(t, "centre"),
		Radius: picked(t, "orbit-round-radius"),
		Period: orbitSeconds(t, "orbit-round-period-seconds"),
		Phase:  nought, Scale: picked(t, "orbit-round-scale")}
}

// orbitCheckRound fails the test when the orbit is not at want at the time,
// at its radius from its centre and at its scale.
func orbitCheckRound(t *testing.T, round Orbit, at time.Duration,
	want maths.Vec) {
	t.Helper()
	state := round.At(at)
	tolerance := orbitTolerance(t)
	if !state.Origin.Near(want, tolerance) || state.Scale != round.Scale {
		t.Errorf("at %v: %v, want %v", at, state.Origin, want)
	}
	distance := state.Origin.Sub(round.Centre).Len()
	if !nearly(distance, round.Radius, tolerance) {
		t.Errorf("at %v the radius is %v", at, distance)
	}
}

// orbitCheckTipped fails the test when an orbit tipped about x leaves the
// plane square to x.
func orbitCheckTipped(t *testing.T) {
	t.Helper()
	tipped := Orbit{Radius: unity, Axis: maths.XAxis, Period: time.Second}
	tolerance := orbitTolerance(t)
	for _, each := range readSheet(t, orbitTippedFile) {
		at := orbitMillis(t, each)
		if x := tipped.At(at).Origin.X; !(math.Abs(x) <= tolerance) {
			t.Errorf("tipped about x, at %v it left the plane: x %v", at, x)
		}
	}
}

// An orbit goes round its centre at its radius, once a period, in the
// plane square to its axis, and keeps the thing's own facing and scale.
func TestOrbitGoesRound(t *testing.T) {
	round := orbitRoundOrbit(t)
	for _, each := range readSheet(t, orbitRoundFile) {
		at, want := orbitMillis(t, each), orbitRowPoint(t, each)
		orbitCheckRound(t, round, at, want)
	}
	orbitCheckTipped(t)
	still := Orbit{Radius: picked(t, "orbit-still-radius"),
		Phase: orbitQuarterTurn}
	first, second := still.At(nought), still.At(time.Hour)
	if first != second {
		t.Error("an orbit with no period moved")
	}
}

// A locked orbit keeps one face to the centre all the way round; an
// unlocked one keeps its facing as it was.
func TestALockedOrbitKeepsOneFaceToTheCentre(t *testing.T) {
	moon := orbitMoon(t, true)
	away := maths.XAxis.Neg()
	tolerance := orbitTolerance(t)
	for k := range pickedCount(t, "orbit-moon-quarters") {
		state := moon.At(time.Duration(k) * time.Second)
		face := state.Orientation.Out(away)
		toCentre := state.Origin.Neg().Unit()
		if face.Sub(toCentre).Len() > tolerance {
			t.Fatalf("at %d s the face points %v, the centre is %v",
				k, face, toCentre)
		}
	}
	unlocked := orbitMoon(t, false)
	face := unlocked.At(time.Second).Orientation.Out(away)
	if face.Sub(away).Len() > orbitTight(t) {
		t.Fatalf("an unlocked orbit turned its thing: %v", face)
	}
}

// A nested subspace goes round with its parent: a point on the far side
// of a locked moon stays on the far side.
func TestNestedGoesRoundWithItsParent(t *testing.T) {
	moon := orbitMoon(t, true)
	inside := orbitFrameAt(orbitPoint(t, "nested-start"))
	nested := Nested{Of: moon, In: inside}
	start := orbitPoint(t, "nested-at-nought")
	quarter := orbitPoint(t, "nested-a-quarter")
	tolerance := orbitTolerance(t)
	if at := nested.At(nought).Origin; at.Sub(start).Len() > tolerance {
		t.Fatalf("at nought the far side is at %v", at)
	}
	at := nested.At(time.Second).Origin
	if at.Sub(quarter).Len() > tolerance {
		t.Fatalf("a quarter round the far side is at %v", at)
	}
}

// A spin turns in place: a quarter of a period is a quarter turn about
// the axis, and the origin stays put.
func TestSpinTurnsInPlace(t *testing.T) {
	centre := orbitPoint(t, "centre")
	spinning := Spin{In: orbitFrameAt(centre),
		Period: orbitSeconds(t, "orbit-spin-period-seconds")}
	at := spinning.At(time.Second)
	if at.Origin != centre {
		t.Fatalf("the origin moved to %v", at.Origin)
	}
	x := at.Orientation.Out(maths.XAxis)
	if x.Sub(orbitPoint(t, "minus-z")).Len() > orbitTolerance(t) {
		t.Fatalf("a quarter turn about up took x to %v", x)
	}
	x = spinning.At(nought).Orientation.Out(maths.XAxis)
	if x.Sub(maths.XAxis).Len() > orbitTight(t) {
		t.Fatalf("at nought x is %v", x)
	}
}

// A growing subspace is as good as nothing before it is born, then grows
// at its rate, carried by its motion.
func TestGrowingGrowsFromItsBirth(t *testing.T) {
	moon := orbitMoon(t, false)
	growing := Growing{Of: moon, From: time.Second,
		Rate: picked(t, "orbit-growing-rate")}
	birth := picked(t, "orbit-birth-tolerance")
	if state := growing.At(nought); state.Scale > birth {
		t.Fatalf("before its birth the scale is %v", state.Scale)
	}
	later := orbitSeconds(t, "orbit-growing-seconds")
	state := growing.At(later)
	if state.Scale != picked(t, "orbit-growing-scale") {
		t.Fatalf("two seconds on the scale is %v", state.Scale)
	}
	parent := moon.At(later).Origin
	if state.Origin.Sub(parent).Len() > orbitTight(t) {
		t.Fatalf("it is at %v, not with its parent", state.Origin)
	}
}

// A beam boosts what comes toward the eye and dims what goes away, the
// same on both sides of the origin, and does nothing at rest.
func TestBeamBoostsTowardTheEye(t *testing.T) {
	beam := Beam{Speed: picked(t, "orbit-beam-speed")}
	upper, lower := orbitPoint(t, "beam-upper"), orbitPoint(t, "beam-lower")
	coming := beam.Boost(maths.Up, upper)
	going := beam.Boost(maths.Up.Neg(), upper)
	side := beam.Boost(maths.XAxis, upper)
	least, most := picked(t, "orbit-beam-coming-least"),
		picked(t, "orbit-beam-going-most")
	if !(coming > least && going < most && side < unity) {
		t.Fatalf("coming %v, going %v, side %v", coming, going, side)
	}
	if other := beam.Boost(maths.Up.Neg(), lower); other != coming {
		t.Fatalf("the lower jet toward the eye boosts %v, the upper %v",
			other, coming)
	}
	if rest := (Beam{}).Boost(maths.Up, upper); rest != unity {
		t.Fatalf("at rest the boost is %v", rest)
	}
}

// A round beam boosts the side of a disc coming toward the eye and
// dims the side going away, mirrored across the axis, and less further
// out where the orbit is slower.
func TestARoundBeamBoostsTheSideComing(t *testing.T) {
	beam := orbitRoundBeam(t, "orbit-round-beam-speed")
	plusX, minusX := orbitPoint(t, "plus-x"), orbitPoint(t, "minus-x")
	minusZ, plusZ := orbitPoint(t, "minus-z"), orbitPoint(t, "plus-z")
	// at plus x the flow is up cross x: minus z
	coming := beam.Boost(minusZ, plusX)
	going := beam.Boost(plusZ, plusX)
	least, most := picked(t, "orbit-round-coming-least"),
		picked(t, "orbit-round-going-most")
	if !(coming > least && going < most) {
		t.Fatalf("coming %v, going %v", coming, going)
	}
	other := beam.Boost(plusZ, minusX)
	if !nearly(other, coming, orbitTolerance(t)) {
		t.Fatalf("across the axis the boost is %v, not %v", other, coming)
	}
	fourX := orbitPoint(t, "four-x")
	if far := beam.Boost(minusZ, fourX); !(far < coming && far > unity) {
		t.Fatalf("four radii out the boost is %v", far)
	}
}

// A beam's flow: a round one goes round prograde, from +X toward -Z,
// slowing outside At by the square root; one along its axis flows away
// from its origin on either side.
func TestABeamFlows(t *testing.T) {
	round := orbitRoundBeam(t, "orbit-flow-speed")
	direction, speed := round.Flow(orbitPoint(t, "plus-x"))
	if direction.Sub(orbitPoint(t, "minus-z")).Len() > orbitTight(t) ||
		speed != picked(t, "orbit-flow-speed") {
		t.Errorf("a round beam at (1,0,0) flows %v at %v", direction, speed)
	}
	_, outer := round.Flow(orbitPoint(t, "four-x"))
	if !nearly(outer, picked(t, "orbit-flow-outer-speed"), orbitTight(t)) {
		t.Errorf("at four times At it flows at %v, not half", outer)
	}
	jet := Beam{Speed: picked(t, "orbit-flow-jet-speed")}
	below := orbitPoint(t, "flow-below")
	if direction, _ := jet.Flow(below); direction != maths.Up.Neg() {
		t.Errorf("below its origin a jet flows %v", direction)
	}
}
