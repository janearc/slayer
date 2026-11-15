package omnipositor

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

func TestAssetFireByteBudgetBeforeLoading(t *testing.T) {
	s := assetSettings(t)
	s.Output.MemoryCacheBytes = 1
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	config := lights.DefaultConfig()
	config.MaxFireBytes = 96
	stage, err := loaded.NewLightStage(config)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, WaterMask: *loaded.Lighting.Water, Cues: cues.Frame{Fire: 1, FireSince: []float64{0, 1.0 / 24}}}
	before := loaded.Identities()
	result, err := stage.Light(context.Background(), request)
	if !errors.Is(err, lights.ErrFireResources) || len(result.Front.Light) != 0 {
		t.Fatalf("resource refusal: %+v, %v", result, err)
	}
	if !reflect.DeepEqual(before, loaded.Identities()) || loaded.CacheBytes() != 0 || loaded.LastFireWorkingSet().Frames != 0 {
		t.Fatal("refused request loaded partial frames")
	}
	request.Cues.FireSince = []float64{0, 0, 0}
	if _, err := stage.Light(context.Background(), request); err != nil {
		t.Fatalf("single frame with smaller retention cache: %v", err)
	}
	if loaded.CacheBytes() != 0 || loaded.LastFireWorkingSet().SelectedBytes != 96 {
		t.Fatal("working-set and retention budgets conflated")
	}
	config.MaxFireBytes = 95
	stage, err = loaded.NewLightStage(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stage.Light(context.Background(), request); !errors.Is(err, lights.ErrFireResources) {
		t.Fatalf("oversized single frame: %v", err)
	}
}

func TestAssetFireConcurrentRefusal(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 1)}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := loaded.NewLightStage(lights.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	loaded.fireRequestMu.Lock()
	defer loaded.fireRequestMu.Unlock()
	request := LightRequest{Width: 2, Height: 2, WaterMask: *loaded.Lighting.Water, Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}}
	if _, err := stage.Light(context.Background(), request); !errors.Is(err, lights.ErrFireResources) {
		t.Fatalf("stage concurrent refusal: %v", err)
	}
	frames, err := loaded.LoadFire(context.Background(), request)
	if !errors.Is(err, lights.ErrFireResources) || frames[0].Images != nil {
		t.Fatalf("loader concurrent refusal: %+v, %v", frames, err)
	}
	if loaded.CacheBytes() != 0 {
		t.Fatal("concurrent request loaded frames")
	}
}
