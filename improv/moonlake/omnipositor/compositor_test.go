package omnipositor

import (
	"context"
	"errors"
	"image"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/develop"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/matte"
	"github.com/janearc/slayer/improv/moonlake/mist"
	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/render/frame"
	"go.uber.org/mock/gomock"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	show, err := cues.Parse([]byte("score:\n  beats:\n    times: [0, 1, 2, 3]\n"), []byte("cues:\n  - at_sec: 0\n    mist: 0.123456\n    master: 0.876543\n    overhead: 0.00001234\n    exposure: 0.123456\n"))
	if err != nil {
		t.Fatal(err)
	}
	moon := lights.Image{Width: 2, Height: 2, Light: []float64{4, 3, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1}}
	water := lights.Image{Width: 2, Height: 2, Light: []float64{0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0}}
	camera := ocean.DefaultConfig()
	return Config{From: 0, To: 3, FPS: 4, Width: 2, Height: 2, Show: show, Ocean: camera, Top: 3, Bottom: -3, Shore: 0, MoonRadius: 1.554, MoonReference: moon, WaterMask: water, MistLayers: 1, MistColour: [3]float64{0.4, 0.5, 0.6}, OverheadMist: 2500, StrobeMist: 0.25, Develop: develop.DefaultOptions()}
}

func noOcean(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error) {
	panic("unexpected ocean render")
}

func TestStretchCallsLightingAndConcreteFloatStages(t *testing.T) {
	cfg := testConfig(t)
	mock := NewMockLightStage(gomock.NewController(t))
	cfg.Light = mock
	sky := lights.Image{Width: 2, Height: 2, Light: []float64{4.00000003, 3, 2, 2, 2, 2, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6}}
	front := lights.Image{Width: 2, Height: 2, Light: []float64{0, 0, 0, 0, 0, 0, 0.3, 0.2, 0.1, 0, 0, 0}}
	sea := &oceantex.Texture{Width: 2, Height: 2, Light: [][3]float64{{0.11, 0.22, 0.33}, {0.44, 0.55, 0.66}, {0.77, 0.88, 0.99}, {1, 2, 3}}}
	var want []*frame.Frame
	var wantImages []*image.RGBA
	for i := 0; i < 3; i++ {
		seconds := float64(i) / 4
		cue, err := cfg.Show.At(seconds, cfg.FPS)
		if err != nil {
			t.Fatal(err)
		}
		cue = cue.Quantized()
		mock.EXPECT().Light(gomock.Any(), LightRequest{Index: i, Seconds: seconds, Width: 2, Height: 2, Cues: cue, Stops: cfg.Develop.Exposure + cue.Exposure, WaterMask: cfg.WaterMask}).Return(LightResult{Sky: sky, Front: front, Strobe: 0.123456}, nil)
		raw := matte.Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 4.00000003, 3, 2, 2, 2, 2}}
		seaImage := matte.Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{0.11, 0.22, 0.33, 0.44, 0.55, 0.66, 0.77, 0.88, 0.99, 1, 2, 3}}
		moon := matte.Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{1, 1, 1, 1, 1, 1, 4, 3, 2, 2, 2, 2}}
		water := matte.Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
		foreground := matte.Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{0.3, 0.2, 0.1, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
		combined, err := matte.Apply(raw, seaImage, matte.Config{Top: 3, Bottom: -3, Shore: 0, MoonRadius: cfg.MoonRadius, MoonRef: &moon, WaterMask: &water, Front: &foreground})
		if err != nil {
			t.Fatal(err)
		}
		moonMist := mist.Image(moon)
		waterMist := mist.Image(water)
		strength := math.Round((cue.Mist*cue.Master+cfg.StrobeMist*0.1235+cue.Overhead*cfg.OverheadMist)*1000) / 1000
		veiled, err := mist.Apply(mist.Image(combined), mist.Config{Top: 3, Bottom: -3, Shore: 0, Strength: strength, Seconds: seconds, Seed: cfg.MistSeed, Colour: cfg.MistColour, Layers: cfg.MistLayers, MoonRef: &moonMist, WaterMask: &waterMist})
		if err != nil {
			t.Fatal(err)
		}
		f := frame.New(2, 2)
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				j := ((1-y)*2 + x) * 3
				p := f.At(x, y)
				p.R, p.G, p.B, p.Cover = veiled.Pixels[j], veiled.Pixels[j+1], veiled.Pixels[j+2], 1
				f.Set(x, y, p)
			}
		}
		options := cfg.Develop
		options.Exposure += cue.Exposure
		picture, err := develop.Develop([]*frame.Frame{f}, options)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, f)
		wantImages = append(wantImages, picture)
	}
	oceanCalls, delivered := 0, 0
	err := run(context.Background(), cfg, func(result Result) error {
		if result.Index != delivered || result.Seconds != float64(delivered)/4 {
			t.Fatalf("timing: %+v", result)
		}
		if !reflect.DeepEqual(result.Linear, want[delivered]) || !reflect.DeepEqual(result.Image, wantImages[delivered]) {
			t.Fatalf("frame %d differs from concrete float stages", delivered)
		}
		delivered++
		return nil
	}, func(c ocean.Config, rgb ocean.RGB, moon *ocean.RGB) (*oceantex.Texture, error) {
		if c.Width != 2 || c.Height != 2 || c.Seconds != float64(oceanCalls)/4 {
			t.Fatalf("ocean config: %+v", c)
		}
		if rgb.Light[0] != float32(0.1) || rgb.Light[6] != 4 || moon.Light[6] != 4 {
			t.Fatalf("ocean orientation: %+v %+v", rgb, moon)
		}
		oceanCalls++
		return sea, nil
	})
	if err != nil || delivered != 3 || oceanCalls != 3 {
		t.Fatalf("err=%v delivered=%d ocean=%d", err, delivered, oceanCalls)
	}
}

