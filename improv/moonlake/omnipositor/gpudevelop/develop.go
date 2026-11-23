package gpudevelop

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"github.com/janearc/slayer/metal"
	ninjacolour "github.com/janearc/slayer/ninjafilm/colour"
	"github.com/janearc/slayer/render/colour"
)

const (
	maxBufferBytes = math.MaxInt32
	widestBloom    = 1e5
	maxExposed     = 1e20
	maxStageBytes  = 1 << 30
)

type Developer struct {
	mu                                         sync.Mutex
	session                                    *metal.Session
	w, h                                       int
	input, over, across, glow, output, weights metal.Buffer
	params                                     []byte
	exposure                                   float64
	bloom                                      bool
}

// allocationBytes returns the bytes the stage needs for w by h pixels and a
// blur reach of reach pixels: 68 bytes a pixel plus 12 bytes for each of the
// 2*reach+1 weights. It refuses a size or reach out of range and a total over
// 1 GiB.
func allocationBytes(w, h, reach int) (int, error) {
	if w <= 0 || h <= 0 || reach < 0 || reach > widestBloom*3 || w > maxStageBytes/68 || h > maxStageBytes/68/w {
		return 0, errors.New("gpudevelop: invalid size or aggregate allocation exceeds 1 GiB")
	}
	bytes := w*h*68 + (2*reach+1)*12
	if bytes > maxStageBytes {
		return 0, errors.New("gpudevelop: aggregate allocation exceeds 1 GiB")
	}
	return bytes, nil
}

// New returns a Developer for frames of w by h pixels. It refuses sizes over
// the buffer ceiling, a tone that is nonfinite or not a float32, a bad
// exposure, a Bloom over 100000 pixels, a Knee that is not a float32
// interval, and a stage over 1 GiB. The shoulder's scale and the bloom's
// weights and reach are the tone's print's, ninjafilm/colour's, made
// float32 for the kernels. The caller must Close it.
func New(w, h int, tone colour.Tone) (*Developer, error) {
	if w <= 0 || h <= 0 || w > maxBufferBytes/16 || h > maxBufferBytes/16/w {
		return nil, errors.New("gpudevelop: dimensions exceed the buffer ceiling")
	}
	for _, v := range []float64{tone.Exposure, tone.Knee, tone.Top, tone.Bloom} {
		if math.IsNaN(v) || math.IsInf(v, 0) || math.IsInf(float64(float32(v)), 0) {
			return nil, errors.New("gpudevelop: tone must be finite and representable in float32")
		}
	}
	e, err := validateExposure(tone.Exposure)
	if err != nil {
		return nil, err
	}
	if tone.Bloom > widestBloom {
		return nil, errors.New("gpudevelop: bloom exceeds 100000 pixels")
	}
	knee := tone.Knee
	if knee > 0 && knee < 1 && (float32(knee) == 0 || float32(knee) == 1) {
		return nil, errors.New("gpudevelop: knee loses its interval in float32")
	}
	// the shoulder's scale is the print's own: the kernel finish restates
	// the shoulder around it and must not solve it again
	scale := tone.Print().Scale()
	weights, reach := []float64{1}, 0 // defined: no bloom is one weight
	if tone.Bloom > 0 {
		weights, reach = ninjacolour.BloomWeights(tone.Bloom)
	}
	if _, err := allocationBytes(w, h, reach); err != nil {
		return nil, err
	}
	s, err := metal.OpenSession(prelude()+kernelBody, "spill", "horizontal", "vertical", "finish")
	if err != nil {
		return nil, fmt.Errorf("gpudevelop: open Metal: %w", err)
	}
	d := &Developer{session: s, w: w, h: h, exposure: e, bloom: tone.Bloom > 0}
	counts := []int{w * h * 3, w * h * 3, w * h * 3, w * h * 3, w * h * 4, 2*reach + 1}
	buffers := []*metal.Buffer{&d.input, &d.over, &d.across, &d.glow, &d.output, &d.weights}
	for i, dst := range buffers {
		*dst, err = s.Buffer(counts[i])
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("gpudevelop: allocate: %w", err)
		}
	}
	for i, v := range weights {
		d.weights.Numbers()[i] = float32(v)
	}
	d.params = make([]byte, 24)
	for i, v := range []uint32{uint32(w), uint32(h), uint32(reach), math.Float32bits(float32(e)), math.Float32bits(float32(knee)), math.Float32bits(float32(scale))} {
		binary.LittleEndian.PutUint32(d.params[i*4:], v)
	}
	return d, nil
}

// validateExposure returns exposure, with zero meaning 1. It refuses NaN,
// infinity, a value that is not a float32, a negative value, and one whose
// float32 reciprocal is zero or infinite.
func validateExposure(exposure float64) (float64, error) {
	if exposure == 0 {
		exposure = 1
	}
	if math.IsNaN(exposure) || math.IsInf(exposure, 0) || math.IsInf(float64(float32(exposure)), 0) {
		return 0, errors.New("gpudevelop: tone must be finite and representable in float32")
	}
	if exposure < 0 || float32(exposure) == 0 || math.IsInf(float64(float32(1/exposure)), 0) {
		return 0, errors.New("gpudevelop: exposure must have a positive finite float32 reciprocal")
	}
	return exposure, nil
}

