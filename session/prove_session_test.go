package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/janearc/slayer/fermat"
	"github.com/janearc/slayer/raw"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/trace"
	"go.uber.org/mock/gomock"
)

// proveWidowSet is widow, a set of a world and no ground, which stages
// with no shelf of sources.
const proveWidowSet = "../sets/widow.yaml"

// The numbers the proof tests pick.
const (
	// proveShutterStep is the time a one glimpse shutter is open
	// before its glimpse: five milliseconds, picked.
	proveShutterStep = 5 * time.Millisecond
	// proveRawWide and proveRawTall are the little raw the sheet tests
	// write, two by two pixels.
	proveRawWide, proveRawTall = 2, 2
	// proveRed, proveGreen and proveBlue are the light of every pixel of
	// that raw, picked; each is exact in a float32.
	proveRed, proveGreen, proveBlue = 0.25, 0.5, 0.75
)

// widowSession is a session of widow, a still on the cpu at the
// session tests' size, opened with a cache of its own.
func widowSession(t *testing.T) *Session {
	t.Helper()
	return widowSessionAt(t, sessionWidth, sessionHeight)
}

// widowSessionAt is a session of widow, a still on the cpu, width by
// height pixels at the session tests' rays, opened with a cache of its
// own.
func widowSessionAt(t *testing.T, width, height int) *Session {
	t.Helper()
	t.Setenv("SLAYER_CACHE", t.TempDir())
	setPath, err := filepath.Abs(proveWidowSet)
	if err != nil {
		t.Fatal(err)
	}
	session, err := Open(context.Background(), jobFrom(t,
		fmt.Sprintf("scene: {set: %s}\nrenderer: cpu\nfilm: {width: %d, "+
			"height: %d, rays: %d}\n", setPath, width, height,
			sessionRays)), Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// a proof of nothing is refused: no pixel named and none picked, or a
// count of picks below nought.
// anti-test of TestAProofOfSomethingIsNotRefusedAsNothing
func TestAProofOfNothingIsRefused(t *testing.T) {
	session := &Session{times: []time.Duration{0}}
	corner := [][2]int{{0, 0}}
	for name, proving := range map[string]Proving{
		"none named, none picked": {Raw: "a.pfm"},
		"picks below nought":      {Raw: "a.pfm", Many: -1},
		"below nought and named": {Raw: "a.pfm", Many: -1,
			Pixels: corner},
	} {
		if err := session.provable(0, proving); !errors.Is(err,
			errProofOfNothing) {
			t.Errorf("%s: %v, not %v", name, err, errProofOfNothing)
		}
	}
}

// a pixel named with no picks is a proof, and so are picks with no
// pixel named: the proof of nothing is not said when its cause is
// absent.
// anti-test of TestAProofOfNothingIsRefused
func TestAProofOfSomethingIsNotRefusedAsNothing(t *testing.T) {
	session := &Session{times: []time.Duration{0}}
	corner := [][2]int{{0, 0}}
	for name, proving := range map[string]Proving{
		"a pixel named, none picked": {Raw: "a.pfm", Pixels: corner},
		"one picked":                 {Raw: "a.pfm", Many: 1},
		"the usual picks":            {Raw: "a.pfm", Many: fermat.Many},
	} {
		if err := session.provable(0, proving); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
}

// a session that draws on the gpu is refused: fermat holds the cpu's
// exact light and the gpu's is an approximation of it.
// anti-test of TestAProofOnTheCpuIsNotRefusedForTheGpu
func TestAProofOnTheGPUIsRefused(t *testing.T) {
	session := &Session{given: asked{onGPU: true},
		times: []time.Duration{0}}
	err := session.provable(0, Proving{Raw: "a.pfm", Many: 1})
	if !errors.Is(err, errProveOnGPU) {
		t.Errorf("a session on the gpu: %v, not %v", err, errProveOnGPU)
	}
}

// a session on the cpu is not refused for the gpu, and a frame that is
// not the job's, or a closed session, is refused as Frame refuses them.
// anti-test of TestAProofOnTheGPUIsRefused
func TestAProofOnTheCpuIsNotRefusedForTheGpu(t *testing.T) {
	session := &Session{times: []time.Duration{0}}
	if err := session.provable(0, Proving{Raw: "a.pfm",
		Many: 1}); err != nil {
		t.Errorf("a session on the cpu: %v", err)
	}
	if err := session.provable(1, Proving{Raw: "a.pfm", Many: 1}); err ==
		nil {
		t.Error("frame 1 of a job of one frame was not refused")
	}
	session.closed = true
	if err := session.provable(0, Proving{Raw: "a.pfm",
		Many: 1}); !errors.Is(err, errClosed) {
		t.Errorf("a closed session: %v, not %v", err, errClosed)
	}
}

// a camera that starts in an explicit stack of media is refused, by the
// words the command line's proof says.
// anti-test of TestAProofOfAnAirCameraIsNotRefusedItsMedia
func TestAProofOfAnExplicitStartStackIsRefused(t *testing.T) {
	err := replayRefusal(asked{initialMedia: []trace.InitialMedium{
		{Index: 1.33}}})
	if !errors.Is(err, errReplayMedia) {
		t.Errorf("an explicit stack: %v, not %v", err, errReplayMedia)
	}
}

// a camera in air, which is no stack, is not refused.
// anti-test of TestAProofOfAnExplicitStartStackIsRefused
func TestAProofOfAnAirCameraIsNotRefusedItsMedia(t *testing.T) {
	if err := replayRefusal(asked{}); err != nil {
		t.Errorf("a camera in air: %v", err)
	}
}

// a session whose source stages no world to replay is refused by name,
// and the proof is made of nothing.
// anti-test of TestAProofOfAStillStagesItsWorld
func TestAProofOfFramesWithNoWorldIsRefused(t *testing.T) {
	control := gomock.NewController(t)
	session := &Session{source: NewMockframer(control),
		times: []time.Duration{0}}
	_, err := session.Prove(0, Proving{Raw: "a.pfm", Many: 1})
	if !errors.Is(err, errProveSource) {
		t.Errorf("a source with no world: %v, not %v", err, errProveSource)
	}
}

// a still stages its world: the frame is the job's own, from the eye the
// set gives and at the job's time, which a job with no time says is
// nought; and a shutter of one glimpse, open for five milliseconds
// before its glimpse, stages the world at that glimpse.
// anti-test of TestAProofOfFramesWithNoWorldIsRefused
func TestAProofOfAStillStagesItsWorld(t *testing.T) {
	session := widowSession(t)
	given, err := session.asking(Ask{})
	if err != nil {
		t.Fatal(err)
	}
	staged, err := session.replayable(0, given, "a.pfm")
	if err != nil {
		t.Fatal(err)
	}
	if staged.world == nil || staged.given.eye == nil ||
		staged.given.at != 0 {
		t.Errorf("staged world %v, eye %v, at %v", staged.world,
			staged.given.eye, staged.given.at)
	}
	session.offsets = []time.Duration{proveShutterStep}
	if staged, err = session.replayable(0, given, "a.pfm"); err != nil ||
		staged.given.at != proveShutterStep {
		t.Errorf("a one glimpse shutter staged at %v (%v), not %v",
			staged.given.at, err, proveShutterStep)
	}
}

// a frame that is the mean of two glimpses is refused, by name.
// anti-test of TestAProofOfAStillStagesItsWorld, whose one glimpse is not
func TestAProofOfAMeanOfGlimpsesIsRefused(t *testing.T) {
	session := widowSession(t)
	session.offsets = []time.Duration{0, proveShutterStep}
	given, err := session.asking(Ask{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = session.replayable(0, given, "a.pfm"); !errors.Is(err,
		errProveShutter) {
		t.Errorf("two glimpses: %v, not %v", err, errProveShutter)
	}
}

// the sheet goes in the folder asked, and where none is asked, beside
// the raw, named for it, with the proof's pages and the raw marked; a
// folder asked for is not the one beside the raw.
func TestTheProofSheetGoesWhereAskedOrBesideTheRaw(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "hill.pfm")
	small := frame.New(proveRawWide, proveRawTall)
	for place := range small.Pix {
		small.Pix[place] = frame.Pixel{R: proveRed, G: proveGreen,
			B: proveBlue}
	}
	if err := raw.Write(path, small); err != nil {
		t.Fatal(err)
	}
	proof, err := fermat.Prove(path, "asked", [][2]int{{0, 0}}, 0,
		func(int, int) (frame.Pixel, []trace.Step, error) {
			return frame.Pixel{R: proveRed, G: proveGreen, B: proveBlue},
				nil, nil
		})
	if err != nil || !proof.Holds() {
		t.Fatalf("the proof: %v, %v", proof, err)
	}
	session := &Session{}
	beside, err := session.Sheet(proof, "")
	if err != nil || beside != filepath.Join(directory, "hill-proof") {
		t.Fatalf("the sheet beside the raw: %q, %v", beside, err)
	}
	elsewhere := filepath.Join(directory, "elsewhere")
	where, err := session.Sheet(proof, elsewhere)
	if err != nil || where != elsewhere {
		t.Fatalf("the sheet asked for: %q, %v", where, err)
	}
	for _, each := range []string{beside, elsewhere} {
		for _, name := range []string{"proof.md", "pixel-0-0.md",
			"marked.png"} {
			if _, err := os.Stat(filepath.Join(each, name)); err != nil {
				t.Error(err)
			}
		}
	}
}
