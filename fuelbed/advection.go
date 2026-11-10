package fuelbed

import (
	"fmt"
	"math"
)

// Transport is an explicitly supplied horizontal heat-transport velocity in
// m/s. East and South follow grid columns/rows. This does not infer a wind
// profile.
type Transport struct{ East, South float64 }

// advectionWeights form a nonnegative explicit upwind heat update.
// Side names identify the neighbour supplying an incoming contribution.
type advectionWeights struct {
	centre, diffuse, west, east, north, south, burn float64
}

// NewAdvected adds the advection term in Mandel et al., equation(1).
// Zero transport retains New's original path and arithmetic exactly.
func NewAdvected(spec Spec, transport Transport) (*Bed, error) {
	if !finite(transport.East) || !finite(transport.South) {
		return nil, fmt.Errorf("fuelbed: heat transport must be finite")
	}
	bed, err := New(spec)
	if err != nil {
		return nil, err
	}
	if transport == (Transport{}) {
		return bed, nil
	}
	weights, err := transport.weights(bed.spec)
	if err != nil {
		return nil, err
	}
	bed.advection = &weights
	return bed, nil
}

// weights budgets diffusion, cooling and both directional Courant
// contributions.
func (transport Transport) weights(spec Spec) (advectionWeights, error) {
	weights := advectionWeights{
		diffuse: spec.Step * spec.Diffusion / (spec.Cell * spec.Cell),
		west:    spec.Step * max(transport.East, 0) / spec.Cell,
		east:    spec.Step * max(-transport.East, 0) / spec.Cell,
		north:   spec.Step * max(transport.South, 0) / spec.Cell,
		south:   spec.Step * max(-transport.South, 0) / spec.Cell,
		burn:    spec.Step * spec.Consumption,
	}
	joint := 4*weights.diffuse + spec.Step*spec.Heating*spec.Cooling +
		weights.west + weights.east + weights.north + weights.south
	weights.centre = 1 - joint
	for _, value := range weights.values() {
		if !finite(value) || value < 0 || value > 1 {
			return advectionWeights{}, fmt.Errorf("fuelbed: heat transport joint positivity sum %.17g must be <= 1 (4d + dt*A*C + (abs(east)+abs(south))*dt/h)", joint)
		}
	}
	if !finite(spec.Heating / spec.Consumption) {
		return advectionWeights{}, fmt.Errorf("fuelbed: heat per consumed fraction is not finite")
	}
	return weights, nil
}

// values is the optional solver parameter order; the legacy layout stays
// intact.
func (weights advectionWeights) values() [7]float64 {
	return [7]float64{weights.centre, weights.diffuse, weights.west, weights.east,
		weights.north, weights.south, weights.burn}
}

// advanceAdvected transports heat with ambient inflow; fuel remains stationary.
func (bed *Bed) advanceAdvected() {
	spec, weights := bed.spec, *bed.advection
	for row := range spec.Depth {
		for column := range spec.Width {
			index := row*spec.Width + column
			cell := bed.now[index]
			left := bed.now[row*spec.Width+max(column-1, 0)].Excess
			right := bed.now[row*spec.Width+min(column+1, spec.Width-1)].Excess
			up := bed.now[max(row-1, 0)*spec.Width+column].Excess
			down := bed.now[min(row+1, spec.Depth-1)*spec.Width+column].Excess
			heat := weights.centre*cell.Excess + weights.diffuse*(left+right+up+down)
			heat += weights.incoming(column, row, spec.Width, spec.Depth, left, right, up, down)
			rate := 0.0
			if cell.Excess > 0 {
				rate = math.Exp(-spec.Activation / cell.Excess)
			}
			fuel := cell.Fuel * (1 - weights.burn*rate)
			burned := cell.Fuel - fuel
			bed.next[index] = Cell{Excess: heat + spec.Heating/spec.Consumption*burned, Fuel: fuel}
		}
	}
	bed.now, bed.next = bed.next, bed.now
}

// incoming uses zero excess temperature outside the finite bed for advection.
// Diffusion separately retains its original insulated boundary values.
func (weights advectionWeights) incoming(column, row, width, depth int,
	left, right, up, down float64,
) float64 {
	if column == 0 {
		left = 0
	}
	if column == width-1 {
		right = 0
	}
	if row == 0 {
		up = 0
	}
	if row == depth-1 {
		down = 0
	}
	return weights.west*left + weights.east*right + weights.north*up + weights.south*down
}
