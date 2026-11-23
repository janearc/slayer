package omnipositor

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/lights"
)

func lightAssets(t *testing.T) lights.Assets {
	t.Helper()
	cfg := testConfig(t)
	mask := lights.Image{Width: 2, Height: 2, Light: []float64{1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0}}
	frames := [3]lights.FireFrames{{Count: 2, Images: map[int]lights.Image{0: cfg.MoonReference, 1: cfg.WaterMask}}}
	return lights.Assets{Plate: cfg.MoonReference, Mask: mask, Water: &cfg.WaterMask, Passes: map[string]lights.Image{"overhead": cfg.MoonReference, "flood-left": cfg.WaterMask}, Fire: &frames}
}

func TestLightRealRendererRequests(t *testing.T) {
	config := lights.DefaultConfig()
	config.Fire.Stops = 23
	assets := lightAssets(t)
	stage, err := NewLightStage(config, assets)
	if err != nil {
		t.Fatal(err)
	}
	var seam LightStage = stage
	renderer, err := lights.NewRenderer(config, assets)
	if err != nil {
		t.Fatal(err)
	}
	for _, seconds := range []float64{0, 0.0417, 0.3333} {
		for _, stops := range []float64{0, 2} {
			cue := cues.Frame{Master: 0.33335, White: -1, Magenta: -1, Fire: 0.8, FireLit: 0.0209, FireSince: []float64{0.0209}, FloodLeft: 0.2, Overhead: 0.1, Colour: [3]float64{0.2, 0.4, 0.6}}
			request := LightRequest{Index: 999, Seconds: seconds, Width: 2, Height: 2, Cues: cue, Stops: stops, WaterMask: *assets.Water}
			before := cue
			before.FireSince = append([]float64(nil), cue.FireSince...)
			got, err := seam.Light(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			complete := lights.CompleteCues(cue)
			complete.FireSince = append([]float64(nil), cue.FireSince...)
			want, err := renderer.Render(lights.Request{Seconds: seconds, Cues: &complete, Stops: &stops, Water: assets.Water})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, LightResult{Sky: want.Sky, Front: want.Front, Strobe: want.Strobe}) {
				t.Fatalf("request conversion at %v, stops %v: got %+v want %+v", seconds, stops, got, want)
			}
			if !reflect.DeepEqual(request.Cues, before) {
				t.Fatal("mutated request cues")
			}
			if got.Front.Light[6] <= 0 {
				t.Fatal("explicit fire frames did not reach front")
			}
		}
	}
}

func TestLightZeroCuesAreComplete(t *testing.T) {
	stage, err := NewLightStage(lights.DefaultConfig(), lightAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, WaterMask: testConfig(t).WaterMask}
	got, err := stage.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	for _, image := range []lights.Image{got.Sky, got.Front} {
		for _, value := range image.Light {
			if value != 0 {
				t.Fatal("complete zero-valued cues were merged with lighting defaults")
			}
		}
	}
	if got.Strobe != 0 {
		t.Fatal("complete zero-valued cues enabled strobes")
	}
}

func TestConfigAcceptsRealLightingWithoutRendering(t *testing.T) {
	config := testConfig(t)
	stage, err := NewLightStage(lights.DefaultConfig(), lightAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	config.Light = stage
	if err := config.validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLightFireUsesRequestStopsAndSelectedFrame(t *testing.T) {
	stage, err := NewLightStage(lights.DefaultConfig(), lightAssets(t))
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Fire: 1, FireSince: []float64{0}}, WaterMask: testConfig(t).WaterMask}
	first, err := stage.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Stops = 2
	exposed, err := stage.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first.Front, exposed.Front) {
		t.Fatal("request stops ignored")
	}
	request.Cues.FireSince = []float64{0.0209}
	next, err := stage.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(exposed.Front, next.Front) || next.Front.Light[0] != 0 || next.Front.Light[6] <= 0 {
		t.Fatal("selected fire frame ignored")
	}
}

