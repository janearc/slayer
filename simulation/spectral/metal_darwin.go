//go:build darwin

package spectral

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/simulation/internal/metalfft"
)

// MetalPlan retains one grid's Fourier kernels, buffers and twiddles.
// One caller uses it at a time; Close ends it and releases its buffers.
type MetalPlan struct {
	shape   Shape
	cells   int
	session *metal.Session
	encoder *metalfft.Encoder
	input   metal.Buffer
	scratch metal.Buffer
	device  string
	closed  bool
	failed  error
}

// OpenMetal prepares a fixed power-of-two grid within its explicit cell
// budget. It refuses invalid shapes and an unavailable Metal device.
func OpenMetal(shape Shape) (*MetalPlan, error) {
	cells, err := checkedShape(shape)
	if err != nil {
		return nil, err
	}
	grid := metalfft.Shape{Width: shape.Width, Height: shape.Height,
		MaxCells: shape.MaxCells}
	if _, err := grid.Cells(); err != nil {
		return nil, err
	}
	device, err := metalDevice()
	if err != nil {
		return nil, err
	}
	session, err := metal.OpenSession(metalfft.Source(),
		metalfft.Kernels()...)
	if err != nil {
		return nil, fmt.Errorf("spectral: open Metal: %w", err)
	}
	plan := newMetalPlan(shape, cells, session, device)
	if err := plan.prepare(grid); err != nil {
		plan.Close()
		return nil, err
	}
	return plan, nil
}

// newMetalPlan assembles a validated plan around its owned session.
func newMetalPlan(shape Shape, cells int, session *metal.Session,
	device string) *MetalPlan {
	return &MetalPlan{shape: shape, cells: cells, session: session,
		device: device}
}

// metalDevice obtains the default device's own name without bridge changes.
// The probe and session compile through the same shared library cache.
func metalDevice() (string, error) {
	probe, err := metal.Open(metalfft.Source(), metalfft.Kernels()[0])
	if err != nil {
		return "", fmt.Errorf("spectral: open Metal device: %w", err)
	}
	defer probe.Close()
	name := probe.Device()
	if name == "" {
		return "", fmt.Errorf("spectral: Metal device has no name")
	}
	return name, nil
}

// prepare creates the encoder's twiddles and two retained complex buffers.
func (plan *MetalPlan) prepare(shape metalfft.Shape) error {
	var err error
	plan.encoder, err = metalfft.New(plan.session, shape)
	if err != nil {
		return err
	}
	plan.input, err = plan.session.Buffer(2 * plan.cells)
	if err != nil {
		return fmt.Errorf("spectral: input buffer: %w", err)
	}
	plan.scratch, err = plan.session.Buffer(2 * plan.cells)
	if err != nil {
		return fmt.Errorf("spectral: scratch buffer: %w", err)
	}
	return nil
}

// Forward returns the unscaled negative-exponent transform in float32
// arithmetic, converted to complex128, without modifying values.
func (plan *MetalPlan) Forward(values []complex128) ([]complex128, error) {
	return plan.transform(values, false)
}

// Inverse returns the positive-exponent transform divided by the cell
// count, in float32 arithmetic, without modifying values.
func (plan *MetalPlan) Inverse(values []complex128) ([]complex128, error) {
	return plan.transform(values, true)
}

// transform packs one input, queues the whole transform, commits once,
// and reads one finished result. Validation refusals leave the plan usable.
func (plan *MetalPlan) transform(values []complex128,
	inverse bool) ([]complex128, error) {
	if err := plan.usable(); err != nil {
		return nil, err
	}
	if err := checkInput(values, plan.cells); err != nil {
		return nil, err
	}
	if err := checkMetalRange(values); err != nil {
		return nil, err
	}
	packed := plan.input.Numbers()
	for index, value := range values {
		packed[2*index], packed[2*index+1] = float32(real(value)),
			float32(imag(value))
	}
	buffer, err := plan.encoder.Append(plan.input, plan.input,
		plan.scratch, inverse)
	if err != nil {
		return plan.failedWith(err)
	}
	if err := plan.session.Commit(); err != nil {
		return plan.failedWith(fmt.Errorf("spectral: Metal batch: %w",
			err))
	}
	result, err := unpackMetal(buffer, plan.cells)
	if err != nil {
		return plan.failedWith(err)
	}
	return result, nil
}

// metalSumLimit leaves a factor of four below float32 overflow for the
// sum of the absolute input components and subsequent butterfly rounding.
const metalSumLimit = math.MaxFloat32 / 4

// checkMetalRange rejects inputs whose unscaled partial sums could exceed
// the conservative float32 range, before a shared buffer is touched.
func checkMetalRange(values []complex128) error {
	var sum float64
	for index, value := range values {
		sum += math.Abs(real(value)) + math.Abs(imag(value))
		if sum > metalSumLimit {
			return fmt.Errorf("spectral: Metal float32 input "+
				"range "+
				"exceeded at value %d", index)
		}
	}
	return nil
}

// unpackMetal copies the final shared buffer and refuses non-finite output.
func unpackMetal(buffer metal.Buffer, cells int) ([]complex128, error) {
	values := buffer.Numbers()
	if len(values) != 2*cells {
		return nil, fmt.Errorf("spectral: unexpected Metal result size")
	}
	result := make([]complex128, cells)
	for index := range result {
		result[index] = complex(float64(values[2*index]),
			float64(values[2*index+1]))
	}
	if err := checkFinite(result); err != nil {
		return nil, fmt.Errorf("spectral: Metal result: %w", err)
	}
	return result, nil
}

// usable rejects nil, zero, closed and failed plans before native memory
// is accessed. A failed plan retains its original cause after closing.
func (plan *MetalPlan) usable() error {
	if plan == nil {
		return fmt.Errorf("spectral: no live Metal plan")
	}
	if plan.failed != nil {
		return fmt.Errorf("spectral: Metal plan failed earlier: %w",
			plan.failed)
	}
	if plan.closed || plan.session == nil || plan.encoder == nil {
		return fmt.Errorf("spectral: Metal plan is closed")
	}
	return nil
}

// failedWith ends a failed batch's plan and returns no partial result.
func (plan *MetalPlan) failedWith(err error) ([]complex128, error) {
	plan.failed = err
	plan.Close()
	return nil, err
}

// Close releases the plan and is harmless for a nil or closed plan.
func (plan *MetalPlan) Close() {
	if plan == nil || plan.closed {
		return
	}
	plan.closed = true
	if plan.session != nil {
		plan.session.Close()
	}
	plan.session, plan.encoder = nil, nil
	plan.input, plan.scratch = metal.Buffer{}, metal.Buffer{}
}

// Device is the device name obtained when the plan was opened.
func (plan *MetalPlan) Device() string {
	if plan == nil {
		return ""
	}
	return plan.device
}
