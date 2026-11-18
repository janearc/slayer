//go:build darwin

package flow

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/janearc/slayer/metal"
	"github.com/janearc/slayer/simulation/internal/metalfft"
)

//go:embed flow.metal
var flowMetalSource string

// spectralPair keeps the two components in independent complex buffers.
type spectralPair struct{ u, v metal.Buffer }

// metalEvolution owns every buffer needed for four spectral RK stages.
type metalEvolution struct {
	config                                         Config
	session                                        *metal.Session
	encoder                                        *metalfft.Encoder
	accepted, stage, candidate, sum, rhs, physical spectralPair
	omega, scratch, coefficients                   metal.Buffer
}

// newMetalStepper checks float32 range and uploads projected initial spectra.
func newMetalStepper(config Config, initial Velocity) (metalStepper, error) {
	shape := metalfft.Shape{Width: config.Shape.Width, Height: config.Shape.Height, MaxCells: config.Shape.MaxCells}
	cells, err := shape.Cells()
	if err != nil {
		return nil, err
	}
	coefficients, err := metalFlowCoefficients(config)
	if err != nil {
		return nil, err
	}
	u, v, err := flowComplexVelocity(config, initial)
	if err != nil {
		return nil, err
	}
	for _, values := range [][]complex128{u, v} {
		if err := metalFlowRange(values); err != nil {
			return nil, err
		}
	}
	names := append(metalfft.Kernels(), "flow_copy", "flow_curl", "flow_rotational", "flow_project_rhs", "flow_rk")
	session, err := metal.OpenSession(metalfft.Source()+flowMetalSource, names...)
	if err != nil {
		return nil, fmt.Errorf("flow: open Metal: %w", err)
	}
	engine := &metalEvolution{config: config, session: session}
	if err := engine.prepare(shape, cells); err != nil {
		engine.close()
		return nil, err
	}
	copy(engine.coefficients.Numbers(), coefficients)
	metalFlowPack(engine.accepted.u, u)
	metalFlowPack(engine.accepted.v, v)
	return engine, nil
}

// metalFlowCoefficients checks float32 arithmetic before opening a device.
func metalFlowCoefficients(config Config) ([]float32, error) {
	values := make([]float32, 4*config.Shape.Width*config.Shape.Height)
	for y := range config.Shape.Height {
		for x := range config.Shape.Width {
			i := y*config.Shape.Width + x
			if !flowRetained(x, config.Shape.Width) || !flowRetained(y, config.Shape.Height) {
				continue
			}
			kx, ky := flowWavenumber(config, x, y)
			square := kx*kx + ky*ky
			for _, value := range []float64{kx, ky, square, config.Viscosity * square} {
				rounded := float32(value)
				if !flowFinite(value) || !flowFinite(float64(rounded)) ||
					(value != 0 && math.Abs(float64(rounded)) < 0x1p-126) {
					return nil, fmt.Errorf("flow: mode %d outside float32 coefficient range", i)
				}
			}
			// Leave room for adding squared components during the GPU projection.
			if square > math.MaxFloat32/16 {
				return nil, fmt.Errorf("flow: wavenumber square outside float32 range")
			}
			values[4*i], values[4*i+1] = float32(kx), float32(ky)
			values[4*i+2], values[4*i+3] = float32(config.Viscosity*square), 1
		}
	}
	return values, nil
}

// metalFlowRange leaves headroom below unnormalized float32 sum overflow.
func metalFlowRange(values []complex128) error {
	var sum float64
	for _, value := range values {
		sum += math.Abs(real(value)) + math.Abs(imag(value))
		if !flowFinite(sum) || sum > math.MaxFloat32/16 {
			return fmt.Errorf("flow: spectrum outside float32 range")
		}
	}
	return nil
}

// metalFlowPack interleaves validated complex values in shared device memory.
func metalFlowPack(buffer metal.Buffer, values []complex128) {
	numbers := buffer.Numbers()
	for i, value := range values {
		numbers[2*i], numbers[2*i+1] = float32(real(value)), float32(imag(value))
	}
}

// prepare retains all stage buffers and mode coefficients on one session.
func (engine *metalEvolution) prepare(shape metalfft.Shape, cells int) error {
	var err error
	engine.encoder, err = metalfft.New(engine.session, shape)
	if err != nil {
		return err
	}
	for _, pair := range []*spectralPair{&engine.accepted, &engine.stage, &engine.candidate, &engine.sum, &engine.rhs, &engine.physical} {
		pair.u, err = engine.session.Buffer(2 * cells)
		if err != nil {
			return err
		}
		pair.v, err = engine.session.Buffer(2 * cells)
		if err != nil {
			return err
		}
	}
	for _, buffer := range []*metal.Buffer{&engine.omega, &engine.scratch} {
		*buffer, err = engine.session.Buffer(2 * cells)
		if err != nil {
			return err
		}
	}
	engine.coefficients, err = engine.session.Buffer(4 * cells)
	return err
}

