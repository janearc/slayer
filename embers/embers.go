// Package embers describes prescribed firebrand trajectories and cooling.
// It is a rendering model, not a combustion or secondary-ignition solver.
package embers

import (
	"errors"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"math"
	"time"
)

// Spec gives the authored source, motion and thermal endpoints in SI units.
type Spec struct {
	Rate                  float64
	Life, Drag            time.Duration
	Radius, Disk, Height  measure.Length
	Velocity, Jitter, Air maths.Vec
	Gravity               float64
	Hot, Cool             measure.Temperature
	Seed                  uint64
}

// Source supplies repeatable firebrand snapshots to a scene or animation.
// New returns an implementation of this contract.
type Source interface {
	At(time.Duration) []Particle
}

type emitter struct{ spec Spec }

// Particle is one living firebrand at the requested snapshot time.
type Particle struct {
	id     int64
	at     maths.Vec
	radius measure.Length
	heat   measure.Temperature
	age    time.Duration
}

// mostLiving limits snapshot work per emitter; it is an engineering bound.
const mostLiving = 8192

// New checks a prescribed particle source.
func New(spec Spec) (emitter, error) {
	valid := spec.Rate > 0 && spec.Rate <= 1e6 && spec.Life > 0 && spec.Drag > 0 &&
		spec.Radius.InMetres() > 0 && spec.Disk.InMetres() >= 0 &&
		spec.Height.InMetres() >= 0 && spec.Gravity >= 0 &&
		spec.Hot.InKelvin() > 0 && spec.Cool.InKelvin() > 0 &&
		spec.Cool.InKelvin() <= spec.Hot.InKelvin() &&
		spec.Jitter.X >= 0 && spec.Jitter.Y >= 0 && spec.Jitter.Z >= 0
	numbers := []float64{spec.Rate, spec.Radius.InMetres(), spec.Disk.InMetres(),
		spec.Height.InMetres(), spec.Gravity, spec.Hot.InKelvin(), spec.Cool.InKelvin(),
		spec.Velocity.X, spec.Velocity.Y, spec.Velocity.Z,
		spec.Jitter.X, spec.Jitter.Y, spec.Jitter.Z, spec.Air.X, spec.Air.Y, spec.Air.Z}
	for _, value := range numbers {
		valid = valid && !math.IsNaN(value) && !math.IsInf(value, 0)
	}
	if !valid || math.Ceil(spec.Rate*spec.Life.Seconds())+2 > mostLiving {
		return emitter{}, errors.New("embers: invalid source or too many living particles")
	}
	return emitter{spec}, nil
}

// At gives living particles at an absolute time since this source started.
// One independently jittered birth per interval avoids emission bands.
func (source emitter) At(when time.Duration) []Particle {
	if when < 0 {
		return nil
	}
	spec := source.spec
	seconds := when.Seconds()
	first := max(int64(0), int64(math.Floor((seconds-spec.Life.Seconds())*spec.Rate))-1)
	last := int64(math.Floor(seconds * spec.Rate))
	var living []Particle
	for id := first; id <= last; id++ {
		birth := time.Duration((float64(id) + share(spec.Seed, id, 0)) / spec.Rate * float64(time.Second))
		age := when - birth
		if age < 0 || age >= spec.Life {
			continue
		}
		living = append(living, source.particle(id, age))
	}
	return living
}

// particle samples a stable disk position, launch velocity and cooling age.
func (source emitter) particle(id int64, age time.Duration) Particle {
	spec := source.spec
	radius := spec.Disk.InMetres() * math.Sqrt(share(spec.Seed, id, 1))
	angle := 2 * math.Pi * share(spec.Seed, id, 2)
	at := maths.V(radius*math.Cos(angle), spec.Height.InMetres(), radius*math.Sin(angle))
	jitter := maths.V(2*share(spec.Seed, id, 3)-1, 2*share(spec.Seed, id, 4)-1, 2*share(spec.Seed, id, 5)-1)
	velocity := spec.Velocity.Add(maths.V(jitter.X*spec.Jitter.X, jitter.Y*spec.Jitter.Y, jitter.Z*spec.Jitter.Z))
	at = position(at, velocity, spec.Air, maths.V(0, -spec.Gravity, 0), age, spec.Drag)
	hot, cool := spec.Hot.InKelvin(), spec.Cool.InKelvin()
	heat := cool + (hot-cool)*(1-age.Seconds()/spec.Life.Seconds())
	return Particle{id: id, at: at, radius: spec.Radius, heat: measure.Kelvin(heat), age: age}
}

// share hashes a birth and attribute independently; it is not evolving state.
// SplitMix64's finalizer supplies a reproducible uniform rendering sample.
func share(seed uint64, id int64, attribute uint64) float64 {
	bits := seed + uint64(id)*0x9e3779b97f4a7c15 + attribute*0x632be59bd9b4e019
	bits = (bits ^ (bits >> 30)) * 0xbf58476d1ce4e5b9
	bits = (bits ^ (bits >> 27)) * 0x94d049bb133111eb
	bits ^= bits >> 31
	return float64(bits>>11) / (1 << 53)
}

// ID is the stable birth identifier.
func (p Particle) ID() int64 { return p.id }

// Position is the current location in source coordinates.
func (p Particle) Position() maths.Vec { return p.at }

// Radius is the physical particle radius.
func (p Particle) Radius() measure.Length { return p.radius }

// Heat is the prescribed surface temperature.
func (p Particle) Heat() measure.Temperature { return p.heat }

// Age is time since this particle's birth.
func (p Particle) Age() time.Duration { return p.age }

// position integrates linear drag toward constant air velocity and gravity.
func position(at, velocity, air, gravity maths.Vec, age, drag time.Duration) maths.Vec {
	seconds, tau := age.Seconds(), drag.Seconds()
	terminal := air.Add(gravity.Scale(tau))
	return at.Add(terminal.Scale(seconds)).Add(velocity.Sub(terminal).Scale(-tau * math.Expm1(-seconds/tau)))
}
