package thing

import (
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// the ranges are tried narrowest first. the widths are worked by hand
// from celestial's iso 21348 limits (tobiska and nusinov 2004) and the
// observer's rows, which run from 360 nm to 830 nm (the cie 1931 table):
//
//	gamma 0.00099, x-ray 9.999, extreme ultraviolet 110, ultraviolet 300,
//	visible 470, infrared 999,240, microwave 14,000,000, radio about 1e11.
func TestRangesRunNarrowestFirst(t *testing.T) {
	if short, long := Visible.Edges(); short != 360 || long != 830 {
		t.Fatalf("the visible runs %v to %v nm, not 360 to 830", short, long)
	}
	want := []radiation.Range{radiation.Gamma(), radiation.XRay(),
		radiation.ExtremeUltraviolet(), radiation.Ultraviolet(), Visible,
		radiation.Infrared(), radiation.Microwave(), radiation.Radio()}
	if len(ranges) != len(want) {
		t.Fatalf("there are %d ranges, want %d", len(ranges), len(want))
	}
	for i, span := range want {
		if ranges[i] != span {
			t.Errorf("range %d is %v, want %v", i, ranges[i], span)
		}
	}
}

// no two ranges are the same width, so that sorting them by width has
// no ties to break: which of two equals came first would otherwise
// decide which answers where they overlap.
func TestNoTwoRangesAreTheSameWidth(t *testing.T) {
	for i, a := range ranges {
		for _, b := range ranges[i+1:] {
			if width(a) == width(b) {
				t.Errorf("%v and %v are the same width", a, b)
			}
		}
	}
}

// a range holds its low end and not its high one, and where ranges
// overlap the narrowest answers. worked by hand from the limits:
//
//	1e-5 gamma's low end; 1e-3 gamma's high end, x-ray's low end
//	10 x-ray's high end, extreme ultraviolet's low end
//	100 in extreme ultraviolet (110 wide) and ultraviolet (300): the first
//	120 extreme ultraviolet's high end: ultraviolet
//	360 in ultraviolet (300 wide) and visible (470): ultraviolet
//	400 ultraviolet's high end: visible
//	760, 829 in visible (470) and infrared (999,240): visible
//	830 visible's high end: infrared
//	1e5 in infrared and radio: infrared
//	1e6 infrared's high end, microwave's low end: microwave
//	1.5e7 microwave's high end: radio
//	1e11 radio's high end, and 1e-6 under gamma: none
func TestRangeEdgesAndOverlaps(t *testing.T) {
	for _, c := range []struct {
		nm   float64
		want radiation.Range
	}{
		{1e-6, radiation.Range{}}, {1e-5, radiation.Gamma()}, {1e-3, radiation.XRay()},
		{10, radiation.ExtremeUltraviolet()}, {100, radiation.ExtremeUltraviolet()},
		{120, radiation.Ultraviolet()}, {360, radiation.Ultraviolet()},
		{400, Visible}, {760, Visible}, {829, Visible}, {830, radiation.Infrared()},
		{1e5, radiation.Infrared()}, {1e6, radiation.Microwave()},
		{1.5e7, radiation.Radio()}, {1e11, radiation.Range{}},
	} {
		if got := Range(maths.Band{Nanometres: c.nm}); got != c.want {
			t.Errorf("%v nm is %v, want %v", c.nm, got, c.want)
		}
	}
}

// visible is from 360 nm up to but not including 830, and the zero band.
func TestSeenHoldsItsLowEndAndNotItsHigh(t *testing.T) {
	for _, c := range []struct {
		nm   float64
		want bool
	}{
		{0, true}, {359.9, false}, {360, true}, {550, true}, {829.9, true},
		{830, false}, {1e6, false},
	} {
		if got := Seen(maths.Band{Nanometres: c.nm}); got != c.want {
			t.Errorf("%v nm seen is %v, want %v", c.nm, got, c.want)
		}
	}
}

// the zero band looks up the visible, and only the visible: with no
// entry for it there is none, and a nanometre, which is an x-ray, is
// not the visible.
func TestLookupOfTheZeroBandIsTheVisible(t *testing.T) {
	table := map[radiation.Range]float64{Visible: 7}
	if v, ok := Lookup(table, maths.Band{}); v != 7 || !ok {
		t.Errorf("the zero band found %v, %v, want 7, true", v, ok)
	}
	if v, ok := Lookup(table, maths.Band{Nanometres: 1}); v != 0 || ok {
		t.Errorf("a nanometre found %v, %v in a table of the visible", v, ok)
	}
	others := map[radiation.Range]float64{radiation.XRay(): 3, radiation.Gamma(): 4,
		radiation.Radio(): 5, radiation.Ultraviolet(): 6}
	if v, ok := Lookup(others, maths.Band{}); v != 0 || ok {
		t.Errorf("the zero band found %v, %v in a table without the visible", v, ok)
	}
	if v, ok := Lookup(others, maths.Band{Nanometres: 1}); v != 3 || !ok {
		t.Errorf("a nanometre found %v, %v, want the x-ray's 3, true", v, ok)
	}
}
