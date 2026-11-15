package gpumist

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/mist"
	"github.com/janearc/slayer/metal"
)

type Image = mist.Image

type Config = mist.Config

const (
	applyGroupSize = 256
	applyMaxBytes  = 1 << 30
	flagWaterMask  = 1
	flagTreeMask   = 2
)

var mistKernelSource = mistSource()

// Apply returns a copy of image with mist blended in around and below the
// shore row, drawn on the GPU through Metal; image is not changed. It refuses
// nonfinite values, Bottom < Shore < Top not holding, Layers under 1, a moon
// reference, water mask or tree mask whose size differs from image, a water
// mask with no positive value, a moon reference with no light, and buffers
// over 1 GiB in all.
func Apply(image Image, config Config) (Image, error) {
	if err := validateImage(image); err != nil {
		return Image{}, fmt.Errorf("gpumist: image: %w", err)
	}
	for _, value := range []float64{config.Top, config.Bottom, config.Shore, config.Strength, config.Seconds, config.Layers, config.Colour[0], config.Colour[1], config.Colour[2]} {
		if !finite(value) {
			return Image{}, errors.New("gpumist: nonfinite mist configuration")
		}
	}
	if config.Top <= config.Bottom || config.Shore <= config.Bottom || config.Shore >= config.Top || config.Layers < 1 {
		return Image{}, errors.New("gpumist: invalid mist configuration")
	}
	if config.MoonRef != nil {
		if err := validateImage(*config.MoonRef); err != nil {
			return Image{}, fmt.Errorf("gpumist: moon reference: %w", err)
		}
		if config.MoonRef.Width != image.Width || config.MoonRef.Height != image.Height {
			return Image{}, fmt.Errorf("gpumist: moon reference is %dx%d and image is %dx%d", config.MoonRef.Width, config.MoonRef.Height, image.Width, image.Height)
		}
	}
	if config.WaterMask != nil {
		if err := validateMask(*config.WaterMask); err != nil {
			return Image{}, fmt.Errorf("gpumist: water mask: %w", err)
		}
		if config.WaterMask.Width != image.Width || config.WaterMask.Height != image.Height {
			return Image{}, fmt.Errorf("gpumist: water mask is %dx%d and image is %dx%d", config.WaterMask.Width, config.WaterMask.Height, image.Width, image.Height)
		}
	}
	if config.TreeMask != nil {
		if err := validateMask(*config.TreeMask); err != nil {
			return Image{}, fmt.Errorf("gpumist: tree mask: %w", err)
		}
		if config.TreeMask.Width != image.Width || config.TreeMask.Height != image.Height {
			return Image{}, fmt.Errorf("gpumist: tree mask is %dx%d and image is %dx%d", config.TreeMask.Width, config.TreeMask.Height, image.Width, image.Height)
		}
	}
	width, height := image.Width, image.Height
	pixels := width * height
	moonPixels := pixels
	if config.MoonRef != nil {
		moonPixels = len(config.MoonRef.Pixels) / 3
	}
	waterFloats := 1
	if config.WaterMask != nil {
		waterFloats = len(config.WaterMask.Pixels)
	}
	treeFloats := 1
	if config.TreeMask != nil {
		treeFloats = len(config.TreeMask.Pixels)
	}
	reductionCount := pixels
	if moonPixels > reductionCount {
		reductionCount = moonPixels
	}
	if waterFloats > reductionCount {
		reductionCount = waterFloats
	}
	if treeFloats > reductionCount {
		reductionCount = treeFloats
	}
	scratchCount := (reductionCount + applyGroupSize - 1) / applyGroupSize
	if scratchCount < 1 {
		scratchCount = 1
	}
	required := int64(len(image.Pixels)+len(image.Pixels)) * 4
	if config.MoonRef != nil {
		required += int64(len(config.MoonRef.Pixels)) * 4
	}
	required += int64(waterFloats+treeFloats) * 4
	required += int64(8) * 4
	required += int64(scratchCount*5) * 4
	required += int64(scratchCount*2) * 4
	if required > applyMaxBytes {
		return Image{}, errors.New("gpumist: aggregate allocation exceeds 1 GiB")
	}
	session, err := metal.OpenSession(mistKernelSource, "max_moon", "max_mask", "reduce_max", "threshold_moon", "reduce_sum", "blend")
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: open Metal: %w", err)
	}
	defer session.Close()

	rawBuf, err := session.Buffer(len(image.Pixels))
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate image buffer: %w", err)
	}
	copy(rawBuf.Numbers(), image.Pixels)

	var moonBuf metal.Buffer
	if config.MoonRef != nil {
		moonBuf, err = session.Buffer(len(config.MoonRef.Pixels))
		if err != nil {
			return Image{}, fmt.Errorf("gpumist: allocate moon buffer: %w", err)
		}
		copy(moonBuf.Numbers(), config.MoonRef.Pixels)
	}

	waterBuf, err := session.Buffer(waterFloats)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate water buffer: %w", err)
	}
	treeBuf, err := session.Buffer(treeFloats)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate tree buffer: %w", err)
	}
	if config.WaterMask != nil {
		copy(waterBuf.Numbers(), config.WaterMask.Pixels)
	} else {
		waterBuf.Numbers()[0] = 0
	}
	if config.TreeMask != nil {
		copy(treeBuf.Numbers(), config.TreeMask.Pixels)
	} else {
		treeBuf.Numbers()[0] = 0
	}

	outBuf, err := session.Buffer(len(image.Pixels))
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate output buffer: %w", err)
	}
	moonMax, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon max buffer: %w", err)
	}
	waterMax, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate water max buffer: %w", err)
	}
	treeMax, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate tree max buffer: %w", err)
	}
	moonRed, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon red buffer: %w", err)
	}
	moonGreen, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon green buffer: %w", err)
	}
	moonBlue, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon blue buffer: %w", err)
	}
	moonCount, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon count buffer: %w", err)
	}
	moonColumn, err := session.Buffer(1)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon column buffer: %w", err)
	}
	moonRedTmp, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon red scratch: %w", err)
	}
	moonGreenTmp, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon green scratch: %w", err)
	}
	moonBlueTmp, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon blue scratch: %w", err)
	}
	moonCountTmp, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon count scratch: %w", err)
	}
	moonColumnTmp, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate moon column scratch: %w", err)
	}
	scratchA, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate scratch buffer: %w", err)
	}
	scratchB, err := session.Buffer(scratchCount)
	if err != nil {
		return Image{}, fmt.Errorf("gpumist: allocate scratch buffer: %w", err)
	}

	moonRows := int((config.Top - config.Shore) / (config.Top - config.Bottom) * float64(height))
	baseParams, err := params(width, height, moonRows, config)
	if err != nil {
		return Image{}, err
	}
	makeParams := func(count int) []byte {
		p := append([]byte(nil), baseParams...)
		setCount(p, count)
		return p
	}

	moonSource := rawBuf
	if config.MoonRef != nil {
		moonSource = moonBuf
	}
	if err := reduceMax(session, "max_moon", moonSource, moonPixels, moonMax, scratchA, scratchB, makeParams(moonPixels)); err != nil {
		return Image{}, err
	}
	if config.WaterMask != nil {
		if err := reduceMax(session, "max_mask", waterBuf, len(config.WaterMask.Pixels), waterMax, scratchA, scratchB, makeParams(len(config.WaterMask.Pixels))); err != nil {
			return Image{}, err
		}
	}
	if config.TreeMask != nil {
		if err := reduceMax(session, "max_mask", treeBuf, len(config.TreeMask.Pixels), treeMax, scratchA, scratchB, makeParams(len(config.TreeMask.Pixels))); err != nil {
			return Image{}, err
		}
	}
	if err := session.Commit(); err != nil {
		return Image{}, fmt.Errorf("gpumist: commit maxima: %w", err)
	}
	if err := validateFiniteBuffer("moon max", moonMax.Numbers()); err != nil {
		return Image{}, err
	}
	if moonMax.Numbers()[0] <= 0 {
		return Image{}, errors.New("gpumist: moon reference produced no positive pixels")
	}
	if config.WaterMask != nil {
		if err := validateFiniteBuffer("water max", waterMax.Numbers()); err != nil {
			return Image{}, err
		}
		if waterMax.Numbers()[0] <= 0 {
			return Image{}, errors.New("gpumist: water mask must contain positive light")
		}
	}
	if config.TreeMask != nil {
		if err := validateFiniteBuffer("tree max", treeMax.Numbers()); err != nil {
			return Image{}, err
		}
	}

	if err := thresholdMoon(session, moonSource, moonMax, moonPixels, moonRed, moonGreen, moonBlue, moonCount, moonColumn, moonRedTmp, moonGreenTmp, moonBlueTmp, moonCountTmp, moonColumnTmp, scratchA, scratchB, makeParams(moonPixels)); err != nil {
		return Image{}, err
	}
	if err := session.Commit(); err != nil {
		return Image{}, fmt.Errorf("gpumist: commit moon threshold: %w", err)
	}
	for name, buffer := range map[string]metal.Buffer{"moon red": moonRed, "moon green": moonGreen, "moon blue": moonBlue, "moon count": moonCount, "moon column": moonColumn} {
		if err := validateFiniteBuffer(name, buffer.Numbers()); err != nil {
			return Image{}, err
		}
	}
	if moonCount.Numbers()[0] <= 0 {
		return Image{}, errors.New("gpumist: moon reference threshold produced no positive pixels")
	}

	blendParams := makeParams(pixels)
	flags := uint32(0)
	if config.WaterMask != nil {
		flags |= flagWaterMask
	}
	if config.TreeMask != nil {
		flags |= flagTreeMask
	}
	setFlags(blendParams, flags)
	if err := session.Dispatch("blend", width, height, blendParams, rawBuf, moonRed, moonGreen, moonBlue, moonCount, moonColumn, waterBuf, waterMax, treeBuf, treeMax, outBuf); err != nil {
		return Image{}, fmt.Errorf("gpumist: dispatch blend: %w", err)
	}
	if err := session.Commit(); err != nil {
		return Image{}, fmt.Errorf("gpumist: commit blend: %w", err)
	}
	if err := validateFiniteBuffer("output image", outBuf.Numbers()); err != nil {
		return Image{}, err
	}
	return Image{Width: width, Height: height, Scale: image.Scale, Pixels: append([]float32(nil), outBuf.Numbers()...)}, nil
}

