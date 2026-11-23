package omnipositor

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

func assetSettings(t *testing.T) settings.Settings {
	t.Helper()
	dir := t.TempDir()
	write := func(name string) string {
		path := filepath.Join(dir, name+".pfm")
		image := lights.Image{Width: 2, Height: 2, Light: []float64{1, 2, 3, 4, 5, 6, 0, 0, 0, 0, 0, 0}}
		if err := lights.Write(path, image, -1); err != nil {
			t.Fatal(err)
		}
		return path
	}
	return settings.Settings{
		Frame:  settings.Frame{Width: 2, Height: 2},
		Passes: settings.Passes{Nodisc: write("plate"), Disc: write("disc"), Mask: write("mask"), Water: write("water"), Trees: write("trees"), BackLeft: write("left"), BackRight: write("right"), Overhead: write("overhead")},
		Night:  settings.Night{FloodPasses: [2]string{"custom-left", "custom-right"}},
		Fire:   settings.Fire{Gain: .1, Decay: 1, FPS: 24, Pulse: 3, Flash: 8, Snap: .12, SwellFall: .3, BurstGain: 1, Top: .6},
		Output: settings.Output{MemoryCacheBytes: 96},
	}
}

func assetFireDir(t *testing.T, count int) string {
	t.Helper()
	dir := t.TempDir()
	for index := 0; index < count; index++ {
		image := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
		image.Light[0] = float64(index + 1)
		framePath := filepath.Join(dir, fmt.Sprintf("%03d.pfm", index))
		if err := lights.Write(framePath, image, -1); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestAssetStaticLoadingAndIdentity(t *testing.T) {
	s := assetSettings(t)
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Lighting.Plate.Light[0] != 1 || loaded.Lighting.Plate.Light[3] != 4 || loaded.Lighting.Plate.Light[6] != 0 {
		t.Fatal("PFM orientation changed")
	}
	if loaded.Lighting.Fire != nil || loaded.Lighting.Disc == nil || loaded.Lighting.Water == nil || loaded.Lighting.Trees == nil {
		t.Fatal("static assets missing")
	}
	for _, name := range []string{"custom-left", "custom-right", "overhead"} {
		if loaded.Lighting.Passes[name].Width != 2 {
			t.Fatalf("missing pass %s", name)
		}
	}
	ids := loaded.Identities()
	raw, err := os.ReadFile(s.Passes.Nodisc)
	if err != nil {
		t.Fatal(err)
	}
	if ids["passes.nodisc"].SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("identity does not hash original file bytes")
	}
	if len(ids) != 8 || ids["passes.nodisc"].SHA256 == "" || !filepath.IsAbs(ids["passes.nodisc"].Path) {
		t.Fatalf("identities: %+v", ids)
	}
	if loaded.MoonReference.Light[0] != loaded.Lighting.Disc.Light[0] {
		t.Fatal("disc not used as default moon reference")
	}
	if err := os.Remove(s.Passes.Nodisc); err != nil {
		t.Fatal(err)
	}
	stage, err := loaded.NewLightStage(lights.DefaultConfig())
	if err != nil || stage == nil {
		t.Fatalf("static assets reloaded: %v", err)
	}
	ids["passes.nodisc"] = AssetIdentity{}
	if loaded.Identities()["passes.nodisc"].SHA256 == "" {
		t.Fatal("identity map leaked")
	}
}

func TestAssetNamedDimensionsAndBadData(t *testing.T) {
	for _, test := range []struct {
		name, match string
		change      func(settings.Settings)
	}{
		{"wrong dimensions", "passes.water", func(s settings.Settings) {
			if err := lights.Write(s.Passes.Water, lights.Image{Width: 1, Height: 1, Light: []float64{1, 1, 1}}, -1); err != nil {
				t.Fatal(err)
			}
		}},
		{"truncated", "passes.mask", func(s settings.Settings) {
			if err := os.WriteFile(s.Passes.Mask, []byte("PF\n2 2\n-1\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"excessive header", "passes.disc", func(s settings.Settings) {
			if err := os.WriteFile(s.Passes.Disc, []byte("PF\n1000000000 1000000000\n-1\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := assetSettings(t)
			test.change(s)
			if _, err := LoadAssets(s); err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("error: %v", err)
			}
		})
	}
	s := assetSettings(t)
	s.Frame = settings.Frame{Width: 3840, Height: 2160}
	if _, err := LoadAssets(s); err == nil || !strings.Contains(err.Error(), "3840x2160") || !strings.Contains(err.Error(), "passes.nodisc") {
		t.Fatalf("4K dimensions: %v", err)
	}
}

func TestAssetOptionalMoonAndFireOff(t *testing.T) {
	s := assetSettings(t)
	s.MoonRef = &settings.OptionalPath{Path: s.Passes.Water}
	s.Fire.Frames = &settings.OptionalPath{Off: true}
	s.Fire.Swell = &settings.OptionalPath{Off: true}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Identities()["moon_ref"].SHA256 == "" {
		t.Fatal("explicit moon reference missing")
	}
	if _, err := loaded.NewLightStage(lights.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
}

func TestCacheFireSelectionBoundAndEviction(t *testing.T) {
	s := assetSettings(t)
	dir := assetFireDir(t, 4)
	s.Fire.Frames = &settings.OptionalPath{Path: dir}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CacheBytes() != 0 {
		t.Fatal("eager fire loading")
	}
	if _, err := loaded.NewLightStage(lights.DefaultConfig()); err != nil {
		t.Fatalf("bounded fire stage unavailable: %v", err)
	}
	request := LightRequest{Seconds: 10, Cues: cues.Frame{Fire: 1, FireSince: []float64{1.0 / 24}}}
	frames, err := loaded.LoadFire(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if frames[0].Count != 4 || len(frames[0].Images) != 1 || frames[0].Images[1].Light[0] != 2 || loaded.CacheBytes() != 96 {
		t.Fatalf("selection/cache: %+v, %d", frames[0], loaded.CacheBytes())
	}
	if err := os.Remove(filepath.Join(dir, "001.pfm")); err != nil {
		t.Fatal(err)
	}
	frames[0].Images[1].Light[0] = 999
	frames, err = loaded.LoadFire(context.Background(), request)
	if err != nil || frames[0].Images[1].Light[0] != 2 {
		t.Fatalf("cache hit/ownership: %v", err)
	}
	request.Cues.FireSince = []float64{2.0 / 24}
	if _, err := loaded.LoadFire(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if loaded.CacheBytes() > s.Output.MemoryCacheBytes {
		t.Fatal("cache exceeded bound")
	}
	request.Cues.FireSince = []float64{1.0 / 24}
	if _, err := loaded.LoadFire(context.Background(), request); err == nil {
		t.Fatal("evicted frame was retained")
	}
	if loaded.Identities()["fire.steady[1]"].SHA256 == "" {
		t.Fatal("selected fire identity missing")
	}
	if _, ok := loaded.Identities()["fire.steady[3]"]; ok {
		t.Fatal("unneeded fire was read")
	}
}

func TestCacheFirePlanningQuantizationAndInactive(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []LightRequest{
		{Cues: cues.Frame{Fire: 0, FireSince: []float64{0}}},
		{Cues: cues.Frame{Fire: 1, FireSince: []float64{10}}},
	} {
		frames, err := loaded.LoadFire(context.Background(), request)
		if err != nil || len(frames[0].Images) != 0 || loaded.CacheBytes() != 0 {
			t.Fatalf("inactive/out-of-range: %v", err)
		}
	}
	request := LightRequest{Cues: cues.Frame{Fire: 1, FireSince: []float64{0.02084, 0.02084}}}
	frames, err := loaded.LoadFire(context.Background(), request)
	if err != nil || len(frames[0].Images) != 1 || frames[0].Images[0].Width != 2 {
		t.Fatalf("quantized and deduplicated selection: %+v, %v", frames[0], err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := loaded.LoadFire(ctx, request); err == nil {
		t.Fatal("cancellation ignored")
	}
	request.Cues.FireSince = []float64{math.NaN()}
	if _, err := loaded.LoadFire(context.Background(), request); err == nil {
		t.Fatal("invalid cue accepted")
	}
}

func TestCachePercussiveSelection(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 4)}
	s.Fire.Swell = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	s.Fire.Burst = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	s.Fire.SteadyFrom = 1
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Seconds: 100, Cues: cues.Frame{Fire: 1, FireLit: .1, FireSince: []float64{.1, .2}}}
	frames, err := loaded.LoadFire(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames[0].Images) != 1 || frames[0].Images[3].Width != 2 || len(frames[1].Images) != 1 || frames[1].Images[2].Width != 2 || len(frames[2].Images) != 1 || frames[2].Images[2].Width != 2 {
		t.Fatalf("percussive selection: %+v", frames)
	}
	if loaded.CacheBytes() > s.Output.MemoryCacheBytes {
		t.Fatal("cache over budget")
	}
	request.Cues.FireLit = -1
	frames, err = loaded.LoadFire(context.Background(), request)
	if err != nil || len(frames[0].Images)+len(frames[1].Images)+len(frames[2].Images) != 0 {
		t.Fatalf("unlit selection: %+v, %v", frames, err)
	}
}

func TestAssetRejectsFireGaps(t *testing.T) {
	s := assetSettings(t)
	dir := assetFireDir(t, 3)
	if err := os.Remove(filepath.Join(dir, "001.pfm")); err != nil {
		t.Fatal(err)
	}
	s.Fire.Frames = &settings.OptionalPath{Path: dir}
	if _, err := LoadAssets(s); err == nil || !strings.Contains(err.Error(), "001.pfm") {
		t.Fatalf("gap: %v", err)
	}
	s.Fire.Frames = nil
	s.Fire.Swell = &settings.OptionalPath{Path: assetFireDir(t, 1)}
	if _, err := LoadAssets(s); err == nil || !strings.Contains(err.Error(), "fire.frames") {
		t.Fatalf("missing steady: %v", err)
	}
}

func TestAssetRejectsSpecialFilesWithoutBlocking(t *testing.T) {
	for _, kind := range []string{"fifo", "device", "directory"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.pfm")
			switch kind {
			case "fifo":
				if err := syscall.Mkfifo(path, 0600); err != nil {
					t.Fatal(err)
				}
			case "device":
				path = "/dev/zero"
			case "directory":
				path = t.TempDir()
			}
			done := make(chan error, 1)
			go func() { _, _, err := readAsset("special", path, 2, 2); done <- err }()
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "regular") {
					t.Fatalf("special file: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("special file open blocked")
			}
		})
	}
}

func TestAssetDetectsMutationWithRestoredModificationTime(t *testing.T) {
	path := assetSettings(t).Passes.Nodisc
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	file, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt([]byte{0}, before.Size()-1); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, before.ModTime(), before.ModTime()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) || before.Size() != after.Size() {
		t.Fatal("test did not restore size and mtime")
	}
	if sameChangeTime(before, after) {
		t.Fatal("mutation hidden by restored mtime")
	}
}

func TestAssetRejectsExcessPixelBytes(t *testing.T) {
	s := assetSettings(t)
	file, err := os.OpenFile(s.Passes.Nodisc, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("extra")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAssets(s); err == nil || !strings.Contains(err.Error(), "passes.nodisc") {
		t.Fatalf("trailing bytes: %v", err)
	}
}

func TestCacheRejectsChangedIdentityAfterEviction(t *testing.T) {
	for _, budget := range []int64{1, 96} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			s := assetSettings(t)
			s.Output.MemoryCacheBytes = budget
			dir := assetFireDir(t, 2)
			s.Fire.Frames = &settings.OptionalPath{Path: dir}
			a, err := LoadAssets(s)
			if err != nil {
				t.Fatal(err)
			}
			request := LightRequest{Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}}
			if _, err := a.LoadFire(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			original := a.Identities()["fire.steady[0]"]
			request.Cues.FireSince = []float64{1.0 / 24}
			if _, err := a.LoadFire(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "000.pfm")
			if err := lights.Write(path, lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}, -1); err != nil {
				t.Fatal(err)
			}
			request.Cues.FireSince = []float64{0}
			if _, err := a.LoadFire(context.Background(), request); err == nil || !strings.Contains(err.Error(), "identity") {
				t.Fatalf("changed identity: %v", err)
			}
			if a.Identities()["fire.steady[0]"] != original {
				t.Fatal("original identity replaced")
			}
			if a.CacheBytes() > budget {
				t.Fatal("cache exceeds budget")
			}
		})
	}
}

func TestAssetFireDirectoryEntryLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 4097; i++ {
		emptyPath := filepath.Join(dir, fmt.Sprint(i))
		if err := os.WriteFile(emptyPath, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := fireFrameCount(dir); err == nil || !strings.Contains(err.Error(), "4096") {
		t.Fatalf("entry limit: %v", err)
	}
}

func TestAssetSpotPasses(t *testing.T) {
	s := assetSettings(t)
	var text strings.Builder
	for _, side := range []string{"left", "right"} {
		for i := 0; i < 3; i++ {
			fmt.Fprintf(&text, "spot_%s_%d: {path: %q}\n", side, i, s.Passes.Nodisc)
		}
	}
	passesText := text.String()
	if err := yaml.Unmarshal([]byte(passesText), &s.Passes); err != nil {
		t.Fatal(err)
	}
	a, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []string{"left", "right"} {
		for i := 0; i < 3; i++ {
			name := fmt.Sprintf("spot-%s-%d", side, i)
			if a.Lighting.Passes[name].Width != 2 {
				t.Fatalf("missing %s", name)
			}
		}
	}
	if err := yaml.Unmarshal([]byte("spot_left_1: {path: /missing/spot.pfm}\n"), &s.Passes); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAssets(s); err == nil || !strings.Contains(err.Error(), "passes.spot_left_1") {
		t.Fatalf("named spot error: %v", err)
	}
}

func TestCacheRequestFireFrameLimitAndWorkingSet(t *testing.T) {
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	a, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Cues: cues.Frame{Fire: 1, FireSince: []float64{0, 1.0 / 24, 1.0 / 24}}}
	if _, err := a.loadFire(context.Background(), request, true, 1); err == nil || !strings.Contains(err.Error(), "maximum 1") {
		t.Fatalf("frame limit: %v", err)
	}
	if a.CacheBytes() != 0 || len(a.Identities()) != 8 {
		t.Fatal("over-limit request read frames")
	}
	frames, err := a.loadFire(context.Background(), request, true, 2)
	if err != nil || len(frames[0].Images) != 2 {
		t.Fatalf("distinct at limit: %v", err)
	}
	metrics := a.LastFireWorkingSet()
	if metrics.Frames != 2 || metrics.SelectedBytes != 192 || metrics.TransientBytes != 432 {
		t.Fatalf("working set: %+v", metrics)
	}
	if metrics.SelectedBytes <= a.CacheBytes() {
		t.Fatal("working set incorrectly bounded by cache")
	}
	request.Cues.FireSince = []float64{0, 1.0 / 24, 2.0 / 24}
	config := lights.DefaultConfig()
	config.MaxFireFrames = 2
	stage, err := a.NewLightStage(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Width, request.Height = 2, 2
	if _, err := stage.Light(context.Background(), request); err == nil || !strings.Contains(err.Error(), "maximum 2") {
		t.Fatalf("stage limit: %v", err)
	}
	if _, ok := a.Identities()["fire.steady[2]"]; ok {
		t.Fatal("stage read over-limit frame")
	}
}

func TestCacheOversizedFrameBypassesCache(t *testing.T) {
	s := assetSettings(t)
	s.Output.MemoryCacheBytes = 1
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 1)}
	loaded, err := LoadAssets(s)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := loaded.LoadFire(context.Background(), LightRequest{Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}})
	if err != nil || len(frames[0].Images) != 1 || loaded.CacheBytes() != 0 {
		t.Fatalf("oversized frame: %v, %d", err, loaded.CacheBytes())
	}
}
