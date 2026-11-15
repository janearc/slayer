//go:build darwin

package gpumatte

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/matte"
	"github.com/janearc/slayer/metal"
)

type Image = matte.Image

type Config = matte.Config

const (
	groupSize     = 256
	maxPixels     = 16777216
	maxStageBytes = 1 << 30
)

var kernelSource = source()

// Apply returns a new image where the sea image is blended into raw below the
// shore row, weighted by the mean brightness of the bright pixels of the moon
// reference (or of raw if there is none). A WaterMask, if set, decides the
// weight per pixel; a Front, if set, is added on top. It refuses nonfinite or
// misordered shore geometry, a MoonRadius of zero or less, a mask or front of
// another size than raw, a bad mask, no positive moon pixels, and over 1 GiB.
func Apply(raw, sea Image, config Config) (Image, error) {
	if err := validateImage(raw); err != nil {
		return Image{}, fmt.Errorf("gpumatte: raw image: %w", err)
	}
	if err := validateImage(sea); err != nil {
		return Image{}, fmt.Errorf("gpumatte: sea image: %w", err)
	}
	if raw.Width != sea.Width || raw.Height != sea.Height {
		return Image{}, fmt.Errorf("gpumatte: raw is %dx%d and sea is %dx%d", raw.Width, raw.Height, sea.Width, sea.Height)
	}
	if !finite(config.Top) || !finite(config.Bottom) || !finite(config.Shore) || !finite(config.MoonRadius) {
		return Image{}, errors.New("gpumatte: nonfinite matte configuration")
	}
	if !(config.Top > config.Bottom) || !(config.Shore > config.Bottom) || !(config.Shore < config.Top) {
		return Image{}, errors.New("gpumatte: invalid shoreline geometry")
	}
	if config.MoonRadius <= 0 {
		return Image{}, errors.New("gpumatte: moon radius must be positive")
	}
	if config.MoonRef != nil {
		if err := validateImage(*config.MoonRef); err != nil {
			return Image{}, fmt.Errorf("gpumatte: moon reference: %w", err)
		}
	}
	if config.WaterMask != nil {
		if err := validateMask(*config.WaterMask); err != nil {
			return Image{}, fmt.Errorf("gpumatte: water mask: %w", err)
		}
		if config.WaterMask.Width != raw.Width || config.WaterMask.Height != raw.Height {
			return Image{}, fmt.Errorf("gpumatte: water mask is %dx%d and image is %dx%d", config.WaterMask.Width, config.WaterMask.Height, raw.Width, raw.Height)
		}
	}
	if config.Front != nil {
		if err := validateImage(*config.Front); err != nil {
			return Image{}, fmt.Errorf("gpumatte: front image: %w", err)
		}
		if config.Front.Width != raw.Width || config.Front.Height != raw.Height {
			return Image{}, fmt.Errorf("gpumatte: front is %dx%d and image is %dx%d", config.Front.Width, config.Front.Height, raw.Width, raw.Height)
		}
	}
	width, height := raw.Width, raw.Height
	pixels := width * height
	moonFloats := 0
	moonPixels := pixels
	if config.MoonRef != nil {
		moonFloats = len(config.MoonRef.Pixels)
		moonPixels = moonFloats / 3
	}
	maskFloats := 0
	if config.WaterMask != nil {
		maskFloats = len(config.WaterMask.Pixels)
	}
	reductionCount := pixels
	if moonPixels > reductionCount {
		reductionCount = moonPixels
	}
	if maskFloats > reductionCount {
		reductionCount = maskFloats
	}
	scratchCount := (reductionCount + groupSize - 1) / groupSize
	if scratchCount <= 0 {
		scratchCount = 1
	}
	required := int64(len(raw.Pixels)+len(sea.Pixels)+len(raw.Pixels)) * 4
	if config.MoonRef != nil {
		required += int64(moonFloats) * 4
	}
	if config.WaterMask != nil {
		required += int64(len(config.WaterMask.Pixels)+1) * 4
	}
	if config.Front != nil {
		required += int64(len(config.Front.Pixels)) * 4
	}
	required += int64(scratchCount*2) * 4
	required += int64(5) * 4
	if required > maxStageBytes {
		return Image{}, errors.New("gpumatte: aggregate allocation exceeds 1 GiB")
	}
	session, err := metal.OpenSession(kernelSource, "max_rgb", "reduce_max", "threshold_sum", "reduce_sum", "blend_nomask", "blend_mask", "add_front")
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: open Metal: %w", err)
	}
	defer session.Close()
	rawBuf, err := session.Buffer(len(raw.Pixels))
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate raw buffer: %w", err)
	}
	seaBuf, err := session.Buffer(len(sea.Pixels))
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate sea buffer: %w", err)
	}
	outBuf, err := session.Buffer(len(raw.Pixels))
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate output buffer: %w", err)
	}
	scratchA, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate scratch buffer: %w", err)
	}
	scratchB, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate scratch buffer: %w", err)
	}
	moonMax, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate moon max buffer: %w", err)
	}
	moonRed, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate moon red buffer: %w", err)
	}
	moonGreen, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate moon green buffer: %w", err)
	}
	moonBlue, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate moon blue buffer: %w", err)
	}
	moonCount, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumatte: allocate moon count buffer: %w", err)
	}
	var maskBuf metal.Buffer
	var maskMax metal.Buffer
	if config.WaterMask != nil {
		maskBuf, err = session.Buffer(len(config.WaterMask.Pixels))
		if err != nil {
			return Image{}, fmt.Errorf("gpumatte: allocate mask buffer: %w", err)
		}
		maskMax, err = session.Buffer(1)
		if err != nil {
			return Image{}, fmt.Errorf("gpumatte: allocate mask max buffer: %w", err)
		}
	}
	var frontBuf metal.Buffer
	if config.Front != nil {
		frontBuf, err = session.Buffer(len(config.Front.Pixels))
		if err != nil {
			return Image{}, fmt.Errorf("gpumatte: allocate front buffer: %w", err)
		}
	}
	baseParams, err := params(width, height, config)
	if err != nil {
		return Image{}, err
	}
	makeParams := func(count int, channel uint32) []byte {
		p := append([]byte(nil), baseParams...)
		setCount(p, count)
		setChannel(p, channel)
		return p
	}
	copy(rawBuf.Numbers(), raw.Pixels)
	copy(seaBuf.Numbers(), sea.Pixels)
	copy(outBuf.Numbers(), raw.Pixels)
	if config.MoonRef != nil {
		moonSource, err := session.Buffer(moonFloats)
		if err != nil {
			return Image{}, fmt.Errorf("gpumatte: allocate moon buffer: %w", err)
		}
		copy(moonSource.Numbers(), config.MoonRef.Pixels)
		if err := reduceMoon(session, moonSource, moonPixels, moonMax, scratchA, scratchB, makeParams(moonPixels, 0)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, moonSource, moonMax, moonPixels, 0, moonRed, scratchA, scratchB, makeParams(moonPixels, 0)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, moonSource, moonMax, moonPixels, 1, moonGreen, scratchA, scratchB, makeParams(moonPixels, 1)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, moonSource, moonMax, moonPixels, 2, moonBlue, scratchA, scratchB, makeParams(moonPixels, 2)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, moonSource, moonMax, moonPixels, 3, moonCount, scratchA, scratchB, makeParams(moonPixels, 3)); err != nil {
			return Image{}, err
		}
	} else {
		if err := reduceMoon(session, rawBuf, pixels, moonMax, scratchA, scratchB, makeParams(pixels, 0)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, rawBuf, moonMax, pixels, 0, moonRed, scratchA, scratchB, makeParams(pixels, 0)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, rawBuf, moonMax, pixels, 1, moonGreen, scratchA, scratchB, makeParams(pixels, 1)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, rawBuf, moonMax, pixels, 2, moonBlue, scratchA, scratchB, makeParams(pixels, 2)); err != nil {
			return Image{}, err
		}
		if err := thresholdChannel(session, rawBuf, moonMax, pixels, 3, moonCount, scratchA, scratchB, makeParams(pixels, 3)); err != nil {
			return Image{}, err
		}
	}
	if config.WaterMask != nil {
		copy(maskBuf.Numbers(), config.WaterMask.Pixels)
		if err := reduceScalar(session, "reduce_max", maskBuf, len(config.WaterMask.Pixels), maskMax, scratchA, scratchB, makeParams(len(config.WaterMask.Pixels), 0)); err != nil {
			return Image{}, err
		}
	}
	blendParams := makeParams(pixels, 0)
	if config.WaterMask == nil {
		if err := session.Dispatch("blend_nomask", width, height, blendParams, rawBuf, seaBuf, moonRed, moonGreen, moonBlue, moonCount, outBuf); err != nil {
			return Image{}, fmt.Errorf("gpumatte: dispatch blend_nomask: %w", err)
		}
	} else {
		if err := session.Dispatch("blend_mask", width, height, blendParams, rawBuf, seaBuf, moonRed, moonGreen, moonBlue, moonCount, maskBuf, maskMax, outBuf); err != nil {
			return Image{}, fmt.Errorf("gpumatte: dispatch blend_mask: %w", err)
		}
	}
	if config.Front != nil {
		copy(frontBuf.Numbers(), config.Front.Pixels)
		if err := session.Dispatch("add_front", width, height, blendParams, outBuf, frontBuf); err != nil {
			return Image{}, fmt.Errorf("gpumatte: dispatch add_front: %w", err)
		}
	}
	if err := session.Commit(); err != nil {
		return Image{}, fmt.Errorf("gpumatte: commit: %w", err)
	}
	if err := validateFiniteBuffer("moon max reduction", moonMax.Numbers()); err != nil {
		return Image{}, err
	}
	if err := validateFiniteBuffer("moon red reduction", moonRed.Numbers()); err != nil {
		return Image{}, err
	}
	if err := validateFiniteBuffer("moon green reduction", moonGreen.Numbers()); err != nil {
		return Image{}, err
	}
	if err := validateFiniteBuffer("moon blue reduction", moonBlue.Numbers()); err != nil {
		return Image{}, err
	}
	if err := validateFiniteBuffer("moon count reduction", moonCount.Numbers()); err != nil {
		return Image{}, err
	}
	if config.WaterMask != nil {
		if err := validateFiniteBuffer("water mask reduction", maskMax.Numbers()); err != nil {
			return Image{}, err
		}
		if maskMax.Numbers()[0] <= 0 {
			return Image{}, errors.New("gpumatte: water mask must contain positive light")
		}
	}
	if err := validateFiniteBuffer("output image", outBuf.Numbers()); err != nil {
		return Image{}, err
	}
	if moonCount.Numbers()[0] <= 0 {
		return Image{}, errors.New("gpumatte: moon reference produced no positive pixels")
	}
	return Image{Width: width, Height: height, Scale: raw.Scale, Pixels: append([]float32(nil), outBuf.Numbers()...)}, nil
}

