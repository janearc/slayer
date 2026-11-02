package plume

import (
	"math"
	"testing"
	"time"
)

// megawattPlume is the plume tests' worked fire, a megawatt in air at
// 293 k, from inside the package.
func megawattPlume(t *testing.T) *Column {
	t.Helper()
	column, err := New(Fire{Heat: 1000, Convective: 0.7, Base: 1.13,
		Yield: 0.0024, Combustion: 21, Extinction: 8.7, Air: 293,
		Density: 1.204, Top: 40})
	if err != nil {
		t.Fatal(err)
	}
	return column
}

// the eddies do not shear: how far apart in phase two heights are does
// not change as time passes.
func TestTheEddiesDoNotShear(t *testing.T) {
	column := megawattPlume(t)
	then := column.phase(20) - column.phase(6)
	later := column.Tick(time.Minute).(*Column)
	if now := later.phase(20) - later.phase(6); !(math.Abs(now-then) <=
		1e-9*math.Abs(then)) {
		t.Errorf("6 m to 20 m apart in phase: %g, a minute on %g", then,
			now)
	}
}

// air rising with the plume keeps its phase: followed up the column in
// small steps of mccaffrey's speed for three seconds, from 5 m, it has
// at its new height and time the phase it had where it started.
func TestAirKeepsItsPhase(t *testing.T) {
	column := megawattPlume(t)
	height, step := 5.0, 1e-4
	for range int(3 / step) {
		half := height + 0.5*step*column.Speed(height)
		height += step * column.Speed(half)
	}
	later := column.Tick(3 * time.Second).(*Column)
	if then, now := column.phase(5), later.phase(height); !(math.Abs(
		now-then) <= 1e-4) {
		t.Errorf("air from 5 m at %g m after 3 s: phase %g, then %g",
			height, now, then)
	}
}
