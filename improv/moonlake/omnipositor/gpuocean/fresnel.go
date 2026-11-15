package gpuocean

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/metal"
)

const MaxSamples = 1 << 20

type FresnelSample struct {
	Cosine float32
	Index  float32
}

// Fresnel returns the share of light reflected, for each sample, as the mean
// of the two polarisations. A cosine is clamped to 0 through 1, and total
// reflection gives 1. It refuses more than MaxSamples samples, a number that
// is not finite, and an Index outside 1e-10 through 1e10.
func Fresnel(samples []FresnelSample) ([]float32, error) {
	if len(samples) > MaxSamples {
		return nil, errors.New("gpuocean: Fresnel sample limit exceeded")
	}
	if len(samples) == 0 {
		return []float32{}, nil
	}
	for _, sample := range samples {
		if math.IsNaN(float64(sample.Cosine)) || math.IsInf(float64(sample.Cosine), 0) || math.IsNaN(float64(sample.Index)) || math.IsInf(float64(sample.Index), 0) || sample.Index < 1e-10 || sample.Index > 1e10 {
			return nil, errors.New("gpuocean: Fresnel requires finite cosine and index in [1e-10,1e10]")
		}
	}
	session, err := metal.OpenSession(fresnelSource(), "fresnel")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open Metal: %w", err)
	}
	defer session.Close()
	input, err := session.Buffer(len(samples) * 2)
	if err != nil {
		return nil, fmt.Errorf("gpuocean: allocate Fresnel input: %w", err)
	}
	output, err := session.Buffer(len(samples))
	if err != nil {
		return nil, fmt.Errorf("gpuocean: allocate Fresnel output: %w", err)
	}
	for index, sample := range samples {
		input.Numbers()[2*index] = sample.Cosine
		input.Numbers()[2*index+1] = sample.Index
	}
	if err := session.Dispatch("fresnel", len(samples), 1, nil, input, output); err != nil {
		return nil, fmt.Errorf("gpuocean: dispatch Fresnel: %w", err)
	}
	if err := session.Commit(); err != nil {
		return nil, fmt.Errorf("gpuocean: commit Fresnel: %w", err)
	}
	result := append([]float32(nil), output.Numbers()...)
	for _, value := range result {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, errors.New("gpuocean: nonfinite Fresnel result")
		}
	}
	return result, nil
}

// fresnelSource is the Metal source for the fresnel kernel. It gives one for
// total reflection and otherwise the mean of the two polarisations.
func fresnelSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, item := range []struct {
		name  string
		value float64
	}{{"zero", 0}, {"one", 1}, {"oneHalf", 0.5}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", item.name, item.value)
	}
	source.WriteString(`kernel void fresnel(device const float2 *input [[buffer(0)]], device float *output [[buffer(1)]], uint id [[thread_position_in_grid]]) {
    float cosine = clamp(input[id].x, zero, one);
    float index = input[id].y;
    float sine2 = (one - cosine * cosine) / (index * index);
    if (sine2 >= one) { output[id] = one; return; }
    float inside = sqrt(one - sine2);
    float across = (cosine - index * inside) / (cosine + index * inside);
    float along = (index * cosine - inside) / (index * cosine + inside);
    output[id] = (across * across + along * along) * oneHalf;
}`)
	return source.String()
}
