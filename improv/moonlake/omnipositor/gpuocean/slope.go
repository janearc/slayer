package gpuocean

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/janearc/slayer/metal"
)

type SlopeSample struct{ Along, Across, VarianceAlong, VarianceAcross float32 }

// SlopeChance returns the density of a surface slope for each sample, as a
// normal curve across and along with the given variances, centred on zero. It
// refuses more than MaxSamples samples, a number that is not finite, and a
// variance outside 1e-10 through 1e10.
func SlopeChance(samples []SlopeSample) ([]float32, error) {
	if len(samples) > MaxSamples {
		return nil, errors.New("gpuocean: slope sample limit exceeded")
	}
	for _, sample := range samples {
		for _, value := range []float32{sample.Along, sample.Across, sample.VarianceAlong, sample.VarianceAcross} {
			if !rayFinite(float64(value)) {
				return nil, errors.New("gpuocean: nonfinite slope sample")
			}
		}
		if sample.VarianceAlong < 1e-10 || sample.VarianceAcross < 1e-10 || sample.VarianceAlong > 1e10 || sample.VarianceAcross > 1e10 || math.Abs(float64(sample.Along)) > 1e10 || math.Abs(float64(sample.Across)) > 1e10 {
			return nil, errors.New("gpuocean: slope sample outside supported float32 range")
		}
	}
	if len(samples) == 0 {
		return []float32{}, nil
	}
	session, err := metal.OpenSession(slopeSource(), "slope_chance")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open slope: %w", err)
	}
	defer session.Close()
	input, err := session.Buffer(len(samples) * 4)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(samples))
	if err != nil {
		return nil, err
	}
	for index, sample := range samples {
		values := input.Numbers()[index*4 : index*4+4]
		values[0], values[1], values[2], values[3] = sample.Along, sample.Across, sample.VarianceAlong, sample.VarianceAcross
	}
	if err := session.Dispatch("slope_chance", len(samples), 1, nil, input, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := append([]float32(nil), output.Numbers()...)
	for _, value := range result {
		if !rayFinite(float64(value)) {
			return nil, errors.New("gpuocean: nonfinite slope chance")
		}
	}
	return result, nil
}

// slopeSource is the Metal source for the slope_chance kernel, which is the
// two dimensional normal density worked out for each sample.
func slopeSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	for _, number := range []struct {
		name  string
		value float64
	}{{"two", 2}, {"pi", math.Pi}} {
		fmt.Fprintf(&source, "constant float %s = %.17ef;\n", number.name, number.value)
	}
	source.WriteString(`kernel void slope_chance(device const float4 *input [[buffer(0)]], device float *output [[buffer(1)]], uint id [[thread_position_in_grid]]) {
    float4 v = input[id];
    output[id] = exp(-(v.x*v.x/v.z+v.y*v.y/v.w)/two)/(two*pi*sqrt(v.z*v.w));
}`)
	return source.String()
}
