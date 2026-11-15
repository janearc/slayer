package template

import (
	"fmt"

	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/swatches"
	"github.com/janearc/slayer/scene/texture"
)

// Neptune is deep blue with a dark storm and white streaks, and triton.
// the real triton goes round it backwards, inclined; here it goes round
// forwards on the equator, since bodies cannot place a moon backwards.
func Neptune() (bodies.Body, error) {
	stripes := bands(0.05, 0.03, 0.3, 8,
		swatches.Light(swatches.NeptuneBlue),
		swatches.Light(swatches.NeptuneDeep),
		swatches.Light(swatches.NeptunePale),
		swatches.Light(swatches.NeptuneDeep),
		swatches.Light(swatches.NeptuneBlue),
		swatches.Light(swatches.NeptuneDark),
		swatches.Light(swatches.NeptuneBlue))
	stripes.Spots = []texture.Spot{{Lat: -0.35, Lon: -1.2, Size: 0.12,
		Stretch: 1.5, Turn: 2,
		Paint: swatches.Light(swatches.NeptuneSpot)}}
	stripes.Streets = []texture.Street{{Lat: -0.5, Count: 4, Size: 0.05,
		Turn: 0.5, Paint: swatches.Light(swatches.NeptuneCloud),
		Seed: 17}}
	stripes.Seed = 17
	gas, err := giantGas(stripes)
	if err != nil {
		return bodies.Body{}, fmt.Errorf("neptune: %w", err)
	}
	triton, err := smallMoon("triton", km(1353.4), kg(2.14e22), km(354759), rad(0))
	if err != nil {
		return bodies.Body{}, err
	}
	neptune, err := bodies.NewBody("neptune", bodies.Giant, km(24622),
		kg(1.024e26), gas, triton)
	if err != nil {
		return bodies.Body{}, err
	}
	return neptune.Turning(hours(16.11), deg(28.32)), nil
}
