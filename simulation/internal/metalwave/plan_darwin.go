//go:build darwin

package metalwave

import (
	_ "embed"
	"encoding/binary"
	"fmt"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/simulation/internal/metalfft"
)

//go:embed wave.metal
var waveSource string

// Plan owns two immutable spectra and a retained paired propagation batch.
// Its buffers and session belong to one caller at a time.
type Plan struct {
	shape                                  metalfft.Shape
	cells                                  int
	bounds                                 amplitudes
	session                                *metal.Session
	encoder                                *metalfft.Encoder
	initialValue, initialRate, coefficient metal.Buffer
	value, rate, scratch                   metal.Buffer
	failed                                 error
}

// Open checks both Cauchy inputs before allocating any native resources.
func Open(shape metalfft.Shape, initial Fields) (*Plan, error) {
	cells, bounds, err := checkedSpectra(shape, initial)
	if err != nil {
		return nil, err
	}
	names := append(metalfft.Kernels(), "wave_combine", "wave_copy")
	session, err := metal.OpenSession(metalfft.Source()+waveSource, names...)
	if err != nil {
		return nil, fmt.Errorf("metalwave: open session: %w", err)
	}
	plan := &Plan{shape: shape, cells: cells, bounds: bounds, session: session}
	if err := plan.prepare(); err != nil {
		plan.Close()
		return nil, err
	}
	packComplex(plan.initialValue, initial.Value)
	packComplex(plan.initialRate, initial.Rate)
	return plan, nil
}

// prepare retains the encoder and every batch buffer for the fixed grid.
func (plan *Plan) prepare() error {
	var err error
	plan.encoder, err = metalfft.New(plan.session, plan.shape)
	if err != nil {
		return err
	}
	for _, buffer := range []*metal.Buffer{&plan.initialValue, &plan.initialRate, &plan.value, &plan.rate, &plan.scratch} {
		*buffer, err = plan.session.Buffer(2 * plan.cells)
		if err != nil {
			return fmt.Errorf("metalwave: complex buffer: %w", err)
		}
	}
	plan.coefficient, err = plan.session.Buffer(4 * plan.cells)
	if err != nil {
		return fmt.Errorf("metalwave: coefficient buffer: %w", err)
	}
	return nil
}

// Evaluate checks and uploads factors, queues the coupled update and both
// inverse transforms, then commits once before reading either final field.
func (plan *Plan) Evaluate(modes []Mode) (Fields, error) {
	if err := plan.Usable(); err != nil {
		return Fields{}, err
	}
	if err := checkedModes(modes, plan.cells, plan.bounds); err != nil {
		return Fields{}, err
	}
	plan.packModes(modes)
	params := plan.parameters()
	err := plan.session.Dispatch("wave_combine", plan.shape.Width, plan.shape.Height, params, plan.initialValue, plan.initialRate, plan.coefficient, plan.value, plan.rate)
	if err != nil {
		return plan.fail(err)
	}
	q, err := plan.encoder.Append(plan.value, plan.value, plan.scratch, true)
	if err != nil {
		return plan.fail(err)
	}
	if &q.Numbers()[0] != &plan.value.Numbers()[0] {
		err = plan.session.Dispatch("wave_copy", plan.shape.Width, plan.shape.Height, params, q, plan.value)
		if err != nil {
			return plan.fail(err)
		}
		q = plan.value
	}
	v, err := plan.encoder.Append(plan.rate, plan.rate, plan.scratch, true)
	if err != nil {
		return plan.fail(err)
	}
	if err := plan.session.Commit(); err != nil {
		return plan.fail(err)
	}
	result, err := plan.read(q, v)
	if err != nil {
		return plan.fail(err)
	}
	return result, nil
}

// packModes stores the three real coefficients and an unused padding word.
func (plan *Plan) packModes(modes []Mode) {
	values := plan.coefficient.Numbers()
	for i, mode := range modes {
		values[4*i], values[4*i+1], values[4*i+2], values[4*i+3] = float32(mode.Cosine), float32(mode.Drift), float32(mode.Acceleration), 0
	}
}

// parameters matches the two-word shader grid without host struct padding.
func (plan *Plan) parameters() []byte {
	var bytes [8]byte
	binary.LittleEndian.PutUint32(bytes[:4], uint32(plan.shape.Width))
	binary.LittleEndian.PutUint32(bytes[4:], uint32(plan.shape.Height))
	return bytes[:]
}

// read copies both completed buffers or returns no partial result.
func (plan *Plan) read(value, rate metal.Buffer) (Fields, error) {
	q, v := value.Numbers(), rate.Numbers()
	if len(q) != 2*plan.cells || len(v) != 2*plan.cells {
		return Fields{}, fmt.Errorf("metalwave: unexpected result buffer length")
	}
	result := Fields{Value: make([]complex128, plan.cells), Rate: make([]complex128, plan.cells)}
	for i := range plan.cells {
		result.Value[i] = complex(float64(q[2*i]), float64(q[2*i+1]))
		result.Rate[i] = complex(float64(v[2*i]), float64(v[2*i+1]))
		for _, number := range [4]float64{real(result.Value[i]), imag(result.Value[i]), real(result.Rate[i]), imag(result.Rate[i])} {
			if !finite(number) {
				return Fields{}, fmt.Errorf("metalwave: non-finite result at cell %d", i)
			}
		}
	}
	return result, nil
}

// packComplex copies validated components into one shared float2 buffer.
func packComplex(buffer metal.Buffer, values []complex128) {
	numbers := buffer.Numbers()
	for i, value := range values {
		numbers[2*i], numbers[2*i+1] = float32(real(value)), float32(imag(value))
	}
}

// Usable checks lifetime and preserves the original cause of a failed batch.
func (plan *Plan) Usable() error {
	if plan == nil {
		return fmt.Errorf("metalwave: nil plan")
	}
	if plan.failed != nil {
		return fmt.Errorf("metalwave: batch failed earlier: %w", plan.failed)
	}
	if plan.session == nil || plan.encoder == nil {
		return fmt.Errorf("metalwave: no live plan")
	}
	return nil
}

// fail closes a failed batch and returns neither Cauchy component.
func (plan *Plan) fail(err error) (Fields, error) {
	plan.failed = err
	plan.Close()
	return Fields{}, err
}

// Close releases owned buffers and is harmless on nil or closed plans.
func (plan *Plan) Close() {
	if plan == nil {
		return
	}
	if plan.session != nil {
		plan.session.Close()
	}
	plan.session, plan.encoder = nil, nil
	plan.initialValue, plan.initialRate, plan.coefficient = metal.Buffer{}, metal.Buffer{}, metal.Buffer{}
	plan.value, plan.rate, plan.scratch = metal.Buffer{}, metal.Buffer{}, metal.Buffer{}
}