// reduceMoon writes to final the largest red plus green plus blue sum among
// the first pixels pixels of source. The first pass runs max_rgb and later
// passes run reduce_max, using scratchA and scratchB. It overwrites the count
// in p.
func reduceMoon(session *metal.Session, source metal.Buffer, pixels int, final metal.Buffer, scratchA, scratchB metal.Buffer, p []byte) error {
	groups := (pixels + groupSize - 1) / groupSize
	if groups == 1 {
		setCount(p, pixels)
		if err := session.Grouped("max_rgb", groupSize, 1, p, source, final); err != nil {
			return fmt.Errorf("gpumatte: dispatch max_rgb: %w", err)
		}
		return nil
	}
	setCount(p, pixels)
	if err := session.Grouped("max_rgb", groupSize, groups, p, source, scratchA); err != nil {
		return fmt.Errorf("gpumatte: dispatch max_rgb: %w", err)
	}
	current := scratchA
	next := scratchB
	count := groups
	for count > groupSize {
		groups = (count + groupSize - 1) / groupSize
		setCount(p, count)
		if err := session.Grouped("reduce_max", groupSize, groups, p, current, next); err != nil {
			return fmt.Errorf("gpumatte: dispatch reduce_max: %w", err)
		}
		current, next = next, current
		count = groups
	}
	setCount(p, count)
	if err := session.Grouped("reduce_max", groupSize, 1, p, current, final); err != nil {
		return fmt.Errorf("gpumatte: dispatch reduce_max: %w", err)
	}
	return nil
}

