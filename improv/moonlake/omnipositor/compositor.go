package omnipositor

import (
	"context"
	"fmt"
	"image"
	"math"
	"strconv"

	"github.com/janearc/slayer/improv/moonlake/cues"
	"github.com/janearc/slayer/improv/moonlake/develop"
	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/matte"
	"github.com/janearc/slayer/improv/moonlake/mist"
	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/oceantex"
	"github.com/janearc/slayer/render/frame"
)

type LightStage interface {
	Light(context.Context, LightRequest) (LightResult, error)
}

type LightRequest struct {
	Index         int
	Seconds       float64
	Width, Height int
	Cues          cues.Frame
	Stops         float64
	WaterMask     lights.Image
	FireSnapshot  *FireSnapshot
}

type LightResult struct {
	Sky, Front lights.Image
	Strobe     float64
}

type Config struct {
	From, To                       int
	FPS                            float64
	Width, Height                  int
	Show                           *cues.Show
	Light                          LightStage
	Ocean                          ocean.Config
	Top, Bottom, Shore, MoonRadius float64
	MoonReference, WaterMask       lights.Image
	TreeMask                       *lights.Image
	MistSeed                       uint64
	MistLayers                     float64
	MistColour                     [3]float64
	OverheadMist, StrobeMist       float64
	Develop                        develop.Options
}

type Result struct {
	Index   int
	Seconds float64
	Cues    cues.Frame
	Linear  *frame.Frame
	Image   *image.RGBA
}

type oceanRender func(ocean.Config, ocean.RGB, *ocean.RGB) (*oceantex.Texture, error)

// Run renders frames From up to but not including To on the CPU, calling
// consume with each one. It stops at the first error or when ctx is done.
func Run(ctx context.Context, config Config, consume func(Result) error) error {
	return runPipeline(ctx, config, consume, ocean.Render, stagePipeline{matte: matte.Apply, mist: mist.Apply, develop: cpuDevelop})
}

// run is Run with the ocean renderer replaced by render.
func run(ctx context.Context, config Config, consume func(Result) error, render oceanRender) error {
	return runPipeline(ctx, config, consume, render, stagePipeline{matte: matte.Apply, mist: mist.Apply, develop: cpuDevelop})
}

// validate refuses a config with a bad frame range or fps, bad size, no
// show or light stage, a nonfinite number, bad geometry or mist values, a
// moon reference or water mask of the wrong size, or a water mask that has
// a negative value or no positive light.
func (c Config) validate() error {
	if !finite(c.FPS) || c.From < 0 || c.To <= c.From || c.FPS <= 0 {
		return fmt.Errorf("expected nonnegative half-open frame range and finite positive fps")
	}
	last := float64(c.To-1) / c.FPS
	if !finite(last) || (c.To-c.From > 1 && float64(c.To-2)/c.FPS == last) {
		return fmt.Errorf("unrepresentable frame timestamps")
	}
	if c.Width <= 0 || c.Height <= 0 || c.Width > mist.MaxPixels/c.Height {
		return fmt.Errorf("invalid or excessive dimensions")
	}
	if c.Show == nil || c.Light == nil {
		return fmt.Errorf("cues and lighting are required")
	}
	for _, value := range []float64{c.Top, c.Bottom, c.Shore, c.MoonRadius, c.MistLayers, c.MistColour[0], c.MistColour[1], c.MistColour[2], c.OverheadMist, c.StrobeMist, c.Develop.Exposure, c.Develop.Knee, c.Develop.Top, c.Develop.Bloom, c.Develop.White} {
		if !finite(value) {
			return fmt.Errorf("nonfinite configuration")
		}
	}
	if c.Develop.Bloom > 100000 {
		return fmt.Errorf("bloom exceeds CPU maximum of 100000 pixels")
	}
	shoreRow := (c.Top - c.Shore) / (c.Top - c.Bottom) * float64(c.Height)
	if c.Top <= c.Bottom || c.Shore <= c.Bottom || c.Shore >= c.Top || !finite(shoreRow) || shoreRow < 1 || shoreRow >= float64(c.Height) || c.MoonRadius <= 0 || c.MistLayers < 1 || c.OverheadMist < 0 || c.StrobeMist < 0 {
		return fmt.Errorf("invalid geometry or mist configuration")
	}
	if _, err := bottomUp(c.MoonReference, c.Width, c.Height); err != nil {
		return fmt.Errorf("moon reference: %w", err)
	}
	if _, err := bottomUp(c.WaterMask, c.Width, c.Height); err != nil {
		return fmt.Errorf("water mask: %w", err)
	}
	most := 0.0
	for _, value := range c.WaterMask.Light {
		if value < 0 {
			return fmt.Errorf("negative water mask")
		}
		most = max(most, float64(float32(value)))
	}
	if most <= 0 {
		return fmt.Errorf("water mask must contain positive light")
	}
	return nil
}

