//go:build darwin

package fuelbed

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"github.com/janearc/slayer/metal"
	"math"
)

//go:embed bed.metal
var stencil string

// fuelSession is the resource boundary used by the retained stencil. Its
// narrow contract also lets failure checks close a real session at each
// allocation or submission boundary without replacing the solver math.
type fuelSession interface {
	Buffer(int) (metal.Buffer, error)
	Dispatch(string, int, int, []byte, ...metal.Buffer) error
	Commit() error
	Close()
}

// Device retains two state buffers and one compiled Metal stencil.
type Device struct {
	session   fuelSession
	spec      Spec
	now, next metal.Buffer
	params    []byte
	tick      int64
	function  string
}

// NewMetal compiles one stencil and retains its two state buffers.
func NewMetal(s Spec) (*Device, error) {
	checked, err := New(s)
	if err != nil {
		return nil, err
	}
	session, err := metal.OpenSession(stencil, "fuel_step")
	if err != nil {
		return nil, err
	}
	return newDevice(checked.spec, session)
}

// newDevice takes ownership of an open session and prepares retained state.
func newDevice(s Spec, session fuelSession) (*Device, error) {
	return newDeviceAdvected(s, nil, session)
}

// newDeviceAdvected keeps the old state allocation and optional transport
// suffix.
func newDeviceAdvected(s Spec, weights *advectionWeights, session fuelSession) (*Device, error) {
	var err error
	d := &Device{session: session, spec: s, params: make([]byte, 32)}
	d.now, err = session.Buffer(len(s.Initial) * 2)
	if err == nil {
		d.next, err = session.Buffer(len(s.Initial) * 2)
	}
	if err != nil {
		session.Close()
		return nil, err
	}
	binary.LittleEndian.PutUint32(d.params, uint32(s.Width))
	binary.LittleEndian.PutUint32(d.params[4:], uint32(s.Depth))
	for i, v := range []float64{s.Step, s.Diffusion / (s.Cell * s.Cell), s.Heating, s.Activation, s.Cooling, s.Consumption} {
		binary.LittleEndian.PutUint32(d.params[8+4*i:], math.Float32bits(float32(v)))
	}
	if weights != nil {
		d.appendAdvection(weights)
	}
	d.reset()
	return d, nil
}

// reset restores initial cells when a caller rewinds the clock.
func (d *Device) reset() {
	for i, c := range d.spec.Initial {
		d.now.Numbers()[2*i] = float32(c.Excess)
		d.now.Numbers()[2*i+1] = float32(c.Fuel)
	}
	d.tick = 0
}

// At evolves on Metal and copies the last complete tick for the caller.
func (d *Device) At(seconds float64) ([]Cell, error) {
	if d.session == nil {
		return nil, fmt.Errorf("fuelbed: closed Metal solver")
	}
	target, err := ticks(seconds, d.spec.Step)
	if err != nil {
		return nil, err
	}
	if target < d.tick {
		d.reset()
	}
	for d.tick < target {
		end := min(target, d.tick+128)
		for d.tick < end {
			if err := d.session.Dispatch(d.stepFunction(), d.spec.Width, d.spec.Depth, d.params, d.now, d.next); err != nil {
				d.Close()
				return nil, err
			}
			d.now, d.next = d.next, d.now
			d.tick++
		}
		if err := d.session.Commit(); err != nil {
			d.Close()
			return nil, err
		}
	}
	cells := make([]Cell, len(d.spec.Initial))
	for i := range cells {
		cells[i] = Cell{float64(d.now.Numbers()[2*i]), float64(d.now.Numbers()[2*i+1])}
	}
	return cells, nil
}

// stepFunction keeps legacy devices and zero-value internal fixtures
// compatible.
func (device *Device) stepFunction() string {
	if device.function == "" {
		return "fuel_step"
	}
	return device.function
}

// Close releases persistent device resources and is safe to repeat.
func (d *Device) Close() {
	if d.session != nil {
		d.session.Close()
		d.session = nil
	}
}
