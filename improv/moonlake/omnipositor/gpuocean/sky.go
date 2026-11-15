package gpuocean

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/improv/moonlake/ocean"
	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/oceantex"
)

type SkyProjection = ocean.Projection

// SkySamples returns the sky colour for each direction, read from the nearest
// pixel of image through projection and divided by its Normalization. A
// direction outside the picture gets the mean of the image's top rows if it
// points up, and black if not. It refuses a bad image, projection axes that
// are not unit and at right angles, and more than MaxSamples directions.
func SkySamples(image ocean.RGB, projection SkyProjection, directions []oceantex.Vec) ([][3]float32, error) {
	if image.Width <= 0 || image.Height <= 0 || image.Width > MaxRayPixels/image.Height || len(image.Light) != image.Width*image.Height*3 || len(directions) > MaxSamples {
		return nil, errors.New("gpuocean: invalid sky image or direction count")
	}
	for _, value := range image.Light {
		if !rayFinite(float64(value)) {
			return nil, errors.New("gpuocean: nonfinite sky image")
		}
	}
	values := []float64{projection.Focal, projection.Normalization, projection.Forward.X, projection.Forward.Y, projection.Forward.Z, projection.Right.X, projection.Right.Y, projection.Right.Z, projection.Up.X, projection.Up.Y, projection.Up.Z}
	for _, value := range values {
		if !rayFinite(value) || math.Abs(value) > 1e12 {
			return nil, errors.New("gpuocean: invalid sky projection")
		}
	}
	if projection.Focal <= 0 || projection.Normalization <= 0 || float32(projection.Focal) == 0 || float32(projection.Normalization) == 0 {
		return nil, errors.New("gpuocean: invalid sky normalization")
	}
	axes := []oceantex.Vec{projection.Forward, projection.Right, projection.Up}
	for _, axis := range axes {
		length := axis.X*axis.X + axis.Y*axis.Y + axis.Z*axis.Z
		if math.Abs(length-1) > 1e-5 {
			return nil, errors.New("gpuocean: sky basis must be orthonormal")
		}
	}
	for first := 0; first < len(axes); first++ {
		for second := first + 1; second < len(axes); second++ {
			left, right := axes[first], axes[second]
			if math.Abs(left.X*right.X+left.Y*right.Y+left.Z*right.Z) > 1e-5 {
				return nil, errors.New("gpuocean: sky basis must be orthonormal")
			}
		}
	}
	for _, direction := range directions {
		length := direction.X*direction.X + direction.Y*direction.Y + direction.Z*direction.Z
		if !rayFinite(length) || length < 1e-20 || length > 1e20 {
			return nil, errors.New("gpuocean: invalid sky direction")
		}
	}
	if len(directions) == 0 {
		return [][3]float32{}, nil
	}
	rows := max(1, image.Height/20)
	var above [3]float64
	for row := 0; row < rows; row++ {
		for column := 0; column < image.Width; column++ {
			index := ((image.Height-1-row)*image.Width + column) * 3
			for channel := 0; channel < 3; channel++ {
				above[channel] += float64(image.Light[index+channel])
			}
		}
	}
	for channel := range above {
		above[channel] /= float64(rows * image.Width)
		if !rayFinite(above[channel]) || math.Abs(above[channel]) > math.MaxFloat32 {
			return nil, errors.New("gpuocean: sky average outside float32 range")
		}
	}
	params := make([]byte, 72)
	binary.LittleEndian.PutUint32(params, uint32(image.Width))
	binary.LittleEndian.PutUint32(params[4:], uint32(image.Height))
	for index, value := range []float64{projection.Focal, projection.Normalization, projection.Forward.X, projection.Forward.Y, projection.Forward.Z, projection.Right.X, projection.Right.Y, projection.Right.Z, projection.Up.X, projection.Up.Y, projection.Up.Z, above[0], above[1], above[2]} {
		binary.LittleEndian.PutUint32(params[8+index*4:], math.Float32bits(float32(value)))
	}
	session, err := metal.OpenSession(skySource(), "sky_samples")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open sky: %w", err)
	}
	defer session.Close()
	pixels, err := session.Buffer(len(image.Light))
	if err != nil {
		return nil, err
	}
	copy(pixels.Numbers(), image.Light)
	input, err := session.Buffer(len(directions) * 3)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(directions) * 3)
	if err != nil {
		return nil, err
	}
	for index, direction := range directions {
		input.Numbers()[index*3], input.Numbers()[index*3+1], input.Numbers()[index*3+2] = float32(direction.X), float32(direction.Y), float32(direction.Z)
	}
	if err := session.Dispatch("sky_samples", len(directions), 1, params, pixels, input, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := make([][3]float32, len(directions))
	for index := range result {
		for channel := 0; channel < 3; channel++ {
			value := output.Numbers()[index*3+channel]
			if !rayFinite(float64(value)) {
				return nil, errors.New("gpuocean: nonfinite sky sample")
			}
			result[index][channel] = value
		}
	}
	return result, nil
}

// skySource is the Metal source for the sky_samples kernel, which finds each
// direction's pixel in the image by the projection.
func skySource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"oneHalf", 0.5}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	source.WriteString(`struct Params { uint width; uint height; float focal; float normalization; packed_float3 forward; packed_float3 right; packed_float3 up; packed_float3 above; };
kernel void sky_samples(device const float *image [[buffer(0)]], device const float *directions [[buffer(1)]], device float *output [[buffer(2)]], constant Params &p [[buffer(3)]], uint id [[thread_position_in_grid]]) {
    uint base=id*3; float3 direction=normalize(float3(directions[base],directions[base+1],directions[base+2]));
    float ahead=dot(direction,float3(p.forward)); float x=-oneHalf, y=-oneHalf;
    if (ahead>zero) { x=float(p.width)*oneHalf+dot(direction,float3(p.right))/ahead*p.focal; y=float(p.height)*oneHalf-dot(direction,float3(p.up))/ahead*p.focal; }
    float3 value=float3(zero);
    if (ahead<=zero || x<zero || y<zero || x>=float(p.width) || y>=float(p.height)) { if (direction.z>zero) value=float3(p.above); }
    else { uint index=((p.height-1-uint(y))*p.width+uint(x))*3; value=float3(image[index],image[index+1],image[index+2]); }
    value/=p.normalization; output[base]=value.x; output[base+1]=value.y; output[base+2]=value.z;
}`)
	return source.String()
}