// reduceScalar reduces the first count floats of source to one value in
// final, running kernel on every pass, with scratchA and scratchB holding
// the partial results. It overwrites the count in p.
func reduceScalar(session *metal.Session, kernel string, source metal.Buffer, count int, final metal.Buffer, scratchA, scratchB metal.Buffer, p []byte) error {
	if count == 1 {
		setCount(p, 1)
		if err := session.Grouped(kernel, groupSize, 1, p, source, final); err != nil {
			return fmt.Errorf("gpumatte: dispatch %s: %w", kernel, err)
		}
		return nil
	}
	groups := (count + groupSize - 1) / groupSize
	current := source
	if groups == 1 {
		setCount(p, count)
		if err := session.Grouped(kernel, groupSize, 1, p, source, final); err != nil {
			return fmt.Errorf("gpumatte: dispatch %s: %w", kernel, err)
		}
		return nil
	}
	setCount(p, count)
	if err := session.Grouped(kernel, groupSize, groups, p, source, scratchA); err != nil {
		return fmt.Errorf("gpumatte: dispatch %s: %w", kernel, err)
	}
	current = scratchA
	next := scratchB
	for groups > groupSize {
		count = groups
		groups = (count + groupSize - 1) / groupSize
		setCount(p, count)
		if err := session.Grouped(kernel, groupSize, groups, p, current, next); err != nil {
			return fmt.Errorf("gpumatte: dispatch %s: %w", kernel, err)
		}
		current, next = next, current
	}
	setCount(p, groups)
	if err := session.Grouped(kernel, groupSize, 1, p, current, final); err != nil {
		return fmt.Errorf("gpumatte: dispatch %s: %w", kernel, err)
	}
	return nil
}

