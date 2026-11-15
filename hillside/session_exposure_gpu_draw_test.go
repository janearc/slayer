//go:build draw && darwin

package hillside

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	cameraFilm "github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/metal"
	scenes "github.com/janearc/slayer/scene/named"
)

// swarmPath is a camera path past the swarm, in the swarm's own unit: a
// second along a line at twelve radii a second, then a second held still.
const swarmPath = `camera:
  unit: the star's radius
  look-at: {x: 0, y: 0, z: 0}
  moves:
    - from: {x: -6, y: 4, z: 26}
      to: {x: 6, y: 4, z: 26}
      duration: {value: 1, unit: s}
    - from: {x: 6, y: 4, z: 26}
      to: {x: 6, y: 4, z: 26}
      duration: {value: 1, unit: s}
`

// The film the gpu's exposure tests draw, picked by the blur slice's
// author, 2026-11-12: two frames a second apart, so the first is on the
// path's move and the second on its hold; a shutter of one glimpse open
// half the interval; three rays a side, so a pixel's samples have nine
// times between them.
const (
	exposureRate = 1
	exposureRays = 3
	// exposureMoving and exposureHeld are the frames on the path's move
	// and on its hold.
	exposureMoving = 0
	exposureHeld   = 1
)

// swarmFilm is a job of the swarm's film on the gpu at the tests' size,
// with what else it says.
func swarmFilm(t *testing.T, more string) job.Job {
	t.Helper()
	return jobFrom(t, fmt.Sprintf("scene: {named: swarm}\nfilm: {width: "+
		"%d, height: %d, rays: %d}\nrenderer: gpu\nfps: %d\nduration: "+
		"{value: 2, unit: s}\n", namedWidth, namedHeight, exposureRays,
		exposureRate)+swarmPath+more)
}

