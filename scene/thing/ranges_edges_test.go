package thing

import (
	"math"
	"testing"

	"github.com/janearc/celestial/radiation"
	"github.com/janearc/slayer/maths"
)

// The signs math.Inf takes for plus and minus infinity.
const (
	rangesPlusSign  = 1  // defined by math.Inf: a sign above nought is plus infinity
	rangesMinusSign = -1 // defined by math.Inf: a sign below nought is minus infinity
)

// the ranges are tried narrowest first. The visible runs from the first
// row of the observer's table to its last, and every other range is
// celestial's, whose edges are asked of the ranges and not written here.
func TestRangesRunNarrowestFirst(t *testing.T) {
	short, long := Visible().Edges()
	if short != picked(t, "ranges-visible-short") ||
		long != picked(t, "ranges-visible-long") {
		t.Fatalf("the visible runs %v to %v nm", short, long)
	}
	want := []radiation.Range{radiation.Gamma(), radiation.XRay(),
		radiation.ExtremeUltraviolet(), radiation.Ultraviolet(), Visible(),
		radiation.Infrared(), radiation.Microwave(), radiation.Radio()}
	if len(ranges) != len(want) {
		t.Fatalf("there are %d ranges, want %d", len(ranges), len(want))
	}
	for place, span := range want {
		if ranges[place] != span {
			t.Errorf("range %d is %v, want %v", place, ranges[place], span)
		}
	}
}

// no two ranges are the same width, so that sorting them by width has
// no ties to break: which of two equals came first would otherwise
// decide which answers where they overlap.
func TestNoTwoRangesAreTheSameWidth(t *testing.T) {
	for place, first := range ranges {
		for _, second := range ranges[place+unity:] {
			if width(first) == width(second) {
				t.Errorf("%v and %v are the same width", first, second)
			}
		}
	}
}

// rangesLow is where a range starts, in nanometres.
func rangesLow(span radiation.Range) float64 {
	short, _ := span.Edges()
	return short
}

// rangesHigh is where a range ends, in nanometres, and what it does not hold.
func rangesHigh(span radiation.Range) float64 {
	_, long := span.Edges()
	return long
}

// below is the wavelength just under another.
func rangesBelow(nanometres float64) float64 {
	return math.Nextafter(nanometres, math.Inf(rangesMinusSign))
}

// rangesHold is whether a wavelength is from a range's low edge up to
// but not including its high one.
func rangesHold(span radiation.Range, nanometres float64) bool {
	return nanometres >= rangesLow(span) && nanometres < rangesHigh(span)
}

// rangesNarrowestAt is the narrowest of the ranges whose edges hold a
// wavelength, or the zero range for none. It is worked from the edges and
// the widths alone, and not from the order Range tries the ranges in.
func rangesNarrowestAt(nanometres float64) radiation.Range {
	var found radiation.Range
	narrowest := math.Inf(rangesPlusSign)
	for _, span := range ranges {
		if rangesHold(span, nanometres) && width(span) < narrowest {
			found, narrowest = span, width(span)
		}
	}
	return found
}

// a range holds its low end and not its high one, and where ranges
// overlap the narrowest answers. Each range's two edges, and the
// wavelength just under each, are put to Range and the answer is the one
// the edges and widths give.
func TestRangeEdgesAndOverlaps(t *testing.T) {
	for _, span := range ranges {
		for _, nanometres := range []float64{rangesLow(span),
			rangesBelow(rangesLow(span)), rangesHigh(span), rangesBelow(rangesHigh(span))} {
			got := Range(maths.NewBand(nanometres))
			if want := rangesNarrowestAt(nanometres); got != want {
				t.Errorf("%v nm is %v, want %v", nanometres, got, want)
			}
		}
	}
}

// the anti-test of the edges: a wavelength just below the lowest edge of
// every range, or at the highest high edge, is in none, and every range
// does answer at its own low edge.
func TestNoRangeHoldsWhatIsPastEveryEdge(t *testing.T) {
	lowest, highest := math.Inf(rangesPlusSign), math.Inf(rangesMinusSign)
	for _, span := range ranges {
		lowest = math.Min(lowest, rangesLow(span))
		highest = math.Max(highest, rangesHigh(span))
		edge := maths.NewBand(rangesLow(span))
		got := Range(edge)
		if got == (radiation.Range{}) {
			t.Errorf("%v nm, the low edge of %v, is in no range",
				rangesLow(span), span)
		}
	}
	for _, nanometres := range []float64{rangesBelow(lowest), highest} {
		got := Range(maths.NewBand(nanometres))
		if got != (radiation.Range{}) {
			t.Errorf("%v nm is in %v, past every range's edges", nanometres, got)
		}
	}
}

