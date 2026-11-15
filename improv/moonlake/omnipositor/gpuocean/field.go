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

type FieldPoint struct{ X, Y float64 }

// SampleField reads the periodic grid values, field.N by field.N, bilinearly
// at each point, wrapping at the edges, and returns one float32 for each. The
// grid repeats every field.Extent in x and y. It refuses a field of N under 2
// or over 4096, a wrong count of values, and numbers beyond float32 range.
func SampleField(field *oceantex.Field, values []float64, points []FieldPoint) ([]float32, error) {
	if field == nil || field.N < 2 || field.N > 4096 || field.N > int(^uint(0)>>1)/field.N || len(values) != field.N*field.N || !rayFinite(field.Extent) || field.Extent <= 0 || float32(field.Extent) == 0 || math.IsInf(float64(float32(field.Extent)), 0) {
		return nil, errors.New("gpuocean: invalid periodic field")
	}
	if len(points) > MaxSamples {
		return nil, errors.New("gpuocean: field sample limit exceeded")
	}
	for _, value := range values {
		if !rayFinite(value) || math.Abs(value) > math.MaxFloat32 {
			return nil, errors.New("gpuocean: field value outside float32 range")
		}
	}
	for _, point := range points {
		if !rayFinite(point.X) || !rayFinite(point.Y) || math.Abs(point.X) > math.MaxFloat32 || math.Abs(point.Y) > math.MaxFloat32 || math.Abs(point.X) > math.MaxFloat32*field.Extent || math.Abs(point.Y) > math.MaxFloat32*field.Extent || math.Abs(point.X/field.Extent*float64(field.N)) > 1<<22 || math.Abs(point.Y/field.Extent*float64(field.N)) > 1<<22 {
			return nil, errors.New("gpuocean: field coordinate outside float32 precision range")
		}
	}
	if len(points) == 0 {
		return []float32{}, nil
	}
	session, err := metal.OpenSession(fieldSource(), "sample_field")
	if err != nil {
		return nil, fmt.Errorf("gpuocean: open field: %w", err)
	}
	defer session.Close()
	grid, err := session.Buffer(len(values))
	if err != nil {
		return nil, err
	}
	positions, err := session.Buffer(len(points) * 2)
	if err != nil {
		return nil, err
	}
	output, err := session.Buffer(len(points))
	if err != nil {
		return nil, err
	}
	for index, value := range values {
		grid.Numbers()[index] = float32(value)
	}
	for index, point := range points {
		positions.Numbers()[index*2] = float32(point.X)
		positions.Numbers()[index*2+1] = float32(point.Y)
	}
	params := make([]byte, 8)
	binary.LittleEndian.PutUint32(params, uint32(field.N))
	binary.LittleEndian.PutUint32(params[4:], math.Float32bits(float32(field.Extent)))
	if err := session.Dispatch("sample_field", len(points), 1, params, grid, positions, output); err != nil {
		return nil, err
	}
	if err := session.Commit(); err != nil {
		return nil, err
	}
	result := append([]float32(nil), output.Numbers()...)
	for _, value := range result {
		if !rayFinite(float64(value)) {
			return nil, errors.New("gpuocean: nonfinite field sample")
		}
	}
	return result, nil
}

// fieldSource is the Metal source for the sample_field kernel, which reads a
// grid bilinearly at a position, wrapping round at the edges.
func fieldSource() string {
	var source strings.Builder
	source.WriteString("#include <metal_stdlib>\nusing namespace metal;\n")
	fmt.Fprintf(&source, "constant float one = %.17ef;\n", 1.0)
	source.WriteString(`struct Params { uint n; float extent; };
kernel void sample_field(device const float *grid [[buffer(0)]], device const float2 *positions [[buffer(1)]], device float *output [[buffer(2)]], constant Params &p [[buffer(3)]], uint id [[thread_position_in_grid]]) {
    float2 at = positions[id] / p.extent * float(p.n);
    float2 base = floor(at);
    float2 fraction = at - base;
    int n = int(p.n);
    int x = ((int(base.x) % n) + n) % n;
    int y = ((int(base.y) % n) + n) % n;
    int nextX = (x + 1) % n; int nextY = (y + 1) % n;
    float top = grid[y*n+x]*(one-fraction.x) + grid[y*n+nextX]*fraction.x;
    float bottom = grid[nextY*n+x]*(one-fraction.x) + grid[nextY*n+nextX]*fraction.x;
    output[id] = top*(one-fraction.y) + bottom*fraction.y;
}`)
	return source.String()
}
