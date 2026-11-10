// Package fuelbed evolves a surface layer's heat and remaining fuel.
package fuelbed

import (
	"fmt"
	"math"
	"slices"
)

// Spec describes a uniform rectangular layer in SI units. Initial contains
// row-major cells; Diffusion is m^2/s, Heating K/s, Activation K,
// Cooling 1/K, Consumption 1/s, Ambient K and Step seconds.
//
// Ambient records the caller's absolute reference temperature. The stencil
// evolves excess above that reference, so Ambient does not enter the equations.
type Spec struct {
	Width, Depth                                                              int
	Cell, Step, Diffusion, Heating, Activation, Cooling, Consumption, Ambient float64
	Initial                                                                   []Cell
}

// Cell keeps excess temperature in kelvin and the remaining fuel fraction.
// Fuel stays in its cell; heat transfer does not replenish it.
type Cell struct{ Excess, Fuel float64 }

// Evolution supplies owned snapshots of a surface fuel model. Both Bed and
// Device implement it; callers release Device with Close when finished.
type Evolution interface {
	At(float64) ([]Cell, error)
}

// Bed owns a CPU reference evolution with immutable returned snapshots.
type Bed struct {
	spec      Spec
	now, next []Cell
	tick      int64
	advection *advectionWeights
}

// New checks the grid and explicit-step stability, then copies its state.
func New(s Spec) (*Bed, error) {
	if s.Width < 1 || s.Depth < 1 || s.Width > 256 || s.Depth > 256 || len(s.Initial) != s.Width*s.Depth {
		return nil, fmt.Errorf("fuelbed: dimensions must be 1..256 with one initial cell each")
	}
	for _, v := range []float64{s.Cell, s.Step, s.Heating, s.Activation, s.Consumption, s.Ambient} {
		if !finite(v) || v <= 0 {
			return nil, fmt.Errorf("fuelbed: positive finite coefficients required")
		}
	}
	for _, v := range []float64{s.Diffusion, s.Cooling} {
		if !finite(v) || v < 0 {
			return nil, fmt.Errorf("fuelbed: negative/non-finite coefficient")
		}
	}
	// Positivity bounds for explicit diffusion, cooling and fuel loss.
	if s.Step*(4*s.Diffusion/(s.Cell*s.Cell)+s.Heating*s.Cooling) > 1 || s.Step*s.Consumption > 1 || s.Step > 1 || s.Step < .0001 {
		return nil, fmt.Errorf("fuelbed: unstable or impractical fixed time step")
	}
	for _, c := range s.Initial {
		if !finite(c.Excess) || c.Excess < 0 || c.Excess > 10000 || !finite(c.Fuel) || c.Fuel < 0 || c.Fuel > 1 {
			return nil, fmt.Errorf("fuelbed: invalid initial cell")
		}
	}
	s.Initial = slices.Clone(s.Initial)
	return &Bed{spec: s, now: slices.Clone(s.Initial), next: make([]Cell, len(s.Initial))}, nil
}

// At advances to the last complete fixed tick, or replays after a rewind.
// Calls on one bed are serial; returned cells belong to the caller.
func (b *Bed) At(seconds float64) ([]Cell, error) {
	target, err := ticks(seconds, b.spec.Step)
	if err != nil {
		return nil, err
	}
	if target < b.tick {
		copy(b.now, b.spec.Initial)
		b.tick = 0
	}
	for b.tick < target {
		b.advance()
		b.tick++
	}
	return slices.Clone(b.now), nil
}

// ticks selects a completed time slice and refuses invalid requests.
func ticks(seconds, step float64) (int64, error) {
	if !finite(seconds) || seconds < 0 || seconds > 86400 {
		return 0, fmt.Errorf("fuelbed: time must be finite within 0..86400s")
	}
	return int64(math.Floor(seconds/step + 1e-9)), nil
}

// advance applies one simultaneous heat and fuel update.
func (b *Bed) advance() {
	if b.advection != nil {
		b.advanceAdvected()
		return
	}
	s := b.spec
	for z := 0; z < s.Depth; z++ {
		for x := 0; x < s.Width; x++ {
			i := z*s.Width + x
			c := b.now[i]
			l := b.now[z*s.Width+max(x-1, 0)].Excess
			r := b.now[z*s.Width+min(x+1, s.Width-1)].Excess
			u := b.now[max(z-1, 0)*s.Width+x].Excess
			d := b.now[min(z+1, s.Depth-1)*s.Width+x].Excess
			burned := s.Step * s.Consumption * Reaction(c, s.Activation)
			heat := s.Diffusion*(l+r+u+d-4*c.Excess)/(s.Cell*s.Cell) - s.Heating*s.Cooling*c.Excess
			b.next[i] = Cell{Excess: c.Excess + s.Step*heat + s.Heating/s.Consumption*burned, Fuel: c.Fuel - burned}
		}
	}
	b.now, b.next = b.next, b.now
}

// Reaction is S exp(-B/u), with zero oxidation at ambient temperature.
// Mandel et al., equations (1-2), https://arxiv.org/abs/0709.0086 .
func Reaction(c Cell, activation float64) float64 {
	if c.Excess <= 0 || c.Fuel <= 0 {
		return 0
	}
	return c.Fuel * math.Exp(-activation/c.Excess)
}

// finite rejects infinities and NaNs before they enter the stencil.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