// reduceMax writes the largest of the first count floats of source to final.
// The first pass runs kernel and later passes run reduce_max, with scratchA
// and scratchB holding the partial results. It overwrites the count in p.
func reduceMax(session *metal.Session, kernel string, source metal.Buffer, count int, final metal.Buffer, scratchA, scratchB metal.Buffer, p []byte) error {
	groups := (count + applyGroupSize - 1) / applyGroupSize
	if groups == 1 {
		setCount(p, count)
		if err := session.Grouped(kernel, applyGroupSize, 1, p, source, final); err != nil {
			return fmt.Errorf("gpumist: dispatch %s: %w", kernel, err)
		}
		return nil
	}
	setCount(p, count)
	if err := session.Grouped(kernel, applyGroupSize, groups, p, source, scratchA); err != nil {
		return fmt.Errorf("gpumist: dispatch %s: %w", kernel, err)
	}
	current := scratchA
	next := scratchB
	currentCount := groups
	for currentCount > applyGroupSize {
		groups = (currentCount + applyGroupSize - 1) / applyGroupSize
		setCount(p, currentCount)
		if err := session.Grouped("reduce_max", applyGroupSize, groups, p, current, next); err != nil {
			return fmt.Errorf("gpumist: dispatch reduce_max: %w", err)
		}
		current, next = next, current
		currentCount = groups
	}
	setCount(p, currentCount)
	if err := session.Grouped("reduce_max", applyGroupSize, 1, p, current, final); err != nil {
		return fmt.Errorf("gpumist: dispatch reduce_max: %w", err)
	}
	return nil
}

