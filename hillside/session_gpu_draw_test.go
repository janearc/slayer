//go:build draw && darwin

package hillside

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	cameraFilm "github.com/janearc/slayer/film"
	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/metal"
	scenes "github.com/janearc/slayer/scene/named"
)

// The scenes the gpu tests draw, each one the gpu draws whole: widow
// has a volume lit by a beam, and smoke 1 a floor, which it does not.
const (
	gpuSet   = "../sets/shoebox.yaml" // defined: a world the gpu draws
	gpuSmoke = 16                     // defined: the rocks, likewise
)

// drawnOrSkipped is the first frame of a job's session, or the test
// skipped where there is no Metal device.
func drawnOrSkipped(t *testing.T, wanted job.Job) Drawn {
	t.Helper()
	drawn, err := opening(t, wanted).Frame(context.Background(), nothing,
		Ask{})
	if errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	if err != nil {
		t.Fatal(err)
	}
	return drawn
}

// on the gpu, a job still of a set is Run's -gpu picture of it.
func TestSessionStillOnTheGPUIsRunsPicture(t *testing.T) {
	wanted := stillOf(t, gpuSet, "", "renderer: gpu\n")
	if drawnOrSkipped(t, wanted).Profile().Device == "" {
		t.Error("a job that says the gpu was drawn on the cpu")
	}
	want := ranPicture(t, gpuSet, append(sized(sessionWidth,
		sessionHeight), "-gpu")...)
	if !bytes.Equal(sessionPicture(t, wanted, Ask{}), want) {
		t.Error("the session and hillside -gpu drew different pictures")
	}
}

// a smoke takes the job's renderer, where a smoke sequence always drew
// on the cpu: on the gpu its frame has a device, on the cpu none.
func TestSessionSmokeTakesTheJobsRenderer(t *testing.T) {
	smoke := fmt.Sprintf("scene: {smoke: %d}\nfilm: {width: %d, height: "+
		"%d, rays: %d}\n", gpuSmoke, sessionSmallWidth,
		sessionSmallHeight, sessionRays)
	onGPU := drawnOrSkipped(t, jobFrom(t, smoke+"renderer: gpu\n"))
	if onGPU.Profile().Device == "" {
		t.Error("a smoke the job puts on the gpu was drawn on the cpu")
	}
	onCPU := drawnOrSkipped(t, jobFrom(t, smoke+"renderer: cpu\n"))
	if onCPU.Profile().Device != "" {
		t.Error("a smoke the job puts on the cpu was drawn on the gpu")
	}
}

// a named scene's film on the gpu, drawn through the session's retained
// drawer, is the per-call draw of each frame: the orbital from far, whose
// beams keep their own time, and the swarm, which keeps nothing.
func TestSessionNamedOnTheGPUIsThePerCallDraw(t *testing.T) {
	film := fmt.Sprintf("fps: %d\nduration: {value: 1, unit: s}\n"+
		"renderer: gpu\n", namedRate)
	for _, each := range []struct {
		scene string
		with  map[string]any
	}{
		{"named: orbital, with: {far: true}", map[string]any{"far": true}},
		{"named: swarm", nil},
	} {
		wanted := jobFrom(t, namedJob(each.scene, film))
		name, _ := wanted.Scene().Named()
		scene, err := scenes.Get(name)
		if err != nil {
			t.Fatal(err)
		}
		built, err := scene.Build(each.with, nil)
		if err != nil {
			t.Fatal(err)
		}
		lens := cameraFilm.Pinhole{Wide: scene.Defaults().Wide()}
		session := opening(t, wanted)
		for index, at := range session.Times() {
			drawn, err := session.Frame(context.Background(), index, Ask{})
			if errors.Is(err, metal.ErrNoDevice) {
				t.Skip("no Metal device")
			}
			if err != nil {
				t.Fatal(err)
			}
			built.Set(at)
			want, err := gpu.Sampled(built.World(), built.Eye(), lens, at,
				namedWidth, namedHeight, namedRays)
			if err != nil || !sameLight(drawn.Light(), want) {
				t.Errorf("%s frame %d at %v is not the per-call draw: %v",
					name, index, at, err)
			}
		}
	}
}

// on the gpu, a rig film's first frame is the per-call rigged still's
// picture at the same time.
func TestRigFilmOnTheGPUIsTheRiggedStill(t *testing.T) {
	rigged := onFixtureRigged(t, rigShutter, false)
	scene := "scene: {set: " + rigged + "}\nrenderer: gpu\n"
	film := opening(t, jobFrom(t, scene+rigFilm))
	if _, err := film.Frame(context.Background(), nothing,
		Ask{}); errors.Is(err, metal.ErrNoDevice) {
		t.Skip("no Metal device")
	}
	still := sessionPicture(t, jobFrom(t, scene), Ask{})
	if !bytes.Equal(filmPicture(t, film, nothing), still) {
		t.Error("the gpu's rig film frame is not its rigged still")
	}
}
