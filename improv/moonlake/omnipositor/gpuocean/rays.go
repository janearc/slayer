package gpuocean

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/oceantex"
)

const MaxRayPixels = 1 << 24

type RayCamera struct {
	Width, Height     int
	Focal, GaugeWidth float64
	Eye               oceantex.Subspace
}

// Rays returns one ray for each pixel of the camera, all starting at the
// eye's origin, in bottom-up row order. It looks along the negative of the
// orientation's Z axis, with the picture GaugeWidth wide at distance Focal.
// It treats a zero orientation as Identity, and refuses a bad size, a Focal
// or GaugeWidth not above zero, and axes not unit length or not at right
// angles.
func Rays(camera RayCamera) ([]oceantex.Ray, error) {
	if camera.Width <= 0 || camera.Height <= 0 || camera.Width > MaxRayPixels/camera.Height {
		return nil, errors.New("gpuocean: invalid ray dimensions")
	}
	if !rayFinite(camera.Focal) || !rayFinite(camera.GaugeWidth) || camera.Focal <= 0 || camera.GaugeWidth <= 0 {
		return nil, errors.New("gpuocean: invalid ray lens")
	}
	orientation := camera.Eye.Orientation
	if orientation == (oceantex.Orientation{}) {
		orientation = oceantex.Identity
	}
	basis := []oceantex.Vec{orientation.X, orientation.Y, orientation.Z, camera.Eye.Origin}
	for _, vector := range basis {
		for _, value := range []float64{vector.X, vector.Y, vector.Z} {
			if !rayFinite(value) || math.Abs(value) > math.MaxFloat32 {
				return nil, errors.New("gpuocean: invalid ray camera basis")
			}
		}
	}
	for _, axis := range basis[:3] {
		length := axis.X*axis.X + axis.Y*axis.Y + axis.Z*axis.Z
		if math.Abs(length-1) > 1e-5 {
			return nil, errors.New("gpuocean: camera basis must be orthonormal")
		}
	}
	for first := 0; first < 3; first++ {
		for second := first + 1; second < 3; second++ {
			left, right := basis[first], basis[second]
			if math.Abs(left.X*right.X+left.Y*right.Y+left.Z*right.Z) > 1e-5 {
				return nil, errors.New("gpuocean: camera basis must be orthonormal")
			}
		}
	}
	reach := camera.GaugeWidth / (2 * camera.Focal)
	if !rayFinite(reach) || reach <= 0 || float32(reach) == 0 || math.IsInf(float64(float32(reach)), 0) {
		return nil, errors.New("gpuocean: ray lens outside float32 range")
	}
	params := make([]byte, 64)
	binary.LittleEndian.PutUint32(params, uint32(camera.Width))
	binary.LittleEndian.PutUint32(params[4:], uint32(camera.Height))
	binary.LittleEndian.PutUint32(params[8:], math.Float32bits(float32(reach)))
	for axis, vector := range basis {
		for component, value := range []float64{vector.X, vector.Y, vector.Z} {
			binary.LittleEndian.PutUint32(params[16+axis*12+component*4:], math.Float32bits(float32(value)))
		}
	}
	session, err := metal.OpenSession(raySource(), "camera_rays")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open rays: %w", err)
	}
	defer session.Close()
	buffer, err := session.Buffer(camera.Width * camera.Height * 3)
	if err != nil {
		return nil, fmt.Errorf("gpuocean: allocate rays: %w", err)
	}
	if err := session.Dispatch("camera_rays", camera.Width, camera.Height, params, buffer); err != nil {
		return nil, fmt.Errorf("gpuocean: dispatch rays: %w", err)
	}
	if err := session.Commit(); err != nil {
		return nil, fmt.Errorf("gpuocean: commit rays: %w", err)
	}
	result := make([]oceantex.Ray, camera.Width*camera.Height)
	for index := range result {
		values := buffer.Numbers()[index*3 : index*3+3]
		if !rayFinite(float64(values[0])) || !rayFinite(float64(values[1])) || !rayFinite(float64(values[2])) {
			return nil, errors.New("gpuocean: nonfinite ray")
		}
		x, y, z := float64(values[0]), float64(values[1]), float64(values[2])
		length := math.Sqrt(x*x + y*y + z*z)
		if !rayFinite(length) || length == 0 {
			return nil, errors.New("gpuocean: invalid ray direction")
		}
		result[index] = oceantex.Ray{At: camera.Eye.Origin, Dir: oceantex.Vec{X: x / length, Y: y / length, Z: z / length}}
	}
	return result, nil
}

// rayFinite reports whether value is neither NaN nor infinite.
func rayFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// raySource is the Metal source for the camera_rays kernel, which writes a
// direction for each pixel into a bottom-up row order.
func raySource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"oneHalf", 0.5}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	source.WriteString(`struct Params { uint width; uint height; float reach; float padding; packed_float3 x; packed_float3 y; packed_float3 z; packed_float3 origin; };
kernel void camera_rays(device float *output [[buffer(0)]], constant Params &p [[buffer(1)]], uint2 point [[thread_position_in_grid]]) {
    if (point.x >= p.width || point.y >= p.height) return;
    float right = (two * (float(point.x) + oneHalf) / float(p.width) - one) * p.reach;
    float up = (one - two * (float(point.y) + oneHalf) / float(p.height)) * p.reach * float(p.height) / float(p.width);
    float3 direction = normalize(float3(p.x) * right + float3(p.y) * up - float3(p.z));
    uint index = ((p.height - 1 - point.y) * p.width + point.x) * 3;
    output[index] = direction.x; output[index + 1] = direction.y; output[index + 2] = direction.z;
}`)
	return source.String()
}
