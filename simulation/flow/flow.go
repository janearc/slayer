// Package flow defines a CPU reference for periodic two-dimensional,
// constant-density incompressible Navier–Stokes flow.
package flow

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/janearc/slayer/simulation/spectral"
)

// Config defines a periodic rectangular domain and its integration limit.
type Config struct {
	Shape          spectral.Shape
	WidthMetres    float64
	HeightMetres   float64
	Viscosity      float64
	MaxStepSeconds float64
}

// Velocity is a row-major horizontal velocity field in metres per second.
type Velocity struct {
	X []float64
	Y []float64
}

// Diagnostics describes the current periodic flow state.
type Diagnostics struct {
	TimeSeconds   float64
	DivergenceL2  float64
	KineticEnergy float64
}

// Flow holds a validated configuration, velocity field and diagnostics.
type Flow struct {
	config      Config
	velocity    Velocity
	diagnostics Diagnostics
	initialized bool
}

// New validates, copies and projects the initial velocity field.
func New(config Config, initial Velocity) (*Flow, error) {
	if err := flowConfigError(config); err != nil {
		return nil, err
	}
	cells := config.Shape.Width * config.Shape.Height
	if len(initial.X) != cells || len(initial.Y) != cells {
		return nil, fmt.Errorf("flow: velocity components have lengths %d and %d, want %d",
			len(initial.X), len(initial.Y), cells)
	}
	if err := flowRealValuesError("initial x velocity", initial.X); err != nil {
		return nil, err
	}
	if err := flowRealValuesError("initial y velocity", initial.Y); err != nil {
		return nil, err
	}

	velocity := flowVelocityCopy(initial)
	velocity, err := flowProject(config, velocity)
	if err != nil {
		return nil, fmt.Errorf("flow: project initial velocity: %w", err)
	}
	diagnostics, err := flowDiagnostics(config, velocity, 0)
	if err != nil {
		return nil, fmt.Errorf("flow: initial diagnostics: %w", err)
	}
	return &Flow{
		config:      config,
		velocity:    velocity,
		diagnostics: diagnostics,
		initialized: true,
	}, nil
}

// Step advances the flow by seconds, refusing invalid or oversized steps.
func (flow *Flow) Step(seconds float64) (Diagnostics, error) {
	if flow == nil || !flow.initialized {
		return Diagnostics{}, fmt.Errorf("flow: cannot step an uninitialized flow")
	}
	if !flowFinite(seconds) || seconds <= 0 {
		return flow.diagnostics, fmt.Errorf("flow: step must be finite and positive: %g", seconds)
	}
	if seconds > flow.config.MaxStepSeconds {
		return flow.diagnostics, fmt.Errorf("flow: step %g exceeds configured maximum %g seconds",
			seconds, flow.config.MaxStepSeconds)
	}
	if err := flowGuard(flow.config, flow.velocity, seconds); err != nil {
		return flow.diagnostics, err
	}
	nextTime := flow.diagnostics.TimeSeconds + seconds
	if !flowFinite(nextTime) {
		return flow.diagnostics, fmt.Errorf("flow: step time overflows")
	}

	// Classical RK4 evaluates the projected, dealiased rotational form.
	k1, err := flowRHS(flow.config, flow.velocity)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: first RK stage: %w", err)
	}
	stage2, err := flowStage(flow.config, flow.velocity, k1, seconds/2)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: second RK state: %w", err)
	}
	k2, err := flowRHS(flow.config, stage2)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: second RK stage: %w", err)
	}
	stage3, err := flowStage(flow.config, flow.velocity, k2, seconds/2)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: third RK state: %w", err)
	}
	k3, err := flowRHS(flow.config, stage3)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: third RK stage: %w", err)
	}
	stage4, err := flowStage(flow.config, flow.velocity, k3, seconds)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: fourth RK state: %w", err)
	}
	k4, err := flowRHS(flow.config, stage4)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: fourth RK stage: %w", err)
	}

	cells := len(flow.velocity.X)
	candidate := Velocity{X: make([]float64, cells), Y: make([]float64, cells)}
	for index := range candidate.X {
		candidate.X[index] = flow.velocity.X[index] + seconds/6*(k1.X[index]+2*k2.X[index]+2*k3.X[index]+k4.X[index])
		candidate.Y[index] = flow.velocity.Y[index] + seconds/6*(k1.Y[index]+2*k2.Y[index]+2*k3.Y[index]+k4.Y[index])
	}
	candidate, err = flowProject(flow.config, candidate)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: project candidate state: %w", err)
	}
	diagnostics, err := flowDiagnostics(flow.config, candidate, nextTime)
	if err != nil {
		return flow.diagnostics, fmt.Errorf("flow: candidate diagnostics: %w", err)
	}

	// Commit only after every stage and diagnostic has succeeded.
	flow.velocity = candidate
	flow.diagnostics = diagnostics
	return diagnostics, nil
}

