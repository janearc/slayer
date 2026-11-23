package main

import (
	"errors"
	"flag"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/cells/mocks"
	"github.com/janearc/slayer/scene/smoke"
	"github.com/janearc/slayer/scene/thing"
)

// what these tests count in, each defined
const (
	noCount  = 0  // defined: no count at all
	aFrame   = 1  // defined: one of a thing
	backward = -1 // defined: one before zero
)

// the words these tests give for a size
const (
	across = "x"
	noSize = "four by two"
)

// inPackage is the picked numbers, read for the tests in package main.
var inPackage = readPicked()

// readPicked reads the numbers file the orbital's tests share.
func readPicked() numbers.Numbers {
	raw, err := os.ReadFile("testdata/numbers.yaml")
	if err != nil {
		panic(err)
	}
	read, err := numbers.Read("testdata/numbers.yaml", raw)
	if err != nil {
		panic(err)
	}
	return read
}

// pick is the picked whole number by its name.
func pick(name string) int {
	number, err := inPackage.Get(name)
	if err != nil {
		panic(err)
	}
	return int(number.Value)
}

// smallSize is the picked size, written as -cells takes it.
func smallSize() string {
	cols := strconv.Itoa(pick("cells-cols"))
	return cols + across + strconv.Itoa(pick("cells-rows"))
}

// goodAsk is flags that ask for cells the way the orbital takes them.
func goodAsk() cellsFlags {
	return cellsFlags{size: smallSize(), set: given{},
		frames: pick("cells-frames"), rate: pick("cells-rate"),
		rays: pick("cells-rays")}
}

// With no size, cells are off. Nothing else is looked at.
func TestCellsAreOffWithNoSize(t *testing.T) {
	mode, err := askCells(cellsFlags{rate: noCount, rays: noCount})
	if err != nil || mode.on {
		t.Fatalf("no size: on %v, %v; want off and no error", mode.on, err)
	}
}

// Asked well, cells turn on with a scene and a stream to draw it.
func TestAGoodAskTurnsCellsOn(t *testing.T) {
	mode, err := askCells(goodAsk())
	if err != nil || !mode.on || mode.scene == nil || mode.stream == nil {
		t.Fatalf("a good ask: on %v, %v", mode.on, err)
	}
}

// anti-tests: cells write no file, so a flag that names one is refused.
func TestCellsRefuseAFlagThatNamesAFile(t *testing.T) {
	for _, named := range []string{outFlag, rawFlag} {
		ask := goodAsk()
		ask.set = given{named: true}
		if _, err := askCells(ask); !errors.Is(err, errFiles) {
			t.Errorf("-%s with -cells: %v; want errFiles", named, err)
		}
	}
}

// anti-tests: every ask that can't be drawn is refused before a scene is
// built.
func TestCellsRefuseWhatCannotBeDrawn(t *testing.T) {
	for _, name := range spoilable {
		if mode, err := askCells(spoiled(name)); err == nil || mode.on {
			t.Errorf("a bad %s: on %v, %v; want refused", name, mode.on,
				err)
		}
	}
}

// spoilable is each part of an ask that spoiled can make wrong.
var spoilable = []string{"size", "rate", "rays", "first", "frames"}

// spoiled is a good ask with the one part named made wrong.
func spoiled(name string) cellsFlags {
	ask := goodAsk()
	switch name {
	case "size":
		ask.size = noSize
	case "rate":
		ask.rate = noCount
	case "rays":
		ask.rays = noCount
	case "first":
		ask.first = backward
	case "frames":
		ask.frames = backward
	}
	return ask
}

// In cells mode the orbital talks on standard error, since frames own
// standard output.
func TestCellsTalkOnStandardError(t *testing.T) {
	if (cellsMode{on: true}).chat() != os.Stderr {
		t.Error("cells mode talks somewhere other than standard error")
	}
}

// anti-tests: out of cells mode the orbital talks on standard output. It
// always has.
func TestPicturesTalkOnStandardOutput(t *testing.T) {
	if (cellsMode{}).chat() != os.Stdout {
		t.Error("picture mode talks somewhere other than standard output")
	}
}

// The lens is as wide as the printing asks, with the pixel given.
func TestTheLensIsThePrintingsWidth(t *testing.T) {
	printed := printing{wide: float64(pick("cells-cols"))}
	pixel := float64(pick("cells-rows"))
	lens := lensFor(printed, pixel)
	if lens.Wide != printed.wide || lens.Pixel != pixel {
		t.Errorf("lens %+v; want wide %v, pixel %v", lens, printed.wide,
			pixel)
	}
}

// The developer exposes by the printing's stops, as two to their power.
func TestTheDeveloperExposesByStops(t *testing.T) {
	stops := float64(pick("cells-frames"))
	tone := developer(printing{stops: stops})
	if tone.Exposure != math.Exp2(stops) {
		t.Errorf("exposure %v at %v stops", tone.Exposure, stops)
	}
}

// A cells mode passes the time of each frame on to the clock it follows.
func TestTheRelayPassesTheTimeOn(t *testing.T) {
	mode, err := askCells(goodAsk())
	if err != nil {
		t.Fatal(err)
	}
	clock := mocks.NewMockClock(gomock.NewController(t))
	at := time.Duration(pick("cells-frames")) * time.Second
	clock.EXPECT().Set(at).Times(aFrame)
	mode.follow(clock)
	mode.relay.Set(at)
}

// anti-tests: before it follows a clock, a relay told the time tells no
// one and does not stop.
func TestARelayWithNoClockTellsNoOne(t *testing.T) {
	mode, err := askCells(goodAsk())
	if err != nil {
		t.Fatal(err)
	}
	frames := pick("cells-frames")
	mode.relay.Set(time.Duration(frames) * time.Second)
}

// anti-tests: -cells draws on the cpu, so -gpu with it is refused before
// a scene is built.
func TestCellsRefuseTheGPU(t *testing.T) {
	ask := goodAsk()
	ask.gpu = true
	if mode, err := askCells(ask); !errors.Is(err, errGPUCells) ||
		mode.on {
		t.Errorf("-gpu with -cells: on %v, %v; want errGPUCells",
			mode.on, err)
	}
}

// Noting a flag gives it. givenFlags gives no flag the tests didn't set.
func TestANotedFlagIsGiven(t *testing.T) {
	found := given{}
	found.note(&flag.Flag{Name: outFlag})
	if !found[outFlag] {
		t.Error("a noted flag isn't given")
	}
	if givenFlags()[outFlag] {
		t.Errorf("-%s is given, and the tests never set it", outFlag)
	}
}

// smokeScene is the picked smoke scene's world and eye.
func smokeScene(t *testing.T) (*thing.World, maths.Subspace) {
	t.Helper()
	chosen, found := smoke.Get(pick("smoke-number"))
	if !found {
		t.Fatalf("there is no smoke scene %d", pick("smoke-number"))
	}
	return chosen.Scene()
}
