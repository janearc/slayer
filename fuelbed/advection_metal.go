//go:build darwin

package fuelbed

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/janearc/slayer/metal"
)

//go:embed advection.metal
var advectionStencil string

// NewMetalAdvected retains an optional upwind stencil and fixed state buffers.
// Zero transport selects the existing NewMetal path and parameter layout.
func NewMetalAdvected(spec Spec, transport Transport) (*Device, error) {
	if transport == (Transport{}) {
		return NewMetal(spec)
	}
	checked, err := NewAdvected(spec, transport)
	if err != nil {
		return nil, err
	}
	if err := validMetalTransport(checked.spec, *checked.advection); err != nil {
		return nil, err
	}
	session, err := metal.OpenSession(stencil+"\n"+advectionStencil, "fuel_step_advected")
	if err != nil {
		return nil, err
	}
	return newDeviceAdvected(checked.spec, checked.advection, session)
}

// validMetalTransport checks coefficients and the rounded thermal weights.
// Tiny weights may round or flush to zero; the declared numerical acceptance
// budget remains necessary and is not a universal conservation guarantee.
func validMetalTransport(spec Spec, weights advectionWeights) error {
	for _, value := range []float64{spec.Heating, spec.Activation, spec.Consumption} {
		encoded := float32(value)
		if !finite(float64(encoded)) || encoded <= 0 {
			return fmt.Errorf("fuelbed: advected coefficient is not representable on Metal")
		}
	}
	ratio := float32(spec.Heating) / float32(spec.Consumption)
	if !finite(float64(ratio)) || ratio <= 0 {
		return fmt.Errorf("fuelbed: advected thermal yield is not representable on Metal")
	}
	return validRoundedTransport(weights)
}

// validRoundedTransport allows four float32 ulps at unity for the accumulated
// rounding of centre, four diffuse copies and four directional weights.
func validRoundedTransport(weights advectionWeights) error {
	var rounded [7]float64
	for index, value := range weights.values() {
		rounded[index] = float64(float32(value))
		if !finite(rounded[index]) || rounded[index] < 0 || rounded[index] > 1 {
			return fmt.Errorf("fuelbed: invalid rounded advection weight %d", index)
		}
	}
	sum := rounded[0] + 4*rounded[1] + rounded[2] + rounded[3] + rounded[4] + rounded[5]
	allowance := 4 * float64(math.Nextafter32(1, 2)-1)
	if sum > 1+allowance {
		return fmt.Errorf("fuelbed: rounded thermal weight sum %.17g exceeds 1 + %.17g", sum, allowance)
	}
	return nil
}

// appendAdvection extends only the new entry's private parameter suffix.
func (device *Device) appendAdvection(weights *advectionWeights) {
	device.function = "fuel_step_advected"
	for _, value := range weights.values() {
		device.params = binary.LittleEndian.AppendUint32(device.params, math.Float32bits(float32(value)))
	}
}
