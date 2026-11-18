package live

import (
	"errors"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/terminal"
	"github.com/janearc/slayer/job"
	"go.uber.org/mock/gomock"
)

// ready is a watcher of a job on a screen the test gives, with the clock
// it reads, read and staged as idle does, so a test can drive it by hand.
func ready(t *testing.T, text string, screen func(*gomock.Controller) Screen) (
	*Watcher, *clock, *gomock.Controller) {
	t.Helper()
	control := gomock.NewController(t)
	now := &clock{now: time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)}
	watching := watcherOn(t, jobFile(t, text), NewMockStager(control),
		screen(control), now, make(chan time.Time))
	watching.reread()
	frames := filmFrames
	if watching.still() {
		frames = one
	}
	watching.opened(factsOf(stageOf(control, frames, nil)))
	watching.pending = Unchanged
	return watching, now, control
}

// quiet is a screen that takes everything and tells nothing.
func quiet(control *gomock.Controller) Screen {
	return screenOf(control, make(chan terminal.Event), nil, nil)
}

// lightHeld is a frame of light as the screen would hold it, at a frame.
func lightHeld(control *gomock.Controller, index int) *held {
	return &held{drawn: drawnOf(control, 40, 30), index: index}
}

// The host's key is in the table of named keys and does nothing: it
// moves no frame, camera or exposure.
func TestTheHostsKeyActionChangesNothing(t *testing.T) {
	watching := idle(t, filmJob)
	before := watching.view
	act, found := byName[terminal.KeyHost]
	if !found {
		t.Fatal("the host's key is not in the table")
	}
	if got := act(watching); got != Unchanged || watching.view != before {
		t.Errorf("the host's key gave %v and left the view %+v, was %+v",
			got, watching.view, before)
	}
}

// With no camera chosen by key, c goes on from the camera the job names:
// far is second of near and far, so the next is near.
func TestNextCameraGoesOnFromTheJobsOwnCamera(t *testing.T) {
	watching := idle(t, stillJob+"camera: {name: far}\n")
	if got := watching.key(pressed(t, "c")); got != Restage {
		t.Errorf("c gave %v, want a restage", got)
	}
	if watching.view.camera != "near" {
		t.Errorf("the camera chosen is %q, want near", watching.view.camera)
	}
	if !strings.Contains(strings.Join(watching.foot.lines(), " "),
		"camera near") {
		t.Errorf("the foot says %q", watching.foot.lines())
	}
}