// Velocity returns independent copies of the current velocity components.
func (flow *Flow) Velocity() Velocity {
	if flow == nil || !flow.initialized {
		return Velocity{}
	}
	return flowVelocityCopy(flow.velocity)
}

// Diagnostics returns a copy of the current scalar diagnostics.
func (flow *Flow) Diagnostics() Diagnostics {
	if flow == nil || !flow.initialized {
		return Diagnostics{}
	}
	return flow.diagnostics
}

// flowConfigError validates the grid, physical units and derived retained modes.
func flowConfigError(config Config) error {
	shape := config.Shape
	if shape.Width < 2 || !flowPowerOfTwo(shape.Width) {
		return fmt.Errorf("flow: width %d must be a power of two of at least 2", shape.Width)
	}
	if shape.Height < 2 || !flowPowerOfTwo(shape.Height) {
		return fmt.Errorf("flow: height %d must be a power of two of at least 2", shape.Height)
	}
	if shape.MaxCells <= 0 {
		return fmt.Errorf("flow: maximum cell budget must be positive: %d", shape.MaxCells)
	}
	if shape.Width > int(^uint(0)>>1)/shape.Height {
		return fmt.Errorf("flow: grid cell count overflows int")
	}
	if cells := shape.Width * shape.Height; cells > shape.MaxCells {
		return fmt.Errorf("flow: grid has %d cells, over budget %d",
			cells, shape.MaxCells)
	}
	if !flowFinite(config.WidthMetres) || config.WidthMetres <= 0 {
		return fmt.Errorf("flow: width must be finite and positive: %g", config.WidthMetres)
	}
	if !flowFinite(config.HeightMetres) || config.HeightMetres <= 0 {
		return fmt.Errorf("flow: height must be finite and positive: %g", config.HeightMetres)
	}
	if !flowFinite(config.Viscosity) || config.Viscosity < 0 {
		return fmt.Errorf("flow: viscosity must be finite and nonnegative: %g", config.Viscosity)
	}
	if !flowFinite(config.MaxStepSeconds) || config.MaxStepSeconds <= 0 {
		return fmt.Errorf("flow: maximum step must be finite and positive: %g",
			config.MaxStepSeconds)
	}
	kx, ky := flowMaxWavenumbers(config)
	if !flowFinite(kx) || !flowFinite(ky) ||
		!flowFinite(kx*kx) || !flowFinite(ky*ky) ||
		!flowFinite(config.Viscosity*(kx*kx+ky*ky)) {
		return fmt.Errorf("flow: domain produces non-finite retained wavenumbers")
	}
	return nil
}

// flowPowerOfTwo accepts radix-two dimensions of at least two.
func flowPowerOfTwo(value int) bool {
	return value >= 2 && value&(value-1) == 0
}

// flowFinite excludes both infinities and NaN.
func flowFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// flowRealValuesError identifies the first non-finite component.
func flowRealValuesError(label string, values []float64) error {
	for index, value := range values {
		if !flowFinite(value) {
			return fmt.Errorf("flow: %s value %d is not finite: %g", label, index, value)
		}
	}
	return nil
}

// flowVelocityCopy makes both components independent of the source.
func flowVelocityCopy(velocity Velocity) Velocity {
	return Velocity{
		X: append([]float64(nil), velocity.X...),
		Y: append([]float64(nil), velocity.Y...),
	}
}

// flowMode maps a Fourier bin to its signed periodic integer mode.
func flowMode(index, size int) int {
	if index > size/2 {
		return index - size
	}
	return index
}

// flowRetained applies the rectangular two-thirds cutoff on one axis.
func flowRetained(index, size int) bool {
	mode := flowMode(index, size)
	if mode < 0 {
		mode = -mode
	}
	return mode <= size/3
}

// flowMaxWavenumbers gives the largest retained physical wavenumbers.
func flowMaxWavenumbers(config Config) (float64, float64) {
	maxX := 2 * math.Pi * float64(config.Shape.Width/3) / config.WidthMetres
	maxY := 2 * math.Pi * float64(config.Shape.Height/3) / config.HeightMetres
	return maxX, maxY
}

// flowWavenumber assigns physical derivatives with zero Nyquist derivative.
func flowWavenumber(config Config, x, y int) (float64, float64) {
	modeX, modeY := flowMode(x, config.Shape.Width), flowMode(y, config.Shape.Height)
	// A real-valued Nyquist component has no independent conjugate partner.
	if x == config.Shape.Width/2 {
		modeX = 0
	}
	if y == config.Shape.Height/2 {
		modeY = 0
	}
	return 2 * math.Pi * float64(modeX) / config.WidthMetres,
		2 * math.Pi * float64(modeY) / config.HeightMetres
}

