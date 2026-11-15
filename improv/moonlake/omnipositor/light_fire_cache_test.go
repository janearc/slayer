package omnipositor

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

func TestCacheFireStageBorrowsWithoutExposingStorage(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 1)}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}}
	borrowed, err := loaded.loadFire(context.Background(), request, false)
	if err != nil {
		t.Fatal(err)
	}
	cached := loaded.lru.Front().Value.(assetCacheEntry).image
	if &borrowed[0].Images[0].Light[0] != &cached.Light[0] {
		t.Fatal("stage path copied cached frame")
	}
	owned, err := loaded.LoadFire(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if &owned[0].Images[0].Light[0] == &cached.Light[0] {
		t.Fatal("public loader exposed cache storage")
	}
}

func TestLightCachedPercussiveFire(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 4)}
	s.Fire.Swell = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	s.Fire.Burst = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	s.Fire.SteadyFrom = 1
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Lighting.Trees = nil
	stage, err := loaded.NewLightStage(lights.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, WaterMask: *loaded.Lighting.Water, Cues: cues.Frame{Fire: 1, FireLit: .1, FireSince: []float64{.1, .2}}}
	got, err := stage.Light(context.Background(), request)
	if err != nil || got.Front.Light[0] <= 0 {
		t.Fatalf("percussive fire: %v", err)
	}
	for _, name := range []string{"fire.steady[3]", "fire.swell[2]", "fire.burst[2]"} {
		if loaded.Identities()[name].SHA256 == "" {
			t.Fatalf("selected layer missing: %s", name)
		}
	}
	if loaded.CacheBytes() > s.Output.MemoryCacheBytes {
		t.Fatal("multiple selected layers exceeded retention budget")
	}
	request.Cues.FireLit = -1
	got, err = stage.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range got.Front.Light {
		if value != 0 {
			t.Fatal("unlit fire rendered")
		}
	}
}

func TestLightCachedFirePerRequest(t *testing.T) {
	for _, budget := range []int64{1, 96} {
		s := assetSettings(t)
		s.Output.MemoryCacheBytes = budget
		s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 4)}
		s.Fire.FPS = 12
		s.Fire.Gain = .7
		loaded, err := LoadAssets(s)
		if err != nil {
			t.Fatal(err)
		}
		loaded.Lighting.Trees = nil
		config := lights.DefaultConfig()
		stage, err := loaded.NewLightStage(config)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.CacheBytes() != 0 {
			t.Fatal("stage eagerly loaded fire")
		}
		request := LightRequest{Width: 2, Height: 2, WaterMask: *loaded.Lighting.Water, Cues: cues.Frame{Fire: 1, FireSince: []float64{1.0 / 12}}, Stops: 2}
		first, err := stage.Light(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		frames, err := loaded.LoadFire(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		config.Fire = loaded.fireConfig
		config.Fixtures.FloodPasses = s.Night.FloodPasses
		reference, err := lights.NewRenderer(config, loaded.Lighting)
		if err != nil {
			t.Fatal(err)
		}
		complete := lights.CompleteCues(request.Cues)
		want, err := reference.Render(lights.Request{Cues: &complete, Stops: &request.Stops, Water: &request.WaterMask, FireFrames: &frames})
		if err != nil || !reflect.DeepEqual(first.Front, want.Front) {
			t.Fatalf("loader/render planning mismatch: %v", err)
		}
		if first.Front.Light[0] <= 0 {
			t.Fatal("fire did not reach output")
		}
		request.Cues.FireSince = []float64{2.0 / 12}
		second, err := stage.Light(context.Background(), request)
		if err != nil || reflect.DeepEqual(first.Front, second.Front) {
			t.Fatalf("frame not refreshed: %v", err)
		}
		if loaded.CacheBytes() > budget || loaded.Lighting.Fire != nil {
			t.Fatal("fire retained outside bounded cache")
		}
		if _, ok := loaded.Identities()["fire.steady[3]"]; ok {
			t.Fatal("unselected frame loaded")
		}
		if err := os.Remove(filepath.Join(s.Fire.Frames.Path, "001.pfm")); err != nil {
			t.Fatal(err)
		}
		request.Cues.FireSince = []float64{1.0 / 12}
		if _, err := stage.Light(context.Background(), request); err == nil {
			t.Fatal("evicted frame retained by renderer")
		}
		request.Cues.Fire = 0
		if _, err := stage.Light(context.Background(), request); err != nil {
			t.Fatalf("inactive fire loaded: %v", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := stage.Light(ctx, request); err == nil {
			t.Fatal("cancelled request accepted")
		}
	}
}
