package door

import (
	"bytes"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/render/frame"
)

// The sizes and times the rows tests are held to.
const (
	testRowWidth = 3                // defined: a few pixels
	testRows     = 40               // defined: more than progress.MinRows
	testDrawn    = 10               // defined: rows drawn so far
	testElapsed  = 20 * time.Second // defined: past progress.MinElapsed
	testForever  = time.Hour        // defined: no tick in a test
)

// testRow is a row whose pixels hold every word kept, a NaN and a
// negative nought among them, which must come back to the bit.
func testRow() []frame.Pixel {
	nan := float32(math.NaN())
	negative := float32(math.Copysign(0, -1))
	return []frame.Pixel{{R: 1, G: 2, B: 3, Cover: 0.5, Depth: 7},
		{R: nan, G: negative, B: -1, Cover: 1,
			Depth: float32(math.Inf(1))},
		{R: 0.1, G: 0.2, B: 0.3, Cover: 0, Depth: 0}}
}

// sameBits is whether two rows are the same to the bit, NaN and all.
func sameBits(got, want []frame.Pixel) bool {
	return bytes.Equal(encodedRow(got), encodedRow(want))
}

// testKept is rows kept in a store under a fresh cache home, keyed by a
// job's name, with a clock that stands still.
func testKept(t *testing.T, name string, at time.Time) (*keptRows,
	*bytes.Buffer) {
	t.Helper()
	store, err := diskcache.Open(t.TempDir(), rowKind)
	if err != nil {
		t.Fatal(err)
	}
	said := &bytes.Buffer{}
	return &keptRows{store: store,
		key: diskcache.NewKey(rowKind).Text("job", name), out: said,
		now: func() time.Time { return at }, every: testForever}, said
}

// a row kept and read back is the same to the bit.
func TestARowKeptIsReadBackToTheBit(t *testing.T) {
	encoded := encodedRow(testRow())
	pixels, whole := decodedRow(encoded, testRowWidth)
	if !whole || !sameBits(pixels, testRow()) {
		t.Errorf("read back %v, whole %t, not %v", pixels, whole,
			testRow())
	}
}

// anti-test of TestARowKeptIsReadBackToTheBit: a row of another width,
// or cut short, is not a row.
func TestARowOfAnotherLengthIsNotARow(t *testing.T) {
	raw := encodedRow(testRow())
	if _, whole := decodedRow(raw, testRowWidth+one); whole {
		t.Error("a row was read as one pixel wider")
	}
	if _, whole := decodedRow(raw[:len(raw)-one], testRowWidth); whole {
		t.Error("a row a byte short was read")
	}
}

// a row drawn is taken up by the next draw of the same still.
func TestARowDrawnIsTakenUp(t *testing.T) {
	kept, _ := testKept(t, "still", time.Now())
	kept.Drawn(one, testRow())
	pixels, found := kept.Taken(one, testRowWidth)
	if !found || !sameBits(pixels, testRow()) || kept.taken != one {
		t.Errorf("took %v, found %t, counted %d", pixels, found,
			kept.taken)
	}
}

// anti-test of TestARowDrawnIsTakenUp: a fresh draw takes nothing up,
// and another job's rows are not this one's.
func TestAFreshDrawOrAnotherJobTakesNothingUp(t *testing.T) {
	kept, _ := testKept(t, "still", time.Now())
	kept.Drawn(one, testRow())
	other := &keptRows{store: kept.store,
		key: diskcache.NewKey(rowKind).Text("job", "another"),
		out: kept.out, now: kept.now, every: testForever}
	if _, found := other.Taken(one, testRowWidth); found {
		t.Error("another job's row was taken up")
	}
	kept.fresh = true
	if _, found := kept.Taken(one, testRowWidth); found {
		t.Error("a fresh draw took a row up")
	}
}

// rows forgotten once the still is written are not taken up again.
func TestRowsForgottenAreNotTakenUp(t *testing.T) {
	kept, _ := testKept(t, "still", time.Now())
	kept.Begin(testRowWidth)
	kept.Drawn(one, testRow())
	kept.close()
	kept.forget()
	if _, found := kept.Taken(one, testRowWidth); found {
		t.Error("a forgotten row was taken up")
	}
}

// the progress line says the rows finished of all, those taken up, the
// time taken and the time left at the pace of the rows drawn.
func TestTheProgressLineSaysHowFarAndHowLong(t *testing.T) {
	began := time.Now()
	kept, _ := testKept(t, "still", began.Add(testElapsed))
	kept.all, kept.began = testRows, began
	kept.taken, kept.drawn = testDrawn, testDrawn
	want := "progress rows=20/40 taken=10 elapsed=20s remaining~40s"
	if got := kept.line(); got != want {
		t.Errorf("the line is %q, not %q", got, want)
	}
}

// anti-test of TestTheProgressLineSaysHowFarAndHowLong: before enough
// rows are drawn the time left is not said.
func TestTheProgressLineSaysNoTimeLeftTooSoon(t *testing.T) {
	began := time.Now()
	kept, _ := testKept(t, "still", began.Add(testElapsed))
	kept.all, kept.began, kept.drawn = testRows, began, one
	if got := kept.line(); strings.Contains(got, "remaining") {
		t.Errorf("after one row the line says the time left: %q", got)
	}
}

// a draw begun and closed says its last line once, and nil rows or rows
// never begun say nothing.
func TestCloseSaysTheLastLine(t *testing.T) {
	kept, said := testKept(t, "still", time.Now())
	kept.close()
	if said.Len() != nothing {
		t.Errorf("rows never begun said %q", said.String())
	}
	kept.Begin(testRows)
	kept.close()
	kept.close()
	if lines := strings.Count(said.String(), "\n"); lines != one {
		t.Errorf("closed twice it said %d lines: %q", lines,
			said.String())
	}
	var none *keptRows
	none.close()
	none.forget()
}
