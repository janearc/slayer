package omnipositor

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"github.com/janearc/slayer/improv/moonlake/develop"
	"github.com/janearc/slayer/improv/moonlake/matte"
	"github.com/janearc/slayer/improv/moonlake/mist"
	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpudevelop"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpumatte"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpumist"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpuocean"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/render/colour"
	"github.com/janearc/slayer/render/frame"
)

type stagePipeline struct {
	matte   func(matte.Image, matte.Image, matte.Config) (matte.Image, error)
	mist    func(mist.Image, mist.Config) (mist.Image, error)
	develop func(int, int, develop.Options, []float32) (*image.RGBA, error)
}

var ErrExperimentalGPUUnsupported = errors.New("experimental gpu compositor: unsupported configuration")

type gpuDevelopCache struct {
	mu    sync.Mutex
	stage *gpudevelop.Developer
	w     int
	h     int
	tone  colour.Tone
}

// closeLocked closes the cached GPU developer, if any, and forgets its size
// and tone. The caller must hold c.mu.
func (c *gpuDevelopCache) closeLocked() {
	if c.stage != nil {
		c.stage.Close()
		c.stage = nil
	}
	c.w = 0
	c.h = 0
	c.tone = colour.Tone{}
}

// Close closes the cached GPU developer, if any, taking the lock first.
func (c *gpuDevelopCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

// develop develops rgb on the GPU, keeping the developer between calls. It
// is reused when the size, knee, top and bloom are the same, and only the
// exposure is changed if that differs. Otherwise a new one replaces it. After
// an error the cached developer is closed.
func (c *gpuDevelopCache) develop(width, height int, options develop.Options, rgb []float32) (*image.RGBA, error) {
	tone := colour.Tone{Exposure: math.Exp2(options.Exposure), Knee: options.Knee, Top: options.Top, Bloom: options.Bloom}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stage != nil && c.w == width && c.h == height && c.tone.Knee == tone.Knee && c.tone.Top == tone.Top && c.tone.Bloom == tone.Bloom {
		if c.tone.Exposure == tone.Exposure {
			picture, err := c.stage.Develop(rgb)
			if err != nil {
				c.closeLocked()
			}
			return picture, err
		}
		picture, err := c.stage.DevelopWithExposure(rgb, tone.Exposure)
		if err != nil {
			c.closeLocked()
			return nil, err
		}
		c.tone.Exposure = tone.Exposure
		return picture, nil
	}
	stage, err := gpudevelop.New(width, height, tone)
	if err != nil {
		return nil, err
	}
	previous := c.stage
	c.stage = stage
	c.w = width
	c.h = height
	c.tone = tone
	if previous != nil {
		previous.Close()
	}
	picture, err := c.stage.Develop(rgb)
	if err != nil {
		c.closeLocked()
	}
	return picture, err
}

// RunExperimentalGPU is Run with the matte, mist and develop stages on the
// GPU. The ocean stage stays on the CPU.
func RunExperimentalGPU(ctx context.Context, config Config, consume func(Result) error) error {
	return RunExperimentalGPUProfiled(ctx, config, consume, nil)
}

// RunExperimentalGPUProfiled is RunExperimentalGPU that also records the time
// of each stage of each frame in profile.
func RunExperimentalGPUProfiled(ctx context.Context, config Config, consume func(Result) error, profile *Profile) error {
	return runExperimentalGPU(ctx, config, consume, profile, false)
}

// RunExperimentalGPUWithOcean is RunExperimentalGPUProfiled with the ocean
// stage on the GPU as well.
func RunExperimentalGPUWithOcean(ctx context.Context, config Config, consume func(Result) error, profile *Profile) error {
	return runExperimentalGPU(ctx, config, consume, profile, true)
}

// runExperimentalGPU is the body of the three RunExperimentalGPU functions;
// gpuOcean picks the GPU ocean stage. It refuses a nonzero Develop.White.
// The GPU developer it makes is closed when the run ends.
func runExperimentalGPU(ctx context.Context, config Config, consume func(Result) error, profile *Profile, gpuOcean bool) error {
	if config.Develop.White != 0 {
		return fmt.Errorf("%w: develop.white is unsupported", ErrExperimentalGPUUnsupported)
	}
	cache := &gpuDevelopCache{}
	defer cache.Close()
	render := ocean.Render
	if gpuOcean {
		render = func(config ocean.Config, image ocean.RGB, moon *ocean.RGB) (*oceantex.Texture, error) {
			return gpuocean.Render(ctx, config, image, moon)
		}
	}
	return runPipelineProfiled(ctx, config, consume, render, stagePipeline{matte: gpumatte.Apply, mist: gpumist.Apply, develop: cache.develop}, profile)
}

// runPipeline is runPipelineProfiled without a profile.
func runPipeline(ctx context.Context, config Config, consume func(Result) error, render oceanRender, stages stagePipeline) error {
	return runPipelineProfiled(ctx, config, consume, render, stages, nil)
}

// runPipelineProfiled renders frames From up to but not including To. Each
// frame goes through cues, lighting, ocean, matte, mist and develop, then to
// consume. ctx is checked between stages, and an error names the frame and
// stage. If profile is not nil each stage's time is recorded in it.
func runPipelineProfiled(ctx context.Context, config Config, consume func(Result) error, render oceanRender, stages stagePipeline, profile *Profile) error {
	if ctx == nil {
		return fmt.Errorf("config: nil context")
	}
	if err := config.validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if consume == nil {
		return fmt.Errorf("config: nil consumer")
	}
	if stages.matte == nil || stages.mist == nil || stages.develop == nil {
		return fmt.Errorf("config: incomplete stage pipeline")
	}
	moon, err := bottomUp(config.MoonReference, config.Width, config.Height)
	if err != nil {
		return fmt.Errorf("config: moon reference: %w", err)
	}
	water, err := bottomUp(config.WaterMask, config.Width, config.Height)
	if err != nil {
		return fmt.Errorf("config: water mask: %w", err)
	}
	var trees *mist.Image
	if config.TreeMask != nil {
		converted, err := bottomUp(*config.TreeMask, config.Width, config.Height)
		if err != nil {
			return fmt.Errorf("config: tree mask: %w", err)
		}
		value := mist.Image(converted)
		trees = &value
	}
	moonRGB := ocean.RGB{Width: moon.Width, Height: moon.Height, Light: moon.Pixels}
	moonMist, waterMist := mist.Image(moon), mist.Image(water)
	for index := config.From; index < config.To; index++ {
		seconds := quantize(float64(index)/config.FPS, 4)
		stageError := func(stage string, err error) error {
			return fmt.Errorf("frame %d at %.9g: %s: %w", index, seconds, stage, err)
		}
		check := func(stage string) error {
			if err := ctx.Err(); err != nil {
				return stageError(stage, err)
			}
			return nil
		}
		if err := check("cues"); err != nil {
			return err
		}
		var started time.Time
		if profile != nil {
			started = time.Now()
		}
		cue, err := config.Show.At(seconds, config.FPS)
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "cues", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if err != nil {
			return stageError("cues", err)
		}
		cue = cue.Quantized()
		options := config.Develop
		options.Exposure = quantize(options.Exposure+cue.Exposure, 3)
		if err := check("lighting"); err != nil {
			return err
		}
		if profile != nil {
			started = time.Now()
		}
		lit, err := config.Light.Light(ctx, LightRequest{Index: index, Seconds: seconds, Width: config.Width, Height: config.Height, Cues: cue, Stops: options.Exposure, WaterMask: config.WaterMask})
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "lighting", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if err != nil {
			return stageError("lighting", err)
		}
		sky, err := bottomUp(lit.Sky, config.Width, config.Height)
		if err != nil {
			return stageError("lighting sky", err)
		}
		front, err := bottomUp(lit.Front, config.Width, config.Height)
		if err != nil {
			return stageError("lighting front", err)
		}
		if !finite(lit.Strobe) || lit.Strobe < 0 {
			return stageError("lighting", fmt.Errorf("invalid strobe level"))
		}
		if err := check("ocean"); err != nil {
			return err
		}
		seaConfig := config.Ocean
		seaConfig.Width, seaConfig.Height, seaConfig.Seconds = config.Width, config.Height, seconds
		if profile != nil {
			started = time.Now()
		}
		sea, err := render(seaConfig, ocean.RGB{Width: sky.Width, Height: sky.Height, Light: sky.Pixels}, &moonRGB)
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "ocean", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if err != nil {
			return stageError("ocean", err)
		}
		seaImage, err := seaFloat(sea, config.Width, config.Height)
		if err != nil {
			return stageError("ocean", err)
		}
		if err := check("matte"); err != nil {
			return err
		}
		if profile != nil {
			started = time.Now()
		}
		combined, err := stages.matte(sky, seaImage, matte.Config{Top: config.Top, Bottom: config.Bottom, Shore: config.Shore, MoonRadius: config.MoonRadius, MoonRef: &moon, WaterMask: &water, Front: &front})
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "matte", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if err != nil {
			return stageError("matte", err)
		}
		if err := validFloat(combined); err != nil {
			return stageError("matte", err)
		}
		strength := quantize(cue.Mist*cue.Master+config.StrobeMist*quantize(lit.Strobe, 4)+cue.Overhead*config.OverheadMist, 3)
		if err := check("mist"); err != nil {
			return err
		}
		if profile != nil {
			started = time.Now()
		}
		veiled, err := stages.mist(mist.Image(combined), mist.Config{Top: config.Top, Bottom: config.Bottom, Shore: config.Shore, Strength: strength, Seconds: seconds, Seed: config.MistSeed, Colour: config.MistColour, Layers: config.MistLayers, MoonRef: &moonMist, WaterMask: &waterMist, TreeMask: trees})
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "mist", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if err != nil {
			return stageError("mist", err)
		}
		if err := validFloat(matte.Image(veiled)); err != nil {
			return stageError("mist", err)
		}
		linear := topDown(veiled)
		if err := check("develop"); err != nil {
			return err
		}
		if profile != nil {
			started = time.Now()
		}
		picture, err := stages.develop(config.Width, config.Height, options, packFrame(linear))
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "develop", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if err != nil {
			return stageError("develop", err)
		}
		if err := check("consume"); err != nil {
			return err
		}
		if profile != nil {
			started = time.Now()
		}
		consumeErr := consume(Result{Index: index, Seconds: seconds, Cues: cue, Linear: linear, Image: picture})
		if profile != nil {
			if profileErr := profile.Record(ProfileSpan{Frame: index, Stage: "consume", Start: started, Duration: time.Since(started)}); profileErr != nil {
				return stageError("profile", profileErr)
			}
		}
		if consumeErr != nil {
			return stageError("consume", consumeErr)
		}
		if err := check("consume"); err != nil {
			return err
		}
	}
	return nil
}

// cpuDevelop develops rgb on the CPU. rgb must be top-down packed RGB, three
// float32 values a pixel at width by height, or it is refused.
func cpuDevelop(width, height int, options develop.Options, rgb []float32) (*image.RGBA, error) {
	linear := frame.New(width, height)
	if len(rgb) != len(linear.Pix)*3 {
		return nil, fmt.Errorf("develop: input must be top-down packed RGB at the stage dimensions")
	}
	for i := range linear.Pix {
		pixel := linear.Pix[i]
		pixel.R, pixel.G, pixel.B = rgb[3*i], rgb[3*i+1], rgb[3*i+2]
		pixel.Cover = 1
		linear.Pix[i] = pixel
	}
	return develop.Develop([]*frame.Frame{linear}, options)
}

// gpuDevelop develops rgb on the GPU with a developer made for this one call
// and closed after it. Nothing in the package calls it.
func gpuDevelop(width, height int, options develop.Options, rgb []float32) (*image.RGBA, error) {
	tone := colour.Tone{Exposure: math.Exp2(options.Exposure), Knee: options.Knee, Top: options.Top, Bloom: options.Bloom}
	stage, err := gpudevelop.New(width, height, tone)
	if err != nil {
		return nil, err
	}
	defer stage.Close()
	return stage.Develop(rgb)
}

// packFrame lays out the red, green and blue of each pixel of input as one
// flat float32 slice, in the frame's row order, leaving out Cover.
func packFrame(input *frame.Frame) []float32 {
	packed := make([]float32, len(input.Pix)*3)
	for i, pixel := range input.Pix {
		packed[3*i], packed[3*i+1], packed[3*i+2] = pixel.R, pixel.G, pixel.B
	}
	return packed
}
