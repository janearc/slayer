package take_test

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/take"
	"github.com/janearc/slayer/take/mocks"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// what the take tests count in, each defined
const (
	once      = 1           // defined: a call that happens exactly once
	fullTurn  = 2 * math.Pi // defined: a whole turn of the shutter's disc is two pi radians
	perSecond = 1000        // defined: a second is 1000 milliseconds
	midStep   = 0.5         // defined: the middle of a step is half of it
	degrees   = 180         // defined: half a turn is 180 degrees, so a degree is pi over 180
	upward    = 1           // defined: math.Inf's code for infinity above all numbers
	downward  = -1          // defined: math.Inf's code for infinity below all numbers
	either    = 0           // defined: math.IsInf's code for infinity of either sign
	nothing   = 0           // defined: no count and no length at all
	single    = 1           // defined: a count of one
	hour      = time.Hour   // defined: an hour is sixty minutes, long enough that no glimpse asked about here lies past the take
)

// the tables the take tests are run over
const (
	framesFile  = "testdata/frames.csv"
	glimpseFile = "testdata/glimpses.csv"
	pacesFile   = "testdata/paces-refused.csv"
)

// what the take tests choose, each picked in testdata/numbers.yaml
var (
	rateLeast     = pickedWhole("take-rate-least")
	rateMost      = pickedWhole("take-rate-most")
	lengthLeast   = pickedWhole("take-length-least")
	lengthMost    = pickedWhole("take-length-most")
	shutterLeast  = pickedFloat("take-shutter-least")
	glimpsesMost  = pickedWhole("take-glimpses-most")
	frameLatest   = pickedWhole("take-frame-most")
	rounding      = pickedWhole("take-rounding")
	oddMost       = pickedFloat("take-odd-most")
	oddLeast      = pickedFloat("take-odd-least")
	originMost    = pickedFloat("take-origin-most")
	stillRate     = pickedFloat("take-still-rate")
	stillGlimpses = pickedWhole("take-still-glimpses")
	stillLength   = time.Duration(pickedWhole("take-still-length")) *
		time.Millisecond
	stillShutter = pickedFloat("take-still-shutter")
)