func TestHalfOpenRangeExcludesEnd(t *testing.T) {
	cfg := testConfig(t)
	cfg.From, cfg.To, cfg.FPS = 4, 6, 4
	mock := NewMockLightStage(gomock.NewController(t))
	cfg.Light = mock
	front := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
	for _, seconds := range []float64{1, 1.25} {
		mock.EXPECT().Light(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, request LightRequest) (LightResult, error) {
			if request.Seconds != seconds || request.Index != int(seconds*4) {
				t.Fatalf("got frame %d at %v want %v", request.Index, request.Seconds, seconds)
			}
			return LightResult{Sky: cfg.MoonReference, Front: front}, nil
		})
	}
	count := 0
	err := run(context.Background(), cfg, func(Result) error { count++; return nil }, func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error) {
		return &oceantex.Texture{Width: 2, Height: 2, Light: make([][3]float64, 4)}, nil
	})
	if err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestAbsoluteFrameTimingAndStaticOceanAreChunkIndependent(t *testing.T) {
	renderChunk := func(from, to int) []Result {
		cfg := testConfig(t)
		cfg.From, cfg.To, cfg.FPS = from, to, 24
		cfg.Ocean.MoonAltitude = 5.75
		mock := NewMockLightStage(gomock.NewController(t))
		cfg.Light = mock
		front := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
		times := map[int]float64{7: 0.2917, 8: 0.3333, 9: 0.375}
		for index := from; index < to; index++ {
			seconds := times[index]
			cue, err := cfg.Show.At(seconds, cfg.FPS)
			if err != nil {
				t.Fatal(err)
			}
			cue = cue.Quantized()
			mock.EXPECT().Light(gomock.Any(), LightRequest{Index: index, Seconds: seconds, Width: 2, Height: 2, Cues: cue, Stops: cue.Exposure, WaterMask: cfg.WaterMask}).Return(LightResult{Sky: cfg.MoonReference, Front: front}, nil)
		}
		oceanIndex := from
		var results []Result
		err := run(context.Background(), cfg, func(result Result) error {
			index := from + len(results)
			if result.Index != index || result.Seconds != times[index] {
				t.Fatalf("absolute result timing: frame %d at %v", result.Index, result.Seconds)
			}
			results = append(results, result)
			return nil
		}, func(config ocean.Config, _ ocean.RGB, _ *ocean.RGB) (*oceantex.Texture, error) {
			if config.Seconds != times[oceanIndex] || config.MoonAltitude != 5.75 {
				t.Fatalf("ocean timing or static altitude changed: %+v", config)
			}
			oceanIndex++
			return &oceantex.Texture{Width: 2, Height: 2, Light: make([][3]float64, 4)}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != to-from || oceanIndex != to {
			t.Fatalf("wrong frame count: %d", len(results))
		}
		return results
	}
	whole := renderChunk(7, 10)
	chunk := renderChunk(8, 10)
	if !reflect.DeepEqual(whole[1:], chunk) {
		t.Fatal("frame output depends on chunk boundaries")
	}
}

func TestPublicRunRejectsMissingLightingWithoutRendering(t *testing.T) {
	cfg := testConfig(t)
	if err := Run(context.Background(), cfg, func(Result) error { return nil }); err == nil {
		t.Fatal("missing LightStage accepted")
	}
}

func TestBloomLimitIsValidatedBeforeLightingOrOcean(t *testing.T) {
	cfg := testConfig(t)
	cfg.Light = NewMockLightStage(gomock.NewController(t))
	cfg.Develop.Bloom = 100000
	if err := cfg.validate(); err != nil {
		t.Fatalf("CPU widest bloom rejected: %v", err)
	}
	cfg.Develop.Bloom = math.Nextafter(100000, math.Inf(1))
	err := run(context.Background(), cfg, func(Result) error { t.Fatal("consumed invalid bloom"); return nil }, noOcean)
	if err == nil || !strings.Contains(err.Error(), "config:") || !strings.Contains(err.Error(), "bloom") {
		t.Fatalf("expected named configuration error: %v", err)
	}
}

func TestValidationNeverRenders(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Config)
	}{
		{"range", func(c *Config) { c.To = c.From }}, {"negative", func(c *Config) { c.From = -1 }}, {"fps", func(c *Config) { c.FPS = 0 }}, {"nan", func(c *Config) { c.FPS = math.NaN() }}, {"infinite fps", func(c *Config) { c.FPS = math.Inf(1) }}, {"unrepresentable time", func(c *Config) { c.FPS = math.SmallestNonzeroFloat64 }}, {"width", func(c *Config) { c.Width = 0 }}, {"huge", func(c *Config) { c.Width = math.MaxInt }}, {"missing lighting", func(c *Config) { c.Light = nil }}, {"missing cues", func(c *Config) { c.Show = nil }}, {"empty water", func(c *Config) { c.WaterMask.Light = make([]float64, 12) }}, {"bad moon", func(c *Config) { c.MoonReference.Light[0] = math.NaN() }}, {"geometry", func(c *Config) { c.Shore = c.Top }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig(t)
			cfg.Light = NewMockLightStage(gomock.NewController(t))
			tc.change(&cfg)
			if err := run(context.Background(), cfg, func(Result) error { return nil }, noOcean); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}

func TestStageErrorsAndCancellation(t *testing.T) {
	sentinel := errors.New("broken")
	for _, stage := range []string{"lighting", "ocean", "matte", "mist", "develop", "consume", "cancel"} {
		t.Run(stage, func(t *testing.T) {
			cfg := testConfig(t)
			cfg.From, cfg.To = 7, 8
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mock := NewMockLightStage(gomock.NewController(t))
			cfg.Light = mock
			sky := cfg.MoonReference
			front := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
			mock.EXPECT().Light(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, LightRequest) (LightResult, error) {
				if stage == "cancel" {
					cancel()
				}
				if stage == "lighting" {
					return LightResult{}, sentinel
				}
				return LightResult{Sky: sky, Front: front}, nil
			})
			if stage == "matte" {
				cfg.MoonRadius = math.MaxFloat64
			}
			if stage == "mist" {
				cfg.MistColour = [3]float64{math.MaxFloat64, math.MaxFloat64, math.MaxFloat64}
			}
			if stage == "develop" {
				cfg.Develop.White = -1
			}
			render := func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error) {
				if stage == "ocean" {
					return nil, sentinel
				}
				if stage == "cancel" {
					t.Fatal("rendered after cancellation")
				}
				return &oceantex.Texture{Width: 2, Height: 2, Light: make([][3]float64, 4)}, nil
			}
			err := run(ctx, cfg, func(Result) error {
				if stage == "consume" {
					return sentinel
				}
				return nil
			}, render)
			if err == nil || !strings.Contains(err.Error(), "frame 7") {
				t.Fatalf("missing frame error: %v", err)
			}
			if stage == "cancel" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			} else if !strings.Contains(err.Error(), stage) {
				t.Fatal(err)
			}
			if stage == "lighting" || stage == "ocean" || stage == "consume" {
				if !errors.Is(err, sentinel) {
					t.Fatal(err)
				}
			}
		})
	}
	cfg := testConfig(t)
	cfg.Light = NewMockLightStage(gomock.NewController(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, cfg, func(Result) error { return nil }, noOcean); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestRunPipelineWithInjectedStagesKeepsTimingAndOrientation(t *testing.T) {
	cfg := testConfig(t)
	cfg.From, cfg.To = 0, 1
	cfg.Develop = develop.Options{Exposure: 0.75, Knee: 0.5, Top: 4}
	mock := NewMockLightStage(gomock.NewController(t))
	cfg.Light = mock
	cue, err := cfg.Show.At(0, cfg.FPS)
	if err != nil {
		t.Fatal(err)
	}
	cue = cue.Quantized()
	sky := lights.Image{Width: 2, Height: 2, Light: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	front := lights.Image{Width: 2, Height: 2, Light: []float64{100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111}}
	mock.EXPECT().Light(gomock.Any(), LightRequest{Index: 0, Seconds: 0, Width: 2, Height: 2, Cues: cue, Stops: cfg.Develop.Exposure + cue.Exposure, WaterMask: cfg.WaterMask}).Return(LightResult{Sky: sky, Front: front, Strobe: 0.125}, nil)
	var matteSeen, mistSeen, developSeen bool
	render := func(config ocean.Config, rgb ocean.RGB, moon *ocean.RGB) (*oceantex.Texture, error) {
		if config.Width != 2 || config.Height != 2 || config.Seconds != 0 {
			t.Fatalf("ocean config: %+v", config)
		}
		if rgb.Light[0] != 7 || rgb.Light[6] != 1 {
			t.Fatalf("sky orientation changed: %+v", rgb.Light)
		}
		if moon.Light[0] != 1 || moon.Light[6] != 4 {
			t.Fatalf("moon orientation changed: %+v", moon.Light)
		}
		return &oceantex.Texture{Width: 2, Height: 2, Light: [][3]float64{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}, {0.7, 0.8, 0.9}, {1, 1, 1}}}, nil
	}
	stages := stagePipeline{
		matte: func(raw, sea matte.Image, config matte.Config) (matte.Image, error) {
			if raw.Width != 2 || raw.Height != 2 || sea.Width != 2 || sea.Height != 2 || raw.Scale != -1 || sea.Scale != -1 {
				t.Fatalf("matte inputs: %+v %+v", raw, sea)
			}
			if raw.Pixels[0] != 7 || raw.Pixels[6] != 1 {
				t.Fatalf("matte saw top-down sky: %v", raw.Pixels)
			}
			if config.Front == nil || config.Front.Pixels[0] != 106 || config.Front.Pixels[6] != 100 {
				t.Fatalf("front orientation changed: %+v", config.Front)
			}
			matteSeen = true
			return matte.Image{Width: 2, Height: 2, Scale: -1, Pixels: []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}, nil
		},
		mist: func(image mist.Image, config mist.Config) (mist.Image, error) {
			if image.Width != 2 || image.Height != 2 || image.Pixels[0] != 1 || image.Pixels[9] != 10 {
				t.Fatalf("mist input: %+v", image)
			}
			if config.Seconds != 0 || config.MoonRef == nil || config.WaterMask == nil {
				t.Fatalf("mist config: %+v", config)
			}
			mistSeen = true
			return image, nil
		},
		develop: func(width, height int, options develop.Options, rgb []float32) (*image.RGBA, error) {
			if width != 2 || height != 2 || len(rgb) != 12 {
				t.Fatalf("develop input: %d %d %d", width, height, len(rgb))
			}
			if options.Exposure != 0.873 || options.Knee != 0.5 || options.Top != 4 {
				t.Fatalf("develop options: %+v", options)
			}
			developSeen = true
			return image.NewRGBA(image.Rect(0, 0, width, height)), nil
		},
	}
	var consumed bool
	if err := runPipeline(context.Background(), cfg, func(result Result) error {
		if result.Index != 0 || result.Seconds != 0 || !reflect.DeepEqual(result.Cues, cue) {
			t.Fatalf("result timing: %+v", result)
		}
		if result.Linear == nil || result.Image == nil {
			t.Fatalf("missing result images: %+v", result)
		}
		if result.Linear.Pix[0].R != 7 || result.Linear.Pix[0].G != 8 || result.Linear.Pix[0].B != 9 {
			t.Fatalf("linear image changed: %+v", result.Linear.Pix[0])
		}
		consumed = true
		return nil
	}, render, stages); err != nil {
		t.Fatal(err)
	}
	if !matteSeen || !mistSeen || !developSeen || !consumed {
		t.Fatalf("stage seam not exercised: matte=%v mist=%v develop=%v consumed=%v", matteSeen, mistSeen, developSeen, consumed)
	}
}

func TestRunExperimentalGPURejectsUnsupportedWhite(t *testing.T) {
	cfg := testConfig(t)
	cfg.From, cfg.To = 0, 1
	cfg.Develop.White = 1
	cfg.Light = NewMockLightStage(gomock.NewController(t))
	if err := RunExperimentalGPU(context.Background(), cfg, func(Result) error { t.Fatal("consume called"); return nil }); err == nil || !strings.Contains(err.Error(), "unsupported") || !strings.Contains(err.Error(), "white") {
		t.Fatalf("unexpected error: %v", err)
	}
}
