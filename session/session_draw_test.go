//go:build draw

package session

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/interfaces/file"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/render/colour"
	"go.uber.org/mock/gomock"
)

const (
	// sessionSmallWidth and sessionSmallHeight are a smaller still, as a
	// live view draws first.
	sessionSmallWidth  = 24
	sessionSmallHeight = 14
	// sessionAngle is a lens narrower than widow's own 40 degrees.
	sessionAngle = "25"
	// sessionTall is an angle down the picture, in degrees.
	sessionTall = "20"
	// sessionExposure is the job's exposure, and sessionSetExposure the
	// set's, which the job's stands over.
	sessionExposure    = "0.5"
	sessionSetExposure = "3"
	// sessionSquare is a pixel of the picture as one pixel of the png,
	// as save writes it.
	sessionSquare = 1
)

// sized is the command line's size for a still w by h with the tests'
// rays.
func sized(width, height int) []string {
	return []string{"-w", strconv.Itoa(width), "-h", strconv.Itoa(height),
		"-n", strconv.Itoa(sessionRays)}
}

// sessionWidow is a set of a world and no ground, which stages with no
// shelf of sources.
const sessionWidow = "../sets/widow.yaml"

// sessionSide is a second camera for widow, off to its east, written
// before its own so that it is the first.
const sessionSide = "  cameras:\n    side:\n      location:\n" +
	"        at: [230, 0, 45]\n      orientation:\n" +
	"        look_at: [0, 0, 0]\n"

// widowFilm is the start of widow's film, where a change to it goes.
const widowFilm = "  film:\n    wide: 40\n"

// widowName is widow's name line, where a scene's renderer goes.
const widowName = "  name: widow\n"

// stillOf is a job of a still of a set at the tests' size, with more
// keys in its film and more lines after it.
func stillOf(t *testing.T, set, inFilm, after string) job.Job {
	t.Helper()
	absolute, err := filepath.Abs(set)
	if err != nil {
		t.Fatal(err)
	}
	return jobFrom(t, fmt.Sprintf("scene: {set: %s}\nfilm: {width: %d, "+
		"height: %d, rays: %d%s}\n%s", absolute, sessionWidth,
		sessionHeight, sessionRays, inFilm, after))
}

