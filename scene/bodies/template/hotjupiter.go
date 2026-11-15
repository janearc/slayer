package template

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// HotJupiter is a giant too near its star: jupiter's gas with its day
// an orbit of eighteen hours, tidally locked, no moons and no aurora,
// and its air streaming away from the star in a tail thirty radii
// long, bent back along its orbit, the gas's own cream.
//
// The shot aims it.
func HotJupiter() (bodies.Body, error) {
	gas, err := giantGas(jupiterStripes())
	if err != nil {
		return bodies.Body{}, fmt.Errorf("a hot jupiter: %w", err)
	}
	tail, err := bodies.NewTail(km(30*jupiterRadiusKm),
		km(2*jupiterRadiusKm), 0.06, 0.25,
		bodies.PerKilometre(0.4/jupiterRadiusKm),
		swatches.Light(swatches.Cream),
		swatches.Light(swatches.TailGlow))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("a hot jupiter: %w", err)
	}
	tail, err = tail.Wisped(maths.Percent(60), 21)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("a hot jupiter: %w", err)
	}
	hot, err := bodies.NewBody("a hot jupiter", bodies.Giant,
		km(jupiterRadiusKm), kg(1.898e27), gas, tail)
	if err != nil {
		return bodies.Body{}, err
	}
	return hot.Turning(hours(18), deg(3.13)), nil
}