// thresholdMoon adds up, over the count pixels of source whose red plus green
// plus blue is at least half of moonMax, their red, green and blue values,
// their number and the sum of their x columns, giving the five totals to red,
// green, blue, total and column. Only the rows p allows through moonRows are
// looked at. It overwrites the count in p.
func thresholdMoon(session *metal.Session, source, moonMax metal.Buffer, count int, red, green, blue, total, column metal.Buffer, redTmp, greenTmp, blueTmp, totalTmp, columnTmp, scratchA, scratchB metal.Buffer, p []byte) error {
	groups := (count + applyGroupSize - 1) / applyGroupSize
	if groups == 1 {
		setCount(p, count)
		if err := session.Grouped("threshold_moon", applyGroupSize, 1, p, source, moonMax, red, green, blue, total, column); err != nil {
			return fmt.Errorf("gpumist: dispatch threshold_moon: %w", err)
		}
		return nil
	}
	setCount(p, count)
	if err := session.Grouped("threshold_moon", applyGroupSize, groups, p, source, moonMax, redTmp, greenTmp, blueTmp, totalTmp, columnTmp); err != nil {
		return fmt.Errorf("gpumist: dispatch threshold_moon: %w", err)
	}
	if err := reduceSum(session, redTmp, groups, red, scratchA, scratchB, p); err != nil {
		return err
	}
	if err := reduceSum(session, greenTmp, groups, green, scratchA, scratchB, p); err != nil {
		return err
	}
	if err := reduceSum(session, blueTmp, groups, blue, scratchA, scratchB, p); err != nil {
		return err
	}
	if err := reduceSum(session, totalTmp, groups, total, scratchA, scratchB, p); err != nil {
		return err
	}
	if err := reduceSum(session, columnTmp, groups, column, scratchA, scratchB, p); err != nil {
		return err
	}
	return nil
}

