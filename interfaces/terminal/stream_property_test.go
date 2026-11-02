package terminal_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// aSide is any number of cells a size may have on a side.
func aSide(least, most int) *rapid.Generator[int] {
	return rapid.IntRange(least, most)
}

// anyShape is any stream the property tests run. Its size, start and
// count are each within the numbers picked.
func anyShape(test *rapid.T) shape {
	return shape{
		first: aSide(nothing, picked("stream-first-most")).Draw(test, "first"),
		count: aSide(once, picked("stream-frames-most")).Draw(test, "count"),
		cols:  aSide(once, picked("stream-cols-most")).Draw(test, "cols"),
		rows:  aSide(once, picked("stream-rows-most")).Draw(test, "rows")}
}

// Any size in range is read.
func TestAnySizeInRangeIsRead(t *testing.T) {
	rapid.Check(t, sizeInRangeIsRead)
}

func sizeInRangeIsRead(test *rapid.T) {
	most := picked("side-most")
	asked := shape{cols: aSide(once, most).Draw(test, "cols"),
		rows: aSide(once, most).Draw(test, "rows")}
	if _, err := terminal.ParseSize(asked.written()); err != nil {
		test.Fatalf("%s: %v", asked.written(), err)
	}
}

// Any size with a side out of range is refused, whichever side it is.
func TestAnySizeOutOfRangeIsRefused(t *testing.T) {
	rapid.Check(t, sizeOutOfRangeIsRefused)
}

func sizeOutOfRangeIsRefused(test *rapid.T) {
	most := picked("side-most")
	bad := rapid.OneOf(aSide(-most, nothing),
		aSide(most+once, most*twice)).Draw(test, "bad")
	good := aSide(once, most).Draw(test, "good")
	refuse(test, shape{cols: bad, rows: good})
	refuse(test, shape{cols: good, rows: bad})
}

// refuse fails the property if the shape is read as a size.
func refuse(test *rapid.T, asked shape) {
	if size, err := terminal.ParseSize(asked.written()); err == nil {
		test.Fatalf("%s was read as %v", asked.written(), size)
	}
}

// For any size, start and count the stream is as many frames as
// counted. The source is asked for each number in order at the size's
// pixels. The bytes are the frames through Chop and Draw, and plain.
func TestAnyStreamIsTheFramesCountedAndPlain(t *testing.T) {
	rapid.Check(t, streamIsCountedAndPlain)
}

func streamIsCountedAndPlain(test *rapid.T) {
	asked := anyShape(test)
	got := run(test, asked)
	if got != drawn(asked) {
		test.Fatalf("%+v: the bytes are not the frames drawn", asked)
	}
	if err := plain(got, asked.rows); err != nil {
		test.Fatalf("%+v: %v", asked, err)
	}
	if begun := strings.Count(got, frameBegin); begun != asked.count {
		test.Fatalf("%+v: %d frames written", asked, begun)
	}
}

// For any size and any frame the source fails at, the stream stops there
// with the source's error and wrote exactly the frames before it.
func TestAnyFailingSourceStopsTheStreamAtItsFrame(t *testing.T) {
	rapid.Check(t, failingSourceStopsTheStream)
}

func failingSourceStopsTheStream(test *rapid.T) {
	asked := anyShape(test)
	broken := errors.New("the scene would not draw")
	before := asked
	before.count = aSide(nothing, asked.count).Draw(test, "before")
	ctrl := gomock.NewController(test)
	defer ctrl.Finish()
	source := mocks.NewMockSource(ctrl)
	last := expectFrames(source, before)
	failing := source.EXPECT().Frame(asked.first+before.count, asked.wide(),
		asked.high()).Return(nil, broken)
	if last != nil {
		failing.After(last)
	}
	stream, err := terminal.NewStream(sizeOf(test, asked), asked.first,
		nothing, source)
	if err != nil {
		test.Fatal(err)
	}
	var out bytes.Buffer
	err = stream.Run(&out)
	begun := strings.Count(out.String(), frameBegin)
	if !errors.Is(err, broken) || begun != before.count {
		test.Fatalf("%+v: %d frames written, %v", asked, begun, err)
	}
}
