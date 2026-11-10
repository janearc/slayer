package medium

import (
	"errors"
	"time"

	"github.com/janearc/slayer/maths"
)

// Layer is one prescribed mist field across a bounded terrain region,
// including clearings. It does not infer weather from the number of trees.
type Layer struct {
	weather GroveWeather
	box     maths.Box
	ground  func(x, z float64) float64
	seconds float64
	base    float64
	level   bool
}

// NewLayer binds weather to explicit world-space bounds and a terrain height.
// Ground must be safe for concurrent reads. Bounds must enclose all intended
// ground heights plus Depth; they also clip the field. Canopy-specific
// GroundFloor and Fade do not mask this regional field.
func NewLayer(weather GroveWeather, bounds maths.Box,
	ground func(x, z float64) float64,
) (*Layer, error) {
	if ground == nil || !weather.validMist() ||
		!weatherFinite(bounds.Min.Len()) ||
		!weatherFinite(bounds.Max.Len()) ||
		bounds.Max.X <= bounds.Min.X || bounds.Max.Y <= bounds.Min.Y ||
		bounds.Max.Z <= bounds.Min.Z {
		return nil, errors.New("medium: invalid mist layer " +
			"or terrain bounds")
	}
	return &Layer{weather: weather, box: bounds, ground: ground}, nil
}

// NewDeck places the height profile above a fixed scene-space base instead
// of following terrain. Ground still clips mist inside the landscape.
// Bounds may clip the profile; their faces remain inside the support.
func NewDeck(weather GroveWeather, bounds maths.Box,
	ground func(x, z float64) float64, base float64,
) (*Layer, error) {
	if !weatherFinite(base) {
		return nil, errors.New("medium: non-finite cloud deck base")
	}
	layer, err := NewLayer(weather, bounds, ground)
	if err != nil {
		return nil, err
	}
	layer.base, layer.level = base, true
	return layer, nil
}

// Bounds is the region supplied by the terrain owner.
func (layer *Layer) Bounds() maths.Box { return layer.box }

// Finest retains the same sampling hint as canopy-bound prescribed mist.
func (layer *Layer) Finest() float64 {
	return layer.weather.Depth / mistTuning.samples
}

// Tick shares terrain and bounds while changing only the noise phase.
func (layer *Layer) Tick(at time.Duration) Medium {
	copy := *layer
	copy.seconds = at.Seconds()
	return &copy
}

// Density extends the prescribed height profile across the entire region.
// Regions are authored weather supports, not simulated condensation volumes.
func (layer *Layer) Density(at maths.Vec) float64 {
	box := layer.box
	if !weatherFinite(at.Len()) ||
		at.X < box.Min.X || at.X > box.Max.X || at.Y < box.Min.Y ||
		at.Y > box.Max.Y || at.Z < box.Min.Z || at.Z > box.Max.Z {
		return weatherZero
	}
	ground := layer.ground(at.X, at.Z)
	if !weatherFinite(ground) || at.Y <= ground {
		return weatherZero
	}
	below := at.Y - ground
	if layer.level {
		below = at.Y - layer.base
	}
	if below <= weatherZero ||
		below >= layer.weather.Depth {
		return weatherZero
	}
	return layer.weather.mistDensity(at, below, layer.seconds, weatherOne)
}
