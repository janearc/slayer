package template

import (
	"fmt"

	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
)

// Uranus is the ice giant on its side: a pale cyan, almost featureless,
// with thin dark rings and three of its five large moons: titania, oberon
// and miranda; ariel and umbriel are not here.
func Uranus() (bodies.Body, error) {
	stripes := bands(0.02, 0.01, 0.15, 6,
		swatches.Light(swatches.UranusCyan),
		swatches.Light(swatches.UranusDeep),
		swatches.Light(swatches.UranusPale),
		swatches.Light(swatches.UranusDeep),
		swatches.Light(swatches.UranusCyan))
	stripes.Seed = 16
	gas, err := giantGas(stripes)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("uranus: %w", err)
	}
	rings, err := bodies.NewRings(1.65, 2.02, km(0.5),
		bodies.PerKilometre(0.5),
		swatches.Light(swatches.UranusRing))
	if err != nil {
		return bodies.Body{}, fmt.Errorf("uranus: %w", err)
	}
	rings = rings.Profiled([]float64{0.2, 0.05, 0.4, 0.05, 0.3, 0.05,
		1, 0.05, 0.5}).Grained(0.4, 40)
	titania, err := smallMoon("titania", km(788.4), kg(3.4e21), km(435910), rad(0))
	if err != nil {
		return bodies.Body{}, err
	}
	oberon, err := smallMoon("oberon", km(761.4), kg(3.08e21), km(583520), rad(2))
	if err != nil {
		return bodies.Body{}, err
	}
	miranda, err := smallMoon("miranda", km(235.8), kg(6.6e19), km(129390), rad(1))
	if err != nil {
		return bodies.Body{}, err
	}
	uranus, err := bodies.NewBody("uranus", bodies.Giant, km(25362),
		kg(8.681e25), gas, rings, titania, oberon, miranda)
	if err != nil {
		return bodies.Body{}, err
	}
	// uranus turns backwards, on its side: the tilt past ninety says
	// so, as pluto's does, and the day is positive
	return uranus.Turning(hours(17.24), deg(97.77)), nil
}
