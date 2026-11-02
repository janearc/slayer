package terminal_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/interfaces/terminal/mocks"
	"go.uber.org/mock/gomock"
)

// what the stream tests count in, each defined
const (
	twice      = 2   // defined: a call or a frame that happens two times
	cellAcross = 2   // defined: a sextant cell is two pixels across
	cellDown   = 3   // defined: a sextant cell is three pixels down
	full       = 255 // defined: the largest value of an 8-bit colour channel
)

// update rewrites the golden stream from what Run writes now. No gate
// passes it: a golden is moved by hand, by a change that meant to move it.
var update = flag.Bool("update", false,
	"rewrite testdata/stream.golden from the stream drawn now")

// what the test pictures change by, each picked in testdata/numbers.yaml
var (
	stepAcross   = picked("step-across")
	stepDown     = picked("step-down")
	stepFrame    = picked("step-frame")
	stepDiagonal = picked("step-diagonal")
)

// shape is a stream under test: where it starts, how many frames it
// runs and how many cells it is across and down.
type shape struct {
	first, count, cols, rows int
}

// standard is the stream the golden holds.
func standard() shape {
	return shape{first: picked("stream-first"),
		count: picked("stream-frames"), cols: picked("stream-cols"),
		rows: picked("stream-rows")}
}

// wide is how many pixels across the stream's frames are.
func (asked shape) wide() int { return asked.cols * cellAcross }

// high is how many pixels down the stream's frames are.
func (asked shape) high() int { return asked.rows * cellDown }

// written is the size as the command line writes it.
func (asked shape) written() string {
	return fmt.Sprintf("%dx%d", asked.cols, asked.rows)
}

// tester is what the helpers need of a test. Both testing.T and
// rapid.T are one.
type tester interface {
	gomock.TestReporter
	Helper()
	Fatal(args ...any)
}

// sizeOf is the shape's size, or the test stops.
func sizeOf(t tester, asked shape) terminal.Size {
	t.Helper()
	size, err := terminal.ParseSize(asked.written())
	if err != nil {
		t.Fatal(err)
	}
	return size
}

// drawnFrame is a picture that differs at every pixel and every frame.
// The wrong frame number shows in the bytes. So does a pixel in the
// wrong cell.
func drawnFrame(frame, wide, high int) (image.Image, error) {
	picture := canvas(wide, high)
	for down := range high {
		for across := range wide {
			picture.SetRGBA(across, down, pixelAt(frame, across, down))
		}
	}
	return picture, nil
}

// canvas is a blank picture wide by high pixels, from the origin.
func canvas(wide, high int) *image.RGBA {
	box := image.Rect(nothing, nothing, wide, high)
	return image.NewRGBA(box)
}

// pixelAt is one pixel of drawnFrame.
func pixelAt(frame, across, down int) color.RGBA {
	return color.RGBA{
		R: uint8(across*stepAcross + frame*stepFrame),
		G: uint8(down * stepDown),
		B: uint8((across + down + frame) * stepDiagonal),
		A: full}
}

// expectFrames expects the source to be asked once for each frame of
// the stream. The order is the stream's and the size is its pixels.
// Each is drawn with drawnFrame. It gives the last call so that a test
// can put more after it.
func expectFrames(source *mocks.MockSource, asked shape) *gomock.Call {
	var before *gomock.Call
	for number := asked.first; number < asked.first+asked.count; number++ {
		call := source.EXPECT().Frame(number, asked.wide(),
			asked.high()).DoAndReturn(drawnFrame)
		if before != nil {
			call.After(before)
		}
		before = call
	}
	return before
}

