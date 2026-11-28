package placement

import (
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
)

// DisplaySkyGradient carries only the producer's plain directional gradient.
// It does not represent stellar, scattering, or composited sky models.
type DisplaySkyGradient sky.Gradient

// WorkingSkyGradient copies the final plain producer sky when it can be shown.
func WorkingSkyGradient(source sky.Sky) *DisplaySkyGradient {
	plain, ok := source.(sky.Gradient)
	if !ok {
		return nil
	}
	gradient := DisplaySkyGradient(plain)
	if !gradient.Valid() {
		return nil
	}
	return &gradient
}

// Valid rejects non-finite or negative radiance and invalid directions.
func (gradient DisplaySkyGradient) Valid() bool {
	for _, light := range [...]maths.Light{gradient.Horizon, gradient.Zenith, gradient.Ground} {
		for _, channel := range [...]float64{light.R, light.G, light.B} {
			if !displayFinite(channel) || channel < 0 {
				return false
			}
		}
	}
	return displayFinite(gradient.Up.X) && displayFinite(gradient.Up.Y) && displayFinite(gradient.Up.Z)
}

// SampleSky evaluates the plain gradient for a continuous normalized working
// pixel. The result is scene-linear and has no photographic development.
// An absent gradient gives zero radiance.
func (scene *DisplayScene) SampleSky(x, y float64) maths.Light {
	if scene == nil || scene.SkyGradient == nil || scene.Width <= 0 || scene.Height <= 0 ||
		!displayFinite(x) || !displayFinite(y) {
		return maths.Light{}
	}
	half, aspect := scene.projectionShape()
	ray := maths.V((2*x-1)*half, (1-2*y)*half/aspect, -1)
	return sky.Gradient(*scene.SkyGradient).Light(scene.Eye.Orientation.Out(ray))
}
