package occultation

import (
	"math"
	"testing"
)

// striped is eight 1 km bins, empty and opaque by turns, from 0.5 km.
func striped(t *testing.T) Profile {
	t.Helper()
	var rows []string
	for k := range 8 {
		depth := 0.0
		if k%2 == 1 {
			depth = 99
		}
		rows = append(rows, row(float64(k+1), depth))
	}
	return table(t, rows...)
}

// eight bins bake to four levels, eight, four, two and one wide, each
// bin the mean of the two below it.
func TestATableHalvesToOneBin(t *testing.T) {
	levels := striped(t).Baked().Levels()
	if len(levels) != 4 {
		t.Fatalf("%d levels, want 4", len(levels))
	}
	for k, want := range []int{8, 4, 2, 1} {
		if len(levels[k]) != want {
			t.Errorf("level %d has %d bins, want %d", k, len(levels[k]), want)
		}
	}
	half := float32((1 + math.Exp(-opaque)) / 2)
	if levels[3][0] != half {
		t.Errorf("the top bin keeps %g, want %g", levels[3][0], half)
	}
}

// read at its own bin's middle across one bin, the table gives the
// bin's depth, as the profile does, to float32.
func TestTheFinestLevelIsTheProfile(t *testing.T) {
	profile := striped(t)
	baked := profile.Baked()
	for k := range 8 {
		middle := float64(k)*1e3 + 1e3
		want := profile.Seen(middle, 0)
		got := float64(baked.Seen(float32(middle), 1e3))
		if math.Abs(got-want) > 1e-5 {
			t.Errorf("bin %d: %g, want %g", k, got, want)
		}
	}
}

// read across four bins, at the middle of four, the table gives what
// light sees across them: the gaps' light, not the bands' mean depth.
func TestAWideRayReadsTheMeanLight(t *testing.T) {
	profile := striped(t)
	baked := profile.Baked()
	want := profile.Seen(2.5e3, 4e3)
	got := float64(baked.Seen(2.5e3, 4e3))
	if math.Abs(got-want) > 1e-5 {
		t.Errorf("%g, want %g", got, want)
	}
	if want > 1 {
		t.Errorf("seen %g across a striped stretch, deeper than its gaps "+
			"allow", want)
	}
}

// a ray wider than the whole table reads its one top bin; outside the
// table there is nothing.
func TestBeyondTheTable(t *testing.T) {
	baked := striped(t).Baked()
	want := float32(-math.Log((1 + math.Exp(-opaque)) / 2))
	if got := baked.Seen(4e3, 1e9); math.Abs(float64(got-want)) > 1e-5 {
		t.Errorf("across everything, %g, want %g", got, want)
	}
	if got := baked.Seen(9e3, 1); got != 0 {
		t.Errorf("outside, %g", got)
	}
}

// an even profile is its depth at every width.
func TestAnEvenTableIsItsDepthAtEveryWidth(t *testing.T) {
	baked := table(t, row(1, 0.4), row(2, 0.4), row(3, 0.4)).Baked()
	for _, across := range []float32{0, 500, 1e3, 1.7e3, 3e3, 1e6} {
		got := baked.Seen(1.8e3, across)
		if math.Abs(float64(got)-0.4) > 1e-5 {
			t.Errorf("across %g m, %g, want 0.4", across, got)
		}
	}
}
