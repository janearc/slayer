//go:build darwin

package gpu

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/janearc/slayer/render/frame"
)

func checkFinishedValue(t *testing.T, got float32, want float64,
	book finishBook,
) {
	t.Helper()
	difference := math.Abs(float64(got) - want)
	limit := max(book.Expected.Absolute, math.Abs(want)*book.Expected.Relative)
	if !(difference <= limit) {
		t.Fatalf("finished value %g, want %g within %g", got, want, limit)
	}
}

// Expected values are closed-form means from the authored book, not the
// old averaging function or another pass through production finishing.
func checkFinishedPixel(t *testing.T, picture *frame.Frame,
	book finishBook, column, row int,
) {
	t.Helper()
	got := picture.At(column, row)
	expected := book.Expected
	base := expected.Base[0]*float64(column) +
		expected.Base[1]*float64(row) + expected.Base[2]
	for channel, value := range []float32{got.R, got.G, got.B} {
		want := base*expected.Scales[channel] + expected.Offsets[channel]
		if column == book.Width-1 && row == book.Height-1 {
			want = expected.Sky[channel]
		}
		checkFinishedValue(t, value, want, book)
	}
	checkFinishedCoverage(t, got, book, column, row)
}

func checkFinishedCoverage(t *testing.T, got frame.Pixel,
	book finishBook, column, row int,
) {
	t.Helper()
	cover := book.Expected.Cover
	depth := float32(book.Expected.Depth[0]*float64(column) +
		book.Expected.Depth[1]*float64(row) + book.Expected.Depth[2])
	if column == book.Width-1 && row == book.Height-1 {
		cover, depth = 0, float32(math.Inf(1))
	}
	checkFinishedValue(t, got.Cover, cover, book)
	if got.Depth != depth {
		t.Fatalf("nearest depth %g, want %g", got.Depth, depth)
	}
}

func consumeFinishFixture(t *testing.T, band *bandFinisher, book finishBook) {
	t.Helper()
	buffer := make([]float32, book.Width*book.Side*64*book.stride())
	for _, batch := range book.Batches {
		// Reuse the same storage as the native bridge. Poisoning exposes
		// any attempt to borrow an earlier band's samples after return.
		for i := range buffer {
			buffer[i] = float32(math.NaN())
		}
		raw := fillFinishBand(buffer, book, batch[0], batch[1])
		if err := band.consume(batch[0], batch[1], raw); err != nil {
			t.Fatal(err)
		}
		checkFinishedPixel(t, band.picture, book, 0, 0)
	}
}

func TestReadbackFinishesPixelsAcrossReused64RowBands(t *testing.T) {
	band, book := finishFixture(t)
	consumeFinishFixture(t, band, book)
	for row := range book.Height {
		for column := range book.Width {
			t.Run(fmt.Sprintf("%d,%d", column, row), func(t *testing.T) {
				checkFinishedPixel(t, band.picture, book, column, row)
			})
		}
	}
	if band.next != book.Height*book.Side || band.bytes != book.Bytes ||
		band.peak != book.Peak || band.native != book.Native {
		t.Fatalf("finished accounting: rows %d, bytes %d/%d, native %v",
			band.next, band.bytes, band.peak, band.native)
	}
}

// finishState excludes wall time: a refusal can spend time validating.
type finishState struct {
	Next        int
	Bytes, Peak int64
	Native      NativeCounts
	Pixels      []frame.Pixel
}

func snapshotFinish(band *bandFinisher) finishState {
	return finishState{Next: band.next, Bytes: band.bytes, Peak: band.peak,
		Native: band.native,
		Pixels: append([]frame.Pixel(nil), band.picture.Pix...)}
}

func TestReadbackRefusalsPreservePixelsAndAccounting(t *testing.T) {
	band, book := finishFixture(t)
	words := book.Width * book.Side * book.stride()
	raw := fillFinishBand(make([]float32, words), book, 0, 1)
	if err := band.consume(0, 1, raw); err != nil {
		t.Fatal(err)
	}
	before := snapshotFinish(band)
	for _, bad := range [][3]int{{0, 1, words}, {2, 1, words},
		{-1, 1, words}, {1, 0, 0}, {1, 70, 70 * words},
		{1, 1, words - 1}, {1, 1, words + 1}} {
		err := band.consume(bad[0], bad[1], make([]float32, bad[2]))
		if err == nil || !reflect.DeepEqual(snapshotFinish(band), before) {
			t.Fatal("refused band changed state", bad, err)
		}
	}
}

func TestReadbackFineSizeRefusesOverflowWithoutAllocating(t *testing.T) {
	for _, bad := range [][3]int{{-1, 1, 1}, {1, -1, 1}, {1, 1, 0},
		{1, 1, math.MaxInt32 + 1}, {math.MaxInt32, 1, 2},
		{1, math.MaxInt32, 2}, {math.MaxInt32, math.MaxInt32, 1}} {
		width, height, err := checkedFineSize(bad[0], bad[1], bad[2])
		if err == nil || width != 0 || height != 0 {
			t.Fatal("hostile sample request escaped", bad, width, height, err)
		}
	}
	for _, good := range [][5]int{{3, 2, 3, 9, 6},
		{0, 2, 3, 0, 6}, {2, 0, 3, 6, 0}} {
		width, height, err := checkedFineSize(good[0], good[1], good[2])
		if err != nil || width != good[3] || height != good[4] {
			t.Fatal("bounded or empty request changed", good, width, height, err)
		}
	}
}