// reduceSum adds up the first count floats of source into final. The first
// pass and later passes all run reduce_sum, with scratchA and scratchB
// holding the partial sums. It overwrites the count in p.
func reduceSum(session *metal.Session, source metal.Buffer, count int, final metal.Buffer, scratchA, scratchB metal.Buffer, p []byte) error {
	groups := (count + applyGroupSize - 1) / applyGroupSize
	if groups == 1 {
		setCount(p, count)
		if err := session.Grouped("reduce_sum", applyGroupSize, 1, p, source, final); err != nil {
			return fmt.Errorf("gpumist: dispatch reduce_sum: %w", err)
		}
		return nil
	}
	setCount(p, count)
	if err := session.Grouped("reduce_sum", applyGroupSize, groups, p, source, scratchA); err != nil {
		return fmt.Errorf("gpumist: dispatch reduce_sum: %w", err)
	}
	current := scratchA
	next := scratchB
	currentCount := groups
	for currentCount > applyGroupSize {
		groups = (currentCount + applyGroupSize - 1) / applyGroupSize
		setCount(p, currentCount)
		if err := session.Grouped("reduce_sum", applyGroupSize, groups, p, current, next); err != nil {
			return fmt.Errorf("gpumist: dispatch reduce_sum: %w", err)
		}
		current, next = next, current
		currentCount = groups
	}
	setCount(p, currentCount)
	if err := session.Grouped("reduce_sum", applyGroupSize, 1, p, current, final); err != nil {
		return fmt.Errorf("gpumist: dispatch reduce_sum: %w", err)
	}
	return nil
}

