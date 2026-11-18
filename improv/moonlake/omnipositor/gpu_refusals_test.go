package omnipositor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/hillside"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/render/colour"
)

// TestGPULightPlansBeforeDrawing checks that the searchlight and laser
// adapters refuse bad plans on the CPU, before anything is sent to the GPU.
func TestGPULightPlansBeforeDrawing(t *testing.T) {
	var base gpuLightBase
	if _, _, err := base.Searchlights(lights.Image{}, lights.Image{}, lights.SearchConfig{}, lights.SearchInput{}); err == nil || !strings.Contains(err.Error(), "search:") {
		t.Errorf("searchlights: %v", err)
	}
	if _, _, err := base.Lasers(lights.Image{}, lights.Image{}, lights.LaserConfig{}, lights.LaserInput{}); err == nil || !strings.Contains(err.Error(), "laser:") {
		t.Errorf("lasers: %v", err)
	}
}

// TestEnableExperimentalGPURefusesAStageWithNoRenderer checks that a stage
// that is not there, or has no renderer, is refused before a device is opened.
func TestEnableExperimentalGPURefusesAStageWithNoRenderer(t *testing.T) {
	var none *RendererLightStage
	if err := none.EnableExperimentalGPU(); err == nil || !strings.Contains(err.Error(), "uninitialized") {
		t.Errorf("nil stage: %v", err)
	}
	if err := (&RendererLightStage{width: 2, height: 2}).EnableExperimentalGPU(); err == nil || !strings.Contains(err.Error(), "uninitialized") {
		t.Errorf("stage with no renderer: %v", err)
	}
}

// TestGPUDevelopCacheCloseForgetsItsSize checks that closing a cache with no
// developer resets its size and tone, and can be done twice.
func TestGPUDevelopCacheCloseForgetsItsSize(t *testing.T) {
	cache := &gpuDevelopCache{w: 5, h: 6, tone: colour.Tone{Exposure: 2, Knee: 1}}
	cache.Close()
	if cache.w != 0 || cache.h != 0 || cache.tone != (colour.Tone{}) || cache.stage != nil {
		t.Errorf("cache after close: %+v", cache)
	}
	cache.Close()
}

// TestExperimentalGPURefusesWhiteBeforeOpeningADevice checks that the three
// public entry points refuse a nonzero white level without drawing.
func TestExperimentalGPURefusesWhiteBeforeOpeningADevice(t *testing.T) {
	cfg := testConfig(t)
	cfg.Develop.White = 1
	consume := func(Result) error { t.Fatal("a frame was delivered"); return nil }
	for name, run := range map[string]func() error{
		"RunExperimentalGPU":          func() error { return RunExperimentalGPU(context.Background(), cfg, consume) },
		"RunExperimentalGPUProfiled":  func() error { return RunExperimentalGPUProfiled(context.Background(), cfg, consume, nil) },
		"RunExperimentalGPUWithOcean": func() error { return RunExperimentalGPUWithOcean(context.Background(), cfg, consume, nil) },
	} {
		if err := run(); !errors.Is(err, ErrExperimentalGPUUnsupported) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestRenderSceneRefusesBeforeOpeningASession checks the refusals that come
// before the GPU session is opened: no context or consumer, a plan the job
// does not allow, and a context that has ended.
func TestRenderSceneRefusesBeforeOpeningASession(t *testing.T) {
	dir := t.TempDir()
	for name, text := range map[string]string{"job.yaml": sceneJobBase, "plan.yaml": scenePlanText} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	plan, wanted, err := ReadScenePlan(filepath.Join(dir, "plan.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	consume := func(SceneResult) error { t.Fatal("a frame was delivered"); return nil }
	stamp := hillside.Stamped("", "", "", "")
	var noContext context.Context
	if err := RenderScene(noContext, plan, wanted, stamp, false, consume); err == nil || !strings.Contains(err.Error(), "context and consumer") {
		t.Errorf("no context: %v", err)
	}
	if err := RenderScene(context.Background(), plan, wanted, stamp, false, nil); err == nil || !strings.Contains(err.Error(), "context and consumer") {
		t.Errorf("no consumer: %v", err)
	}
	bad := plan
	bad.Scale = 3
	if err := RenderScene(context.Background(), bad, wanted, stamp, false, consume); err == nil || !strings.Contains(err.Error(), "invalid range") {
		t.Errorf("bad plan: %v", err)
	}
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RenderScene(ended, plan, wanted, stamp, false, consume); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
}
