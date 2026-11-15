//go:build darwin

package gpulights

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/metal"
)

const DefaultMaxBytes int64 = 1 << 30

type Effect uint32

const (
	EffectMoon Effect = 1 << iota
	EffectFixtures
	EffectForeground
)

const SupportedEffects = EffectMoon | EffectFixtures | EffectForeground

var ErrUnsupportedEffect = errors.New("gpulights: unsupported effect flags")

type Renderer struct {
	mu       sync.Mutex
	width    int
	height   int
	maxBytes int64
}

type FixtureLayer struct {
	Name   string
	Amount float64
	Colour [3]float64
}

type Request struct {
	Plate         lights.Image
	Open          []bool
	Moon          *lights.Image
	MoonShift     float64
	MoonLevel     float64
	Master        float64
	Foreground    *lights.Image
	FixtureLayers []FixtureLayer
	Passes        map[string]lights.Image
	Effects       Effect
}

type Result struct {
	Sky   lights.Image
	Front lights.Image
}

// New is a Renderer for frames of width by height with the DefaultMaxBytes
// budget.
func New(width, height int) (*Renderer, error) {
	return NewWithBudget(width, height, DefaultMaxBytes)
}

// NewWithBudget is a Renderer for frames of width by height that refuses any
// request needing more than maxBytes of GPU buffers. It refuses bad dimensions
// and a budget that is not above zero.
func NewWithBudget(width, height int, maxBytes int64) (*Renderer, error) {
	if !dimensions(width, height) {
		return nil, fmt.Errorf("gpulights: invalid dimensions")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("gpulights: invalid buffer budget")
	}
	return &Renderer{width: width, height: height, maxBytes: maxBytes}, nil
}

