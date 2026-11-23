package omnipositor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/omnipositor/identity"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/settings"
)

// never is the stop of a frame that does not stop: no context at all,
// which the snapshot and the light stage refuse.
var never context.Context

func executionBinary(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := identity.HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func executionDevice() string { return "cpu/" + runtime.GOOS + "/" + runtime.GOARCH }

func executionConfig(t *testing.T) Config {
	c := testConfig(t)
	c.Show, c.Light, c.TreeMask = nil, nil, nil
	c.MoonReference, c.WaterMask = lights.Image{}, lights.Image{}
	return c
}

func TestExecutionFireBudgetNormalization(t *testing.T) {
	s := assetSettings(t)
	s.Output.MemoryCacheBytes = 2 * lights.DefaultMaxFireBytes
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 1)}
	var baseline identity.Key
	for i, configured := range []int64{lights.DefaultMaxFireBytes, s.Output.MemoryCacheBytes, 96} {
		l := lights.DefaultConfig()
		l.MaxFireBytes = configured
		e, err := NewExecutionSnapshot(context.Background(), s, executionConfig(t), l, []byte("score: {beats: {times: [0, 1, 2, 3]}}"), []byte("cues: [{at_sec: 0, fire: 1}]"), executionBinary(t), "cpu", executionDevice())
		if err != nil {
			t.Fatal(err)
		}
		expected := lights.FireByteBudget(configured, s.Output.MemoryCacheBytes)
		if e.stage.maxFireBytes != expected {
			t.Fatalf("preparation budget: got %d, want %d", e.stage.maxFireBytes, expected)
		}
		rendererBudget := reflect.ValueOf(e.stage.renderer).Elem().FieldByName("config").FieldByName("MaxFireBytes").Int()
		if rendererBudget != expected {
			t.Fatalf("renderer budget: got %d, want %d", rendererBudget, expected)
		}
		var recipe struct{ Lighting lights.Config }
		if err := json.Unmarshal(e.spec.Settings, &recipe); err != nil {
			t.Fatal(err)
		}
		if recipe.Lighting.MaxFireBytes != expected {
			t.Fatalf("identity budget: got %d, want %d", recipe.Lighting.MaxFireBytes, expected)
		}
		prepared, err := e.PrepareFrame(context.Background(), 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(prepared.request.FireSnapshot.frames[0].Images) == 0 {
			t.Fatal("selected fire missing")
		}
		if i == 0 {
			baseline = prepared.Key()
		} else if i == 1 && prepared.Key() != baseline {
			t.Fatal("equivalent effective budgets changed identity")
		} else if i == 2 && prepared.Key() == baseline {
			t.Fatal("different effective budget shared identity")
		}
	}
}