// swarmBuilt is the swarm as its defaults build it.
func swarmBuilt(t *testing.T) *scenes.Built {
	t.Helper()
	scene, err := scenes.Get("swarm")
	if err != nil {
		t.Fatal(err)
	}
	built, err := scene.Build(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return built
}

// frameOrSkipped is a session's frame, or the test skipped where there
// is no Metal device.
func frameOrSkipped(t *testing.T, session *Session, index int) Drawn {
	t.Helper()
	drawn, err := session.Frame(context.Background(), index, Ask{})
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// on the gpu a film's shutter holds the kernel's shutter open across the
// glimpse's share while the camera moves: the frame is the drawer's own
// open draw from the path's eye at the share's start to its eye at the
// share's end, at the glimpse's time, to the bit
func TestSessionShutterOnTheGPUIsTheKernelsOverTheGlimpsesShare(t *testing.T) {
	wanted := swarmFilm(t, shutterFilmKey(shutterHalf, singleRay))
	session := opening(t, wanted)
	got := frameOrSkipped(t, session, exposureMoving)
	plan := cameraPlan(wanted)
	middle := session.offsets[nothing]
	opens, err := plan.camera.At(middle - session.share/shareHalves)
	if err != nil {
		t.Fatal(err)
	}
	closes, err := plan.camera.At(middle + session.share/shareHalves)
	if err != nil {
		t.Fatal(err)
	}
	built := swarmBuilt(t)
	frames, err := gpu.NewFrames(gpu.Software)
	if err != nil {
		t.Fatal(err)
	}
	defer frames.Close()
	lens := cameraFilm.Pinhole{Wide: session.given.wide * maths.Degree}
	want, _, err := frames.DrawOpen(built.World(), wholeGeneration, *opens,
		*closes, lens, session.Times()[exposureMoving]+middle, namedWidth,
		namedHeight, exposureRays)
	if err != nil {
		t.Fatal(err)
	}
	if !sameLight(got.Light(), want) {
		t.Error("a shuttered frame is not the kernel's open draw across " +
			"the glimpse's share")
	}
	plain := frameOrSkipped(t, opening(t, swarmFilm(t, "")), exposureMoving)
	if sameLight(got.Light(), plain.Light()) {
		t.Error("an open shutter over a moving camera changed nothing")
	}
}

// and over a camera that is held still the shutter changes nothing: the
// frame on the path's hold is the frame with no shutter drawn at the
// glimpse's own time, to the bit. The swarm's rings turn, so the plain
// film is started the glimpse's offset later to be the same scene.
func TestSessionShutterOnTheGPUOverAHeldCameraIsThePlainFrame(t *testing.T) {
	open := opening(t, swarmFilm(t, shutterFilmKey(shutterHalf, singleRay)))
	plain := opening(t, swarmFilm(t, fmt.Sprintf("start: {value: %s, "+
		"unit: s}\n", secondsWritten(open.offsets[nothing]))))
	got := frameOrSkipped(t, open, exposureHeld)
	want := frameOrSkipped(t, plain, exposureHeld)
	if !sameLight(got.Light(), want.Light()) {
		t.Error("a shutter over a held camera is not the plain frame")
	}
	early := frameOrSkipped(t, opening(t, swarmFilm(t, "")), exposureHeld)
	if sameLight(got.Light(), early.Light()) {
		t.Error("a glimpse was drawn at its frame's time, not its own")
	}
}

// on the gpu a job's lens is drawn by the kernel: the frame is the
// per-call draw through the lens the session made, to the bit, and it is
// not the pinhole's frame
func TestSessionLensOnTheGPUIsThePerCallDrawThroughIt(t *testing.T) {
	wanted := jobFrom(t, lensJob("renderer: gpu\n"))
	session := opening(t, wanted)
	got := frameOrSkipped(t, session, nothing)
	given := asAsked(t, session)
	if err := (sequencePlan{}).exposing(&given, nothing); err != nil {
		t.Fatal(err)
	}
	built := swarmBuilt(t)
	world := built.World()
	given.footprint(world)
	want, err := gpu.Sampled(world, built.Eye(), *given.optics.opened,
		session.Times()[nothing], namedWidth, namedHeight, namedRays)
	if err != nil {
		t.Fatal(err)
	}
	if !sameLight(got.Light(), want) {
		t.Error("a job's lens on the gpu is not the per-call draw " +
			"through it")
	}
	pinhole := frameOrSkipped(t, opening(t, jobFrom(t, namedJob(
		"named: swarm", "renderer: gpu\n"))), nothing)
	if sameLight(got.Light(), pinhole.Light()) {
		t.Error("a job's lens drew the pinhole's frame")
	}
}

// shutterFilmKey is a job's shutter key alone, open for an angle with so
// many glimpses.
func shutterFilmKey(angle, glimpses int) string {
	return fmt.Sprintf("shutter: {angle: {value: %d, unit: deg}, "+
		"glimpses: %d}\n", angle, glimpses)
}

// a smoke's film keeps no drawer between frames, so its open shutter is
// drawn by one made for the frame: on the gpu, and not the frame with no
// shutter
func TestSessionShutterOnTheGPUIsDrawnForASmokeToo(t *testing.T) {
	smoke := fmt.Sprintf("scene: {smoke: %d}\nfilm: {width: %d, height: "+
		"%d, rays: %d}\nrenderer: gpu\nfps: %d\nduration: {value: 1, "+
		"unit: s}\n", gpuSmoke, sessionSmallWidth, sessionSmallHeight,
		exposureRays, shutterRate) + movingCamera(t)
	open := frameOrSkipped(t, opening(t, jobFrom(t, smoke+shutterFilmKey(
		shutterWhole, singleRay))), nothing)
	if open.Profile().Device == "" {
		t.Error("a smoke's open shutter was drawn on the cpu")
	}
	plain := opening(t, jobFrom(t, smoke+fmt.Sprintf("start: {value: %s, "+
		"unit: s}\n", secondsWritten(time.Second/shutterRate/shareHalves))))
	if sameLight(open.Light(), frameOrSkipped(t, plain, nothing).Light()) {
		t.Error("a smoke's open shutter over a moving camera changed " +
			"nothing")
	}
}
