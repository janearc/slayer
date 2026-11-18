// Package gravitational propagates weak vacuum plane-front strain profiles.
package gravitational

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/simulation/spectral"
	"github.com/janearc/slayer/simulation/waves"
)

// LightSpeed is the defined speed of light in metres per second.
const LightSpeed = 299792458

// Config specifies a periodic profile, its direction and explicit bounds.
type Config struct {
	Samples, MaxSamples int
	LengthMetres        float64
	Direction           [3]float64
	MaxStrain           float64
}

// Profiles contains plus and cross strain and their rates per second.
type Profiles struct {
	Plus, Cross         []float64
	PlusRate, CrossRate []float64
}

// Field owns immutable initial profiles and a transverse polarization frame.
type Field struct {
	config Config
	wave   *waves.Field
	p, q   [3]float64
}

// Tensor is a spatial metric perturbation, without the identity matrix.
type Tensor [3][3]float64

// New validates, copies and prepares two vacuum polarization profiles.
func New(config Config, initial Profiles) (*Field, error) {
	if err := planeInputError(config, initial); err != nil {
		return nil, err
	}
	p, q, err := polarizationFrame(config.Direction)
	if err != nil {
		return nil, err
	}
	frequency, err := vacuumFrequencies(config)
	if err != nil {
		return nil, err
	}
	state := packPolarizations(config.Samples, initial)
	shape := spectral.Shape{Width: config.Samples, Height: 2, MaxCells: 2 * config.Samples}
	wave, err := waves.New(shape, frequency, state)
	if err != nil {
		return nil, fmt.Errorf("gravitational: prepare profiles: %w", err)
	}
	return &Field{config: config, wave: wave, p: p, q: q}, nil
}

// planeInputError checks budgets, physical units and every initial component
// before creating the two-row numerical representation.
func planeInputError(config Config, initial Profiles) error {
	n := config.Samples
	if n < 2 || n&(n-1) != 0 || n > int(^uint(0)>>1)/2 {
		return fmt.Errorf("gravitational: sample count must fit a doubled radix-two grid")
	}
	if config.MaxSamples <= 0 || n > config.MaxSamples {
		return fmt.Errorf("gravitational: sample budget exceeded")
	}
	if !planeFinite(config.LengthMetres) || config.LengthMetres <= 0 {
		return fmt.Errorf("gravitational: length must be finite and positive")
	}
	if !planeFinite(config.MaxStrain) || config.MaxStrain <= 0 || config.MaxStrain > 0.01 {
		return fmt.Errorf("gravitational: strain bound must be positive and at most the picked cap 0.01")
	}
	for _, component := range [][]float64{initial.Plus, initial.Cross, initial.PlusRate, initial.CrossRate} {
		if len(component) != n {
			return fmt.Errorf("gravitational: profile length %d, want %d", len(component), n)
		}
		for _, value := range component {
			if !planeFinite(value) {
				return fmt.Errorf("gravitational: initial profile contains non-finite data")
			}
		}
	}
	for i := range n {
		if math.Hypot(initial.Plus[i], initial.Cross[i]) > config.MaxStrain {
			return fmt.Errorf("gravitational: initial strain exceeds the chosen bound")
		}
	}
	return nil
}

// vacuumFrequencies keeps each distinct c*|k| while sharing frequency
// between opposite signed modes. Both identical rows use that same table;
// the unused transverse row carries no independent physics.
func vacuumFrequencies(config Config) ([]float64, error) {
	step := 2 * math.Pi / config.LengthMetres
	if !planeFinite(step) {
		return nil, fmt.Errorf("gravitational: profile wavenumber overflows")
	}
	values := make([]float64, 2*config.Samples)
	for i := range config.Samples {
		mode := i
		if mode > config.Samples/2 {
			mode -= config.Samples
		}
		omega := LightSpeed * math.Abs(step*float64(mode))
		if !planeFinite(omega) {
			return nil, fmt.Errorf("gravitational: mode frequency overflows")
		}
		values[i], values[config.Samples+i] = omega, omega
	}
	return values, nil
}

// packPolarizations copies the two real Cauchy profiles into one complex
// field, repeated on two rows for the shared Fourier propagator.
func packPolarizations(samples int, initial Profiles) waves.State {
	state := waves.State{Value: make([]complex128, 2*samples), Rate: make([]complex128, 2*samples)}
	for i := range samples {
		value := complex(initial.Plus[i], initial.Cross[i])
		rate := complex(initial.PlusRate[i], initial.CrossRate[i])
		state.Value[i], state.Value[samples+i] = value, value
		state.Rate[i], state.Rate[samples+i] = rate, rate
	}
	return state
}

// At returns profiles at finite elapsed seconds, including negative times.
func (field *Field) At(seconds float64) (Profiles, error) {
	if field == nil || field.wave == nil {
		return Profiles{}, fmt.Errorf("gravitational: no initialized field")
	}
	state, err := field.wave.At(seconds)
	if err != nil {
		return Profiles{}, err
	}
	return field.unpack(state)
}