// the overlaps radiant.go names are there: the extreme ultraviolet's last
// wavelength inside the ultraviolet, the microwave's first and last inside
// the radio, the visible's first inside the ultraviolet and its last inside
// the infrared, and the infrared's last inside the radio. At each, both
// ranges hold the wavelength and the narrower one answers.
func TestTheNamedOverlapsAreOverlapsAndTheNarrowerAnswers(t *testing.T) {
	euv, uv := radiation.ExtremeUltraviolet(), radiation.Ultraviolet()
	micro, radio := radiation.Microwave(), radiation.Radio()
	infrared := radiation.Infrared()
	visibleHigh := rangesHigh(Visible())
	for _, each := range []struct {
		what         string
		at           float64
		inner, outer radiation.Range
	}{
		{"extreme ultraviolet's last", rangesBelow(rangesHigh(euv)), euv, uv},
		{"microwave's first", rangesLow(micro), micro, radio},
		{"microwave's last", rangesBelow(rangesHigh(micro)), micro, radio},
		{"visible's first", rangesLow(Visible()), Visible(), uv},
		{"visible's last", rangesBelow(visibleHigh), Visible(), infrared},
		{"infrared's last", rangesBelow(rangesHigh(infrared)), infrared, radio},
	} {
		if !rangesHold(each.inner, each.at) ||
			!rangesHold(each.outer, each.at) {
			t.Errorf("%s, at %v nm, is not in both ranges", each.what, each.at)
			continue
		}
		want := each.inner
		if width(each.outer) < width(each.inner) {
			want = each.outer
		}
		if got := Range(maths.NewBand(each.at)); got != want {
			t.Errorf("%s, at %v nm, is %v, want the narrower, %v", each.what,
				each.at, got, want)
		}
	}
}

// visible is from its low edge up to but not including its high edge, and
// the zero band. The anti-test is the wavelength just under the low edge,
// the high edge itself, and a wavelength well past it. The ultraviolet
// runs up to 400 nm, over the visible's first stretch, and is the narrower
// of the two there (300 nm to 470), so a band is the visible's by Range
// only from the ultraviolet's high edge up; the edges themselves are the
// range's own.
func TestVisibleHoldsItsLowEndAndNotItsHigh(t *testing.T) {
	short, long := Visible().Edges()
	_, ultravioletHigh := radiation.Ultraviolet().Edges()
	for _, each := range []struct {
		nanometres float64
		want       bool
	}{
		{rangesBelow(short), false}, {short, true},
		{(short + long) / pair, true}, {rangesBelow(long), true},
		{long, false}, {long * pair, false},
	} {
		if got := Visible().Holds(each.nanometres); got != each.want {
			t.Errorf("%v nm held by the visible is %v, want %v",
				each.nanometres, got, each.want)
		}
	}
	for _, each := range []struct {
		nanometres float64
		want       bool
	}{
		{nought, true}, {short, false}, {rangesBelow(ultravioletHigh), false},
		{ultravioletHigh, true}, {(short + long) / pair, true},
		{rangesBelow(long), true}, {long, false}, {long * pair, false},
	} {
		got := Range(maths.NewBand(each.nanometres)) == Visible()
		if got != each.want {
			t.Errorf("%v nm seen is %v, want %v", each.nanometres, got,
				each.want)
		}
	}
}

// the zero band looks up the visible, and only the visible: with no
// entry for it there is none, and a nanometre, which is an x-ray, is
// not the visible.
func TestLookupOfTheZeroBandIsTheVisible(t *testing.T) {
	marked := picked(t, "ranges-mark-visible")
	xrayNanometre := maths.NewBand(picked(t, "ranges-xray-nanometre"))
	table := map[radiation.Range]float64{Visible(): marked}
	if found, ok := Lookup(table, maths.NewBand(nought)); found != marked || !ok {
		t.Errorf("the zero band found %v, %v, want %v, true", found, ok,
			marked)
	}
	if found, ok := Lookup(table, xrayNanometre); found != nought || ok {
		t.Errorf("a nanometre found %v, %v in a table of the visible", found,
			ok)
	}
	xray := picked(t, "ranges-mark-xray")
	others := map[radiation.Range]float64{
		radiation.XRay():        xray,
		radiation.Gamma():       picked(t, "ranges-mark-gamma"),
		radiation.Radio():       picked(t, "ranges-mark-radio"),
		radiation.Ultraviolet(): picked(t, "ranges-mark-ultraviolet"),
	}
	if found, ok := Lookup(others, maths.NewBand(nought)); found != nought || ok {
		t.Errorf("the zero band found %v, %v in a table without the visible",
			found, ok)
	}
	if found, ok := Lookup(others, xrayNanometre); found != xray || !ok {
		t.Errorf("a nanometre found %v, %v, want the x-ray's %v, true", found,
			ok, xray)
	}
}
