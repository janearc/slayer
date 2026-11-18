package hillside

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/slayer/gpu"
	"github.com/janearc/slayer/job"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/scene/thing"
)

// wireJob is a job of smoke one saying a renderer and a film mode, read
// from a file of its own.
func wireJob(t *testing.T, renderer, mode string) job.Job {
	t.Helper()
	path := filepath.Join(t.TempDir(), "job.yaml")
	text := "$schema: https://github.com/janearc/slayer/schemas/job-v1.json\n" +
		"scene: {smoke: 1}\nrenderer: " + renderer + "\n" +
		"film: {width: 16, height: 16, mode: " + mode + "}\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	wanted, err := job.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	return wanted
}

// TestWireIsRefusedOnTheGPU holds that a wire asked of the gpu is
// refused by name wherever a draw could reach the gpu: a job that says
// renderer gpu, a draw on the gpu, a composite pass and a retained
// drawer. Nothing is drawn some other way.
func TestWireIsRefusedOnTheGPU(t *testing.T) {
	if _, err := jobAsked(wireJob(t, "gpu", "wire")); !errors.Is(err,
		errWireOnGPU) {
		t.Errorf("a job's wire on the gpu gave %v", err)
	}
	for name, given := range map[string]asked{
		"gpu":       {wire: true, onGPU: true},
		"composite": {wire: true, composite: &gpu.Frames{}},
	} {
		_, _, err := drawMeasured(thing.NewWorld(), given, 0)
		if !errors.Is(err, errWireOnGPU) {
			t.Errorf("%s: a wire gave %v", name, err)
		}
	}
	_, _, err := asked{wire: true}.drawnBy(nil, thing.NewWorld(),
		wholeGeneration, maths.Subspace{})
	if !errors.Is(err, errWireOnGPU) {
		t.Errorf("a retained drawer's wire gave %v", err)
	}
}

// anti-test of TestWireIsRefusedOnTheGPU
// TestWirePassesOnTheCPUAndLightOnTheGPU holds that the refusal is the
// wire's on the gpu and nothing else's: a wire on the cpu and light on
// the gpu both pass, and a job's film mode reaches what is asked.
func TestWirePassesOnTheCPUAndLightOnTheGPU(t *testing.T) {
	for _, given := range []asked{{wire: true}, {onGPU: true}} {
		if err := wireRefusal(given); err != nil {
			t.Errorf("%+v was refused: %v", given.wire, err)
		}
	}
	for _, each := range []struct {
		renderer, mode string
		wire           bool
	}{{"cpu", "wire", true}, {"gpu", "light", false}} {
		given, err := jobAsked(wireJob(t, each.renderer, each.mode))
		if err != nil || given.wire != each.wire {
			t.Errorf("%s on %s: wire %v, %v", each.mode, each.renderer,
				given.wire, err)
		}
	}
}

// TestWireSessionDevelopsThroughThePlainTone holds that a wire's session
// gives the plain tone its light is made for, whatever the scene's.
func TestWireSessionDevelopsThroughThePlainTone(t *testing.T) {
	scene := colour.Tone{Exposure: 7, Knee: 0.5, Bloom: 2}
	wanted := wireJob(t, "cpu", "wire")
	if got := newSession(nil, asked{wire: true}, wanted,
		scene).Tone(); got != wireTone {
		t.Errorf("a wire develops through %+v", got)
	}
}

// anti-test of TestWireSessionDevelopsThroughThePlainTone
// TestLightSessionKeepsTheScenesTone holds that a session of light keeps
// the scene's own tone.
func TestLightSessionKeepsTheScenesTone(t *testing.T) {
	scene := colour.Tone{Exposure: 7, Knee: 0.5, Bloom: 2}
	wanted := wireJob(t, "cpu", "light")
	if got := newSession(nil, asked{}, wanted, scene).Tone(); got != scene {
		t.Errorf("light develops through %+v", got)
	}
}