// run writes the stream of asked, from a source that draws drawnFrame.
func run(t tester, asked shape) string {
	t.Helper()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	source := mocks.NewMockSource(ctrl)
	expectFrames(source, asked)
	stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
		asked.count, source)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := stream.Run(&out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// drawn is what the stream of asked should write, worked from the
// frames by Chop and Draw.
func drawn(asked shape) string {
	var want bytes.Buffer
	for number := asked.first; number < asked.first+asked.count; number++ {
		picture, _ := drawnFrame(number, asked.wide(), asked.high())
		terminal.Draw(&want, terminal.Chop(picture))
	}
	return want.String()
}

// Columns by rows is that many cells. The source is asked for each
// frame at two pixels across a cell and three down.
func TestASizeIsColumnsByRowsOfCells(t *testing.T) {
	for _, row := range casesOf(t, sizesRead) {
		size, err := terminal.ParseSize(textIn(t, row, "written"))
		if err != nil {
			t.Fatal(err)
		}
		source := mocks.NewMockSource(gomock.NewController(t))
		wide, high := wholeIn(t, row, "wide"), wholeIn(t, row, "high")
		blank := canvas(wide, high)
		source.EXPECT().Frame(nothing, wide, high).Return(blank, nil)
		stream, err := terminal.NewStream(size, nothing, once, source)
		if err != nil {
			t.Fatal(err)
		}
		if err := stream.Run(io.Discard); err != nil {
			t.Errorf("%s: %v", textIn(t, row, "written"), err)
		}
	}
}

// The smallest and largest sizes are read.
func TestTheSmallestAndLargestSizesAreRead(t *testing.T) {
	for _, row := range casesOf(t, sizesEdge) {
		written := textIn(t, row, "written")
		if _, err := terminal.ParseSize(written); err != nil {
			t.Errorf("%s: %v", written, err)
		}
	}
}

// Only two whole numbers from one to the most joined by x make a size.
// Anything else is refused. The refusal says it is not a size and names
// the side that was wrong.
func TestWhatIsNotASizeIsRefused(t *testing.T) {
	for _, row := range casesOf(t, sizesRefused) {
		written, side := textIn(t, row, "written"), textIn(t, row, "side")
		_, err := terminal.ParseSize(written)
		if err == nil {
			t.Errorf("%q was read as a size", written)
			continue
		}
		says := err.Error()
		if !strings.Contains(says, "not a size") {
			t.Errorf("%q: %v; want it said not to be a size", written, err)
		}
		if side != "either" && !strings.Contains(says, side) {
			t.Errorf("%q: %v; want %s named", written, err, side)
		}
	}
}

// Each row of the table is a stream that cannot run. It has no size or
// a negative start or count. It runs past the last number or has no
// source. All are refused.
func TestAStreamIsRefusedWhatItCannotRun(t *testing.T) {
	source := mocks.NewMockSource(gomock.NewController(t))
	for _, row := range casesOf(t, streamsBad) {
		var size terminal.Size
		if written := textIn(t, row, "size"); written != "" {
			var err error
			if size, err = terminal.ParseSize(written); err != nil {
				t.Fatal(err)
			}
		}
		var given terminal.Source
		if textIn(t, row, "source") == yes {
			given = source
		}
		stream, err := terminal.NewStream(size, wholeIn(t, row, "first"),
			wholeIn(t, row, "frames"), given)
		if err == nil || stream != nil {
			t.Errorf("row %v gave %v, %v; want a refusal", row, stream, err)
		}
	}
}

// Nought frames is a stream with no end. Any count up to a few makes a
// stream too.
func TestAStreamIsMadeFromWhatItNeeds(t *testing.T) {
	asked := standard()
	source := mocks.NewMockSource(gomock.NewController(t))
	for frames := nothing; frames <= asked.count; frames++ {
		stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
			frames, source)
		if err != nil || stream == nil {
			t.Errorf("%d frames: %v, %v", frames, stream, err)
		}
	}
}

// The stream is each frame from the source in order through Chop and
// Draw. The source is asked for the first frame on as many as counted.
func TestAStreamIsEachFrameDrawnInOrder(t *testing.T) {
	asked := standard()
	got, want := run(t, asked), drawn(asked)
	if got != want {
		t.Errorf("the stream is not each frame, Chop and Draw: got %d "+
			"bytes, want %d", len(got), len(want))
	}
}

