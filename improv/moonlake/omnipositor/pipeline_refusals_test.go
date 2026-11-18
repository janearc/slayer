package omnipositor

import (
	"context"
	"errors"
	"image"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/slayer/improv/moonlake/develop"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/matte"
	"github.com/janearc/slayer/improv/moonlake/mist"
	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/oceantex"
)

// funcLight is a light stage that is a function.
type funcLight func(context.Context, LightRequest) (LightResult, error)

// Light calls the function.
func (f funcLight) Light(ctx context.Context, r LightRequest) (LightResult, error) { return f(ctx, r) }

// stubStages are the pieces of a pipeline that a test can break one at a time.
type stubStages struct {
	light   func(context.Context, LightRequest) (LightResult, error)
	ocean   func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error)
	matte   func(matte.Image, matte.Image, matte.Config) (matte.Image, error)
	mist    func(mist.Image, mist.Config) (mist.Image, error)
	develop func(int, int, develop.Options, []float32) (*image.RGBA, error)
	consume func(Result) error
}

// goodStubs returns stages that all succeed on a two by two frame.
func goodStubs(cfg Config) stubStages {
	flat := lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
	return stubStages{
		light: func(context.Context, LightRequest) (LightResult, error) {
			return LightResult{Sky: cfg.MoonReference, Front: flat, Strobe: 0.5}, nil
		},
		ocean: func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error) {
			return &oceantex.Texture{Width: 2, Height: 2, Light: make([][3]float64, 4)}, nil
		},
		matte: func(sky, _ matte.Image, _ matte.Config) (matte.Image, error) { return sky, nil },
		mist:  func(in mist.Image, _ mist.Config) (mist.Image, error) { return in, nil },
		develop: func(w, h int, o develop.Options, rgb []float32) (*image.RGBA, error) {
			return image.NewRGBA(image.Rect(0, 0, w, h)), nil
		},
		consume: func(Result) error { return nil },
	}
}

// runStubs runs the pipeline for frame 7 with the stub stages and profile.
func runStubs(ctx context.Context, cfg Config, s stubStages, profile *Profile) error {
	cfg.From, cfg.To = 7, 8
	cfg.Light = funcLight(s.light)
	return runPipelineProfiled(ctx, cfg, s.consume, s.ocean, stagePipeline{matte: s.matte, mist: s.mist, develop: s.develop}, profile)
}

// TestPipelineStubsSucceed checks that the stub pipeline runs, so that each
// refusal below is caused by the one stage it breaks. It also checks what
// reaches the consumer for frame 7 at 4 frames a second.
func TestPipelineStubsSucceed(t *testing.T) {
	cfg := testConfig(t)
	s := goodStubs(cfg)
	var got Result
	s.consume = func(r Result) error { got = r; return nil }
	if err := runStubs(context.Background(), cfg, s, nil); err != nil {
		t.Fatal(err)
	}
	if got.Index != 7 || got.Seconds != 1.75 || got.Image == nil || got.Linear == nil {
		t.Errorf("result %+v", got)
	}
}

