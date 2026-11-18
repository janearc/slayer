//go:build darwin

// Package metalfield composes a retained spectral field, complex transfer,
// and inverse transform into one Metal batch.
package metalfield

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/simulation/internal/metalfft"
)

// Plan owns immutable spectral data and mutable propagation buffers.
// It is used by one caller at a time.
type Plan struct {
	session                       *metal.Session
	encoder                       *metalfft.Encoder
	shape                         metalfft.Shape
	base, transfer, work, scratch metal.Buffer
	sum                           float64
	failed                        error
}

// Open validates every input before allocating device resources.
func Open(shape metalfft.Shape, spectrum []complex128) (*Plan, error) {
	cells, err := shape.Cells()
	if err != nil {
		return nil, err
	}
	if len(spectrum) != cells {
		return nil, fmt.Errorf("metalfield: spectrum length %d, want %d", len(spectrum), cells)
	}
	sum, err := componentSum(spectrum)
	if err != nil {
		return nil, err
	}
	if sum > math.MaxFloat32/8 {
		return nil, fmt.Errorf("metalfield: spectrum exceeds float32 sum range")
	}
	names := append(metalfft.Kernels(), "field_transfer")
	session, err := metal.OpenSession(metalfft.Source()+transferSource, names...)
	if err != nil {
		return nil, fmt.Errorf("metalfield: open session: %w", err)
	}
	plan := &Plan{session: session, shape: shape, sum: sum}
	if err := plan.prepare(cells); err != nil {
		plan.Close()
		return nil, err
	}
	pack(plan.base, spectrum)
	return plan, nil
}

// prepare retains all buffers for the lifetime of the fixed-grid plan.
func (plan *Plan) prepare(cells int) error {
	var err error
	plan.encoder, err = metalfft.New(plan.session, plan.shape)
	if err != nil {
		return err
	}
	for _, buffer := range []*metal.Buffer{&plan.base, &plan.transfer, &plan.work, &plan.scratch} {
		*buffer, err = plan.session.Buffer(2 * cells)
		if err != nil {
			return fmt.Errorf("metalfield: allocate: %w", err)
		}
	}
	return nil
}

// Evaluate applies one bounded complex transfer, commits once, and copies
// the final field. Invalid factors leave the plan usable.
func (plan *Plan) Evaluate(factors []complex128) ([]complex128, error) {
	if err := plan.Usable(); err != nil {
		return nil, err
	}
	cells := plan.shape.Width * plan.shape.Height
	if len(factors) != cells {
		return nil, fmt.Errorf("metalfield: transfer length %d, want %d", len(factors), cells)
	}
	if err := plan.checkFactors(factors); err != nil {
		return nil, err
	}
	pack(plan.transfer, factors)
	var params [8]byte
	binary.LittleEndian.PutUint32(params[:4], uint32(plan.shape.Width))
	binary.LittleEndian.PutUint32(params[4:], uint32(plan.shape.Height))
	err := plan.session.Dispatch("field_transfer", plan.shape.Width, plan.shape.Height, params[:], plan.base, plan.transfer, plan.work)
	if err != nil {
		return plan.fail(err)
	}
	result, err := plan.encoder.Append(plan.work, plan.work, plan.scratch, true)
	if err != nil {
		return plan.fail(err)
	}
	if err = plan.session.Commit(); err != nil {
		return plan.fail(err)
	}
	values := result.Numbers()
	output := make([]complex128, cells)
	for i := range output {
		output[i] = complex(float64(values[2*i]), float64(values[2*i+1]))
	}
	if _, err := componentSum(output); err != nil {
		return plan.fail(err)
	}
	return output, nil
}

// checkFactors bounds products and unnormalized partial inverse sums.
func (plan *Plan) checkFactors(factors []complex128) error {
	for i, value := range factors {
		if !finite(real(value)) || !finite(imag(value)) {
			return fmt.Errorf("metalfield: non-finite transfer at %d", i)
		}
		bound := math.Abs(real(value)) + math.Abs(imag(value))
		if bound > math.MaxFloat32/8 || plan.sum*bound > math.MaxFloat32/8 {
			return fmt.Errorf("metalfield: transfer exceeds float32 range at %d", i)
		}
	}
	return nil
}

// componentSum refuses non-finite values and overflowing sums.
func componentSum(values []complex128) (float64, error) {
	var sum float64
	for i, value := range values {
		if !finite(real(value)) || !finite(imag(value)) {
			return 0, fmt.Errorf("metalfield: non-finite value at %d", i)
		}
		sum += math.Abs(real(value)) + math.Abs(imag(value))
		if !finite(sum) {
			return 0, fmt.Errorf("metalfield: component sum overflows")
		}
	}
	return sum, nil
}

// finite excludes both infinities and NaN.
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// pack copies validated complex values into an interleaved shared buffer.
func pack(buffer metal.Buffer, values []complex128) {
	numbers := buffer.Numbers()
	for i, value := range values {
		numbers[2*i], numbers[2*i+1] = float32(real(value)), float32(imag(value))
	}
}

// Usable checks resource lifetime before any native memory is accessed.
func (plan *Plan) Usable() error {
	if plan == nil {
		return fmt.Errorf("metalfield: nil plan")
	}
	if plan.failed != nil {
		return fmt.Errorf("metalfield: failed earlier: %w", plan.failed)
	}
	if plan.session == nil || plan.encoder == nil {
		return fmt.Errorf("metalfield: closed plan")
	}
	return nil
}

// fail closes a failed batch and retains its cause without a partial result.
func (plan *Plan) fail(err error) ([]complex128, error) {
	plan.failed = err
	plan.Close()
	return nil, err
}

// Close releases every buffer and is harmless on nil or closed plans.
func (plan *Plan) Close() {
	if plan == nil {
		return
	}
	if plan.session != nil {
		plan.session.Close()
	}
	plan.session, plan.encoder = nil, nil
	plan.base, plan.transfer, plan.work, plan.scratch = metal.Buffer{}, metal.Buffer{}, metal.Buffer{}, metal.Buffer{}
}

const transferSource = `
struct FieldGrid { uint width; uint height; };
kernel void field_transfer(
 device const float2 *base [[buffer(0)]],
 device const float2 *factor [[buffer(1)]],
 device float2 *result [[buffer(2)]],
 constant FieldGrid &grid [[buffer(3)]],
 uint2 at [[thread_position_in_grid]]) {
 if (at.x >= grid.width || at.y >= grid.height) { return; }
 uint i=at.y*grid.width+at.x;
 float2 a=base[i], b=factor[i];
 result[i]=float2(a.x*b.x-a.y*b.y, a.x*b.y+a.y*b.x);
}
`