// widowWith is widow's set file with each pair of texts, the first
// changed to the second, written to a file of its own.
func widowWith(t *testing.T, pairs ...string) string {
	t.Helper()
	raw, err := os.ReadFile(sessionWidow)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for i := 0; i+1 < len(pairs); i += 2 {
		changed := strings.Replace(text, pairs[i], pairs[i+1], 1)
		if changed == text {
			t.Fatalf("widow has no %q to change", pairs[i])
		}
		text = changed
	}
	path := filepath.Join(t.TempDir(), "widow.yaml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// ranPicture is the png Run writes for a set and settings.
func ranPicture(t *testing.T, set string, settings ...string) []byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), "run.png")
	arguments := append([]string{"-set", set, "-out", out}, settings...)
	var wrong bytes.Buffer
	if code := Run(arguments, io.Discard, &wrong, Stamp{}); code !=
		nothing {
		t.Fatalf("hillside %v gave %d: %s", arguments, code, wrong.String())
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// opening is the session a job opens, closed when the test ends.
func opening(t *testing.T, wanted job.Job) *Session {
	t.Helper()
	session, err := Open(context.Background(), wanted, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// sessionPicture is the png of a still's frame as Run writes one: the
// light developed by the scene's own tone, its hud drawn, written as
// save writes it.
func sessionPicture(t *testing.T, wanted job.Job, ask Ask) []byte {
	t.Helper()
	session := opening(t, wanted)
	drawn, err := session.Frame(context.Background(), nothing, ask)
	if err != nil {
		t.Fatal(err)
	}
	picture := session.Tone().Develop(drawn.Light())
	if err := drawn.Overlay(picture); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "session.png")
	if err := file.PNG(out, picture, sessionSquare); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// a job still of a set is Run's picture of it, byte for byte.
func TestSessionStillOfASetIsRunsPicture(t *testing.T) {
	want := ranPicture(t, sessionWidow, sized(sessionWidth,
		sessionHeight)...)
	got := sessionPicture(t, stillOf(t, sessionWidow, "", ""), Ask{})
	if !bytes.Equal(got, want) {
		t.Error("the session and hillside wrote different pictures")
	}
}

// an ask's size stands over the job's, as -w, -h and -n do, without the
// scene staged again; the zero ask is the job's own.
func TestSessionAskStandsOverTheJobsSize(t *testing.T) {
	ask, err := NewAsk(sessionSmallWidth, sessionSmallHeight, sessionRays,
		nil)
	if err != nil {
		t.Fatal(err)
	}
	want := ranPicture(t, sessionWidow, sized(sessionSmallWidth,
		sessionSmallHeight)...)
	wanted := stillOf(t, sessionWidow, "", "")
	if got := sessionPicture(t, wanted, ask); !bytes.Equal(got, want) {
		t.Error("the ask's size did not stand over the job's")
	}
	if got := sessionPicture(t, wanted, Ask{}); bytes.Equal(got, want) {
		t.Error("the zero ask drew the ask's size, not the job's")
	}
}

// a camera named in the job gives that camera's picture, and it differs
// from the set's first camera's.
func TestSessionNamedCameraIsThatCamerasPicture(t *testing.T) {
	twin := widowWith(t, "  cameras:\n", sessionSide)
	want := ranPicture(t, sessionWidow, sized(sessionWidth,
		sessionHeight)...)
	named := sessionPicture(t, stillOf(t, twin, "", "camera: {name: eye}\n"),
		Ask{})
	first := sessionPicture(t, stillOf(t, twin, "", ""), Ask{})
	if !bytes.Equal(named, want) {
		t.Error("the named camera is not that camera's picture")
	}
	if bytes.Equal(first, named) {
		t.Error("the set's first camera drew the named one's picture")
	}
}

// a camera the set has not is refused, with the names it has.
func TestSessionRefusesACameraTheSetHasNot(t *testing.T) {
	twin := widowWith(t, "  cameras:\n", sessionSide)
	wanted := stillOf(t, twin, "", "camera: {name: nobody}\n")
	_, err := Open(context.Background(), wanted, Stamp{})
	if err == nil || !strings.Contains(err.Error(), `"nobody"`) ||
		!strings.Contains(err.Error(), "side, eye") {
		t.Errorf("a camera the set has not gave %v", err)
	}
}

// the job's exposure stands over the set's: in a set drawn from its world
// alone it is in the print, and the picture is the set's with the job's
// exposure written in it.
func TestSessionJobExposureStandsOverTheSets(t *testing.T) {
	said := widowWith(t, widowFilm, widowFilm+"    exposure: "+
		sessionSetExposure+"\n")
	jobs := widowWith(t, widowFilm, widowFilm+"    exposure: "+
		sessionExposure+"\n")
	wanted := stillOf(t, said, ", exposure: "+sessionExposure, "")
	size := sized(sessionWidth, sessionHeight)
	got := sessionPicture(t, wanted, Ask{})
	if !bytes.Equal(got, ranPicture(t, jobs, size...)) {
		t.Error("the job's exposure did not stand over the set's")
	}
	if bytes.Equal(got, ranPicture(t, said, size...)) {
		t.Error("the set's exposure drew the same picture as the job's")
	}
}

// the job's lens stands over the set's.
func TestSessionJobWideStandsOverTheSets(t *testing.T) {
	jobs := widowWith(t, widowFilm, "  film:\n    wide: "+sessionAngle+"\n")
	size := sized(sessionWidth, sessionHeight)
	got := sessionPicture(t, stillOf(t, sessionWidow, ", wide: "+
		sessionAngle, ""), Ask{})
	if !bytes.Equal(got, ranPicture(t, jobs, size...)) {
		t.Error("the job's wide did not stand over the set's")
	}
	if bytes.Equal(got, ranPicture(t, sessionWidow, size...)) {
		t.Error("the set's wide drew the same picture as the job's")
	}
}

// the job's renderer stands over the set's: a set that says the gpu is
// drawn on the cpu when the job says so, and on the gpu when it says
// nothing.
func TestSessionJobRendererStandsOverTheSets(t *testing.T) {
	gpuSet := widowWith(t, widowName, widowName+"  renderer: gpu\n")
	size := sized(sessionWidth, sessionHeight)
	got := sessionPicture(t, stillOf(t, gpuSet, "", "renderer: cpu\n"),
		Ask{})
	if !bytes.Equal(got, ranPicture(t, sessionWidow, size...)) {
		t.Error("the job's cpu did not stand over the set's gpu")
	}
	unsaid := opening(t, stillOf(t, gpuSet, "", ""))
	if !unsaid.given.onGPU {
		t.Error("a job that says no renderer did not leave the set's")
	}
}

// on the command line the set stands over the flags: -wide and -gpu are
// the set's to say, and -exposure is refused beside -set.
func TestRunLetsTheSetStandOverItsFilm(t *testing.T) {
	size := sized(sessionWidth, sessionHeight)
	own := ranPicture(t, sessionWidow, size...)
	if !bytes.Equal(ranPicture(t, sessionWidow, append(size, "-wide",
		sessionAngle)...), own) {
		t.Error("-wide stood over the set's lens")
	}
	cpuSet := widowWith(t, widowName, widowName+"  renderer: cpu\n")
	if !bytes.Equal(ranPicture(t, cpuSet, append(size, "-gpu")...), own) {
		t.Error("-gpu stood over the set's renderer")
	}
	arguments := append([]string{"-set", sessionWidow, "-exposure",
		sessionExposure}, size...)
	if code := Run(arguments, io.Discard, io.Discard, Stamp{}); code !=
		exitUsage {
		t.Errorf("-exposure beside -set gave %d, not %d", code, exitUsage)
	}
}

// exitUsage is defined as the number Run gives for flags it cannot take.
const exitUsage = 2

// tall is the wide it converts to, for the frame's shape; a frame of
// another shape works it out again.
func TestSessionTallIsTheWideItConvertsTo(t *testing.T) {
	tall := stillOf(t, sessionWidow, ", tall: "+sessionTall, "")
	wideFor := func(width, height int) job.Job {
		wide, _ := tall.Film().WideFor(width, height)
		return stillOf(t, sessionWidow, ", wide: "+
			strconv.FormatFloat(wide, 'g', -1, 64), "")
	}
	if !bytes.Equal(sessionPicture(t, tall, Ask{}),
		sessionPicture(t, wideFor(sessionWidth, sessionHeight), Ask{})) {
		t.Error("tall is not the wide it converts to")
	}
	upright, err := NewAsk(sessionHeight, sessionWidth, sessionRays, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := sessionPicture(t, tall, upright)
	if !bytes.Equal(got, sessionPicture(t, wideFor(sessionHeight,
		sessionWidth), upright)) {
		t.Error("tall was not worked out for the asked frame's shape")
	}
	if bytes.Equal(got, sessionPicture(t, wideFor(sessionWidth,
		sessionHeight), upright)) {
		t.Error("tall kept the job's shape's angle in another shape")
	}
}

// sequenced runs the sequence verb on a plan into a directory of its
// own, and gives the directory.
func sequenced(t *testing.T, plan string) string {
	t.Helper()
	output := filepath.Join(t.TempDir(), "frames")
	arguments := []string{"sequence", "--plan", plan, "--output", output}
	var wrong bytes.Buffer
	if code := sequenceRun(arguments, io.Discard, &wrong); code !=
		nothing {
		t.Fatalf("sequence gave %d: %s", code, wrong.String())
	}
	return output
}

// sameAsSequenced holds each frame a plan's lifted job draws to be the
// sequence verb's frame file, developed by the plan's print and written
// by the sequence's own writer, and its times to be the plan's.
func sameAsSequenced(t *testing.T, path, output string) {
	t.Helper()
	plan, err := readSequence(path)
	if err != nil {
		t.Fatal(err)
	}
	session := opening(t, plan.lifted)
	times := session.Times()
	if len(times) != plan.frames {
		t.Fatalf("the session has %d frames, the plan %d", len(times),
			plan.frames)
	}
	tone := colour.Tone{Knee: plan.knee, Top: plan.top, Bloom: plan.bloom}
	for index, at := range times {
		if at != plan.frameTime(index) {
			t.Errorf("frame %d at %v, the plan's at %v", index, at,
				plan.frameTime(index))
		}
		drawn, err := session.Frame(context.Background(), index, Ask{})
		if err != nil {
			t.Fatal(err)
		}
		picture := tone.Develop(drawn.Light())
		if err = drawn.Overlay(picture); err != nil {
			t.Fatal(err)
		}
		written := filepath.Join(t.TempDir(), "frame.png")
		if _, err = file.SequencePNGHash(written, picture); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(written)
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("frame-%06d.png", index)
		want, err := os.ReadFile(filepath.Join(output, name))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("frame %d is not the sequence's: %v", index, err)
		}
	}
}

// a job film of a smoke, lifted from the camera smoke's plan, draws the
// sequence verb's frames, byte for byte, and the manifest a plan writes
// holds no key.
func TestSessionFilmOfASmokeIsTheSequencesFrames(t *testing.T) {
	path := shortSmokePlan(t, sessionPlainSmoke)
	output := sequenced(t, path)
	sameAsSequenced(t, path, output)
	manifest, err := os.ReadFile(filepath.Join(output, "sequence.json"))
	if err != nil || bytes.Contains(manifest, []byte(`"Key"`)) {
		t.Errorf("a plan's manifest holds a key, or is not there: %v", err)
	}
}

// a job film of a set, lifted from a plan, draws the sequence verb's
// frames, its sections kept from frame to frame as the sequence keeps
// them.
func TestSessionFilmOfASetIsTheSequencesFrames(t *testing.T) {
	scene := sectionsSet(t, true)
	dir := t.TempDir()
	output := filepath.Join(dir, "frames")
	path := filepath.Join(dir, "clip.yaml")
	text := fmt.Sprintf(seqPlan, scene, output)
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	sameAsSequenced(t, path, sequenced(t, path))
}

// a world-only film draws authored motion at each absolute frame time,
// while keeping the set's own print and camera.
func TestSessionFilmOfAWorldOnlySetDrawsKeyedMotionAtEachTime(t *testing.T) {
	raw, err := os.ReadFile(sessionWidow)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "    motions:\n")
	end := strings.Index(text, "    wisps:\n")
	if start < 0 || end < start {
		t.Fatal("widow has no authored orbit block")
	}
	text = text[:start] + "    motions:\n      orbit:\n" +
		"        seconds: 1\n        ease: linear\n        hold: true\n" +
		"        frames:\n          - {percent: 0, at: [-45, 0, 0]}\n" +
		"          - {percent: 100, at: [45, 0, 0]}\n" + text[end:]
	text = strings.Replace(text,
		"print: {knee: 0.7, top: 8, source: eye}",
		"print: {knee: 0.5, top: 4, source: eye}", 1)
	scene := filepath.Join(t.TempDir(), "widow.yaml")
	if err := os.WriteFile(scene, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted := jobFrom(t, fmt.Sprintf("scene: {set: %s}\nfilm: {width: %d, "+
		"height: %d, rays: 1}\nfps: 1\nduration: {value: 2, unit: s}\n",
		scene, sessionSmallWidth, sessionSmallHeight))
	session := opening(t, wanted)
	if len(session.Times()) != 2 || session.Times()[0] != 0 ||
		session.Times()[1] != time.Second {
		t.Fatalf("the world-only film times are %v", session.Times())
	}
	first, err := session.Frame(context.Background(), 0, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := session.Frame(context.Background(), 1, Ask{})
	if err != nil {
		t.Fatal(err)
	}
	if first.At() != 0 || second.At() != time.Second {
		t.Fatalf("frames are at %v and %v", first.At(), second.At())
	}
	if slices.Equal(first.Light().Pix, second.Light().Pix) {
		t.Fatal("the keyed world thing did not move between film times")
	}
	if tone := session.Tone(); tone.Knee != 0.5 || tone.Top != 4 {
		t.Fatalf("the world-only film has tone %+v, want its authored print", tone)
	}
}

// a smoke with bands is refused by name in a job, and still drawn as
// before by a sequence plan, which draws it without them.
func TestSessionRefusesABandedSmokeAPlanDraws(t *testing.T) {
	wanted := jobFrom(t, fmt.Sprintf("scene: {smoke: %d}\nfilm: {width: "+
		"%d, height: %d, rays: %d}\n", sessionBandedSmoke,
		sessionSmallWidth, sessionSmallHeight, sessionRays))
	_, err := Open(context.Background(), wanted, Stamp{})
	if err == nil || !strings.Contains(err.Error(), "bands") {
		t.Errorf("a banded smoke in a job gave %v", err)
	}
	output := sequenced(t, shortSmokePlan(t, sessionBandedSmoke))
	if _, err := os.Stat(filepath.Join(output, "frame-000001.png")); err !=
		nil {
		t.Errorf("a plan did not draw the banded smoke: %v", err)
	}
}

// a still through a set's rig is Run's picture of it; an ask's width
// gives the rig's rows for it, and a height that is not those rows is
// refused.
func TestSessionStillThroughARigIsRunsPicture(t *testing.T) {
	rigged := onFixtureRigged(t, "0.5", false)
	want := ranPicture(t, rigged)
	wanted := jobFrom(t, "scene: {set: "+rigged+"}\n")
	if got := sessionPicture(t, wanted, Ask{}); !bytes.Equal(got, want) {
		t.Error("the session and hillside drew the rig differently")
	}
	session := opening(t, wanted)
	wider, err := NewAsk(sessionWidth, nothing, nothing, nil)
	if err != nil {
		t.Fatal(err)
	}
	drawn, err := session.Frame(context.Background(), nothing, wider)
	rows := session.given.frame.Rows(sessionWidth)
	if err != nil || drawn.Light().W != sessionWidth ||
		drawn.Light().H != rows {
		t.Fatalf("an ask's width on a rig gave %v", err)
	}
	wrong, err := NewAsk(sessionWidth, rows+partsOnce, nothing, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Frame(context.Background(), nothing,
		wrong); err == nil || !strings.Contains(err.Error(), "rig") {
		t.Errorf("a height a rig does not have gave %v", err)
	}
}

// a camera named in a job on a set with ground is that camera's picture,
// and a name the set has not is refused with the names it has.
func TestSessionNamedCameraOnGround(t *testing.T) {
	scene := onFixture(t, "", "")
	want := ranPicture(t, scene)
	named := jobFrom(t, "scene: {set: "+scene+"}\ncamera: {name: stone}\n")
	if !bytes.Equal(sessionPicture(t, named, Ask{}), want) {
		t.Error("the named camera on ground is not its picture")
	}
	nobody := jobFrom(t, "scene: {set: "+scene+"}\ncamera: {name: nobody}\n")
	_, err := Open(context.Background(), nobody, Stamp{})
	if err == nil || !strings.Contains(err.Error(), "its cameras are stone") {
		t.Errorf("a camera the ground set has not gave %v", err)
	}
}

// a film through a rig is refused, as a sequence refuses it, and the
// job's lens or exposure on a rig is refused, since the rig says them.
func TestOpenRefusesWhatARigSays(t *testing.T) {
	rigged := onFixtureRigged(t, "0.5", false)
	for _, each := range []struct{ document, says string }{
		{"fps: 2\nduration: {value: 1, unit: s}\nshutter: {angle: " +
			"{value: 180, unit: deg}, glimpses: 2}\n", "leave the job's " +
			"shutter out"},
		{"film: {width: 48, height: 48}\n", "which is wide"},
		{"film: {wide: " + sessionAngle + "}\n", "with a rig"},
		{"film: {exposure: " + sessionExposure + "}\n", "with a rig"},
		{"camera: {name: stone, zoom: {from: {value: 20, unit: deg}, " +
			"to: {value: 60, unit: deg}, duration: {value: 8, unit: s}}}\n",
			"leave the zoom out"},
	} {
		wanted := jobFrom(t, "scene: {set: "+rigged+"}\n"+each.document)
		_, err := Open(context.Background(), wanted, Stamp{})
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("%q on a rig gave %v", each.document, err)
		}
	}
}

// a film of a set whose file has gone since the job was read is refused
// with the reader's error.
func TestOpenRefusesAFilmOfASetThatHasGone(t *testing.T) {
	gone := widowWith(t)
	wanted := jobFrom(t, "scene: {set: "+gone+"}\n"+
		"fps: 2\nduration: {value: 1, unit: s}\n")
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), wanted, Stamp{}); err == nil {
		t.Error("a film of a set that has gone was opened")
	}
}

// a context cancelled while the scene is staged gives its error, and
// the session it staged is closed, not given.
func TestOpenCancelledWhileStagingGivesNoSession(t *testing.T) {
	ctx := NewMockContext(gomock.NewController(t))
	gomock.InOrder(ctx.EXPECT().Err().Return(nil),
		ctx.EXPECT().Err().Return(context.Canceled))
	session, err := Open(ctx, stillOf(t, sessionWidow, "", ""), Stamp{})
	if !errors.Is(err, context.Canceled) || session != nil {
		t.Errorf("a cancel while staging gave %v and %v", session, err)
	}
}
