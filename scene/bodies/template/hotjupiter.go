package template

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// hotJupiterRadiusKm is the hot jupiter's radius in kilometres: what
// scene/generate gives for jupiter's mass and age at an orbit of
// eighteen hours round a Sun. That is the cold radius of Fortney, Marley
// & Barnes 2007 (tables 2 to 4) at that light, 78618 km, and Sestovic,
// Demory & Queloz 2018's inflation (their equation 3 and table 1) of
// about 0.47 Jupiter radii on top, for a mass in its 0.98 to 2.5 Jupiter
// mass bin. It was 69911 km, jupiter's own. The tail is said in this
// radius, not jupiter's.
const hotJupiterRadiusKm = 111962

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
	tail, err := bodies.NewTail(km(30*hotJupiterRadiusKm),
		km(2*hotJupiterRadiusKm), 0.06, 0.25,
		bodies.PerKilometre(0.4/hotJupiterRadiusKm),
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
		km(hotJupiterRadiusKm), kg(1.898e27), gas, tail)
	if err != nil {
		return bodies.Body{}, err
	}
	return hot.Turning(hours(18), deg(3.13)), nil
}