// readable is who may read the golden file. It is defined as the owner
// writing and everyone reading.
const readable os.FileMode = 0o644

// The stream is held byte for byte to the golden. go test -update
// rewrites the golden.
func TestTheStreamIsHeldToItsGolden(t *testing.T) {
	got := run(t, standard())
	if *update {
		if err := os.WriteFile(goldenFile, []byte(got), readable); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("the stream is not its golden: got %d bytes, want %d; "+
			"if the change meant to move it, go test -update", len(got),
			len(want))
	}
}

// solid is a picture of one colour.
func solid(asked shape, fill color.RGBA) image.Image {
	picture := canvas(asked.wide(), asked.high())
	draw.Draw(picture, picture.Bounds(), image.NewUniform(fill),
		image.Point{}, draw.Src)
	return picture
}

// One solid cell is these bytes, spelled out here and not by Draw.
// The frame begins and goes to row one. Both colours are set. Then a
// space, a reset and the end. Red first and then blue.
func TestASolidCellIsTheStandardsBytes(t *testing.T) {
	asked := shape{first: nothing, count: twice, cols: once, rows: once}
	red := color.RGBA{R: full, A: full}
	blue := color.RGBA{B: full, A: full}
	source := mocks.NewMockSource(gomock.NewController(t))
	redFrame := source.EXPECT().Frame(nothing, asked.wide(), asked.high()).
		Return(solid(asked, red), nil)
	source.EXPECT().Frame(once, asked.wide(), asked.high()).
		Return(solid(asked, blue), nil).After(redFrame)
	stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
		asked.count, source)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := stream.Run(&out); err != nil {
		t.Fatal(err)
	}
	redCell := rgb{red: full}
	blueCell := rgb{blue: full}
	want := frameBegin + place(firstRow) + colours(redCell, redCell) +
		" " + reset + frameEnd + frameBegin + place(firstRow) +
		colours(blueCell, blueCell) + " " + reset + frameEnd
	if out.String() != want {
		t.Errorf("got %q\nwant %q", out.String(), want)
	}
}

// The stream is plain ANSI and nothing else. Each frame is synchronised
// output with a move to each row. Colours, sextants and resets fill it.
func TestTheStreamIsPlainANSI(t *testing.T) {
	asked := standard()
	got := run(t, asked)
	if err := plain(got, asked.rows); err != nil {
		t.Error(err)
	}
	if begun := strings.Count(got, frameBegin); begun != asked.count {
		t.Errorf("%d frames begin; want %d", begun, asked.count)
	}
}

// Failing source: the stream stops at that frame and says which. The
// error is the source's own. Earlier frames were written whole. Nothing
// was written for the failed one.
func TestASourceThatFailsStopsTheStream(t *testing.T) {
	broken := errors.New("the scene would not draw")
	asked := standard()
	source := mocks.NewMockSource(gomock.NewController(t))
	good := source.EXPECT().Frame(asked.first, asked.wide(), asked.high()).
		DoAndReturn(drawnFrame)
	source.EXPECT().Frame(asked.first+once, asked.wide(), asked.high()).
		Return(nil, broken).After(good)
	stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
		nothing, source)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = stream.Run(&out)
	says := fmt.Sprintf("frame %d", asked.first+once)
	if !errors.Is(err, broken) || !strings.Contains(err.Error(), says) {
		t.Errorf("got %v; want %s and the source's error", err, says)
	}
	if written := strings.Count(out.String(), frameBegin); written != once {
		t.Errorf("%d frames written; want one", written)
	}
}

