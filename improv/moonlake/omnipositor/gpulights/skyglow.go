package gpulights

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/metal"
)

// SkyGlow adds a blue-white glow to sky and returns it as a new image. The
// glow is a band that is brightest at each column's Rim row and falls off above
// it, plus a halo round the moon when the moon is in front of the camera. Only
// pixels where Open is set change. It returns a copy if Gain is not above
// zero, and refuses bad sizes, non-finite values and an over-budget frame.
func SkyGlow(sky lights.Image, input lights.SkyGlowInput) (lights.Image, error) {
	width, height := sky.Width, sky.Height
	if !dimensions(width, height) || len(input.Open) != width*height || len(input.Rim) != width {
		return lights.Image{}, errors.New("gpulights: invalid sky glow dimensions")
	}
	if err := validateImage(sky, width, height); err != nil {
		return lights.Image{}, err
	}
	if int64(width)*int64(height)*16+int64(width)*4 > DefaultMaxBytes {
		return lights.Image{}, errors.New("gpulights: sky glow allocation exceeds budget")
	}
	values := []float64{input.Gain, input.Azimuth, input.MoonAltitude, input.MoonScale, input.MoonLevel, input.Master, input.Camera.Focal, input.Camera.Width, input.Camera.Height, input.Moon[0], input.Moon[1], input.Moon[2], input.Camera.Forward.X, input.Camera.Forward.Y, input.Camera.Forward.Z, input.Camera.Right.X, input.Camera.Right.Y, input.Camera.Right.Z, input.Camera.Up.X, input.Camera.Up.Y, input.Camera.Up.Z}
	for _, value := range values {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return lights.Image{}, errors.New("gpulights: invalid sky glow parameter")
		}
	}
	if input.Gain <= 0 {
		return lights.Image{Width: width, Height: height, Light: append([]float64(nil), sky.Light...)}, nil
	}
	moonX, moonY, ahead := input.Camera.Project(lights.Toward(input.Azimuth, input.MoonAltitude).Scale(1e6))
	radius := math.Tan(input.MoonScale*0.259*math.Pi/180) * input.Camera.Focal
	luminance := (input.Moon[0] + input.Moon[1] + input.Moon[2]) / 3
	for _, value := range []float64{moonX, moonY, radius, luminance} {
		if !finite(value) || math.Abs(value) > math.MaxFloat32 {
			return lights.Image{}, errors.New("gpulights: sky glow geometry outside float32 range")
		}
	}
	if ahead && (radius <= 0 || float32(radius) == 0) {
		return lights.Image{}, errors.New("gpulights: invalid sky glow radius")
	}
	params := make([]byte, 40)
	binary.LittleEndian.PutUint32(params, uint32(width))
	binary.LittleEndian.PutUint32(params[4:], uint32(height))
	if ahead {
		binary.LittleEndian.PutUint32(params[8:], 1)
	}
	for index, value := range []float64{moonX, moonY, radius, input.Gain, luminance, input.MoonLevel, input.Master} {
		binary.LittleEndian.PutUint32(params[12+index*4:], math.Float32bits(float32(value)))
	}
	session, err := metal.OpenSession(skyGlowSource(), "sky_glow")
	if err != nil {
		return lights.Image{}, fmt.Errorf("gpulights: open sky glow: %w", err)
	}
	defer session.Close()
	image, err := session.Buffer(width * height * 3)
	if err != nil {
		return lights.Image{}, err
	}
	mask, err := session.Buffer(width * height)
	if err != nil {
		return lights.Image{}, err
	}
	rim, err := session.Buffer(width)
	if err != nil {
		return lights.Image{}, err
	}
	if err := uploadImage(image.Numbers(), sky); err != nil {
		return lights.Image{}, err
	}
	uploadMask(mask.Numbers(), input.Open)
	for index, value := range input.Rim {
		rim.Numbers()[index] = float32(value)
	}
	if err := session.Dispatch("sky_glow", width, height, params, image, mask, rim); err != nil {
		return lights.Image{}, err
	}
	if err := session.Commit(); err != nil {
		return lights.Image{}, err
	}
	if err := validateFiniteBuffer("sky glow", image.Numbers()); err != nil {
		return lights.Image{}, err
	}
	return decodeImage(image.Numbers(), width, height), nil
}

// skyGlowSource is the Metal source for the sky_glow kernel, which adds a band
// of blue-white light along the rim and a halo round the moon.
func skyGlowSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"bandBase", 0.4}, {"bandGain", 0.6}, {"bandWidth", 0.18}, {"haloGain", 3}, {"red", 0.55}, {"green", 0.66}, {"blue", 0.95}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	source.WriteString(`struct Params { uint width; uint height; uint ahead; float moonX; float moonY; float radius; float gain; float luminance; float moonLevel; float master; };
kernel void sky_glow(device float *sky [[buffer(0)]], device const float *open [[buffer(1)]], device const float *rim [[buffer(2)]], constant Params &p [[buffer(3)]], uint2 point [[thread_position_in_grid]]) {
    if (point.x >= p.width || point.y >= p.height) return;
    uint pixel = point.y * p.width + point.x;
    if (open[pixel] <= zero) return;
    float above = rim[point.x] - float(point.y);
    float band = bandBase + bandGain * exp(-above / (bandWidth * float(p.height)));
    float halo = zero;
    if (p.ahead != 0) { float distance = length(float2(point) - float2(p.moonX, p.moonY)) / p.radius; halo = haloGain * exp(-max(zero, distance-one)/haloGain); }
    float glow = p.gain * p.luminance * (band + halo * p.moonLevel) * p.master;
    uint index = pixel * 3; sky[index] += glow * red; sky[index+1] += glow * green; sky[index+2] += glow * blue;
}`)
	return source.String()
}
