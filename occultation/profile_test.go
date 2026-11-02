package occultation

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// row is a table's row in the pds's columns for a radius in km and a
// median depth, its other columns plausible.
func row(km, depth float64) string {
	return fmt.Sprintf("%9.2f, 10.000, 1.000,%7.4f, 0.0000, 1.0000\r\n", km,
		depth)
}

// table is a profile read from rows, or the test stops.
func table(t *testing.T, rows ...string) Profile {
	t.Helper()
	profile, err := Read(strings.NewReader(strings.Join(rows, "")))
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

// a row's radius is its bin's middle, in km: three 1 km rows from 100
// km make a profile from 99.5 to 102.5 km, in metres.
func TestATableIsBinsAboutEachRadius(t *testing.T) {
	profile := table(t, row(100, 0.5), row(101, 1), row(102, 2))
	if profile.Inner() != 99.5e3 || profile.Outer() != 102.5e3 {
		t.Errorf("from %g to %g m, want 99500 to 102500", profile.Inner(),
			profile.Outer())
	}
	for _, x := range []struct{ at, want float64 }{
		{99.6e3, 0.5}, {100.9e3, 1}, {102.4e3, 2},
	} {
		if got := profile.Seen(x.at, 0); got != x.want {
			t.Errorf("at %g m, depth %g, want %g", x.at, got, x.want)
		}
	}
}

// 99 is opaque, read as six; a depth less than zero is an empty gap's noise,
// read as zero.
func TestTheTablesMarksAreRead(t *testing.T) {
	profile := table(t, row(1, 99), row(2, -0.02), row(3, 0.3))
	if got := profile.Seen(1e3, 0); got != opaque {
		t.Errorf("99 read as %g, want %g", got, float64(opaque))
	}
	if got := profile.Seen(2e3, 0); got != 0 {
		t.Errorf("-0.02 read as %g, want nought", got)
	}
}

// a row with no depth, a row of the wrong shape, uneven or falling
// radii and a single row are all refused.
func TestWhatCannotBeAProfileIsRefused(t *testing.T) {
	for name, rows := range map[string][]string{
		"no depth": {row(1, 1), "     2.00,  0.000,99.000, 0.0000,-9.0000," +
			" 0.0000\r\n", row(3, 1)},
		"five fields":  {row(1, 1), "2.00, 1, 1, 1, 1\r\n"},
		"not a number": {row(1, 1), "2.00, 1, 1, one, 1, 1\r\n"},
		"uneven":       {row(1, 1), row(2, 1), row(4, 1)},
		"falling":      {row(3, 1), row(2, 1)},
		"one row":      {row(1, 1)},
	} {
		_, err := Read(strings.NewReader(strings.Join(rows, "")))
		if err == nil {
			t.Errorf("%s: read", name)
		}
	}
}

// across a stretch of one depth, the depth seen is that depth.
func TestAnEvenStretchIsSeenAsItsDepth(t *testing.T) {
	profile := table(t, row(1, 0.7), row(2, 0.7), row(3, 0.7), row(4, 0.7))
	if got := profile.Seen(2.5e3, 2e3); math.Abs(got-0.7) > 1e-12 {
		t.Errorf("seen %g, want 0.7", got)
	}
}

// half empty and half opaque, the light gets through the empty half:
// seen as -ln of the mean of 1 and e^-6, about 0.69, not their mean
// depth of three.
func TestAGapLetsItsLightThrough(t *testing.T) {
	profile := table(t, row(1, 0), row(2, 99))
	want := -math.Log((1 + math.Exp(-opaque)) / 2)
	if got := profile.Seen(1.5e3, 2e3); math.Abs(got-want) > 1e-12 {
		t.Errorf("seen %g, want %g", got, want)
	}
	if got := profile.Mean(0.5e3, 2.5e3); math.Abs(got-opaque/2.0) > 1e-12 {
		t.Errorf("mean %g, want %g", got, opaque/2.0)
	}
}

// part of a bin counts by the part: a quarter of a 2 deep bin and three
// quarters of a zero deep one mean half a depth.
func TestPartOfABinCountsByThePart(t *testing.T) {
	profile := table(t, row(1, 0), row(2, 2))
	if got := profile.Mean(0.75e3, 1.75e3); math.Abs(got-0.5) > 1e-12 {
		t.Errorf("mean %g, want 0.5", got)
	}
}

// outside the profile, there is nothing; a stretch reaching past it is
// read over the part inside.
func TestOutsideTheProfileIsEmpty(t *testing.T) {
	profile := table(t, row(1, 1), row(2, 1))
	if got := profile.Seen(3e3, 0); got != 0 {
		t.Errorf("outside, %g", got)
	}
	if got := profile.Seen(2.4e3, 2e3); math.Abs(got-1) > 1e-12 {
		t.Errorf("reaching past the edge, %g, want 1", got)
	}
}

// profiles join end to end only where one starts as the last ends, in
// bins of the same width.
func TestProfilesJoinOnlyEndToEnd(t *testing.T) {
	first := table(t, row(1, 1), row(2, 1))
	next := table(t, row(3, 2), row(4, 2))
	joined, err := Joined(first, next)
	if err != nil {
		t.Fatal(err)
	}
	if joined.Inner() != 0.5e3 || joined.Outer() != 4.5e3 ||
		joined.Seen(3.2e3, 0) != 2 {
		t.Errorf("joined from %g to %g m", joined.Inner(), joined.Outer())
	}
	apart := table(t, row(5, 2), row(6, 2))
	if _, err := Joined(first, apart); err == nil {
		t.Errorf("joined with a gap between")
	}
	wider := table(t, row(3, 2), row(5, 2))
	if _, err := Joined(first, wider); err == nil {
		t.Errorf("joined bins of two widths")
	}
	if _, err := Joined(); err == nil {
		t.Errorf("joined nothing")
	}
}

// saturn's rings in voyager 2's delta sco profile, pinned on the shelf,
// against nasa's saturnian rings fact sheet: the c ring from 74,658 km
// at 0.05 to 0.35; the b ring to 117,507 km at 0.4 to 2.5; the cassini
// division at 0 to 0.1; the a ring from 122,340 km at 0.4 to 1; the
// encke gap from 133,410 km. held loosely, inside each.
func TestSaturnsRingsAreWhereNasaSays(t *testing.T) {
	shelf := os.Getenv("SLAYER_SHELF")
	if shelf == "" {
		t.Skip("no shelf named in SLAYER_SHELF")
	}
	var parts []Profile
	for _, from := range []string{"072", "080", "090", "100", "110", "120",
		"130", "140"} {
		file, err := os.Open(filepath.Join(shelf,
			"vg2-pps-delta-sco-km001-"+from))
		if err != nil {
			t.Skip(err)
		}
		part, err := Read(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, part)
	}
	saturn, err := Joined(parts...)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []struct {
		name               string
		from, to, low, top float64
	}{
		{"the c ring", 76e3, 90e3, 0.02, 0.4},
		{"the b ring", 95e3, 115e3, 0.8, opaque},
		{"the cassini division", 118.5e3, 121.5e3, 0, 0.2},
		{"the a ring", 123e3, 133e3, 0.3, 1.2},
		{"the encke gap", 133.48e3, 133.68e3, 0, 0.1},
	} {
		mean := saturn.Mean(x.from*1e3, x.to*1e3)
		if mean < x.low || mean > x.top {
			t.Errorf("%s: mean depth %g, want %g to %g", x.name, mean,
				x.low, x.top)
		}
	}
}