// Render lights one frame. It scales the plate by Master, then, for each effect
// flag set, adds the moon (where Open), adds each fixture layer, and copies
// the foreground. It returns the sky, and the front, which is empty unless the
// foreground flag is set. It refuses a nil renderer, an invalid request, and
// a request needing more buffers than the budget.
func (r *Renderer) Render(request Request) (Result, error) {
	if r == nil {
		return Result{}, fmt.Errorf("gpulights: nil renderer")
	}
	if !dimensions(r.width, r.height) {
		return Result{}, fmt.Errorf("gpulights: invalid renderer dimensions")
	}
	if err := r.validate(request); err != nil {
		return Result{}, err
	}
	required := request.allocations(r.width, r.height)
	if required > r.maxBytes {
		return Result{}, fmt.Errorf("gpulights: aggregate allocation exceeds %d bytes", r.maxBytes)
	}
	session, err := metal.OpenSession(kernelSource(), "scale_plate", "moon_add", "add_fixture", "copy_front")
	if err != nil {
		return Result{}, fmt.Errorf("gpulights: open Metal: %w", err)
	}
	defer session.Close()

	pixels := r.width * r.height
	skyIn, err := session.Buffer(pixels * 3)
	if err != nil {
		return Result{}, fmt.Errorf("gpulights: allocate sky input buffer: %w", err)
	}
	skyOut, err := session.Buffer(pixels * 3)
	if err != nil {
		return Result{}, fmt.Errorf("gpulights: allocate sky output buffer: %w", err)
	}
	if err := uploadImage(skyIn.Numbers(), request.Plate); err != nil {
		return Result{}, fmt.Errorf("gpulights: plate: %w", err)
	}
	plateParams, err := plateParams(r.width, r.height, request.Master)
	if err != nil {
		return Result{}, err
	}
	if err := session.Dispatch("scale_plate", r.width, r.height, plateParams, skyIn, skyOut); err != nil {
		return Result{}, fmt.Errorf("gpulights: dispatch scale_plate: %w", err)
	}
	current := skyOut
	currentIsOut := true
	if request.Effects&EffectMoon != 0 {
		moonBuf, err := session.Buffer(pixels * 3)
		if err != nil {
			return Result{}, fmt.Errorf("gpulights: allocate moon buffer: %w", err)
		}
		if err := uploadImage(moonBuf.Numbers(), *request.Moon); err != nil {
			return Result{}, fmt.Errorf("gpulights: moon reference: %w", err)
		}
		openBuf, err := session.Buffer(pixels)
		if err != nil {
			return Result{}, fmt.Errorf("gpulights: allocate moon mask buffer: %w", err)
		}
		uploadMask(openBuf.Numbers(), request.Open)
		next := skyIn
		if currentIsOut {
			next = skyIn
		} else {
			next = skyOut
		}
		moonParams, err := moonParams(r.width, r.height, request.MoonShift, request.MoonLevel)
		if err != nil {
			return Result{}, err
		}
		if err := session.Dispatch("moon_add", r.width, r.height, moonParams, current, moonBuf, openBuf, next); err != nil {
			return Result{}, fmt.Errorf("gpulights: dispatch moon_add: %w", err)
		}
		current = next
		currentIsOut = !currentIsOut
	}
	if request.Effects&EffectFixtures != 0 {
		if len(request.FixtureLayers) > 0 {
			passBufs := make([]metal.Buffer, 0, len(request.FixtureLayers))
			for range request.FixtureLayers {
				passBuf, err := session.Buffer(pixels * 3)
				if err != nil {
					return Result{}, fmt.Errorf("gpulights: allocate fixture buffer: %w", err)
				}
				passBufs = append(passBufs, passBuf)
			}
			for i, layer := range request.FixtureLayers {
				pass, found := request.Passes[layer.Name]
				if !found {
					return Result{}, fmt.Errorf("gpulights: missing fixture pass %q", layer.Name)
				}
				if err := uploadImage(passBufs[i].Numbers(), pass); err != nil {
					return Result{}, fmt.Errorf("gpulights: fixture %q: %w", layer.Name, err)
				}
				next := skyIn
				if !currentIsOut {
					next = skyOut
				}
				fixtureParams, err := fixtureParams(r.width, r.height, layer)
				if err != nil {
					return Result{}, fmt.Errorf("gpulights: fixture %q: %w", layer.Name, err)
				}
				if err := session.Dispatch("add_fixture", r.width, r.height, fixtureParams, current, passBufs[i], next); err != nil {
					return Result{}, fmt.Errorf("gpulights: dispatch add_fixture %q: %w", layer.Name, err)
				}
				current = next
				currentIsOut = !currentIsOut
			}
		}
	}
	front := lights.Image{Width: r.width, Height: r.height, Light: make([]float64, pixels*3)}
	if request.Effects&EffectForeground != 0 {
		frontIn, err := session.Buffer(pixels * 3)
		if err != nil {
			return Result{}, fmt.Errorf("gpulights: allocate foreground buffer: %w", err)
		}
		if err := uploadImage(frontIn.Numbers(), *request.Foreground); err != nil {
			return Result{}, fmt.Errorf("gpulights: foreground: %w", err)
		}
		frontOut, err := session.Buffer(pixels * 3)
		if err != nil {
			return Result{}, fmt.Errorf("gpulights: allocate foreground output buffer: %w", err)
		}
		copyParams, err := copyParams(r.width, r.height)
		if err != nil {
			return Result{}, err
		}
		if err := session.Dispatch("copy_front", r.width, r.height, copyParams, frontIn, frontOut); err != nil {
			return Result{}, fmt.Errorf("gpulights: dispatch copy_front: %w", err)
		}
		if err := session.Commit(); err != nil {
			return Result{}, fmt.Errorf("gpulights: commit: %w", err)
		}
		if err := validateFiniteBuffer("foreground", frontOut.Numbers()); err != nil {
			return Result{}, err
		}
		if err := validateFiniteBuffer("sky", current.Numbers()); err != nil {
			return Result{}, err
		}
		return Result{Sky: decodeImage(current.Numbers(), r.width, r.height), Front: decodeImage(frontOut.Numbers(), r.width, r.height)}, nil
	}
	if err := session.Commit(); err != nil {
		return Result{}, fmt.Errorf("gpulights: commit: %w", err)
	}
	if err := validateFiniteBuffer("sky", current.Numbers()); err != nil {
		return Result{}, err
	}
	return Result{Sky: decodeImage(current.Numbers(), r.width, r.height), Front: front}, nil
}

