package span_test

import (
	"os"
	"slices"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/span"
	"pgregory.net/rapid"
)

// what these tests count in, each defined
const (
	lit  = 1 // defined: a lamp that shines counts one
	dark = 0 // defined: a lamp that does not shine counts zero
	next = 1 // defined: the nanometre after another is one on
)

// the tables the tests are run over
const (
	holdsFile   = "testdata/holds.csv"
	refusedFile = "testdata/refused.csv"
)

// what these tests choose, each picked in picked-numbers.yml
var (
	shortest   = data.Whole("span-shortest")
	longest    = data.Whole("span-longest")
	red        = data.Whole("span-red")
	green      = data.Whole("span-green")
	blue       = data.Whole("span-blue")
	margin     = data.Whole("span-margin")
	marginMost = data.Whole("span-margin-most")
	marginWide = data.Whole("span-margin-wide")
)

// table is the rows of a table of cases in testdata, or the test stops.
func table(t *testing.T, name string) []data.Row {
	t.Helper()
	text, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.Parse(string(text))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// whole is the whole number in a column of a row, or the test stops.
func whole(t *testing.T, row data.Row, column string) int {
	t.Helper()
	number, err := row.Whole(column)
	if err != nil {
		t.Fatal(err)
	}
	return number
}

// spanned is the span between two wavelengths, or the test stops.
func spanned(t stopper, from, to int) span.Span {
	t.Helper()
	made, err := span.New(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// stopper is a test that can be stopped: a testing.T, or a rapid.T.
type stopper interface {
	Helper()
	Fatal(args ...any)
}

// lamps is the light a span holds by the plain statement of it: each
// lamp is held when its wavelength is inside the span.
func lamps(from, to int) maths.Light {
	return maths.RGB(shines(from, to, red), shines(from, to, green),
		shines(from, to, blue))
}

// shines is one if a wavelength is inside a span and zero if not.
func shines(from, to, wavelength int) float64 {
	if from <= wavelength && wavelength <= to {
		return lit
	}
	return dark
}

// ends are the two ends of a span, drawn from a little wider than the
// observer sees. The first is at or before the second.
func ends(t *rapid.T) (from, to int) {
	from = rapid.IntRange(shortest-margin, longest+margin).Draw(t, "from")
	to = rapid.IntRange(from, longest+marginMost).Draw(t, "to")
	return from, to
}

func TestSpansRunForwardsInsideWhatIsSeen(t *testing.T) {
	spanned(t, shortest, longest)
	spanned(t, red, red)
	spanned(t, shortest, shortest)
	spanned(t, longest, longest)
}

// anti-tests: a backward span makes no span. Nor does one that starts
// before the observer sees or ends past it. What comes back is the zero
// span.
func TestASpanRefusesWhatItCannotBe(t *testing.T) {
	var zero span.Span
	for _, row := range table(t, refusedFile) {
		from, to := whole(t, row, "from"), whole(t, row, "to")
		got, err := span.New(from, to)
		if err == nil || got != zero {
			t.Errorf("a span of %d to %d was not refused: %v, %v", from,
				to, got, err)
		}
	}
}

// Whether a span is made is the plain statement of it: its ends run
// forwards, and both are inside what is seen.
func TestASpanIsMadeExactlyWhenItRunsForwardsInsideWhatIsSeen(
	t *testing.T) {
	rapid.Check(t, spanIsMadeWhenItCan)
}

func spanIsMadeWhenItCan(t *rapid.T) {
	from := rapid.IntRange(shortest-marginWide, longest+marginWide).Draw(t,
		"from")
	to := rapid.IntRange(shortest-marginWide, longest+marginWide).Draw(t,
		"to")
	want := from <= to && from >= shortest && to <= longest
	if _, err := span.New(from, to); (err == nil) != want {
		t.Fatalf("a span of %d to %d: made is %v, want %v", from, to,
			err == nil, want)
	}
}

func TestASpanHoldsTheLampsThatShineInsideIt(t *testing.T) {
	for _, row := range table(t, holdsFile) {
		from, to := whole(t, row, "from"), whole(t, row, "to")
		onRed, onGreen := whole(t, row, "red"), whole(t, row, "green")
		onBlue := whole(t, row, "blue")
		want := maths.RGB(float64(onRed), float64(onGreen), float64(onBlue))
		if got := spanned(t, from, to).Holds(); got != want {
			t.Errorf("%d to %d holds %v, want %v", from, to, got, want)
		}
	}
}

// Each lamp shines at one wavelength of the spectrum and no other. Lit
// one nanometre at a time, red is found at 611. Green is found at 549
// and blue at 464.
func TestTheLampsShineAtTheirOwnHues(t *testing.T) {
	found := map[maths.Light][]int{}
	var none maths.Light
	for nanometre := shortest; nanometre <= longest; nanometre++ {
		one := spanned(t, nanometre, nanometre)
		if lit := one.Holds(); lit != none {
			found[lit] = append(found[lit], nanometre)
		}
	}
	want := map[maths.Light]int{maths.RGB(lit, dark, dark): red,
		maths.RGB(dark, lit, dark): green, maths.RGB(dark, dark, lit): blue}
	if len(found) != len(want) {
		t.Errorf("%d kinds of lamp were found: %v", len(found), found)
	}
	for lamp, nanometre := range want {
		if got := found[lamp]; !slices.Equal(got, []int{nanometre}) {
			t.Errorf("lamp %v shines at %v, want %d", lamp, got,
				nanometre)
		}
	}
}

// Whatever span is drawn holds the lamps that are inside it.
func TestASpanHoldsExactlyTheLampsInsideIt(t *testing.T) {
	rapid.Check(t, spanHoldsItsLamps)
}

func spanHoldsItsLamps(t *rapid.T) {
	from, to := ends(t)
	made, err := span.New(from, to)
	if err != nil {
		t.Skip("a span past what is seen")
	}
	if got, want := made.Holds(), lamps(from, to); got != want {
		t.Fatalf("%d to %d holds %v, want %v", from, to, got, want)
	}
}

// Spans that split the spectrum between them split the lamps between
// them. Cut in two, a span holds in its halves the lamps it holds whole.
func TestSpansThatSplitTheSpectrumSplitTheLamps(t *testing.T) {
	rapid.Check(t, splitSpansSplitLamps)
}

func splitSpansSplitLamps(t *rapid.T) {
	from := rapid.IntRange(shortest, longest-next).Draw(t, "from")
	to := rapid.IntRange(from+next, longest).Draw(t, "to")
	cut := rapid.IntRange(from, to-next).Draw(t, "cut")
	whole := spanned(t, from, to)
	before, after := spanned(t, from, cut), spanned(t, cut+next, to)
	halves := before.Holds().Add(after.Holds())
	if halves != whole.Holds() {
		t.Fatalf("%d to %d, cut after %d: halves hold %v, whole holds %v",
			from, to, cut, halves, whole.Holds())
	}
}

// Inside a span, another holds no lamp that the outer one does not.
func TestASpanInsideAnotherHoldsNoMoreThanIt(t *testing.T) {
	rapid.Check(t, nestedSpansHoldLess)
}

func nestedSpansHoldLess(t *rapid.T) {
	from := rapid.IntRange(shortest, longest).Draw(t, "from")
	to := rapid.IntRange(from, longest).Draw(t, "to")
	inner := rapid.IntRange(from, to).Draw(t, "inner")
	last := rapid.IntRange(inner, to).Draw(t, "last")
	out, in := spanned(t, from, to).Holds(), spanned(t, inner, last).Holds()
	if in.R > out.R || in.G > out.G || in.B > out.B {
		t.Fatalf("%d to %d holds %v, and %d to %d inside it holds %v",
			from, to, out, inner, last, in)
	}
}