// unpack validates the candidate regime before returning independent profiles.
func (field *Field) unpack(state waves.State) (Profiles, error) {
	n := field.config.Samples
	if len(state.Value) != 2*n || len(state.Rate) != 2*n {
		return Profiles{}, fmt.Errorf("gravitational: unexpected evolved profile length")
	}
	result := Profiles{Plus: make([]float64, n), Cross: make([]float64, n), PlusRate: make([]float64, n), CrossRate: make([]float64, n)}
	for i := range n {
		plus, cross := real(state.Value[i]), imag(state.Value[i])
		plusRate, crossRate := real(state.Rate[i]), imag(state.Rate[i])
		if !planeFinite(plus) || !planeFinite(cross) || !planeFinite(plusRate) || !planeFinite(crossRate) || math.Hypot(plus, cross) > field.config.MaxStrain {
			return Profiles{}, fmt.Errorf("gravitational: evolved profile exceeds finite weak-field bounds")
		}
		result.Plus[i], result.Cross[i] = plus, cross
		result.PlusRate[i], result.CrossRate[i] = plusRate, crossRate
	}
	return result, nil
}

// Tensor returns the transverse traceless strain in the configured frame.
func (field *Field) Tensor(plus, cross float64) (Tensor, error) {
	if field == nil || field.wave == nil {
		return Tensor{}, fmt.Errorf("gravitational: no initialized field")
	}
	if !planeFinite(plus) || !planeFinite(cross) || math.Hypot(plus, cross) > field.config.MaxStrain {
		return Tensor{}, fmt.Errorf("gravitational: tensor strain exceeds finite weak-field bounds")
	}
	var tensor Tensor
	for i := range 3 {
		for j := range 3 {
			tensor[i][j] = plus*(field.p[i]*field.p[j]-field.q[i]*field.q[j]) + cross*(field.p[i]*field.q[j]+field.q[i]*field.p[j])
		}
	}
	return tensor, nil
}

// polarizationFrame selects a deterministic transverse basis. The least
// aligned axis keeps the projection safely away from zero.
func polarizationFrame(direction [3]float64) ([3]float64, [3]float64, error) {
	n, err := planeUnit(direction)
	if err != nil {
		return [3]float64{}, [3]float64{}, err
	}
	axis := 0
	for i := 1; i < 3; i++ {
		if math.Abs(n[i]) < math.Abs(n[axis]) {
			axis = i
		}
	}
	var p [3]float64
	for i := range 3 {
		p[i] = -n[axis] * n[i]
	}
	p[axis] += 1
	// The least-aligned component has magnitude at most 1/sqrt(3),
	// so this projection has norm at least sqrt(2/3).
	projectedNorm := math.Hypot(math.Hypot(p[0], p[1]), p[2])
	for i := range p {
		p[i] /= projectedNorm
	}
	q := [3]float64{n[1]*p[2] - n[2]*p[1], n[2]*p[0] - n[0]*p[2], n[0]*p[1] - n[1]*p[0]}
	return p, q, nil
}

// planeUnit rescales before forming the norm, avoiding unnecessary overflow
// and underflow for finite nonzero directions.
func planeUnit(vector [3]float64) ([3]float64, error) {
	var scale float64
	for _, value := range vector {
		if !planeFinite(value) {
			return [3]float64{}, fmt.Errorf("gravitational: direction must be finite")
		}
		scale = math.Max(scale, math.Abs(value))
	}
	if scale == 0 {
		return [3]float64{}, fmt.Errorf("gravitational: direction must be nonzero")
	}
	for i := range vector {
		vector[i] /= scale
	}
	norm := math.Hypot(math.Hypot(vector[0], vector[1]), vector[2])
	for i := range vector {
		vector[i] /= norm
	}
	return vector, nil
}

// DifferentialStrain projects the perturbation along two arm directions.
func (tensor Tensor) DifferentialStrain(first, second [3]float64) (float64, error) {
	if err := tensor.finiteError(); err != nil {
		return 0, err
	}
	a, err := planeUnit(first)
	if err != nil {
		return 0, err
	}
	b, err := planeUnit(second)
	if err != nil {
		return 0, err
	}
	var response float64
	for i := range 3 {
		for j := range 3 {
			response += 0.5 * (a[i]*a[j] - b[i]*b[j]) * tensor[i][j]
		}
	}
	if !planeFinite(response) {
		return 0, fmt.Errorf("gravitational: detector response overflows")
	}
	return response, nil
}

// Offset returns first-order tidal displacement, h times position over two.
func (tensor Tensor) Offset(position [3]float64) ([3]float64, error) {
	if err := tensor.finiteError(); err != nil {
		return [3]float64{}, err
	}
	for _, value := range position {
		if !planeFinite(value) {
			return [3]float64{}, fmt.Errorf("gravitational: position must be finite")
		}
	}
	var offset [3]float64
	for i := range 3 {
		for j := range 3 {
			offset[i] += (0.5 * tensor[i][j]) * position[j]
		}
		if !planeFinite(offset[i]) {
			return [3]float64{}, fmt.Errorf("gravitational: offset overflows")
		}
	}
	return offset, nil
}

// finiteError rejects non-finite entries even in an observation null direction.
func (tensor Tensor) finiteError() error {
	for _, row := range tensor {
		for _, value := range row {
			if !planeFinite(value) {
				return fmt.Errorf("gravitational: tensor must be finite")
			}
		}
	}
	return nil
}

// planeFinite excludes both infinities and NaN.
func planeFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