// validate refuses a request that does not fit the renderer: an image of
// another size, a number that is not finite float32, unknown effect flags,
// more than 64 fixture layers or one with an empty name, a moon or foreground
// effect with its input missing, and inputs given for an effect that is off.
func (r *Renderer) validate(request Request) error {
	if err := validateImage(request.Plate, r.width, r.height); err != nil {
		return fmt.Errorf("gpulights: plate: %w", err)
	}
	if !finite(request.Master) || math.Abs(request.Master) > math.MaxFloat32 {
		return fmt.Errorf("gpulights: master outside finite float32 range")
	}
	if request.Effects&^SupportedEffects != 0 {
		return fmt.Errorf("%w: 0x%x", ErrUnsupportedEffect, uint32(request.Effects&^SupportedEffects))
	}
	moonData := request.Moon != nil || len(request.Open) != 0 || request.MoonShift != 0 || request.MoonLevel != 0
	if moonData && request.Effects&EffectMoon == 0 {
		return fmt.Errorf("gpulights: moon inputs require the moon effect flag")
	}
	if request.Effects&EffectMoon != 0 {
		if request.Moon == nil {
			return fmt.Errorf("gpulights: moon effect requires a reference image")
		}
		if err := validateImage(*request.Moon, r.width, r.height); err != nil {
			return fmt.Errorf("gpulights: moon reference: %w", err)
		}
		if len(request.Open) != r.width*r.height {
			return fmt.Errorf("gpulights: moon mask has %d pixels, the frame has %d", len(request.Open), r.width*r.height)
		}
		if !finite(request.MoonShift) || math.Abs(request.MoonShift) > math.MaxFloat32 {
			return fmt.Errorf("gpulights: moon shift outside finite float32 range")
		}
		if !finite(request.MoonLevel) || math.Abs(request.MoonLevel) > math.MaxFloat32 {
			return fmt.Errorf("gpulights: moon level outside finite float32 range")
		}
	}
	fixtureData := len(request.FixtureLayers) != 0 || len(request.Passes) != 0
	if fixtureData && request.Effects&EffectFixtures == 0 {
		return fmt.Errorf("gpulights: fixture inputs require the fixture effect flag")
	}
	if request.Effects&EffectFixtures != 0 {
		if len(request.FixtureLayers) > 64 {
			return fmt.Errorf("gpulights: fixture layer count exceeds 64")
		}
		for _, layer := range request.FixtureLayers {
			if layer.Name == "" {
				return fmt.Errorf("gpulights: empty fixture pass name")
			}
			if !finite(layer.Amount) || math.Abs(layer.Amount) > math.MaxFloat32 {
				return fmt.Errorf("gpulights: fixture %q amount outside finite float32 range", layer.Name)
			}
			for _, channel := range layer.Colour {
				if !finite(channel) || math.Abs(channel) > math.MaxFloat32 {
					return fmt.Errorf("gpulights: fixture %q colour outside finite float32 range", layer.Name)
				}
			}
		}
	}
	if request.Foreground != nil && request.Effects&EffectForeground == 0 {
		return fmt.Errorf("gpulights: foreground image requires the foreground effect flag")
	}
	if request.Effects&EffectForeground != 0 {
		if request.Foreground == nil {
			return fmt.Errorf("gpulights: foreground effect requires an image")
		}
		if err := validateImage(*request.Foreground, r.width, r.height); err != nil {
			return fmt.Errorf("gpulights: foreground: %w", err)
		}
	}
	return nil
}

// allocations is the bytes of GPU buffers the request needs for a frame of
// width by height, at four bytes a number, counting only the buffers for the
// effects that are switched on. Render compares it with the budget.
func (request Request) allocations(width, height int) int64 {
	pixels := int64(width) * int64(height)
	required := 2 * pixels * 3 * 4
	if request.Effects&EffectMoon != 0 {
		required += 2 * pixels * 4
		required += pixels * 3 * 4
	}
	if request.Effects&EffectFixtures != 0 {
		required += int64(len(request.FixtureLayers)) * pixels * 3 * 4
	}
	if request.Effects&EffectForeground != 0 {
		required += 2 * pixels * 3 * 4
	}
	required += 4 * 4
	return required
}

