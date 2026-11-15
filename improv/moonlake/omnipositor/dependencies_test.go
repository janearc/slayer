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

func TestCacheFireSnapshotAliases(t *testing.T) {
	s := assetSettings(t)
	dir := assetFireDir(t, 3)
	s.Fire.Frames = &settings.OptionalPath{Path: dir}
	s.Fire.Swell = &settings.OptionalPath{Path: dir}
	a, err := LoadStaticAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Fire: 1, FireSince: []float64{1.0 / 24}}}
	first, err := a.prepareFire(context.Background(), request, 100, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if first.inputs["fire.swell[1]"].SHA256 == "" {
		t.Fatal("swell identity missing")
	}
	if !reflect.DeepEqual(first.CacheHits(), []string{"fire.swell[1]"}) {
		t.Fatalf("actual within-request alias hit missing: %v", first.CacheHits())
	}
	a.mu.Lock()
	delete(a.identities, "fire.swell[1]")
	a.mu.Unlock()
	second, err := a.prepareFire(context.Background(), request, 100, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if second.inputs["fire.swell[1]"].SHA256 == "" {
		t.Fatal("cache alias identity missing")
	}
	if !reflect.DeepEqual(second.CacheHits(), []string{"fire.steady[1]", "fire.swell[1]"}) {
		t.Fatalf("actual cache hits: %v", second.CacheHits())
	}
	frames := first.Frames()
	frames[0].Images[1].Light[0] = 999
	if first.Frames()[0].Images[1].Light[0] != 2 {
		t.Fatal("snapshot storage exposed")
	}
}

func TestCacheFireSnapshotEvictionIsNotHit(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	s.Fire.Swell = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	s.Fire.Burst = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	a, err := LoadStaticAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Fire: 1, FireLit: .1, FireSince: []float64{1.0 / 24}}}
	for i := 0; i < 2; i++ {
		snapshot, err := a.prepareFire(context.Background(), request, 100, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.frames[0].Images) == 0 || len(snapshot.frames[1].Images) == 0 || len(snapshot.frames[2].Images) == 0 {
			t.Fatal("test did not select all sequences")
		}
		if len(snapshot.CacheHits()) != 0 {
			t.Fatalf("evicted entry reported as hit: %v", snapshot.CacheHits())
		}
	}
}

func TestIdentityRejectsUnboundExecution(t *testing.T) {
	s := assetSettings(t)
	a, err := LoadStaticAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	base := Config{Width: 2, Height: 2, FPS: 24, From: 0, To: 2}
	for _, mutate := range []func(*Config){
		func(c *Config) {},
		func(c *Config) { c.MoonReference = lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)} },
		func(c *Config) { c.WaterMask = lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)} },
		func(c *Config) { c.TreeMask = &lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)} },
		func(c *Config) { c.Light = &RendererLightStage{} },
	} {
		c := base
		mutate(&c)
		builder, err := NewFrameIdentity(c, lights.DefaultConfig(), a, "unused", []byte("unbound score"), []byte("unbound cues"), "cpu", "device")
		if builder != nil || !errors.Is(err, ErrUnboundExecutionSnapshot) {
			t.Fatalf("unbound configuration accepted: %v", err)
		}
	}
	var builder FrameIdentity
	dependencies, err := builder.Prepare(context.Background(), LightRequest{})
	if dependencies.Fire != nil || !errors.Is(err, ErrUnboundExecutionSnapshot) {
		t.Fatalf("unbound preparation accepted: %v", err)
	}
}

func TestLightRejectsUnboundFireSnapshot(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	a, err := LoadStaticAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Fire: 1, FireSince: []float64{1.0 / 24}}}
	snapshot, err := a.prepareFire(context.Background(), request, 100, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := a.NewLightStage(lights.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	request.FireSnapshot = snapshot
	for _, water := range []lights.Image{{}, {Width: 2, Height: 2, Light: make([]float64, 12)}} {
		request.WaterMask = water
		if _, err := stage.Light(context.Background(), request); !errors.Is(err, ErrUnboundExecutionSnapshot) {
			t.Fatalf("unbound snapshot rendered: %v", err)
		}
	}
}