// casesOf is the rows of a table of cases in testdata, or the test
// stops.
func casesOf(t *testing.T, name string) []data.Row {
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

// numberIn is the number in a column of a row, or the test stops.
func numberIn(t *testing.T, row data.Row, column string) float64 {
	t.Helper()
	amount, err := row.Float(column)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

// openingAt is the time of a glimpse of a frame, worked here from the
// words of the pace and not from the code. The shutter is open for its
// share of the frame. The glimpses stand evenly through it, each at the
// middle of its share.
func openingAt(rate, shutter float64, glimpses, frame,
	index int) time.Duration {
	open := shutter / fullTurn
	step := (float64(index) + midStep) / float64(glimpses)
	seconds := (float64(frame) + open*step) / rate
	nanoseconds := math.Round(seconds * float64(time.Second))
	return time.Duration(nanoseconds)
}

// anyPath is a mock path that answers every call with the same place.
func anyPath(ctrl *gomock.Controller) *mocks.MockMotion {
	path := mocks.NewMockMotion(ctrl)
	var place maths.Subspace
	path.EXPECT().At(gomock.Any()).Return(place).AnyTimes()
	return path
}

func TestAGlimpseIsCaughtAtTheMiddleOfItsShareOfTheOpenShutter(
	t *testing.T) {
	ctrl := gomock.NewController(t)
	for _, row := range casesOf(t, glimpseFile) {
		rate := numberIn(t, row, "rate")
		shutter := numberIn(t, row, "turns") * fullTurn
		glimpses := int(numberIn(t, row, "glimpses"))
		frame := int(numberIn(t, row, "frame"))
		index := int(numberIn(t, row, "index"))
		want := time.Duration(numberIn(t, row, "want_ns"))
		cranked, err := take.Cranked(rate, shutter, glimpses)
		if err != nil {
			t.Fatal(err)
		}
		made, err := take.Take(anyPath(ctrl), cranked, hour)
		if err != nil {
			t.Fatal(err)
		}
		caught := made.Glimpses(frame)
		if at := caught[index].At(); at != want {
			t.Errorf("glimpse %d of frame %d at %v a second is at %v, "+
				"want %v", index, frame, rate, at, want)
		}
	}
}

// anti-tests: a camera is not cranked if it does not turn or turns
// backwards. Nor is it if the shutter is shut or is open more than all
// the way round. Nor is it if it catches nothing.
func TestACameraIsNotCrankedWhereItCannotBe(t *testing.T) {
	for _, row := range casesOf(t, pacesFile) {
		rate := numberIn(t, row, "rate")
		shutter := numberIn(t, row, "shutter")
		glimpses := int(numberIn(t, row, "glimpses"))
		if _, err := take.Cranked(rate, shutter, glimpses); err == nil {
			label, _ := row.Text("label")
			t.Errorf("%s was cranked", label)
		}
	}
}

// oddNumber draws an ordinary number or an odd one.
func oddNumber() *rapid.Generator[float64] {
	odd := rapid.SampledFrom([]float64{math.NaN(), math.Inf(upward),
		math.Inf(downward), nothing})
	return rapid.OneOf(rapid.Float64Range(oddLeast, oddMost), odd)
}

// Whether a camera is cranked follows from its numbers. The rate is greater
// than zero and finite. The shutter is greater than zero and no more than a
// whole turn. There is one glimpse or more.
func TestACameraIsCrankedExactlyWhenItCanBe(t *testing.T) {
	rapid.Check(t, cameraIsCrankedWhenItCan)
}

func cameraIsCrankedWhenItCan(t *rapid.T) {
	rate, shutter := oddNumber().Draw(t, "rate"), oddNumber().Draw(t,
		"shutter")
	glimpses := rapid.IntRange(-glimpsesMost, glimpsesMost).Draw(t,
		"glimpses")
	want := rate > nothing && !math.IsInf(rate, either) &&
		shutter > nothing && shutter <= fullTurn && glimpses >= single
	if _, err := take.Cranked(rate, shutter, glimpses); (err == nil) !=
		want {
		t.Fatalf("a rate of %v, a shutter of %v and %d glimpses: "+
			"cranked is %v, want %v", rate, shutter, glimpses,
			err == nil, want)
	}
}

// anti-tests: a take with no path is not made. Nor is one with no pace
// or no length. The path is a mock with no expectations. So any reach
// for it fails the test.
func TestATakeNeedsAPathAPaceAndALength(t *testing.T) {
	cranked, err := take.Cranked(stillRate, fullTurn, stillGlimpses)
	if err != nil {
		t.Fatal(err)
	}
	unpaced, refusal := take.Cranked(nothing, nothing, nothing)
	if refusal == nil {
		t.Fatal("a camera that does not turn was cranked")
	}
	path := mocks.NewMockMotion(gomock.NewController(t))
	if _, err := take.Take(nil, cranked, stillLength); err == nil {
		t.Error("a take with no path was made")
	}
	if _, err := take.Take(path, unpaced, stillLength); err == nil {
		t.Error("a take with no pace was made")
	}
	for _, length := range []time.Duration{nothing, -stillLength} {
		if _, err := take.Take(path, cranked, length); err == nil {
			t.Errorf("a take of %v was made", length)
		}
	}
	if _, err := take.Take(path, cranked, time.Nanosecond); err != nil {
		t.Errorf("the shortest take there is was refused: %v", err)
	}
}

// No take at all holds no frames and has no glimpses, and does not ask
// the path anything.
func TestNoTakeHoldsNoFramesAndNoGlimpses(t *testing.T) {
	cranked, err := take.Cranked(stillRate, fullTurn, stillGlimpses)
	if err != nil {
		t.Fatal(err)
	}
	none, refusal := take.Take(nil, cranked, stillLength)
	if refusal == nil {
		t.Fatal("a take with no path was made")
	}
	if got := none.Frames(); got != nothing {
		t.Errorf("no take holds %d frames", got)
	}
	if got := none.Glimpses(single); len(got) != nothing {
		t.Errorf("no take has %d glimpses", len(got))
	}
}

// Takes hold the whole frames their length has room for. That is the
// length in nanoseconds times the rate over a second, with any part of
// a frame left off. The path is a mock with no expectations. Counting
// frames asks it nothing.
func TestATakeHoldsWholeFrames(t *testing.T) {
	path := mocks.NewMockMotion(gomock.NewController(t))
	for _, row := range casesOf(t, framesFile) {
		rate := numberIn(t, row, "rate")
		length := time.Duration(numberIn(t, row, "length_ns"))
		want := int(numberIn(t, row, "want"))
		cranked, err := take.Cranked(rate, fullTurn, single)
		if err != nil {
			t.Fatal(err)
		}
		made, err := take.Take(path, cranked, length)
		if err != nil {
			t.Fatal(err)
		}
		if got := made.Frames(); got != want {
			t.Errorf("%v at %v a second is %d frames, want %d", length,
				rate, got, want)
		}
	}
}

// Takes hold every frame they have room for at any rate. Whole frames a
// second for whole milliseconds hold the product over a thousand with
// the part of a frame left off. Float rates times lengths
// in seconds can land a hair under the whole number they mean.
func TestATakeHoldsEveryFrameItHasRoomFor(t *testing.T) {
	rapid.Check(t, takeHoldsEveryFrame)
}

func takeHoldsEveryFrame(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rate := rapid.IntRange(rateLeast, rateMost).Draw(t, "rate")
	milliseconds := rapid.IntRange(lengthLeast, lengthMost).Draw(t,
		"milliseconds")
	cranked, err := take.Cranked(float64(rate), fullTurn, single)
	if err != nil {
		t.Fatal(err)
	}
	length := time.Duration(milliseconds) * time.Millisecond
	path := mocks.NewMockMotion(ctrl)
	made, err := take.Take(path, cranked, length)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := made.Frames(), rate*milliseconds/perSecond; got !=
		want {
		t.Fatalf("%d a second for %v is %d frames, want %d", rate, length,
			got, want)
	}
}

// subspaceAt is a place for a glimpse of a mock path to be told, by the
// order of the glimpse.
func subspaceAt(index int) maths.Subspace {
	return maths.Subspace{Origin: maths.V(float64(index), nothing, nothing)}
}

// Takes ask the path where the camera is at each instant the shutter is open.
// They ask once. Each glimpse gets the time asked for and the place told.
// Glimpses are in order. Frame number zero is the first.
func TestGlimpsesAskThePathForTheCameraAtEachInstant(t *testing.T) {
	rapid.Check(t, glimpsesAskThePath)
}

// drawnPace is a pace, a frame and the number of glimpses a take could
// be asked for.
func drawnPace(t *rapid.T) (rate, shutter float64, many, frame int) {
	least, most := float64(rateLeast), float64(rateMost)
	rate = rapid.Float64Range(least, most).Draw(t, "rate")
	shutter = rapid.Float64Range(shutterLeast, fullTurn).Draw(t, "shutter")
	many = rapid.IntRange(single, glimpsesMost).Draw(t, "glimpses")
	frame = rapid.IntRange(nothing, frameLatest).Draw(t, "frame")
	return rate, shutter, many, frame
}

func glimpsesAskThePath(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rate, shutter, many, frame := drawnPace(t)
	path := mocks.NewMockMotion(ctrl)
	var times []time.Duration
	for index := range many {
		at := openingAt(rate, shutter, many, frame, index)
		path.EXPECT().At(at).Return(subspaceAt(index)).Times(once)
		times = append(times, at)
	}
	cranked, err := take.Cranked(rate, shutter, many)
	if err != nil {
		t.Fatal(err)
	}
	made, err := take.Take(path, cranked, hour)
	if err != nil {
		t.Fatal(err)
	}
	caught := made.Glimpses(frame)
	if len(caught) != many {
		t.Fatalf("frame %d has %d glimpses, want %d", frame, len(caught),
			many)
	}
	for index, each := range caught {
		if each.At() != times[index] || each.Eye() != subspaceAt(index) {
			t.Fatalf("glimpse %d is at %v from %v, want %v from %v",
				index, each.At(), each.Eye(), times[index],
				subspaceAt(index))
		}
	}
}

// The glimpses of a frame fall evenly through the time its shutter is
// open, and no earlier than the frame opens nor later than it closes.
// The times are rounded to a nanosecond, so the gaps are even to a
// couple of them.
func TestGlimpsesFallEvenlyThroughTheOpenShutter(t *testing.T) {
	rapid.Check(t, glimpsesFallEvenly)
}

func glimpsesFallEvenly(t *rapid.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rate, shutter, many, frame := drawnPace(t)
	cranked, err := take.Cranked(rate, shutter, many)
	if err != nil {
		t.Fatal(err)
	}
	made, err := take.Take(anyPath(ctrl), cranked, hour)
	if err != nil {
		t.Fatal(err)
	}
	open := shutter / fullTurn
	second := float64(time.Second)
	opens := time.Duration(float64(frame) / rate * second)
	closes := time.Duration((float64(frame) + open) / rate * second)
	slack := time.Duration(rounding) * time.Nanosecond
	gap := time.Duration(open / rate / float64(many) * second)
	caught := made.Glimpses(frame)
	for index, each := range caught {
		if each.At() < opens-slack || each.At() > closes+slack {
			t.Fatalf("glimpse %d at %v is outside %v to %v", index,
				each.At(), opens, closes)
		}
		if index > nothing {
			apart := each.At() - caught[index-single].At()
			if apart-gap > slack || gap-apart > slack {
				t.Fatalf("glimpses %d and %d are %v apart, want %v",
					index-single, index, apart, gap)
			}
		}
	}
}

// Takes along one of slayer's own motions put the camera where the motion
// puts it at every glimpse. The glimpses come in time.
func TestATakeAlongSlayersMotion(t *testing.T) {
	rapid.Check(t, takeAlongStillMotion)
}

func takeAlongStillMotion(t *rapid.T) {
	coordinate := rapid.Float64Range(-originMost, originMost)
	origin := maths.V(coordinate.Draw(t, "x"), coordinate.Draw(t, "y"),
		coordinate.Draw(t, "z"))
	place := maths.Subspace{Origin: origin}
	shutter := stillShutter * math.Pi / degrees
	steady, err := take.Cranked(stillRate, shutter, stillGlimpses)
	if err != nil {
		t.Fatal(err)
	}
	made, err := take.Take(thing.Still(place), steady, stillLength)
	if err != nil {
		t.Fatal(err)
	}
	room := math.Floor(stillRate * stillLength.Seconds())
	if want := int(room); made.Frames() != want {
		t.Fatalf("a take of %v at %v a second holds %d frames, want %d",
			stillLength, stillRate, made.Frames(), want)
	}
	last := -time.Nanosecond
	for number := range made.Frames() {
		for _, each := range made.Glimpses(number) {
			if each.Eye() != place || each.At() <= last ||
				each.At() >= stillLength {
				t.Fatalf("frame %d: a glimpse at %v from %v", number,
					each.At(), each.Eye())
			}
			last = each.At()
		}
	}
}