// validateImage refuses an image that is not exactly width by height with
// three values a pixel, or that holds a value that is not finite or is beyond
// float32 range.
func validateImage(image lights.Image, width, height int) error {
	if image.Width != width || image.Height != height || len(image.Light) != width*height*3 {
		return fmt.Errorf("invalid image dimensions")
	}
	for _, value := range image.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return fmt.Errorf("light outside finite float32 range")
		}
	}
	return nil
}

// uploadImage copies image.Light into dst as float32. It refuses a dst of
// another length and any value that is not finite or is beyond float32 range.
func uploadImage(dst []float32, image lights.Image) error {
	if len(dst) != len(image.Light) {
		return fmt.Errorf("invalid image dimensions")
	}
	for i, value := range image.Light {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return fmt.Errorf("light outside finite float32 range")
		}
		dst[i] = float32(value)
	}
	return nil
}

// uploadMask writes 1 into dst for each true in mask and 0 for each false.
// dst must be at least as long as mask.
func uploadMask(dst []float32, mask []bool) {
	for i, value := range mask {
		if value {
			dst[i] = 1
		} else {
			dst[i] = 0
		}
	}
}

// decodeImage widens src to a width by height image of float64. It checks
// nothing; src must hold width*height*3 values.
func decodeImage(src []float32, width, height int) lights.Image {
	out := lights.Image{Width: width, Height: height, Light: make([]float64, len(src))}
	for i, value := range src {
		out.Light[i] = float64(value)
	}
	return out
}

// validateFiniteBuffer refuses a buffer read back from the GPU that holds a NaN
// or infinity, naming the buffer and the first bad channel in the error.
func validateFiniteBuffer(name string, values []float32) error {
	for i, value := range values {
		if !finite(float64(value)) {
			return fmt.Errorf("gpulights: %s channel %d is not finite", name, i)
		}
	}
	return nil
}

// plateParams is the 12 byte parameter block for scale_plate: width, height,
// then master as a float32, all little-endian. It refuses a master that is not
// finite float32.
func plateParams(width, height int, master float64) ([]byte, error) {
	out := make([]byte, 12)
	binary.LittleEndian.PutUint32(out[0:], uint32(width))
	binary.LittleEndian.PutUint32(out[4:], uint32(height))
	if err := setFiniteFloat32(out, 8, master); err != nil {
		return nil, err
	}
	return out, nil
}

// moonParams is the 16 byte parameter block for moon_add: width, height, the
// row shift, then the level. It refuses a shift or level that is not finite
// float32.
func moonParams(width, height int, shift, level float64) ([]byte, error) {
	out := make([]byte, 16)
	binary.LittleEndian.PutUint32(out[0:], uint32(width))
	binary.LittleEndian.PutUint32(out[4:], uint32(height))
	if err := setFiniteFloat32(out, 8, shift); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(out, 12, level); err != nil {
		return nil, err
	}
	return out, nil
}

