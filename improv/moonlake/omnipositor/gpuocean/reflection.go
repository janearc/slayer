package gpuocean

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/oceantex"
)

type ReflectionSample struct {
	Eye            oceantex.Vec
	SlopeX, SlopeY float32
	Index          float32
}

type ReflectionResult struct {
	Direction oceantex.Vec
	Weight    float32
}

// Reflections mirrors each eye direction about the surface normal made from
// its slopes, and returns the direction, with z not below zero, and the
// unpolarised Fresnel weight for its Index. The eye points from the surface
// toward the viewer. It refuses more than MaxSamples samples, an eye that is
// not unit length, a slope that is not finite, and an Index outside 1e-10
// through 1e10.
func Reflections(samples []ReflectionSample) ([]ReflectionResult, error) {
	if len(samples) > MaxSamples {
		return nil, errors.New("gpuocean: reflection sample limit exceeded")
	}
	for _, sample := range samples {
		length := sample.Eye.X*sample.Eye.X + sample.Eye.Y*sample.Eye.Y + sample.Eye.Z*sample.Eye.Z
		if !rayFinite(length) || math.Abs(length-1) > 1e-5 || !rayFinite(float64(sample.SlopeX)) || !rayFinite(float64(sample.SlopeY)) || math.Abs(float64(sample.SlopeX)) > 1e10 || math.Abs(float64(sample.SlopeY)) > 1e10 || !rayFinite(float64(sample.Index)) || sample.Index < 1e-10 || sample.Index > 1e10 {
			return nil, errors.New("gpuocean: invalid reflection sample")
		}
	}
	if len(samples) == 0 {
		return []ReflectionResult{}, nil
	}
	session, err := metal.OpenSession(reflectionSource(), "reflection")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open reflection: %w", err)
	}
	defer session.Close()
	input, err := session.Buffer(len(samples) * 6)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(samples) * 4)
	if err != nil {
		return nil, err
	}
	for index, sample := range samples {
		values := input.Numbers()[index*6 : index*6+6]
		values[0], values[1], values[2] = float32(sample.Eye.X), float32(sample.Eye.Y), float32(sample.Eye.Z)
		values[3], values[4], values[5] = sample.SlopeX, sample.SlopeY, sample.Index
	}
	if err := session.Dispatch("reflection", len(samples), 1, nil, input, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := make([]ReflectionResult, len(samples))
	for index := range result {
		values := output.Numbers()[index*4 : index*4+4]
		for _, value := range values {
			if !rayFinite(float64(value)) {
				return nil, errors.New("gpuocean: nonfinite reflection")
			}
		}
		result[index] = ReflectionResult{Direction: oceantex.Vec{X: float64(values[0]), Y: float64(values[1]), Z: float64(values[2])}, Weight: values[3]}
	}
	return result, nil
}

// reflectionSource is the Metal source for the reflection kernel. It builds a
// normal from the slopes and mirrors the eye about it, with z not below zero.
func reflectionSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"two", 2}, {"oneHalf", 0.5}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	source.WriteString(`kernel void reflection(device const float *input [[buffer(0)]], device float *output [[buffer(1)]], uint id [[thread_position_in_grid]]) {
    uint base=id*6; float3 eye=float3(input[base],input[base+1],input[base+2]); float3 normal=normalize(float3(-input[base+3],-input[base+4],one)); float index=input[base+5];
    float cosine=max(dot(normal,eye),zero); float bounded=clamp(cosine,zero,one); float sine2=(one-bounded*bounded)/(index*index); float weight=one;
    if (sine2<one) { float inside=sqrt(one-sine2); float across=(bounded-index*inside)/(bounded+index*inside); float along=(index*bounded-inside)/(index*bounded+inside); weight=(across*across+along*along)*oneHalf; }
    float3 direction=two*cosine*normal-eye; direction.z=max(direction.z,zero); if (length(direction)>zero) direction=normalize(direction);
    uint out=id*4; output[out]=direction.x; output[out+1]=direction.y; output[out+2]=direction.z; output[out+3]=weight;
}`)
	return source.String()
}
