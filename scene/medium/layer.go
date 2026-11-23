package medium

import (
	"errors"
	"math"
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
	thin    thinning
}

// thinning is where a field is thickest, its x and z, how far from it the
// field has thinned to its least, and that least as a share; a thinning
// with no reach holds the whole field everywhere.
type thinning struct {
	near         maths.Vec
	reach, least float64
}

// newThinning is a thinning toward a point over a reach above nought, to
// a least from nought to one.
func newThinning(point maths.Vec, reach, least float64) (thinning, error) {
	if !weatherFinite(point.X+point.Z) || !weatherFinite(reach) ||
		reach <= weatherZero || !(least >= weatherZero && least <= weatherOne) {
		return thinning{}, errors.New("medium: a field thins toward a " +
			"finite point over a reach above nought, to a share from " +
			"nought to one")
	}
	return thinning{near: point, reach: reach, least: least}, nil
}

// share is the share of itself the field holds over a point: one with no
// reach, else one at the near point thinning by a smooth step to the least.
func (thin thinning) share(at maths.Vec) float64 {
	if thin.reach <= weatherZero {
		return weatherOne
	}
	away := math.Hypot(at.X-thin.near.X, at.Z-thin.near.Z)
	fall := maths.Smooth(weatherZero, thin.reach, away)
	return weatherOne - (weatherOne-thin.least)*fall
}

// Toward makes the layer thickest over a point, by its x and z, thinning
// by a smooth step away from it to least of itself at reach and beyond.
// Reach MUST be above nought and least from nought to one.
func (layer *Layer) Toward(point maths.Vec, reach, least float64) error {
	thin, err := newThinning(point, reach, least)
	if err != nil {
		return err
	}
	layer.thin = thin
	return nil
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

// Asymmetry is the Henyey-Greenstein asymmetry the field scatters with:
// nought scatters evenly every way.
func (layer *Layer) Asymmetry() float64 { return layer.weather.Asymmetry }

// Bounds is the region supplied by the terrain owner.
func (layer *Layer) Bounds() maths.Box { return layer.box }

// Finest retains the same sampling hint as canopy-bound prescribed mist.
func (layer *Layer) Finest() float64 {
	return layer.weather.Depth / mistTuning.samples
}

// Tick shares terrain and bounds while changing only the noise phase.
func (layer *Layer) Tick(at time.Duration) Medium {
	ticked := *layer
	ticked.seconds = at.Seconds()
	return &ticked
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
	return layer.weather.mistDensity(at, below, layer.seconds,
		layer.thin.share(at))
}