// A watcher with no status folder writes none and counts no pictures.
func TestPublishWithNoFolderWritesNothing(t *testing.T) {
	watching := idle(t, stillJob)
	watching.given.Folder = ""
	picture := image.NewRGBA(image.Rect(0, 0, 4, 3))
	cells := terminal.Chop(canvas(picture, placed(4, 3, paneOf(testScreen()))))
	watching.publish(picture, cells, testPixel, true)
	if watching.kept.pictures != none {
		t.Errorf("%d pictures counted with no folder", watching.kept.pictures)
	}
	entries, err := os.ReadDir(filepath.Dir(watching.given.Path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != one || entries[none].Name() != "test.yaml" {
		t.Errorf("the job's folder holds %d entries, want only the job",
			len(entries))
	}
}

// The status says stopped, staging, drawing or shown by what the watcher
// is doing, in that order of precedence.
func TestStateSaysWhatTheWatcherIsDoing(t *testing.T) {
	watching := idle(t, stillJob)
	if got := watching.state(); got != stateShown {
		t.Errorf("an idle watcher is %q, want %q", got, stateShown)
	}
	watching.flight = &flight{}
	if got := watching.state(); got != stateDrawing {
		t.Errorf("a watcher with a plan in flight is %q, want %q", got,
			stateDrawing)
	}
	watching.words = stagingWords
	if got := watching.state(); got != stateStaging {
		t.Errorf("a watcher staging is %q, want %q", got, stateStaging)
	}
	watching.stopped = true
	if got := watching.state(); got != stateStopped {
		t.Errorf("a stopped watcher is %q, want %q", got, stateStopped)
	}
	if got := watching.report().State; got != stateStopped {
		t.Errorf("the report of a stopped watcher says %q", got)
	}
}

// A status that cannot be written as JSON is refused and leaves nothing
// at its path.
func TestWriteWholeRefusesAStatusItCannotMarshal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	if err := writeWhole(path, report{Time: math.NaN()}); err == nil {
		t.Error("a status with a NaN time was written")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("a status file was left for a refused status")
	}
	if err := writeWhole(path, report{Time: 1.5}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("a good status was not written: %v", err)
	}
}

// s says why when the job file cannot be written, and keeps what the keys
// chose.
func TestSaveSaysWhenTheFileCannotBeWritten(t *testing.T) {
	watching := idle(t, stillJob)
	watching.key(pressed(t, "c"))
	if err := os.Remove(watching.given.Path); err != nil {
		t.Fatal(err)
	}
	if got := watching.key(pressed(t, "s")); got != Unchanged {
		t.Errorf("s with the file gone gave %v", got)
	}
	if !strings.Contains(strings.Join(watching.foot.lines(), " "),
		"no such file") {
		t.Errorf("the foot says %q", watching.foot.lines())
	}
	if watching.view.camera != "near" {
		t.Errorf("the camera chosen was lost: %q", watching.view.camera)
	}
}

// writeEdits refuses a path that is a folder, which can be seen but not
// read as a file.
func TestWriteEditsRefusesAFolder(t *testing.T) {
	dir := t.TempDir()
	if err := writeEdits(dir, edits{camera: "far"}); err == nil {
		t.Error("a folder was edited as a job file")
	}
}

// s says so when the job it has just written cannot be read back, and
// holds the trouble at the foot.
func TestSaveSaysWhenTheWrittenJobCannotBeRead(t *testing.T) {
	watching := idle(t, stillJob)
	watching.key(pressed(t, "c"))
	watching.given.Read = func(string) (job.Job, error) {
		return job.Job{}, errors.New("the job reads no more")
	}
	if got := watching.key(pressed(t, "s")); got != Unchanged {
		t.Errorf("s gave %v when the job would not read back", got)
	}
	if !strings.Contains(watching.foot.trouble, "reads no more") {
		t.Errorf("the trouble is %q", watching.foot.trouble)
	}
	raw, err := os.ReadFile(watching.given.Path)
	if err != nil || !strings.Contains(string(raw), "near") {
		t.Errorf("the camera was not written: %s %v", raw, err)
	}
}

// Help mode shows the help text in the foot, with the status row last,
// and no picture.
func TestHelpModeShowsTheHelpAtTheFoot(t *testing.T) {
	footed := make(chan []string, 8)
	shown := make(chan *terminal.Cells, 8)
	watching, _, control := ready(t, stillJob,
		func(control *gomock.Controller) Screen {
			return screenOf(control, make(chan terminal.Event), shown, footed)
		})
	watching.shown = lightHeld(control, none)
	watching.view.mode = helping
	watching.show()
	lines := await(t, footed, "the foot")
	if !strings.Contains(strings.Join(lines, "\n"), helpLines[none]) ||
		!strings.Contains(lines[len(lines)-one], "? help") {
		t.Errorf("the foot in help mode says %q", lines)
	}
	if len(shown) != none {
		t.Error("a picture was presented in help mode")
	}
}

// A screen that cannot be written to is said at the foot, and the status
// row is still sent.
func TestAScreenThatRefusesIsSaidAtTheFoot(t *testing.T) {
	var footed int
	watching, _, control := ready(t, stillJob,
		func(control *gomock.Controller) Screen {
			screen := NewMockScreen(control)
			screen.EXPECT().Events().Return(
				(<-chan terminal.Event)(make(chan terminal.Event))).AnyTimes()
			screen.EXPECT().Screen().Return(testScreen()).AnyTimes()
			screen.EXPECT().Present(gomock.Any()).
				Return(errors.New("the terminal went away")).AnyTimes()
			screen.EXPECT().Foot(gomock.Any()).DoAndReturn(
				func([]string) error {
					footed++
					return nil
				}).AnyTimes()
			return screen
		})
	watching.shown = lightHeld(control, none)
	watching.show()
	if !strings.Contains(strings.Join(watching.foot.lines(), " "),
		"the terminal went away") {
		t.Errorf("the foot says %q", watching.foot.lines())
	}
	if footed != one {
		t.Errorf("the foot was written %d times, want once", footed)
	}
}

// brighter is how much more light the middle pixel of one picture has
// than another's, in the red channel.
func brighter(more, less *image.RGBA) int {
	at := image.Pt(more.Bounds().Dx()/2, more.Bounds().Dy()/2)
	return int(more.RGBAAt(at.X, at.Y).R) - int(less.RGBAAt(at.X, at.Y).R)
}

// An exposure the keys chose is laid on the light before it is developed:
// three thirds of a stop is a stop more, so the picture is brighter.
func TestPictureDevelopsAtTheExposureTheKeysChose(t *testing.T) {
	watching, _, control := ready(t, stillJob, quiet)
	watching.shown = lightHeld(control, none)
	plain, fresh, err := watching.picture()
	if err != nil || !fresh {
		t.Fatalf("the picture gave fresh %v, %v", fresh, err)
	}
	watching.view.thirds = 3
	lifted, fresh, err := watching.picture()
	if err != nil || !fresh {
		t.Fatalf("the lifted picture gave fresh %v, %v", fresh, err)
	}
	if brighter(lifted, plain) <= 0 {
		t.Errorf("a stop more light gave a middle pixel of %v against %v",
			lifted.RGBAAt(20, 15), plain.RGBAAt(20, 15))
	}
	again, fresh, err := watching.picture()
	if err != nil || fresh || again != lifted {
		t.Errorf("the same view was developed again: fresh %v, %v", fresh, err)
	}
}

// A hud that cannot be laid on the picture is a refusal from picture, and
// show says it at the foot and presents nothing.
func TestAHudThatFailsLeavesNoPicture(t *testing.T) {
	shown := make(chan *terminal.Cells, 4)
	watching, _, control := ready(t, stillJob,
		func(control *gomock.Controller) Screen {
			return screenOf(control, make(chan terminal.Event), shown, nil)
		})
	broken := NewMockDrawn(control)
	broken.EXPECT().Light().Return(lit(40, 30)).AnyTimes()
	broken.EXPECT().Overlay(gomock.Any()).
		Return(errors.New("the hud would not draw")).AnyTimes()
	watching.shown = &held{drawn: broken}
	if _, _, err := watching.picture(); err == nil {
		t.Error("a hud that failed gave a picture")
	}
	watching.show()
	if !strings.Contains(strings.Join(watching.foot.lines(), " "),
		"hud would not draw") {
		t.Errorf("the foot says %q", watching.foot.lines())
	}
	if len(shown) != none {
		t.Error("a picture was presented for a failed hud")
	}
}

// A job with titles has them drawn on its picture, and the same job
// without them has not.
func TestTitlesAreDrawnOnThePicture(t *testing.T) {
	titles := "titles: {elements: {a: {kind: text, words: HELLO, ink: " +
		"ink, across: 0.5, down: 0.5}}}\n"
	bare, _, control := ready(t, stillJob, quiet)
	bare.shown = &held{drawn: drawnOf(control, 160, 96)}
	if bare.titles != nil {
		t.Fatal("a job with no titles holds titles")
	}
	plain, _, err := bare.picture()
	if err != nil {
		t.Fatal(err)
	}
	titled, _, control := ready(t, stillJob+titles, quiet)
	titled.shown = &held{drawn: drawnOf(control, 160, 96)}
	if titled.titles == nil {
		t.Fatal("a job with titles holds none")
	}
	drawn, _, err := titled.picture()
	if err != nil {
		t.Fatal(err)
	}
	differing := 0
	for i := range plain.Pix {
		if plain.Pix[i] != drawn.Pix[i] {
			differing++
		}
	}
	if differing == none {
		t.Error("the titles changed no pixel of the picture")
	}
}

// A status row begins with the word passing through it.
func TestStatusBeginsWithThePassingWord(t *testing.T) {
	watching := idle(t, stillJob)
	watching.words = "saved"
	if got := watching.status(nil); !strings.HasPrefix(got, "saved  ") {
		t.Errorf("the status row says %q, want it to begin with saved", got)
	}
}

// A tick while a film plays and nothing is drawn moves on to the frame
// the clock has reached: three quarters of a second at four frames a
// second is three frames.
func TestTickMovesPlayOnToTheFrameTheClockReached(t *testing.T) {
	watching, now, _ := ready(t, filmJob, quiet)
	watching.key(pressed(t, " "))
	watching.pending = Unchanged
	now.pass(750 * time.Millisecond)
	watching.tick(now.Now())
	if watching.view.index != 3 || watching.pending != Redraw {
		t.Errorf("after 750 ms of play the frame is %d, owing %v; want 3 "+
			"and a redraw", watching.view.index, watching.pending)
	}
	now.pass(time.Second)
	watching.tick(now.Now())
	if watching.view.index != 7 {
		t.Errorf("after 1750 ms of play the frame is %d, want 7",
			watching.view.index)
	}
	now.pass(500 * time.Millisecond)
	watching.tick(now.Now())
	if watching.view.index != 1 {
		t.Errorf("after 2250 ms of play the frame is %d, want it round "+
			"to 1 of 8", watching.view.index)
	}
}

// A refusal told by the worker is held at the foot as the trouble.
func TestHeardKeepsARefusalAtTheFoot(t *testing.T) {
	watching := idle(t, stillJob)
	watching.heard(message{err: errors.New("the stage fell over")})
	if !strings.Contains(watching.foot.trouble, "stage fell over") {
		t.Errorf("the trouble is %q", watching.foot.trouble)
	}
}

// The files followed include the theme of the job's titles, and a file
// that is not there has the zero stamp so its coming back is a change.
func TestFollowedIncludesTheTitlesThemeAndStampsSayWhatIsGone(t *testing.T) {
	theme := filepath.Join(t.TempDir(), "theme.yaml")
	watching := idle(t, stillJob+"titles: {theme: "+theme+", elements: "+
		"{a: {kind: text, words: hi, ink: ink, across: 0.5, down: 0.5}}}\n")
	got := watching.followed()
	if len(got) != pair || got[none] != watching.given.Path || got[one] != theme {
		t.Errorf("the files followed are %v, want the job and %s", got, theme)
	}
	read := stampsOf(got)
	if read[watching.given.Path] == (stamp{}) || read[theme] != (stamp{}) {
		t.Errorf("the stamps are %v: the job's is zero, or the missing "+
			"theme's is not", read)
	}
}

// A moment the job cannot take is said, and let go, with the job shown
// the file's own.
func TestRelaidLetsGoAMomentTheJobCannotTake(t *testing.T) {
	watching := idle(t, stillJob)
	watching.view.said, watching.view.moment = true, -time.Second
	if got := watching.relaid(); got != Restage {
		t.Errorf("relaid gave %v, want a restage", got)
	}
	if watching.view.said || !strings.Contains(
		strings.Join(watching.foot.lines(), " "), "minimum") {
		t.Errorf("the moment was kept or not said: %+v, %q", watching.view,
			watching.foot.lines())
	}
}

// Titles that do not fit the picture's height are refused by picture,
// naming the hud, and nothing is kept as developed.
func TestTitlesThatDoNotFitThePictureAreRefused(t *testing.T) {
	watching, _, control := ready(t, stillJob+"titles: {elements: {a: "+
		"{kind: text, words: HELLO, ink: ink, across: 0.5, down: 0.5}}}\n",
		quiet)
	watching.shown = lightHeld(control, none)
	picture, fresh, err := watching.picture()
	if err == nil || picture != nil || fresh ||
		!strings.Contains(err.Error(), "30 high") {
		t.Errorf("a picture 30 high gave %v, fresh %v, %v; want a refusal "+
			"naming the height", picture, fresh, err)
	}
	if watching.developed.picture != nil {
		t.Error("a refused picture was kept as developed")
	}
}