func TestIdentityOwnedExecution(t *testing.T) {
	s := assetSettings(t)
	c := executionConfig(t)
	score := []byte("score:\n  beats:\n    times: [0, 1, 2, 3]\n")
	sheet := []byte("cues: []\n")
	makeSnapshot := func(s settings.Settings, c Config, l lights.Config, score, sheet []byte) *ExecutionSnapshot {
		t.Helper()
		e, err := NewExecutionSnapshot(context.Background(), s, c, l, score, sheet, executionBinary(t), "cpu", executionDevice())
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	base := makeSnapshot(s, c, lights.DefaultConfig(), score, sheet)
	first, err := base.PrepareFrame(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*settings.Settings, *Config, *lights.Config, *[]byte, *[]byte){
		func(s *settings.Settings, c *Config, l *lights.Config, a, b *[]byte) { c.Develop.Exposure++ },
		func(s *settings.Settings, c *Config, l *lights.Config, a, b *[]byte) { l.HazeGain++ },
		func(s *settings.Settings, c *Config, l *lights.Config, a, b *[]byte) {
			copied := append([]byte(nil), (*a)...)
			*a = append(copied, '\n')
		},
		func(s *settings.Settings, c *Config, l *lights.Config, a, b *[]byte) {
			*b = []byte("cues: [{at_sec: 0, exposure: 1}]\n")
		},
	} {
		ss, cc, ll, a, b := s, c, lights.DefaultConfig(), score, sheet
		change(&ss, &cc, &ll, &a, &b)
		other, err := makeSnapshot(ss, cc, ll, a, b).PrepareFrame(context.Background(), 0)
		if err != nil {
			t.Fatal(err)
		}
		if first.Key().String() == other.Key().String() {
			t.Fatal("changed execution shared identity")
		}
	}
	score[0] = '!'
	sheet[0] = '!'
	if err := os.Remove(s.Passes.Disc); err != nil {
		t.Fatal(err)
	}
	again, err := base.PrepareFrame(context.Background(), 0)
	if err != nil || again.Key() != first.Key() {
		t.Fatalf("owned execution changed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := base.PrepareFrame(ctx, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := base.PrepareFrame(never, 0); err == nil {
		t.Fatal("nil context accepted")
	}
}

func TestIdentityRejectsRuntimeClaims(t *testing.T) {
	s := assetSettings(t)
	hash := executionBinary(t)
	for _, claim := range [][3]string{
		{sourceDigest(nil), "cpu", executionDevice()},
		{hash, "gpu", executionDevice()},
		{hash, "cpu", "cpu"},
		{hash, "cpu", "cpu/other/architecture"},
		{"", "", ""},
	} {
		e, err := NewExecutionSnapshot(context.Background(), s, executionConfig(t), lights.DefaultConfig(), []byte("score: {}"), []byte("cues: []"), claim[0], claim[1], claim[2])
		if e != nil || !errors.Is(err, ErrUnboundExecutionSnapshot) {
			t.Fatalf("unsupported runtime claim accepted: %v", err)
		}
	}
	e, err := NewExecutionSnapshot(context.Background(), s, executionConfig(t), lights.DefaultConfig(), []byte("score: {}"), []byte("cues: []"), hash, "cpu", executionDevice())
	if err != nil {
		t.Fatal(err)
	}
	if e.spec.BinarySHA256 != hash || e.spec.Backend != "cpu" || e.spec.Device != executionDevice() {
		t.Fatal("identity not bound to executable and CPU runtime")
	}
}

func TestIdentityRejectsCallerExecutionObjects(t *testing.T) {
	for _, change := range []func(*Config){
		func(c *Config) { c.Show = testConfig(t).Show },
		func(c *Config) { c.Light = &RendererLightStage{} },
		func(c *Config) { c.MoonReference = testConfig(t).MoonReference },
		func(c *Config) { c.WaterMask = testConfig(t).WaterMask },
		func(c *Config) { c.TreeMask = &lights.Image{} },
	} {
		c := executionConfig(t)
		change(&c)
		e, err := NewExecutionSnapshot(context.Background(), assetSettings(t), c, lights.DefaultConfig(), []byte("score: {}"), []byte("cues: []"), executionBinary(t), "cpu", executionDevice())
		if e != nil || !errors.Is(err, ErrUnboundExecutionSnapshot) {
			t.Fatalf("caller execution accepted: %v", err)
		}
	}
}

func TestIdentityPreparedFrameOwnership(t *testing.T) {
	var p PreparedFrame
	if err := p.Render(context.Background(), func(Result) error { return nil }); !errors.Is(err, ErrUnboundExecutionSnapshot) {
		t.Fatalf("zero frame accepted: %v", err)
	}
	s := assetSettings(t)
	s.Fire.Frames = &settings.OptionalPath{Path: assetFireDir(t, 3)}
	e, err := NewExecutionSnapshot(context.Background(), s, executionConfig(t), lights.DefaultConfig(), []byte("score: {beats: {times: [0, 1, 2, 3]}}"), []byte("cues: [{at_sec: 0, fire: 1}]"), executionBinary(t), "cpu", executionDevice())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := e.PrepareFrame(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared.request.FireSnapshot.frames[0].Images) == 0 {
		t.Fatal("selected fire missing")
	}
	for _, change := range []func(*LightRequest){
		func(r *LightRequest) { r.Seconds++ },
		func(r *LightRequest) { r.Stops++ },
		func(r *LightRequest) { r.Index++ },
		func(r *LightRequest) { r.Cues.Fire = 0 },
		func(r *LightRequest) { r.WaterMask = lights.Image{} },
		func(r *LightRequest) { r.Width++ },
	} {
		request := prepared.request
		request.FireSnapshot = nil
		change(&request)
		if _, err := (preparedLight{p: prepared}).Light(context.Background(), request); !errors.Is(err, ErrUnboundExecutionSnapshot) {
			t.Fatalf("mismatched request accepted: %v", err)
		}
	}
	e.prepareMu.Lock()
	if _, err := e.PrepareFrame(context.Background(), 0); !errors.Is(err, lights.ErrFireResources) {
		t.Fatalf("overlap preparation: %v", err)
	}
	e.prepareMu.Unlock()
	e.renderMu.Lock()
	if err := prepared.Render(context.Background(), func(Result) error { return nil }); !errors.Is(err, lights.ErrFireResources) {
		t.Fatalf("overlap rendering: %v", err)
	}
	e.renderMu.Unlock()
}

func TestIdentityStaticDependencyChanges(t *testing.T) {
	for _, name := range []string{"moon", "water", "trees", "plate", "pass"} {
		t.Run(name, func(t *testing.T) {
			s := assetSettings(t)
			makeFrame := func() *PreparedFrame {
				t.Helper()
				e, err := NewExecutionSnapshot(context.Background(), s, executionConfig(t), lights.DefaultConfig(), []byte("score: {}"), []byte("cues: []"), executionBinary(t), "cpu", executionDevice())
				if err != nil {
					t.Fatal(err)
				}
				p, err := e.PrepareFrame(context.Background(), 0)
				if err != nil {
					t.Fatal(err)
				}
				return p
			}
			first := makeFrame()
			path := map[string]string{"moon": s.Passes.Disc, "water": s.Passes.Water, "trees": s.Passes.Trees, "plate": s.Passes.Nodisc, "pass": s.Passes.BackLeft}[name]
			changed := lights.Image{Width: 2, Height: 2, Light: []float64{2, 2, 3, 4, 5, 6, 0, 0, 0, 0, 0, 0}}
			if err := lights.Write(path, changed, -1); err != nil {
				t.Fatal(err)
			}
			second := makeFrame()
			if first.Key() == second.Key() {
				t.Fatal("static image change shared identity")
			}
			retained, err := first.owner.PrepareFrame(context.Background(), 0)
			if err != nil || retained.Key() != first.Key() {
				t.Fatalf("static snapshot changed: %v", err)
			}
		})
	}
}