// finite reports whether value is neither NaN nor an infinity.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// quantize rounds value to digits places after the decimal point by writing
// it out as text and reading it back.
func quantize(value float64, digits int) float64 {
	result, _ := strconv.ParseFloat(strconv.FormatFloat(value, 'f', digits, 64), 64)
	return result
}

// bottomUp turns a top-down light image into a matte image with its rows
// reversed and its values as float32. It refuses a wrong size or a value
// that is not finite or does not fit in a float32.
func bottomUp(input lights.Image, width, height int) (matte.Image, error) {
	if input.Width != width || input.Height != height || len(input.Light) != width*height*3 {
		return matte.Image{}, fmt.Errorf("invalid top-down light dimensions")
	}
	out := matte.Image{Width: width, Height: height, Scale: -1, Pixels: make([]float32, len(input.Light))}
	for y := 0; y < height; y++ {
		for x := 0; x < width*3; x++ {
			value := input.Light[y*width*3+x]
			if !finite(value) || math.Abs(value) > math.MaxFloat32 {
				return matte.Image{}, fmt.Errorf("light outside finite float32 range")
			}
			out.Pixels[(height-1-y)*width*3+x] = float32(value)
		}
	}
	return out, nil
}

// seaFloat turns an ocean texture, whose rows are already bottom-up, into a
// matte image in the same row order. It refuses a wrong size or a value that
// is not finite or does not fit in a float32.
func seaFloat(input *oceantex.Texture, width, height int) (matte.Image, error) {
	if input == nil || input.Width != width || input.Height != height || len(input.Light) != width*height {
		return matte.Image{}, fmt.Errorf("invalid bottom-up ocean dimensions")
	}
	out := matte.Image{Width: width, Height: height, Scale: -1, Pixels: make([]float32, width*height*3)}
	for i, pixel := range input.Light {
		for c, value := range pixel {
			if !finite(value) || math.Abs(value) > math.MaxFloat32 {
				return matte.Image{}, fmt.Errorf("ocean light outside finite float32 range")
			}
			out.Pixels[i*3+c] = float32(value)
		}
	}
	return out, nil
}

// validFloat refuses an image that has a NaN or an infinity in a pixel.
func validFloat(input matte.Image) error {
	for _, value := range input.Pixels {
		if !finite(float64(value)) {
			return fmt.Errorf("nonfinite float stage output")
		}
	}
	return nil
}

// topDown turns a bottom-up mist image into a frame with its rows reversed
// and every pixel's Cover set to 1.
func topDown(input mist.Image) *frame.Frame {
	out := frame.New(input.Width, input.Height)
	for y := 0; y < input.Height; y++ {
		for x := 0; x < input.Width; x++ {
			i := ((input.Height-1-y)*input.Width + x) * 3
			pixel := out.At(x, y)
			pixel.R, pixel.G, pixel.B, pixel.Cover = input.Pixels[i], input.Pixels[i+1], input.Pixels[i+2], 1
			out.Set(x, y, pixel)
		}
	}
	return out
}