// thresholdChannel adds up one channel over the pixels of source whose red
// plus green plus blue is at least half of maxValue, and writes the total to
// final. Channel 0, 1 or 2 picks red, green or blue; any other value counts
// the pixels. It overwrites the count and channel in p.
func thresholdChannel(session *metal.Session, source, maxValue metal.Buffer, pixels int, channel uint32, final metal.Buffer, scratchA, scratchB metal.Buffer, p []byte) error {
	groups := (pixels + groupSize - 1) / groupSize
	if groups == 1 {
		setCount(p, pixels)
		setChannel(p, channel)
		if err := session.Grouped("threshold_sum", groupSize, 1, p, source, maxValue, final); err != nil {
			return fmt.Errorf("gpumatte: dispatch threshold_sum: %w", err)
		}
		return nil
	}
	setCount(p, pixels)
	setChannel(p, channel)
	if err := session.Grouped("threshold_sum", groupSize, groups, p, source, maxValue, scratchA); err != nil {
		return fmt.Errorf("gpumatte: dispatch threshold_sum: %w", err)
	}
	current := scratchA
	next := scratchB
	count := groups
	for count > groupSize {
		groups = (count + groupSize - 1) / groupSize
		setCount(p, count)
		if err := session.Grouped("reduce_sum", groupSize, groups, p, current, next); err != nil {
			return fmt.Errorf("gpumatte: dispatch reduce_sum: %w", err)
		}
		current, next = next, current
		count = groups
	}
	setCount(p, count)
	if err := session.Grouped("reduce_sum", groupSize, 1, p, current, final); err != nil {
		return fmt.Errorf("gpumatte: dispatch reduce_sum: %w", err)
	}
	return nil
}

// params returns the 32-byte block of settings the matte kernels read: width,
// height, the shore row in pixels, a row scale of one eighth, and the moon
// disc, which is pi times the square of MoonRadius taken as degrees and
// converted to radians. Count and channel are left zero. It fails if a value
// does not fit in a float32.
func params(width, height int, config Config) ([]byte, error) {
	b := make([]byte, 32)
	setUint32(b, 8, uint32(width))
	setUint32(b, 12, uint32(height))
	if err := setFiniteFloat32(b, 16, (config.Top-config.Shore)/(config.Top-config.Bottom)*float64(height)); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 20, 1.0/(2*4.0)); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 24, math.Pi*math.Pow(config.MoonRadius*math.Pi/180, 2)); err != nil {
		return nil, err
	}
	return b, nil
}

// setCount stores count in the first four bytes of the parameter block p.
func setCount(p []byte, count int) {
	setUint32(p, 0, uint32(count))
}

// setChannel stores channel at byte 4 of p; threshold_sum reads it to choose
// what to add up.
func setChannel(p []byte, channel uint32) {
	setUint32(p, 4, channel)
}

// setUint32 writes value little-endian into the four bytes of p at off.
func setUint32(p []byte, off int, value uint32) {
	binary.LittleEndian.PutUint32(p[off:], value)
}

// setFloat32 writes the bits of value little-endian into the four bytes of p
// at off.
func setFloat32(p []byte, off int, value float32) {
	binary.LittleEndian.PutUint32(p[off:], math.Float32bits(value))
}