// params returns the 64-byte block of settings the mist kernels read: count,
// width, height, moonRows, flags (left zero here), shore row, rise, lake,
// strength, layers, seconds, the three colour values and the seed in two
// 32-bit halves. It fails if a derived value does not fit in a float32.
func params(width, height, moonRows int, config Config) ([]byte, error) {
	b := make([]byte, 64)
	setCount(b, width*height)
	setUint32(b, 4, uint32(width))
	setUint32(b, 8, uint32(height))
	setUint32(b, 12, uint32(moonRows))
	setUint32(b, 16, 0)
	if err := setFiniteFloat32(b, 20, (config.Top-config.Shore)/(config.Top-config.Bottom)*float64(height)); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 24, 1.6/(config.Top-config.Bottom)*float64(height)); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 28, float64(height)-((config.Top-config.Shore)/(config.Top-config.Bottom))*float64(height)); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 32, config.Strength); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 36, config.Layers); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 40, config.Seconds); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 44, config.Colour[0]); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 48, config.Colour[1]); err != nil {
		return nil, err
	}
	if err := setFiniteFloat32(b, 52, config.Colour[2]); err != nil {
		return nil, err
	}
	setUint32(b, 56, uint32(config.Seed))
	setUint32(b, 60, uint32(config.Seed>>32))
	return b, nil
}

// setCount stores count in the first four bytes of the parameter block p.
func setCount(p []byte, count int) {
	setUint32(p, 0, uint32(count))
}

// setFlags stores flags at byte 16 of p; blend reads it to see which masks
// are present.
func setFlags(p []byte, flags uint32) {
	setUint32(p, 16, flags)
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
		return errors.New("gpumist: derived parameters are not representable as float32")
	}
	setFloat32(p, off, converted)
	return nil
}