// setExposureLocked checks exposure and then stores it on d and in the
// parameter bytes the kernels read. d is unchanged on error. The caller must
// hold d.mu.
func (d *Developer) setExposureLocked(exposure float64) error {
	next, err := validateExposure(exposure)
	if err != nil {
		return err
	}
	d.exposure = next
	binary.LittleEndian.PutUint32(d.params[12:], math.Float32bits(float32(next)))
	return nil
}

// developLocked turns rgb, packed top-down RGB floats at the stage size, into
// an RGBA image on the GPU. It refuses a wrong length and any value that is
// NaN, infinite or over 1e20 after exposure. Without bloom it clears the glow
// and runs only finish. A failed dispatch or commit closes the session, so
// later calls report the stage closed. The caller must hold d.mu.
func (d *Developer) developLocked(rgb []float32) (*image.RGBA, error) {
	if len(rgb) != d.w*d.h*3 {
		return nil, errors.New("gpudevelop: input must be top-down packed RGB at the stage dimensions")
	}
	for _, v := range rgb {
		x := float64(v) * d.exposure
		if math.IsNaN(x) || math.IsInf(x, 0) || math.Abs(x) > maxExposed {
			return nil, errors.New("gpudevelop: nonfinite or out-of-range exposed input")
		}
	}
	copy(d.input.Numbers(), rgb)
	steps := []struct {
		name    string
		buffers []metal.Buffer
	}{
		{"spill", []metal.Buffer{d.input, d.over}},
		{"horizontal", []metal.Buffer{d.over, d.weights, d.across}},
		{"vertical", []metal.Buffer{d.across, d.weights, d.glow}},
		{"finish", []metal.Buffer{d.input, d.glow, d.output}},
	}
	if !d.bloom {
		clear(d.glow.Numbers())
		steps = steps[3:]
	}
	for _, step := range steps {
		if err := d.session.Dispatch(step.name, d.w, d.h, d.params, step.buffers...); err != nil {
			d.session.Close()
			d.session = nil
			return nil, fmt.Errorf("gpudevelop: dispatch %s: %w", step.name, err)
		}
	}
	if err := d.session.Commit(); err != nil {
		d.session.Close()
		d.session = nil
		return nil, fmt.Errorf("gpudevelop: commit: %w", err)
	}
	out := image.NewRGBA(image.Rect(0, 0, d.w, d.h))
	for i, v := range d.output.Numbers() {
		out.Pix[i] = uint8(v)
	}
	return out, nil
}

// Develop develops rgb at the current exposure and returns the RGBA image. It
// refuses a closed stage.
func (d *Developer) Develop(rgb []float32) (*image.RGBA, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session == nil {
		return nil, errors.New("gpudevelop: stage is closed")
	}
	return d.developLocked(rgb)
}

// DevelopWithExposure sets the exposure and then develops rgb. The new
// exposure stays in force for later calls. It refuses a closed stage, and an
// invalid exposure before any drawing.
func (d *Developer) DevelopWithExposure(rgb []float32, exposure float64) (*image.RGBA, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session == nil {
		return nil, errors.New("gpudevelop: stage is closed")
	}
	if err := d.setExposureLocked(exposure); err != nil {
		return nil, err
	}
	return d.developLocked(rgb)
}

// GPUTime returns the GPU time the session reports, or zero once the stage is
// closed.
func (d *Developer) GPUTime() time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session == nil {
		return 0
	}
	return d.session.GPUTime()
}

// Close ends the session. Develop calls after it refuse; closing twice does
// nothing.
func (d *Developer) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.session != nil {
		d.session.Close()
		d.session = nil
	}
}

// prelude returns the start of the Metal source: the include, the int
// constants zero to four, and each float constant along with its Hi part (the
// float32 value) and its Lo part (what that value leaves out).
func prelude() string {
	source := "#include <metal_stdlib>\nusing namespace metal;\n"
	for i, name := range []string{"zero", "one", "two", "three", "four"} {
		source += fmt.Sprintf("constant int %s = %d;\n", name, i)
	}
	for _, number := range []struct {
		name  string
		value float64
	}{
		{"black", 0}, {"white", 1}, {"third", 1.0 / 3},
		{"red", 0.2126}, {"green", 0.7152}, {"blue", 0.0722},
		{"srgbKnee", 0.0031308}, {"srgbSlope", 12.92}, {"srgbScale", 1.055},
		{"srgbPower", 1.0 / 2.4}, {"srgbOffset", 0.055}, {"byteTop", 255}, {"halfStep", 0.5},
	} {
		source += fmt.Sprintf("constant float %s = %.17ef;\n", number.name, number.value)
		hi := float64(float32(number.value))
		lo := number.value - hi
		source += fmt.Sprintf("constant float %sHi = %.17ef;\nconstant float %sLo = %.17ef;\n", number.name, hi, number.name, lo)
	}
	return source
}