// TestPipelineStageFailures checks that a failure in each stage, or a result
// that stage returns that cannot be used, stops the run and names the stage.
func TestPipelineStageFailures(t *testing.T) {
	sentinel := errors.New("stage broke")
	nan := float32(math.NaN())
	for _, c := range []struct {
		name   string
		break_ func(s *stubStages, cfg Config)
		want   string
	}{
		{"light fails", func(s *stubStages, _ Config) {
			s.light = func(context.Context, LightRequest) (LightResult, error) { return LightResult{}, sentinel }
		}, "lighting: stage broke"},
		{"sky of the wrong size", func(s *stubStages, _ Config) {
			s.light = func(context.Context, LightRequest) (LightResult, error) {
				return LightResult{Sky: lights.Image{Width: 1, Height: 1, Light: make([]float64, 3)}, Front: lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}}, nil
			}
		}, "lighting sky"},
		{"front of the wrong size", func(s *stubStages, cfg Config) {
			s.light = func(context.Context, LightRequest) (LightResult, error) {
				return LightResult{Sky: cfg.MoonReference, Front: lights.Image{}}, nil
			}
		}, "lighting front"},
		{"negative strobe", func(s *stubStages, cfg Config) {
			s.light = func(context.Context, LightRequest) (LightResult, error) {
				return LightResult{Sky: cfg.MoonReference, Front: lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}, Strobe: -1}, nil
			}
		}, "invalid strobe level"},
		{"ocean fails", func(s *stubStages, _ Config) {
			s.ocean = func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error) { return nil, sentinel }
		}, "ocean: stage broke"},
		{"ocean of the wrong size", func(s *stubStages, _ Config) {
			s.ocean = func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error) {
				return &oceantex.Texture{Width: 1, Height: 1, Light: make([][3]float64, 1)}, nil
			}
		}, "invalid bottom-up ocean dimensions"},
		{"matte fails", func(s *stubStages, _ Config) {
			s.matte = func(matte.Image, matte.Image, matte.Config) (matte.Image, error) { return matte.Image{}, sentinel }
		}, "matte: stage broke"},
		{"matte gives NaN", func(s *stubStages, cfg Config) {
			s.matte = func(sky, _ matte.Image, _ matte.Config) (matte.Image, error) {
				bad := sky
				bad.Pixels = append([]float32(nil), sky.Pixels...)
				bad.Pixels[0] = nan
				return bad, nil
			}
		}, "nonfinite float stage output"},
		{"mist fails", func(s *stubStages, _ Config) {
			s.mist = func(mist.Image, mist.Config) (mist.Image, error) { return mist.Image{}, sentinel }
		}, "mist: stage broke"},
		{"mist gives NaN", func(s *stubStages, _ Config) {
			s.mist = func(in mist.Image, _ mist.Config) (mist.Image, error) {
				in.Pixels = append([]float32(nil), in.Pixels...)
				in.Pixels[1] = nan
				return in, nil
			}
		}, "nonfinite float stage output"},
		{"develop fails", func(s *stubStages, _ Config) {
			s.develop = func(int, int, develop.Options, []float32) (*image.RGBA, error) { return nil, sentinel }
		}, "develop: stage broke"},
		{"consume fails", func(s *stubStages, _ Config) {
			s.consume = func(Result) error { return sentinel }
		}, "consume: stage broke"},
	} {
		cfg := testConfig(t)
		s := goodStubs(cfg)
		c.break_(&s, cfg)
		err := runStubs(context.Background(), cfg, s, nil)
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "frame 7") {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// TestPipelineStopsWhenContextEnds checks that a context that ends inside
// each stage is seen before the next stage starts, and that the next stage is
// not run.
func TestPipelineStopsWhenContextEnds(t *testing.T) {
	order := []string{"light", "ocean", "matte", "mist", "develop", "consume"}
	for at, name := range order {
		cfg := testConfig(t)
		s := goodStubs(cfg)
		ctx, cancel := context.WithCancel(context.Background())
		var ran []string
		note := func(stage string) {
			ran = append(ran, stage)
			if stage == name {
				cancel()
			}
		}
		light, oc, mt, ms, dv := s.light, s.ocean, s.matte, s.mist, s.develop
		s.light = func(c context.Context, r LightRequest) (LightResult, error) { note("light"); return light(c, r) }
		s.ocean = func(a ocean.Config, b ocean.RGB, c *ocean.RGB) (*oceantex.Texture, error) {
			note("ocean")
			return oc(a, b, c)
		}
		s.matte = func(a, b matte.Image, c matte.Config) (matte.Image, error) { note("matte"); return mt(a, b, c) }
		s.mist = func(a mist.Image, b mist.Config) (mist.Image, error) { note("mist"); return ms(a, b) }
		s.develop = func(a, b int, c develop.Options, d []float32) (*image.RGBA, error) {
			note("develop")
			return dv(a, b, c, d)
		}
		s.consume = func(Result) error { note("consume"); return nil }
		err := runStubs(ctx, cfg, s, nil)
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Errorf("ended in %s: %v", name, err)
		}
		if !reflect.DeepEqual(ran, order[:at+1]) {
			t.Errorf("ended in %s: stages run %v", name, ran)
		}
	}
	cfg := testConfig(t)
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	s := goodStubs(cfg)
	s.light = func(context.Context, LightRequest) (LightResult, error) {
		t.Fatal("lit after the context ended")
		return LightResult{}, nil
	}
	if err := runStubs(ended, cfg, s, nil); !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "cues") {
		t.Errorf("ended before the first stage: %v", err)
	}
}