// flowComplexVelocity independently transforms both real velocity components.
func flowComplexVelocity(config Config, velocity Velocity) ([]complex128, []complex128, error) {
	u := make([]complex128, len(velocity.X))
	v := make([]complex128, len(velocity.Y))
	for index := range u {
		u[index] = complex(velocity.X[index], 0)
		v[index] = complex(velocity.Y[index], 0)
	}
	uHat, err := spectral.Forward(config.Shape, u)
	if err != nil {
		return nil, nil, fmt.Errorf("transform x velocity: %w", err)
	}
	vHat, err := spectral.Forward(config.Shape, v)
	if err != nil {
		return nil, nil, fmt.Errorf("transform y velocity: %w", err)
	}
	return uHat, vHat, nil
}

// flowProject removes longitudinal and unresolved velocity modes.
func flowProject(config Config, velocity Velocity) (Velocity, error) {
	uHat, vHat, err := flowComplexVelocity(config, velocity)
	if err != nil {
		return Velocity{}, err
	}
	flowProjectSpectra(config, uHat, vHat)
	u, err := flowInverseReal(config.Shape, uHat)
	if err != nil {
		return Velocity{}, fmt.Errorf("inverse x velocity: %w", err)
	}
	v, err := flowInverseReal(config.Shape, vHat)
	if err != nil {
		return Velocity{}, fmt.Errorf("inverse y velocity: %w", err)
	}
	return Velocity{X: u, Y: v}, nil
}

// flowProjectSpectra applies the Leray projector and cutoff in place.
func flowProjectSpectra(config Config, u, v []complex128) {
	width, height := config.Shape.Width, config.Shape.Height
	for y := range height {
		for x := range width {
			index := y*width + x
			if !flowRetained(x, width) || !flowRetained(y, height) {
				u[index], v[index] = 0, 0
				continue
			}
			kx, ky := flowWavenumber(config, x, y)
			k2 := kx*kx + ky*ky
			if k2 == 0 {
				continue
			}
			longitudinal := (complex(kx, 0)*u[index] + complex(ky, 0)*v[index]) / complex(k2, 0)
			u[index] -= complex(kx, 0) * longitudinal
			v[index] -= complex(ky, 0) * longitudinal
		}
	}
}

// flowInverseReal checks the inverse result before taking its real component.
func flowInverseReal(shape spectral.Shape, values []complex128) ([]float64, error) {
	transformed, err := spectral.Inverse(shape, values)
	if err != nil {
		return nil, err
	}
	realValues := make([]float64, len(transformed))
	for index, value := range transformed {
		if !flowFinite(real(value)) || !flowFinite(imag(value)) {
			return nil, fmt.Errorf("inverse value %d is non-finite: %v", index, value)
		}
		realValues[index] = real(value)
	}
	return realValues, nil
}

// flowRHS forms projected rotational acceleration and viscous diffusion.
func flowRHS(config Config, velocity Velocity) (Velocity, error) {
	uHat, vHat, err := flowComplexVelocity(config, velocity)
	if err != nil {
		return Velocity{}, err
	}
	cells := config.Shape.Width * config.Shape.Height
	omegaHat := make([]complex128, cells)
	for y := range config.Shape.Height {
		for x := range config.Shape.Width {
			index := y*config.Shape.Width + x
			if !flowRetained(x, config.Shape.Width) ||
				!flowRetained(y, config.Shape.Height) {
				uHat[index], vHat[index] = 0, 0
				continue
			}
			kx, ky := flowWavenumber(config, x, y)
			omegaHat[index] = complex(0, kx)*vHat[index] -
				complex(0, ky)*uHat[index]
		}
	}
	omega, err := flowInverseReal(config.Shape, omegaHat)
	if err != nil {
		return Velocity{}, fmt.Errorf("inverse vorticity: %w", err)
	}
	nonlinearU := make([]complex128, cells)
	nonlinearV := make([]complex128, cells)
	for index := range omega {
		nonlinearU[index] = complex(velocity.Y[index]*omega[index], 0)
		nonlinearV[index] = complex(-velocity.X[index]*omega[index], 0)
	}
	nonlinearUHat, err := spectral.Forward(config.Shape, nonlinearU)
	if err != nil {
		return Velocity{}, fmt.Errorf("transform rotational x term: %w", err)
	}
	nonlinearVHat, err := spectral.Forward(config.Shape, nonlinearV)
	if err != nil {
		return Velocity{}, fmt.Errorf("transform rotational y term: %w", err)
	}
	for y := range config.Shape.Height {
		for x := range config.Shape.Width {
			index := y*config.Shape.Width + x
			if !flowRetained(x, config.Shape.Width) ||
				!flowRetained(y, config.Shape.Height) {
				nonlinearUHat[index], nonlinearVHat[index] = 0, 0
				continue
			}
			kx, ky := flowWavenumber(config, x, y)
			k2 := kx*kx + ky*ky
			nonlinearUHat[index] -= complex(config.Viscosity*k2, 0) * uHat[index]
			nonlinearVHat[index] -= complex(config.Viscosity*k2, 0) * vHat[index]
		}
	}
	// Periodic momentum is conserved, so roundoff in the product's mean
	// must not turn into a change of the spatially uniform velocity.
	nonlinearUHat[0], nonlinearVHat[0] = 0, 0
	flowProjectSpectra(config, nonlinearUHat, nonlinearVHat)
	u, err := flowInverseReal(config.Shape, nonlinearUHat)
	if err != nil {
		return Velocity{}, fmt.Errorf("inverse x acceleration: %w", err)
	}
	v, err := flowInverseReal(config.Shape, nonlinearVHat)
	if err != nil {
		return Velocity{}, fmt.Errorf("inverse y acceleration: %w", err)
	}
	return Velocity{X: u, Y: v}, nil
}