const kernelBody = `
struct Params { uint w; uint h; uint reach; float exposure; float knee; float scale; };
#pragma clang fp reassociate(off) contract(off)
float2 twoSum(float a, float b) {
    volatile float sum = a+b;
    volatile float part = sum-a;
    volatile float remainder = (a-(sum-part))+(b-part);
    return float2(sum,remainder);
}
float2 weighted(float value, float hi, float lo) {
    volatile float product = value*hi;
    volatile float error = fma(value,hi,-product);
    volatile float tail = fma(value,lo,error);
    return twoSum(product,tail);
}
float2 addPair(float2 a, float2 b) {
    float2 sum = twoSum(a.x,b.x);
    volatile float tail = (a.y+b.y)+sum.y;
    return twoSum(sum.x,tail);
}
float luminance(float3 l) {
    float2 r = weighted(l.r,redHi,redLo);
    float2 g = weighted(l.g,greenHi,greenLo);
    float2 b = weighted(l.b,blueHi,blueLo);
    float2 sum = addPair(addPair(r,g),b);
    return sum.x+sum.y;
}
float3 readRGB(device const float *in, uint i) {
    return float3(in[three*i], in[three*i+one], in[three*i+two]);
}
void writeRGB(device float *out, uint i, float3 v) {
    out[three*i] = v.r; out[three*i+one] = v.g; out[three*i+two] = v.b;
}
kernel void spill(device const float *in [[buffer(zero)]], device float *out [[buffer(one)]],
                  constant Params &p [[buffer(two)]], uint2 at [[thread_position_in_grid]]) {
    uint i = at.y*p.w+at.x;
    float3 l = readRGB(in,i)*p.exposure;
    float y = luminance(l);
    float3 v = float3(black);
    if (y > white) { v = l * (((y-white)/y)*third); }
    writeRGB(out,i,v);
}
kernel void horizontal(device const float *in [[buffer(zero)]], device const float *weights [[buffer(one)]],
                       device float *out [[buffer(two)]], constant Params &p [[buffer(three)]],
                       uint2 at [[thread_position_in_grid]]) {
    float3 sum = float3(black);
    int first = max(zero,int(p.reach)-int(at.x));
    int last = min(two*int(p.reach),int(p.reach)+int(p.w)-one-int(at.x));
    for (int k=first; k<=last; k++) {
        int x = int(at.x)+k-int(p.reach);
        float wt = weights[k];
        if (wt != black) { sum += readRGB(in,at.y*p.w+x)*wt; }
    }
    writeRGB(out,at.y*p.w+at.x,sum);
}
kernel void vertical(device const float *in [[buffer(zero)]], device const float *weights [[buffer(one)]],
                     device float *out [[buffer(two)]], constant Params &p [[buffer(three)]],
                     uint2 at [[thread_position_in_grid]]) {
    float3 sum = float3(black);
    int first = max(zero,int(p.reach)-int(at.y));
    int last = min(two*int(p.reach),int(p.reach)+int(p.h)-one-int(at.y));
    for (int k=first; k<=last; k++) {
        int y = int(at.y)+k-int(p.reach);
        float wt = weights[k];
        if (wt != black) { sum += readRGB(in,y*p.w+at.x)*wt; }
    }
    writeRGB(out,at.y*p.w+at.x,sum);
}
float encode(float v) {
    if (isnan(v)) { return black; }
    v = clamp(v,black,white);
    float srgb = v<=srgbKnee ? srgbSlope*v : srgbScale*pow(v,srgbPower)-srgbOffset;
    return floor(srgb*byteTop+halfStep);
}
kernel void finish(device const float *in [[buffer(zero)]], device const float *glow [[buffer(one)]],
                   device float *out [[buffer(two)]], constant Params &p [[buffer(three)]],
                   uint2 at [[thread_position_in_grid]]) {
    uint i = at.y*p.w+at.x;
    float3 l = readRGB(in,i)*p.exposure+readRGB(glow,i);
    // the shoulder is defined once, by Tone.Squeeze and Tone.Scale in
    // ninjafilm/colour; this restates it for the GPU, which cannot call
    // Go, and p.scale is that Scale. TestTheKernelsShoulderIsTheTones
    // holds the two together.
    if (p.knee>black && p.knee<white) {
        float y = luminance(l);
        float squeezed = y;
        if (y>p.knee) {
            squeezed = min(white,p.knee+p.scale*log(one+(y-p.knee)/p.scale));
            l *= squeezed/y;
        }
        float peak = max(l.r,max(l.g,l.b));
        if (peak>white) { l = float3(squeezed)+(l-float3(squeezed))*((white-squeezed)/(peak-squeezed)); }
    }
    out[four*i] = encode(l.r);
    out[four*i+one] = encode(l.g);
    out[four*i+two] = encode(l.b);
    out[four*i+three] = byteTop;
}
`