func TestLightOwnsStaticAndFireAssetsAndResults(t *testing.T) {
	assets := lightAssets(t)
	stage, err := NewLightStage(lights.DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	request := LightRequest{Width: 2, Height: 2, Cues: cues.Frame{Master: 1, FloodLeft: 0.2, Overhead: 0.1, Fire: 1, FireSince: []float64{0}}, WaterMask: testConfig(t).WaterMask}
	want, err := stage.Light(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	assets.Plate.Light[0] = 999
	assets.Mask.Light[0] = 0
	assets.Water.Light[6] = 0
	delete(assets.Passes, "overhead")
	delete(assets.Fire[0].Images, 0)
	got, err := stage.Light(context.Background(), request)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("caller assets retained: %v", err)
	}
	got.Sky.Light[0], got.Front.Light[0] = 999, 999
	again, err := stage.Light(context.Background(), request)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatalf("result storage retained: %v", err)
	}
}

func TestLightConstructorPrevalidates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*lights.Config, *lights.Assets)
		match  string
	}{
		{"config", func(c *lights.Config, a *lights.Assets) { c.Focal = 0 }, "config"},
		{"plate", func(c *lights.Config, a *lights.Assets) { a.Plate = lights.Image{} }, "plate"},
		{"mask", func(c *lights.Config, a *lights.Assets) { a.Mask = lights.Image{} }, "mask"},
		{"water", func(c *lights.Config, a *lights.Assets) { a.Water = &lights.Image{} }, "water"},
		{"unused pass", func(c *lights.Config, a *lights.Assets) { a.Passes["unused"] = lights.Image{} }, "unused"},
		{"nonfinite pass", func(c *lights.Config, a *lights.Assets) {
			p := a.Passes["overhead"]
			p.Light = append([]float64(nil), p.Light...)
			p.Light[0] = math.NaN()
			a.Passes["overhead"] = p
		}, "overhead"},
		{"unused fire", func(c *lights.Config, a *lights.Assets) { a.Fire[0].Images[1] = lights.Image{} }, "fire steady frame 1"},
		{"fire count", func(c *lights.Config, a *lights.Assets) { a.Fire[0].Count = -1 }, "fire"},
		{"fire bound", func(c *lights.Config, a *lights.Assets) { c.MaxFireFrames = 1 }, "maximum"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, assets := lights.DefaultConfig(), lightAssets(t)
			tc.change(&config, &assets)
			stage, err := NewLightStage(config, assets)
			if err == nil || stage != nil || !strings.Contains(err.Error(), tc.match) {
				t.Fatalf("stage=%v error=%v", stage, err)
			}
		})
	}
}

func TestLightRequestValidationAndCancellation(t *testing.T) {
	assets := lightAssets(t)
	assets.Fire[0].Count = 3
	stage, err := NewLightStage(lights.DefaultConfig(), assets)
	if err != nil {
		t.Fatal(err)
	}
	base := LightRequest{Width: 2, Height: 2, WaterMask: testConfig(t).WaterMask}
	for _, tc := range []struct {
		name   string
		change func(*LightRequest)
		match  string
	}{
		{"dimensions", func(r *LightRequest) { r.Width = 1 }, "dimensions"},
		{"seconds", func(r *LightRequest) { r.Seconds = math.NaN() }, "seconds"},
		{"stops", func(r *LightRequest) { r.Stops = math.Inf(1) }, "stops"},
		{"water", func(r *LightRequest) { r.WaterMask = lights.Image{} }, "water"},
		{"cues", func(r *LightRequest) { r.Cues.Master = math.NaN() }, "cues"},
		{"missing pass", func(r *LightRequest) { r.Cues.FloodRight = 1 }, "flood-right"},
		{"missing fire", func(r *LightRequest) { r.Cues.Fire = 1; r.Cues.FireSince = []float64{2.0 / 24} }, "fire steady frame 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := base
			tc.change(&request)
			if _, err := stage.Light(context.Background(), request); err == nil || !strings.Contains(err.Error(), tc.match) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stage.Light(ctx, base); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := stage.Light(never, base); err == nil {
		t.Fatal("nil context accepted")
	}
	var absent *RendererLightStage
	if _, err := absent.Light(context.Background(), base); err == nil {
		t.Fatal("nil stage accepted")
	}
	if _, err := new(RendererLightStage).Light(context.Background(), base); err == nil {
		t.Fatal("uninitialized stage accepted")
	}
}
