//go:build draw

package hillside

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/frame"
	"github.com/janearc/slayer/render/progress"
	milestone "github.com/janearc/slayer/scene/smoke"
	"go.uber.org/mock/gomock"
)

// mockSession is a session of two frames whose source is a mock, staged
// partsWidth by partsHeight.
func mockSession(t *testing.T) (*Session, *Mockframer) {
	t.Helper()
	source := NewMockframer(gomock.NewController(t))
	given := asked{width: partsWidth, height: partsHeight,
		samples: sessionRays}
	return &Session{source: source, given: given,
		times: []time.Duration{nothing, partsSecond}}, source
}

// mockDrawn is what a mock source gives: a frame the size it was asked.
func mockDrawn(at time.Duration, given asked) (drawnLight, error) {
	given.at = at
	return drawnLight{drawn: frame.New(given.width, given.height),
		given: given}, nil
}

// a cancelled context stops before the next frame: the source is not
// asked to draw, and the session draws again once asked with a context
// that is not done.
func TestSessionCancelledContextStopsBeforeTheNextFrame(t *testing.T) {
	session, source := mockSession(t)
	source.EXPECT().light(time.Duration(nothing), gomock.Any()).
		DoAndReturn(mockDrawn).Times(1)
	if _, err := session.Frame(context.Background(), nothing,
		Ask{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := session.Frame(ctx, partsOnce, Ask{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("a cancelled context gave %v", err)
	}
	source.EXPECT().light(partsSecond, gomock.Any()).
		DoAndReturn(mockDrawn).Times(1)
	drawn, err := session.Frame(context.Background(), partsOnce, Ask{})
	if err != nil || drawn.At() != partsSecond {
		t.Errorf("the session did not draw again after a cancel: %v", err)
	}
}

// Open with a cancelled context stages nothing.
func TestOpenWithACancelledContextStagesNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	session, err := Open(ctx, stillOf(t, sessionWidow, "", ""), Stamp{})
	if !errors.Is(err, context.Canceled) || session != nil {
		t.Errorf("a cancelled open gave %v", err)
	}
}

// an ask's size and rays stand over the staged ones, and a zero leaves
// them; the sink is told once a frame, of the whole frame.
func TestSessionAskStandsOverWhatWasStaged(t *testing.T) {
	session, source := mockSession(t)
	var bands []progress.Band
	ask, err := NewAsk(partsAskedWidth, nothing, nothing,
		func(band progress.Band) { bands = append(bands, band) })
	if err != nil {
		t.Fatal(err)
	}
	asking := gomock.Cond(func(given asked) bool {
		return given.width == partsAskedWidth &&
			given.height == partsHeight && given.samples == sessionRays
	})
	source.EXPECT().light(time.Duration(nothing), asking).
		DoAndReturn(mockDrawn).Times(1)
	if _, err := session.Frame(context.Background(), nothing,
		ask); err != nil {
		t.Fatal(err)
	}
	whole, _ := progress.NewBand(partsAskedWidth, partsHeight, nothing,
		partsHeight)
	if len(bands) != partsOnce || bands[nothing] != whole {
		t.Errorf("the sink was told %v, not the whole frame once", bands)
	}
}

// a session with no sink tells nothing, and draws.
func TestSessionWithNoSinkTellsNothing(t *testing.T) {
	session, source := mockSession(t)
	source.EXPECT().light(gomock.Any(), gomock.Any()).
		DoAndReturn(mockDrawn).Times(1)
	if _, err := session.Frame(context.Background(), nothing,
		Ask{}); err != nil {
		t.Error(err)
	}
}

// a frame the source cannot draw gives the source's error.
func TestSessionGivesTheSourcesError(t *testing.T) {
	session, source := mockSession(t)
	failed := errors.New("the source failed")
	source.EXPECT().light(gomock.Any(), gomock.Any()).
		Return(drawnLight{}, failed).Times(1)
	if _, err := session.Frame(context.Background(), nothing,
		Ask{}); !errors.Is(err, failed) {
		t.Errorf("a failed source gave %v", err)
	}
}

// a frame the job has not is refused, and the source is not asked.
func TestSessionRefusesAFrameTheJobHasNot(t *testing.T) {
	session, _ := mockSession(t)
	for _, index := range []int{partsBelow, len(session.times)} {
		_, err := session.Frame(context.Background(), index, Ask{})
		if err == nil || !strings.Contains(err.Error(),
			fmt.Sprintf("no frame %d", index)) {
			t.Errorf("frame %d gave %v", index, err)
		}
	}
}

// a closed session draws nothing, and closing twice is harmless.
func TestSessionClosedDrawsNothing(t *testing.T) {
	session := opening(t, stillOf(t, sessionWidow, "", ""))
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if err := session.Close(); err != nil {
		t.Errorf("closing twice gave %v", err)
	}
	if _, err := session.Frame(context.Background(), nothing,
		Ask{}); !errors.Is(err, errClosed) {
		t.Errorf("a closed session drew, or gave %v", err)
	}
}

// the hud is drawn on a picture of the frame's size, and refused on any
// other, or for a Drawn no session drew.
func TestDrawnOverlayTakesAPictureOfTheFrame(t *testing.T) {
	session := opening(t, stillOf(t, sessionWidow, "", ""))
	drawn, err := session.Frame(context.Background(), nothing, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	right := image.NewRGBA(image.Rect(nothing, nothing, sessionWidth,
		sessionHeight))
	if err := drawn.Overlay(right); err != nil {
		t.Errorf("a picture of the frame gave %v", err)
	}
	wrong := image.NewRGBA(image.Rect(nothing, nothing, sessionHeight,
		sessionWidth))
	for _, picture := range []*image.RGBA{wrong, nil} {
		if err := drawn.Overlay(picture); err == nil {
			t.Error("the hud was drawn on a picture of another size")
		}
	}
	if err := (Drawn{}).Overlay(right); !errors.Is(err, errNotDrawn) {
		t.Errorf("a Drawn no session drew gave %v", err)
	}
}

// a still's frame is at its time and drawn on the cpu with no profile.
func TestSessionStillIsAtItsTime(t *testing.T) {
	wanted := stillOf(t, sessionWidow, "", "at: {value: "+partsAtWritten+
		", unit: s}\n")
	session := opening(t, wanted)
	times := session.Times()
	if len(times) != partsOnce || times[nothing] != partsAt {
		t.Fatalf("a still at ten minutes has times %v", times)
	}
	times[nothing] = nothing
	if session.Times()[nothing] == nothing {
		t.Error("Times gave the session's own slice to change")
	}
	drawn, err := session.Frame(context.Background(), nothing, Ask{})
	if err != nil || drawn.At() != partsAt ||
		drawn.Profile().Device != "" {
		t.Errorf("the still's frame is at %v, %+v: %v", drawn.At(),
			drawn.Profile(), err)
	}
}

// quality is carried, exact where the job says none.
func TestSessionCarriesQuality(t *testing.T) {
	for _, each := range []struct{ said, want string }{
		{"quality: preview\n", "preview"}, {"", "exact"},
	} {
		session := opening(t, stillOf(t, sessionWidow, "", each.said))
		if session.Quality() != each.want {
			t.Errorf("quality %q read as %q", each.said, session.Quality())
		}
	}
}

// a smoke is staged with its own lens, rays and tone where the job says
// none, and the job's exposure where it says one.
func TestSessionSmokeTakesWhatTheJobLeaves(t *testing.T) {
	scene, _ := milestone.Get(sessionPlainSmoke)
	plain := opening(t, jobFrom(t, fmt.Sprintf("scene: {smoke: %d}\n",
		sessionPlainSmoke)))
	if plain.given.wide != scene.Wide/maths.Degree ||
		plain.given.samples != max(scene.Samples, singleRay) ||
		plain.Tone() != scene.Tone {
		t.Errorf("an unsaid smoke is %+v", plain.given)
	}
	said := opening(t, jobFrom(t, fmt.Sprintf("scene: {smoke: %d}\n"+
		"film: {rays: %d, wide: %s, exposure: %s}\n", sessionPlainSmoke,
		partsWidth, sessionAngle, sessionExposure)))
	if said.given.samples != partsWidth ||
		said.given.wide == scene.Wide/maths.Degree ||
		said.Tone().Exposure == scene.Tone.Exposure {
		t.Errorf("a smoke the job says is %+v, %+v", said.given,
			said.Tone())
	}
}

// a camera path and a zoom on a still are sampled at the still's own
// moment, and move what it sees.
func TestSessionStillTakesACameraPath(t *testing.T) {
	plain := sessionPicture(t, stillOf(t, sessionWidow, "", ""), Ask{})
	for _, camera := range []string{
		"camera:\n  unit: m\n  look-at: {x: 0, y: 0, z: 0}\n" +
			"  up: {x: 0, y: 0, z: 1}\n  moves:\n" +
			"    - from: {x: 230, y: 0, z: 45}\n" +
			"      to: {x: 230, y: 0, z: 45}\n" +
			"      duration: {value: 1, unit: s}\n",
		"camera:\n  name: eye\n  zoom: {from: {value: 20, unit: deg}, " +
			"to: {value: 60, unit: deg}, duration: {value: 8, unit: s}}\n",
	} {
		moved := sessionPicture(t, stillOf(t, sessionWidow, "", camera),
			Ask{})
		if bytes.Equal(moved, plain) {
			t.Errorf("a still's camera did not move: %q", camera)
		}
	}
}

// what a session does not draw is refused at Open, by name.
func TestOpenRefusesWhatASessionDoesNotDraw(t *testing.T) {
	widow, err := filepath.Abs(sessionWidow)
	if err != nil {
		t.Fatal(err)
	}
	path := "camera:\n  unit: m\n  look-at: {x: 0, y: 0, z: 0}\n" +
		"  moves:\n    - from: {x: 0, y: 0, z: 6}\n" +
		"      to: {x: 0, y: 0, z: 2}\n      speed: {value: 2, unit: m/s}\n"
	set := "scene: {set: " + widow + "}\n"
	film := "fps: 2\nduration: {value: 1, unit: s}\n"
	for _, each := range []struct{ document, says string }{
		{"scene: {named: fire-night}\n", "named"},
		{"scene: {frames: drawn}\n", "frames"},
		{set + strings.Replace(path, "unit: m", "unit: km", 1), "metres"},
		{set + film, "ground"},
		{fmt.Sprintf("scene: {smoke: %d}\n", partsNoSmoke), "no smoke"},
		{fmt.Sprintf("scene: {smoke: %d}\ncamera: {name: eye}\n",
			sessionPlainSmoke), "one camera"},
		{fmt.Sprintf("scene: {smoke: %d}\n", sessionGaugedSmoke), "a gauge"},
	} {
		_, err := Open(context.Background(), jobFrom(t, each.document),
			Stamp{})
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("%q gave %v, not a refusal saying %q", each.document,
				err, each.says)
		}
	}
}

// a smoke whose inputs are not there is refused at Open, in the smoke's
// own words.
func TestOpenRefusesASmokeWithoutItsInputs(t *testing.T) {
	t.Setenv("SLAYER_MARBLE", "")
	wanted := jobFrom(t, fmt.Sprintf("scene: {smoke: %d}\n", partsMarble))
	_, err := Open(context.Background(), wanted, Stamp{})
	if err == nil || !strings.Contains(err.Error(), "SLAYER_MARBLE") {
		t.Errorf("a smoke without its inputs gave %v", err)
	}
}

// a world of a set's pieces whose film says no print is developed by
// hillside's own tone, at the job's exposure where it says one.
func TestSessionExposureOnASetWithNoPrint(t *testing.T) {
	bare := widowWith(t, "    print: {knee: 0.7, top: 8, source: eye}\n",
		"")
	said := opening(t, stillOf(t, bare, ", exposure: "+sessionExposure,
		""))
	want := developing
	want.Exposure, _ = strconv.ParseFloat(sessionExposure, 64)
	if said.Tone() != want {
		t.Errorf("the job's exposure gave the tone %+v", said.Tone())
	}
	if unsaid := opening(t, stillOf(t, bare, "", "")); unsaid.Tone() !=
		developing {
		t.Errorf("no exposure gave the tone %+v", unsaid.Tone())
	}
}

// a session says the exposure it was staged at and the size a frame
// takes when asked none: the job's where it says them.
func TestSessionSaysItsExposureAndSize(t *testing.T) {
	bare := widowWith(t, "    print: {knee: 0.7, top: 8, source: eye}\n",
		"")
	said := opening(t, stillOf(t, bare, ", exposure: "+sessionExposure,
		""))
	want, _ := strconv.ParseFloat(sessionExposure, 64)
	if said.Exposure() != want {
		t.Errorf("the job's exposure %v was staged as %v", want,
			said.Exposure())
	}
	width, height := said.Size()
	if width <= nothing || height <= nothing {
		t.Errorf("a session says it draws %d by %d", width, height)
	}
}

// anti-test: a job that says no exposure is staged at another one, the
// set's or the default, not the job's.
func TestSessionExposureUnsaidIsNotTheJobs(t *testing.T) {
	bare := widowWith(t, "    print: {knee: 0.7, top: 8, source: eye}\n",
		"")
	unsaid := opening(t, stillOf(t, bare, "", ""))
	said, _ := strconv.ParseFloat(sessionExposure, 64)
	if unsaid.Exposure() == said || unsaid.Exposure() <= nothing {
		t.Errorf("no exposure was staged as %v", unsaid.Exposure())
	}
}