// TestPipelineProfileRecordsEachStage checks that a profile gets one span for
// each of the seven stages in order, and that a profile that fills up stops
// the run at the stage that could not be recorded.
func TestPipelineProfileRecordsEachStage(t *testing.T) {
	stages := []string{"cues", "lighting", "ocean", "matte", "mist", "develop", "consume"}
	cfg := testConfig(t)
	profile, err := NewProfile(100)
	if err != nil {
		t.Fatal(err)
	}
	if err := runStubs(context.Background(), cfg, goodStubs(cfg), profile); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, span := range profile.Spans() {
		if span.Frame != 7 {
			t.Errorf("span for frame %d", span.Frame)
		}
		got = append(got, span.Stage)
	}
	if !reflect.DeepEqual(got, stages) {
		t.Fatalf("stages recorded %v, want %v", got, stages)
	}
	for k := range stages {
		full, err := NewProfile(k + 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := full.Record(ProfileSpan{Frame: 0, Stage: "earlier", Start: profile.Spans()[0].Start}); err != nil {
			t.Fatal(err)
		}
		err = runStubs(context.Background(), cfg, goodStubs(cfg), full)
		if err == nil || !strings.Contains(err.Error(), "profile: profile: span limit reached") {
			t.Errorf("profile full at %s: %v", stages[k], err)
			continue
		}
		var recorded []string
		for _, span := range full.Spans()[1:] {
			recorded = append(recorded, span.Stage)
		}
		if strings.Join(recorded, ",") != strings.Join(stages[:k], ",") {
			t.Errorf("profile full at %s: recorded %v", stages[k], recorded)
		}
	}
}

// TestPipelineRefusesBadSetup checks the refusals before the first frame: no
// context, an invalid config, no consumer, an incomplete pipeline, and a tree
// mask that does not fit.
func TestPipelineRefusesBadSetup(t *testing.T) {
	cfg := testConfig(t)
	cfg.Light = funcLight(goodStubs(cfg).light)
	full := goodStubs(cfg)
	stages := stagePipeline{matte: full.matte, mist: full.mist, develop: full.develop}
	ok := func(Result) error { return nil }
	var noContext context.Context
	if err := runPipelineProfiled(noContext, cfg, ok, full.ocean, stages, nil); err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Errorf("nil context: %v", err)
	}
	broken := cfg
	broken.FPS = 0
	if err := runPipelineProfiled(context.Background(), broken, ok, full.ocean, stages, nil); err == nil || !strings.HasPrefix(err.Error(), "config: ") {
		t.Errorf("bad config: %v", err)
	}
	if err := runPipelineProfiled(context.Background(), cfg, nil, full.ocean, stages, nil); err == nil || !strings.Contains(err.Error(), "nil consumer") {
		t.Errorf("nil consumer: %v", err)
	}
	for name, s := range map[string]stagePipeline{
		"no matte":   {mist: full.mist, develop: full.develop},
		"no mist":    {matte: full.matte, develop: full.develop},
		"no develop": {matte: full.matte, mist: full.mist},
	} {
		if err := runPipelineProfiled(context.Background(), cfg, ok, full.ocean, s, nil); err == nil || !strings.Contains(err.Error(), "incomplete stage pipeline") {
			t.Errorf("%s: %v", name, err)
		}
	}
	trees := lights.Image{Width: 3, Height: 1, Light: make([]float64, 9)}
	cfg.TreeMask = &trees
	if err := runPipelineProfiled(context.Background(), cfg, ok, full.ocean, stages, nil); err == nil || !strings.Contains(err.Error(), "tree mask") {
		t.Errorf("tree mask: %v", err)
	}
	cfg.TreeMask = &lights.Image{Width: 2, Height: 2, Light: make([]float64, 12)}
	if err := runPipelineProfiled(context.Background(), cfg, ok, full.ocean, stages, nil); err != nil {
		t.Errorf("good tree mask: %v", err)
	}
}

// TestCPUDevelopRefusesShortInput checks that input of the wrong length is
// refused, and that a developed black frame is opaque black.
func TestCPUDevelopRefusesShortInput(t *testing.T) {
	if _, err := cpuDevelop(2, 2, develop.DefaultOptions(), make([]float32, 11)); err == nil || !strings.Contains(err.Error(), "top-down packed RGB") {
		t.Errorf("short input: %v", err)
	}
	picture, err := cpuDevelop(2, 2, develop.DefaultOptions(), make([]float32, 12))
	if err != nil {
		t.Fatal(err)
	}
	if picture.Bounds().Dx() != 2 || picture.Bounds().Dy() != 2 {
		t.Errorf("bounds %v", picture.Bounds())
	}
	if r, g, b, a := picture.At(0, 0).RGBA(); r != 0 || g != 0 || b != 0 || a != 0xffff {
		t.Errorf("black frame developed to %d %d %d %d", r, g, b, a)
	}
}

// TestPrepareFireRefusals checks that fire preparation refuses while another
// request is running and for a context that has ended.
func TestPrepareFireRefusals(t *testing.T) {
	e, _ := newSnapshot(t)
	request := LightRequest{Width: 2, Height: 2}
	e.assets.fireRequestMu.Lock()
	if _, err := e.assets.prepareFire(context.Background(), request, 10, 1<<20); !errors.Is(err, lights.ErrFireResources) {
		t.Errorf("overlap: %v", err)
	}
	e.assets.fireRequestMu.Unlock()
	ended, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.assets.prepareFire(ended, request, 10, 1<<20); !errors.Is(err, context.Canceled) {
		t.Errorf("ended context: %v", err)
	}
}