// validateFiniteBuffer fails, naming name, if any of values is NaN or
// infinite. It is used on what the GPU wrote back.
func validateFiniteBuffer(name string, values []float32) error {
	for _, value := range values {
		if !finite(float64(value)) {
			return fmt.Errorf("gpumist: %s produced nonfinite value", name)
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

// validateMask applies validateImage to image and also refuses any negative
// value.
func validateMask(image Image) error {
	if err := validateImage(image); err != nil {
		return err
	}
	for _, value := range image.Pixels {
		if value < 0 {
			return errors.New("negative mask value")
		}
	}
	return nil
}

// dimensions reports whether width and height are positive, give no more
// than mist.MaxPixels pixels, and leave width times height times three
// within the range of int.
func dimensions(width, height int) bool {
	return width > 0 && height > 0 && width <= mist.MaxPixels/height && width <= int(^uint(0)>>1)/height/3
}

// finite reports whether value is neither NaN nor infinite.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// mistSource returns the Metal source for the mist kernels: the constants,
// the helper functions, the three max kernels, threshold_moon, reduce_sum and
// blend. The group size in it is applyGroupSize.
func mistSource() string {
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
	}{{"zeroU", 0}, {"oneU", 1}, {"twoU", 2}, {"threeU", 3}, {"fourU", 4}, {"fiveU", 5}} {
		fmt.Fprintf(&b, "constant uint %s = %du;\n", number.name, number.value)
	}
	for _, constant := range []struct {
		name  string
		value uint64
	}{{"latticeX", 0x9E3779B97F4A7C15}, {"latticeY", 0xC2B2AE3D27D4EB4F}, {"latticeSeed", 0x165667B19E3779F9}, {"latticeMix", 0xff51afd7ed558ccd}} {
		fmt.Fprintf(&b, "constant ulong %s = %dul;\n", constant.name, constant.value)
	}
	fmt.Fprintf(&b, "constant uint latticeShift = %du;\n", 33)
	fmt.Fprintf(&b, "constant float latticeScale = %.17ef;\n", math.Ldexp(1, -53))
	fmt.Fprintf(&b, "constant uint block = %du;\n", applyGroupSize)
	b.WriteString(`
struct Params { uint count; uint width; uint height; uint moonRows; uint flags; float shoreRow; float rise; float lake; float strength; float layers; float seconds; float colour0; float colour1; float colour2; uint seedLo; uint seedHi; };
float3 readRGB(device const float *in, uint i) { return float3(in[3*i], in[3*i+1], in[3*i+2]); }
void writeRGB(device float *out, uint i, float3 v) { out[3*i] = v.x; out[3*i+1] = v.y; out[3*i+2] = v.z; }
ulong seed(constant Params &p) { return ((ulong)p.seedHi << 32u) | (ulong)p.seedLo; }
float smooth(float a, float b, float x) { float t = clamp((x - a) / (b - a), zero, one); return t * t * (three - two * t); }
float thickened(float d, float n) { return one - pow(one - min(one, d), n) + max(zero, d - one); }
float lattice(long x, long y, ulong s) { ulong h = (ulong)x * latticeX ^ (ulong)y * latticeY ^ s * latticeSeed; h ^= h >> latticeShift; h *= latticeMix; h ^= h >> latticeShift; return float(h >> 11u) * latticeScale; }
float noise(float u, float v, ulong s) { float x0 = floor(u); float y0 = floor(v); float fx = u - x0; float fy = v - y0; float sx = fx * fx * (three - two * fx); float sy = fy * fy * (three - two * fy); long ix = (long)x0; long iy = (long)y0; float a = lattice(ix, iy, s); float b = lattice(ix + oneU, iy, s); float c = lattice(ix, iy + oneU, s); float d = lattice(ix + oneU, iy + oneU, s); return a + (b - a) * sx + (c - a) * sy + (a - b - c + d) * sx * sy; }
float fbm(float u, float v, ulong s) { float sum = zero; float amp = 0.5f; float norm = zero; for (uint octave = zeroU; octave < fiveU; octave++) { sum += amp * noise(u, v, s + (ulong)octave * 7919u); norm += amp; u *= 2.03f; v *= 2.03f; amp *= 0.5f; } return sum / norm; }
float curled(float u, float v, float seconds, ulong s) { float qx = fbm(u + 0.09f * seconds, v, s); float qy = fbm(u + 5.2f - 0.07f * seconds, v + 1.3f, s + oneU); return fbm(u + 1.9f * qx, v + 0.8f * qy, s + twoU); }
kernel void max_moon(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant Params &p [[buffer(2)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
    threadgroup float a[block];
    uint i = gid * block + tid;
    float v = zero;
    if (i < p.count) {
        uint row = i / p.width;
        uint top = p.height - oneU - row;
        if (top < p.moonRows) {
            float3 c = readRGB(in, i);
            v = c.x + c.y + c.z;
        }
    }
    a[tid] = v;
    threadgroup_barrier(mem_flags::mem_threadgroup);
    for (uint span = block / twoU; span > zeroU; span >>= oneU) {
        if (tid < span) { a[tid] = max(a[tid], a[tid + span]); }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (tid == zeroU) { out[gid] = a[zeroU]; }
}
kernel void max_mask(device const float *in [[buffer(0)]], device float *out [[buffer(1)]], constant Params &p [[buffer(2)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
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
kernel void threshold_moon(device const float *in [[buffer(0)]], device const float *limit [[buffer(1)]], device float *red [[buffer(2)]], device float *green [[buffer(3)]], device float *blue [[buffer(4)]], device float *count [[buffer(5)]], device float *column [[buffer(6)]], constant Params &p [[buffer(7)]], uint tid [[thread_position_in_threadgroup]], uint gid [[threadgroup_position_in_grid]]) {
    threadgroup float r[block];
    threadgroup float g[block];
    threadgroup float b[block];
    threadgroup float c[block];
    threadgroup float x[block];
    uint i = gid * block + tid;
    float vr = zero;
    float vg = zero;
    float vb = zero;
    float vc = zero;
    float vx = zero;
    if (i < p.count) {
        uint row = i / p.width;
        uint top = p.height - oneU - row;
        if (top < p.moonRows) {
            float3 p3 = readRGB(in, i);
            float sum = p3.x + p3.y + p3.z;
            if (sum >= limit[0] * halfValue) {
                vr = p3.x;
                vg = p3.y;
                vb = p3.z;
                vc = one;
                vx = float(i % p.width);
            }
        }
    }
    r[tid] = vr;
    g[tid] = vg;
    b[tid] = vb;
    c[tid] = vc;
    x[tid] = vx;
    threadgroup_barrier(mem_flags::mem_threadgroup);
    for (uint span = block / twoU; span > zeroU; span >>= oneU) {
        if (tid < span) {
            r[tid] += r[tid + span];
            g[tid] += g[tid + span];
            b[tid] += b[tid + span];
            c[tid] += c[tid + span];
            x[tid] += x[tid + span];
        }
        threadgroup_barrier(mem_flags::mem_threadgroup);
    }
    if (tid == zeroU) {
        red[gid] = r[zeroU];
        green[gid] = g[zeroU];
        blue[gid] = b[zeroU];
        count[gid] = c[zeroU];
        column[gid] = x[zeroU];
    }
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
kernel void blend(device const float *raw [[buffer(0)]], device const float *moonRed [[buffer(1)]], device const float *moonGreen [[buffer(2)]], device const float *moonBlue [[buffer(3)]], device const float *moonCount [[buffer(4)]], device const float *moonColumn [[buffer(5)]], device const float *waterMask [[buffer(6)]], device const float *waterMax [[buffer(7)]], device const float *treeMask [[buffer(8)]], device const float *treeMax [[buffer(9)]], device float *out [[buffer(10)]], constant Params &p [[buffer(11)]], uint2 at [[thread_position_in_grid]]) {
    if (at.x >= p.width || at.y >= p.height) { return; }
    uint i = (p.height - oneU - at.y) * p.width + at.x;
    float row = float(at.y);
    float below = row - p.shoreRow;
    float3 base = readRGB(raw, i);
    if (below < -p.rise) {
        writeRGB(out, i, base);
        return;
    }
    float lake = p.lake;
    float near = max(zero, below / lake);
    float up = smooth(-p.rise, zero, below);
    float far = 0.35f + 0.65f * exp(-near * 1.6f);
    float column = moonCount[0] > zero ? moonColumn[0] / moonCount[0] : zero;
    float off = (float(at.x) - column) / (0.28f * float(p.width));
    float forward = 0.25f + 0.75f * exp(-off * off);
    float k = float(p.width) / 1280.0f;
    float u = float(at.x) / ((260.0f + 500.0f * near) * k) + 0.05f * p.seconds;
    float v = row / ((30.0f + 120.0f * near) * k);
    float veil = 0.55f + 0.45f * fbm(u, v, seed(p) + 11u);
    float wu = float(at.x) / ((90.0f + 300.0f * near) * k) + 0.12f * p.seconds;
    float wv = row / ((14.0f + 60.0f * near) * k) - 0.04f * p.seconds;
    float wisps = smooth(0.42f, 0.72f, curled(wu, wv, p.seconds, seed(p)));
    float d = up * far * forward * (0.45f * veil + 0.75f * wisps);
    if ((p.flags & 2u) != zeroU) {
        float3 t = readRGB(treeMask, i);
        if (t.x + t.y + t.z > 0.3f * treeMax[0]) {
            writeRGB(out, i, base);
            return;
        }
    }
    if ((p.flags & 1u) != zeroU && below > zero) {
        float3 w = readRGB(waterMask, i);
        float on = (w.x + w.y + w.z) / (3.0f * waterMax[0]);
        float ease = smooth(zero, 30.0f, below);
        d *= (one - ease) + ease * min(one, 1.25f * on);
    }
    d = thickened(d, p.layers);
    float moon = (moonRed[0] + moonGreen[0] + moonBlue[0]) / (three * moonCount[0]);
    float3 mist = float3(p.strength * d * p.colour0 * moon, p.strength * d * p.colour1 * moon, p.strength * d * p.colour2 * moon);
    float mix = one - 0.6f * min(one, d);
    writeRGB(out, i, base * mix + mist);
}
`)
	return b.String()
}