// flowStage builds and projects one finite intermediate RK state.
func flowStage(config Config, base, derivative Velocity, scale float64) (Velocity, error) {
	stage := Velocity{X: make([]float64, len(base.X)), Y: make([]float64, len(base.Y))}
	for index := range stage.X {
		stage.X[index] = base.X[index] + scale*derivative.X[index]
		stage.Y[index] = base.Y[index] + scale*derivative.Y[index]
		if !flowFinite(stage.X[index]) || !flowFinite(stage.Y[index]) {
			return Velocity{}, fmt.Errorf("RK stage is non-finite at cell %d", index)
		}
	}
	return flowProject(config, stage)
}

// flowGuard checks the picked advective and viscous step bound.
func flowGuard(config Config, velocity Velocity, seconds float64) error {
	maxU, maxV := 0.0, 0.0
	for index := range velocity.X {
		maxU = math.Max(maxU, math.Abs(velocity.X[index]))
		maxV = math.Max(maxV, math.Abs(velocity.Y[index]))
	}
	kx, ky := flowMaxWavenumbers(config)
	measure := seconds * (maxU*kx + maxV*ky +
		config.Viscosity*(kx*kx+ky*ky))
	if !flowFinite(measure) || measure > 0.5 {
		return fmt.Errorf("flow: step guard %g exceeds picked limit 0.5", measure)
	}
	return nil
}

// flowDiagnostics measures energy and spectral divergence of accepted velocity.
func flowDiagnostics(config Config, velocity Velocity, time float64) (Diagnostics, error) {
	var energy float64
	for index := range velocity.X {
		u, v := velocity.X[index], velocity.Y[index]
		if !flowFinite(u) || !flowFinite(v) {
			return Diagnostics{}, fmt.Errorf("velocity is non-finite at cell %d", index)
		}
		energy += 0.5 * (u*u + v*v)
		if !flowFinite(energy) {
			return Diagnostics{}, fmt.Errorf("kinetic energy overflows")
		}
	}
	energy /= float64(len(velocity.X))
	if !flowFinite(time) || !flowFinite(energy) {
		return Diagnostics{}, fmt.Errorf("time or kinetic energy is non-finite")
	}
	uHat, vHat, err := flowComplexVelocity(config, velocity)
	if err != nil {
		return Diagnostics{}, err
	}
	divergenceHat := make([]complex128, len(uHat))
	for y := range config.Shape.Height {
		for x := range config.Shape.Width {
			index := y*config.Shape.Width + x
			kx, ky := flowWavenumber(config, x, y)
			divergenceHat[index] = complex(0, kx)*uHat[index] +
				complex(0, ky)*vHat[index]
		}
	}
	divergence, err := spectral.Inverse(config.Shape, divergenceHat)
	if err != nil {
		return Diagnostics{}, fmt.Errorf("inverse divergence: %w", err)
	}
	var square float64
	for index, value := range divergence {
		magnitude := cmplx.Abs(value)
		square += magnitude * magnitude
		if !flowFinite(square) {
			return Diagnostics{}, fmt.Errorf("divergence norm overflows at cell %d", index)
		}
	}
	divergenceL2 := math.Sqrt(square / float64(len(divergence)))
	if !flowFinite(divergenceL2) {
		return Diagnostics{}, fmt.Errorf("divergence norm is non-finite")
	}
	return Diagnostics{
		TimeSeconds:   time,
		DivergenceL2:  divergenceL2,
		KineticEnergy: energy,
	}, nil
}