// step does not replace accepted buffers until candidate readback succeeds.
func (engine *metalEvolution) step(seconds, nextTime float64) (Velocity, Diagnostics, error) {
	current := engine.accepted
	for stage := range 4 {
		params := engine.params(seconds, stage)
		if err := engine.appendRHS(current, params); err != nil {
			return Velocity{}, Diagnostics{}, err
		}
		next := engine.stage
		if stage == 3 {
			next = engine.candidate
		}
		err := engine.dispatch("flow_rk", params, engine.accepted.u, engine.accepted.v, engine.rhs.u, engine.rhs.v, engine.sum.u, engine.sum.v, next.u, next.v)
		if err != nil {
			return Velocity{}, Diagnostics{}, err
		}
		current = next
	}
	params := engine.params(seconds, 0)
	for _, pair := range [][2]metal.Buffer{{engine.candidate.u, engine.physical.u}, {engine.candidate.v, engine.physical.v}} {
		if err := engine.transform(pair[0], pair[1], true, params); err != nil {
			return Velocity{}, Diagnostics{}, err
		}
	}
	if err := engine.session.Commit(); err != nil {
		return Velocity{}, Diagnostics{}, fmt.Errorf("flow: Metal step: %w", err)
	}
	velocity, diagnostics, err := engine.readCandidate(nextTime)
	if err != nil {
		return Velocity{}, Diagnostics{}, err
	}
	engine.accepted, engine.candidate = engine.candidate, engine.accepted
	return velocity, diagnostics, nil
}

// appendRHS keeps all five transforms and the nonlinear product on-device.
func (engine *metalEvolution) appendRHS(current spectralPair, params []byte) error {
	if err := engine.dispatch("flow_curl", params, current.u, current.v, engine.coefficients, engine.omega); err != nil {
		return err
	}
	for _, pair := range [][2]metal.Buffer{{engine.omega, engine.omega}, {current.u, engine.physical.u}, {current.v, engine.physical.v}} {
		if err := engine.transform(pair[0], pair[1], true, params); err != nil {
			return err
		}
	}
	if err := engine.dispatch("flow_rotational", params, engine.physical.u, engine.physical.v, engine.omega, engine.rhs.u, engine.rhs.v); err != nil {
		return err
	}
	for _, buffer := range []metal.Buffer{engine.rhs.u, engine.rhs.v} {
		if err := engine.transform(buffer, buffer, false, params); err != nil {
			return err
		}
	}
	return engine.dispatch("flow_project_rhs", params, current.u, current.v, engine.coefficients, engine.rhs.u, engine.rhs.v)
}

// transform ensures the final result owns its requested buffer before the
// next transform borrows shared scratch. This handles either stage parity.
func (engine *metalEvolution) transform(input, output metal.Buffer, inverse bool, params []byte) error {
	result, err := engine.encoder.Append(input, output, engine.scratch, inverse)
	if err != nil {
		return err
	}
	if &result.Numbers()[0] != &output.Numbers()[0] {
		return engine.dispatch("flow_copy", params, result, output)
	}
	return nil
}

// params packs dimensions, the rounded interval and RK stage without padding.
func (engine *metalEvolution) params(seconds float64, stage int) []byte {
	var data [16]byte
	for i, value := range [4]uint32{uint32(engine.config.Shape.Width), uint32(engine.config.Shape.Height), math.Float32bits(float32(seconds)), uint32(stage)} {
		binary.LittleEndian.PutUint32(data[4*i:], value)
	}
	return data[:]
}

// dispatch appends one flow kernel over the fixed grid without committing.
func (engine *metalEvolution) dispatch(name string, params []byte, buffers ...metal.Buffer) error {
	return engine.session.Dispatch(name, engine.config.Shape.Width, engine.config.Shape.Height, params, buffers...)
}

// readCandidate computes diagnostics in float64 after the single commit.
// Parseval converts the spectral divergence norm to grid RMS directly.
func (engine *metalEvolution) readCandidate(nextTime float64) (Velocity, Diagnostics, error) {
	cells := engine.config.Shape.Width * engine.config.Shape.Height
	velocity := Velocity{X: make([]float64, cells), Y: make([]float64, cells)}
	physicalU, physicalV := engine.physical.u.Numbers(), engine.physical.v.Numbers()
	u, v := engine.candidate.u.Numbers(), engine.candidate.v.Numbers()
	var energy, divergence float64
	for y := range engine.config.Shape.Height {
		for x := range engine.config.Shape.Width {
			i := y*engine.config.Shape.Width + x
			for _, value := range []float32{physicalU[2*i], physicalU[2*i+1], physicalV[2*i], physicalV[2*i+1], u[2*i], u[2*i+1], v[2*i], v[2*i+1]} {
				if !flowFinite(float64(value)) {
					return Velocity{}, Diagnostics{}, fmt.Errorf("flow: non-finite Metal candidate at %d", i)
				}
			}
			vx, vy := float64(physicalU[2*i]), float64(physicalV[2*i])
			velocity.X[i], velocity.Y[i] = vx, vy
			energy += 0.5 * (vx*vx + vy*vy)
			kx, ky := flowWavenumber(engine.config, x, y)
			realPart := kx*float64(u[2*i]) + ky*float64(v[2*i])
			imaginary := kx*float64(u[2*i+1]) + ky*float64(v[2*i+1])
			divergence = math.Hypot(divergence, math.Hypot(realPart, imaginary))
		}
	}
	energy /= float64(cells)
	divergence /= float64(cells)
	if !flowFinite(energy) || !flowFinite(divergence) {
		return Velocity{}, Diagnostics{}, fmt.Errorf("flow: Metal diagnostics overflow")
	}
	return velocity, Diagnostics{TimeSeconds: nextTime, KineticEnergy: energy, DivergenceL2: divergence}, nil
}

// close releases the owned native session and prevents further dispatches.
func (engine *metalEvolution) close() {
	if engine != nil && engine.session != nil {
		engine.session.Close()
		engine.session = nil
		engine.encoder = nil
	}
}