// setFiniteFloat32 narrows value to a float32 and writes it into p at off. It
// fails if the result is NaN or infinite, or if a nonzero value became zero.
func setFiniteFloat32(p []byte, off int, value float64) error {
	converted := float32(value)
	if !finite(float64(converted)) || (value != 0 && converted == 0) {
		return errors.New("gpumatte: derived parameters are not representable as float32")
	}
	setFloat32(p, off, converted)
	return nil
}

// validateFiniteBuffer fails, naming name, if any of values is NaN or
// infinite. It is used on what the GPU wrote back.
func validateFiniteBuffer(name string, values []float32) error {
	for _, value := range values {
		if !finite(float64(value)) {
			return fmt.Errorf("gpumatte: %s produced nonfinite value", name)
		}
	}
	return nil
}

// validateImage refuses an image with bad dimensions, a nonfinite Scale, a
// Pixels length other than width times height times three, or any pixel that
// is NaN or infinite.
func validateImage(image Image) error {
	if !dimensions(image.Width, image.Height) || !finite(image.Scale) || len(image.Pixels) != image.Width*image.Height*3 {
		return errors.New("invalid image")
	}
	for _, value := range image.Pixels {
		if !finite(float64(value)) {
			return errors.New("nonfinite pixel")
		}
	}
	return nil
}

// validateMask applies validateImage to image and also refuses negative
// values and positive subnormal values.
func validateMask(image Image) error {
	if err := validateImage(image); err != nil {
		return err
	}
	for _, value := range image.Pixels {
		if value < 0 {
			return errors.New("negative mask value")
		}
		if isPositiveSubnormalMaskValue(value) {
			return errors.New("water mask contains positive subnormal values")
		}
	}
	return nil
}

// isPositiveSubnormalMaskValue reports whether value is a positive float32
// with an all-zero exponent and a nonzero fraction.
func isPositiveSubnormalMaskValue(value float32) bool {
	bits := math.Float32bits(value)
	return bits != 0 && bits&0x80000000 == 0 && bits&0x7f800000 == 0
}

// dimensions reports whether width and height are positive, give no more
// than maxPixels pixels, and leave width times height times three within the
// range of int.
func dimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= maxPixels/height && width <= int(^uint(0)>>1)/height/3
}