// Nothing drawn stops the stream. So does a picture of another size.
// Neither writes anything.
func TestASourceThatDrawsTheWrongThingStopsTheStream(t *testing.T) {
	asked := standard()
	for _, row := range casesOf(t, picturesBad) {
		var wrong image.Image
		if textIn(t, row, "drawn") == yes {
			wide := asked.wide() + wholeIn(t, row, "wide_off")
			high := asked.high() + wholeIn(t, row, "high_off")
			wrong = canvas(wide, high)
		}
		source := mocks.NewMockSource(gomock.NewController(t))
		source.EXPECT().Frame(asked.first, asked.wide(), asked.high()).
			Return(wrong, nil)
		stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
			asked.count, source)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		err = stream.Run(&out)
		if err == nil || out.Len() != nothing {
			t.Errorf("%s: got %v and %d bytes; want a refusal and none",
				textIn(t, row, "name"), err, out.Len())
		}
	}
}

// cropped is a picture of the shape cut from a bigger one, so it does
// not start at the origin.
func cropped(asked shape) image.Image {
	left, top := picked("crop-left"), picked("crop-top")
	right, bottom := left+asked.wide(), top+asked.high()
	bigger := canvas(right, bottom)
	for down := range bottom {
		for across := range right {
			bigger.SetRGBA(across, down, pixelAt(nothing, across, down))
		}
	}
	piece := image.Rect(left, top, right, bottom)
	return bigger.SubImage(piece)
}

// Cropped pictures do not start at the origin. This one is the right
// size and is drawn from its own corner.
func TestAPictureNotAtTheOriginIsDrawnFromItsCorner(t *testing.T) {
	asked := shape{first: nothing, count: once, cols: picked("stream-cols"),
		rows: picked("stream-rows")}
	source := mocks.NewMockSource(gomock.NewController(t))
	source.EXPECT().Frame(nothing, asked.wide(), asked.high()).
		Return(cropped(asked), nil)
	stream, err := terminal.NewStream(sizeOf(t, asked), nothing, once, source)
	if err != nil {
		t.Fatal(err)
	}
	var out, want bytes.Buffer
	if err := stream.Run(&out); err != nil {
		t.Fatal(err)
	}
	piece := cropped(asked)
	terminal.Draw(&want, terminal.Chop(piece))
	if out.String() != want.String() {
		t.Error("a picture off the origin was not drawn from its corner")
	}
}

// accepted is a write that takes everything it is given.
func accepted(given []byte) (int, error) { return len(given), nil }

// Failing write: the stream stops at once. The source is not asked for
// another frame. The error is the writer's and names the frame.
func TestAWriteThatFailsStopsTheStream(t *testing.T) {
	broken := errors.New("the window closed")
	asked := standard()
	ctrl := gomock.NewController(t)
	writer := mocks.NewMockWriter(ctrl)
	writer.EXPECT().Write(gomock.Any()).Return(nothing, broken).Times(once)
	source := mocks.NewMockSource(ctrl)
	source.EXPECT().Frame(asked.first, asked.wide(), asked.high()).
		DoAndReturn(drawnFrame).Times(once)
	stream, err := terminal.NewStream(sizeOf(t, asked), asked.first,
		asked.count, source)
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Run(writer)
	if !errors.Is(err, broken) || !strings.Contains(err.Error(), "writing it") {
		t.Errorf("got %v; want the writer's error, on writing", err)
	}
}

// Endless stream: it runs until a write fails. The writer takes two
// frames and refuses the third. The source is asked for three.
func TestAStreamWithNoEndStopsWhenTheWriterDoes(t *testing.T) {
	broken := errors.New("the window closed")
	asked := standard()
	asked.count = twice + once
	ctrl := gomock.NewController(t)
	writer := mocks.NewMockWriter(ctrl)
	taken := writer.EXPECT().Write(gomock.Any()).DoAndReturn(accepted).
		Times(twice)
	writer.EXPECT().Write(gomock.Any()).Return(nothing, broken).
		Times(once).After(taken)
	source := mocks.NewMockSource(ctrl)
	expectFrames(source, asked)
	stream, err := terminal.NewStream(sizeOf(t, asked), asked.first, nothing,
		source)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Run(writer); !errors.Is(err, broken) {
		t.Errorf("got %v; want the writer's error", err)
	}
}