// fixtureParams is the 24 byte parameter block for add_fixture: width, height,
// the layer's amount, then its three colour channels. It refuses an amount or
// channel that is not finite float32.
func fixtureParams(width, height int, layer FixtureLayer) ([]byte, error) {
	out := make([]byte, 24)
	binary.LittleEndian.PutUint32(out[0:], uint32(width))
	binary.LittleEndian.PutUint32(out[4:], uint32(height))
	if err := setFiniteFloat32(out, 8, layer.Amount); err != nil {
		return nil, err
	}
	for i, channel := range layer.Colour {
		if err := setFiniteFloat32(out, 12+i*4, channel); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// copyParams is the 8 byte parameter block for copy_front: width, then height,
// as little-endian uint32. The error is always nil; it matches the others.
func copyParams(width, height int) ([]byte, error) {
	out := make([]byte, 8)
	binary.LittleEndian.PutUint32(out[0:], uint32(width))
	binary.LittleEndian.PutUint32(out[4:], uint32(height))
	return out, nil
}

// setFiniteFloat32 writes value as a little-endian float32 at offset off in
// dst. It refuses a value that is not finite or is beyond float32 range.
func setFiniteFloat32(dst []byte, off int, value float64) error {
	if !finite(value) || math.Abs(value) > math.MaxFloat32 {
		return fmt.Errorf("gpulights: value outside finite float32 range")
	}
	binary.LittleEndian.PutUint32(dst[off:], math.Float32bits(float32(value)))
	return nil
}

// dimensions reports whether width and height are both positive and the frame
// has at most 1<<24 pixels. It divides rather than multiplies so it cannot
// overflow.
func dimensions(width, height int) bool {
	const maxPixels = 1 << 24
	return width > 0 && height > 0 && width <= maxPixels/height
}

// finite reports whether value is neither NaN nor infinite.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// kernelSource is the Metal source for the four kernels Render runs:
// scale_plate (multiply by master), moon_add (add the moon, shifted down by
// shift rows and blended between rows, where open), add_fixture (add pass
// scaled by amount and colour) and copy_front (copy the foreground).
func kernelSource() string {
	var b strings.Builder
	b.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	b.WriteString("struct PlateParams { uint width; uint height; float master; };\n")
	b.WriteString("struct MoonParams { uint width; uint height; float shift; float level; };\n")
	b.WriteString("struct FixtureParams { uint width; uint height; float amount; float r; float g; float b; };\n")
	b.WriteString("struct CopyParams { uint width; uint height; };\n")
	b.WriteString("kernel void scale_plate(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant PlateParams &p [[buffer(2)]], uint2 gid [[thread_position_in_grid]]) { if (gid.x >= p.width || gid.y >= p.height) { return; } uint i = gid.y * p.width + gid.x; uint base = i * 3u; out[base] = in[base] * p.master; out[base + 1u] = in[base + 1u] * p.master; out[base + 2u] = in[base + 2u] * p.master; }\n")
	b.WriteString("kernel void moon_add(device const float *sky [[buffer(0)]], device const float *moon [[buffer(1)]], device const float *open [[buffer(2)]], device float *out [[buffer(3)]], constant MoonParams &p [[buffer(4)]], uint2 gid [[thread_position_in_grid]]) { if (gid.x >= p.width || gid.y >= p.height) { return; } uint i = gid.y * p.width + gid.x; uint base = i * 3u; out[base] = sky[base]; out[base + 1u] = sky[base + 1u]; out[base + 2u] = sky[base + 2u]; if (open[i] <= 0.5f) { return; } float srcY = float(gid.y) - p.shift; if (srcY < 0.0f || srcY >= float(p.height - 1u)) { return; } int low = int(floor(srcY)); float f = srcY - float(low); uint ai = uint(low) * p.width + gid.x; uint bi = uint(low + 1) * p.width + gid.x; uint a = ai * 3u; uint b = bi * 3u; float wa = open[ai] > 0.5f ? 1.0f - f : 0.0f; float wb = open[bi] > 0.5f ? f : 0.0f; out[base] += p.level * (wa * moon[a] + wb * moon[b]); out[base + 1u] += p.level * (wa * moon[a + 1u] + wb * moon[b + 1u]); out[base + 2u] += p.level * (wa * moon[a + 2u] + wb * moon[b + 2u]); }\n")
	b.WriteString("kernel void add_fixture(device const float *sky [[buffer(0)]], device const float *pass [[buffer(1)]], device float *out [[buffer(2)]], constant FixtureParams &p [[buffer(3)]], uint2 gid [[thread_position_in_grid]]) { if (gid.x >= p.width || gid.y >= p.height) { return; } uint i = gid.y * p.width + gid.x; uint base = i * 3u; out[base] = sky[base] + p.amount * p.r * pass[base]; out[base + 1u] = sky[base + 1u] + p.amount * p.g * pass[base + 1u]; out[base + 2u] = sky[base + 2u] + p.amount * p.b * pass[base + 2u]; }\n")
	b.WriteString("kernel void copy_front(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant CopyParams &p [[buffer(2)]], uint2 gid [[thread_position_in_grid]]) { if (gid.x >= p.width || gid.y >= p.height) { return; } uint i = gid.y * p.width + gid.x; uint base = i * 3u; out[base] = in[base]; out[base + 1u] = in[base + 1u]; out[base + 2u] = in[base + 2u]; }\n")
	return b.String()
}