// finite reports whether value is neither NaN nor infinite.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// source returns the Metal source for the matte kernels: the constants, the
// helper functions, the max, sum and threshold kernels, the two blends and
// add_front. The group size in it is groupSize.
func source() string {
	var b strings.Builder
	b.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"three", 3}, {"halfValue", 0.5}, {"feather", 4}, {"boost", 1.25}} {
		fmt.Fprintf(&b, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	for _, number := range []struct {
		name  string
		value uint32
	}{{"zeroU", 0}, {"oneU", 1}, {"twoU", 2}, {"threeU", 3}, {"fourU", 4}} {
		fmt.Fprintf(&b, "constant uint %s = %du;\n", number.name, number.value)
	}
	fmt.Fprintf(&b, "constant uint block = %d;\n", groupSize)
	b.WriteString(`
struct Params { uint count; uint channel; uint width; uint height; float shoreRow; float rowScale; float disc; float pad; };
float3 readRGB(device const float *in, uint i) { return float3(in[3*i], in[3*i+1], in[3*i+2]); }
void writeRGB(device float *out, uint i, float3 v) { out[3*i] = v.x; out[3*i+1] = v.y; out[3*i+2] = v.z; }
kernel void max_rgb(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant Params &p [[buffer(2)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
    threadgroup float a[block];
    uint i = gid * block + tid;
    float v = zero;
    if (i < p.count) { float3 c = readRGB(in, i); v = c.x + c.y + c.z; }
    a[tid] = v;
    threadgroup_barrier(mem_flags::mem_threadgroup);
    for (uint span = block / twoU; span > zeroU; span >>= oneU) {
        if (tid < span) { a[tid] = max(a[tid], a[tid + span]); }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (tid == zeroU) { out[gid] = a[zeroU]; }
}
kernel void reduce_max(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant Params &p [[buffer(2)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
    threadgroup float a[block];
    uint i = gid * block + tid;
    float v = zero;
    if (i < p.count) { v = in[i]; }
    a[tid] = v;
    threadgroup_barrier(mem_flags::mem_threadgroup);
    for (uint span = block / twoU; span > zeroU; span >>= oneU) {
        if (tid < span) { a[tid] = max(a[tid], a[tid + span]); }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (tid == zeroU) { out[gid] = a[zeroU]; }
}
kernel void threshold_sum(device const float *in [[buffer(0)]], device const float *limit [[buffer(1)]], device float *out [[buffer(2)]], constant Params &p [[buffer(3)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
    threadgroup float a[block];
    uint i = gid * block + tid;
    float v = zero;
    if (i < p.count) {
        float3 c = readRGB(in, i);
        float sum = c.x + c.y + c.z;
        if (sum >= limit[0] * halfValue) {
            if (p.channel == zeroU) { v = c.x; }
            else if (p.channel == oneU) { v = c.y; }
            else if (p.channel == twoU) { v = c.z; }
            else { v = one; }
        }
    }
    a[tid] = v;
    threadgroup_barrier(mem_flags::mem_threadgroup);
    for (uint span = block / twoU; span > zeroU; span >>= oneU) {
        if (tid < span) { a[tid] += a[tid + span]; }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (tid == zeroU) { out[gid] = a[zeroU]; }
}
kernel void reduce_sum(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant Params &p [[buffer(2)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
    threadgroup float a[block];
    uint i = gid * block + tid;
    float v = zero;
    if (i < p.count) { v = in[i]; }
    a[tid] = v;
    threadgroup_barrier(mem_flags::mem_threadgroup);
    for (uint span = block / twoU; span > zeroU; span >>= oneU) {
        if (tid < span) { a[tid] += a[tid + span]; }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (tid == zeroU) { out[gid] = a[zeroU]; }
}
kernel void blend_nomask(device const float *raw [[buffer(0)]], device const float *sea [[buffer(1)]], device const float *moonRed [[buffer(2)]], device const float *moonGreen [[buffer(3)]], device const float *moonBlue [[buffer(4)]], device const float *moonCount [[buffer(5)]], device float *out [[buffer(6)]], constant Params &p [[buffer(7)]], uint2 at [[thread_position_in_grid]]) {
    if (at.x >= p.width || at.y >= p.height) { return; }
    float row = float(p.height - oneU - at.y);
    float w = clamp((row - p.shoreRow + feather) * p.rowScale, zero, one);
    if (w == zero) { return; }
    uint i = at.y * p.width + at.x;
    float3 base = readRGB(raw, i);
    float3 ocean = readRGB(sea, i);
    float count = moonCount[0];
    float moon = count > zero ? (moonRed[0] + moonGreen[0] + moonBlue[0]) / (three * count) : zero;
    float3 water = ocean * moon * p.disc;
    writeRGB(out, i, base * (one - w) + water * w);
}
kernel void blend_mask(device const float *raw [[buffer(0)]], device const float *sea [[buffer(1)]], device const float *moonRed [[buffer(2)]], device const float *moonGreen [[buffer(3)]], device const float *moonBlue [[buffer(4)]], device const float *moonCount [[buffer(5)]], device const float *mask [[buffer(6)]], device const float *maskMax [[buffer(7)]], device float *out [[buffer(8)]], constant Params &p [[buffer(9)]], uint2 at [[thread_position_in_grid]]) {
    if (at.x >= p.width || at.y >= p.height) { return; }
    uint i = at.y * p.width + at.x;
    float3 base = readRGB(raw, i);
    float3 ocean = readRGB(sea, i);
    float count = moonCount[0];
    float moon = count > zero ? (moonRed[0] + moonGreen[0] + moonBlue[0]) / (three * count) : zero;
    float3 m = readRGB(mask, i);
    float sum = m.x + m.y + m.z;
    float w = zero;
    if (maskMax[0] > zero && sum > zero) {
        w = min(one, sum * boost / (three * maskMax[0]));
    }
    if (w == zero) { return; }
    float3 water = ocean * moon * p.disc;
    writeRGB(out, i, base * (one - w) + water * w);
}
kernel void add_front(device float *out [[buffer(0)]], device const float *front [[buffer(1)]], constant Params &p [[buffer(2)]], uint2 at [[thread_position_in_grid]]) {
    if (at.x >= p.width || at.y >= p.height) { return; }
    uint i = at.y * p.width + at.x;
    float3 v = readRGB(out, i) + readRGB(front, i);
    writeRGB(out, i, v);
}
`)
	return b.String()
}
