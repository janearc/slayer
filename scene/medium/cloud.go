package medium

import (
	"errors"
	"time"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// Cloud is a deck of separate clouds between a level base and a level top
// over a bounded region. Seeded noise cut at a threshold makes bodies with
// clear sky between them, and the share of the deck the bodies take
// narrows from the weather's coverage at the base to nothing at the top,
// so a body is widest at its flat base and rounds off upward, as heaped
// cloud does.
//
// Inside a body the density is the weather's extinction; at
// its edge it falls over the noise's shoulder.
//
// It is prescribed weather: nothing condenses, and the bodies keep their
// shapes but for the wind carrying them.
type Cloud struct {
	weather   GroveWeather
	box       maths.Box
	base, top float64
	seconds   float64
	thin      thinning
}

// Toward makes the deck's cover greatest over a point, by its x and z, and
// thinning by a smooth step away from it to least of itself at reach and
// beyond, so the bodies are fewer and smaller there. Reach MUST be above
// nought and least from nought to one.
func (cloud *Cloud) Toward(point maths.Vec, reach, least float64) error {
	thin, err := newThinning(point, reach, least)
	if err != nil {
		return err
	}
	cloud.thin = thin
	return nil
}

// NewCloud is a deck of clouds from base to top, in scene heights, over
// the bounds' x and z. The weather's extinction, coverage, seed, patch
// scale, wind and asymmetry keep their meanings; its depth is the deck's.
// The top MUST be above the base and the coverage above nought.
func NewCloud(weather GroveWeather, bounds maths.Box, base,
	top float64) (*Cloud, error) {
	weather.Depth = top - base
	if !weatherFinite(base) || !weatherFinite(top) ||
		!weather.validMist() || weather.Coverage <= weatherZero ||
		bounds.Max.X <= bounds.Min.X || bounds.Max.Z <= bounds.Min.Z ||
		!weatherFinite(bounds.Min.Len()) ||
		!weatherFinite(bounds.Max.Len()) {
		return nil, errors.New("medium: a cloud deck has a top above " +
			"its base, a coverage above nought and finite bounds")
	}
	bounds.Min.Y, bounds.Max.Y = base, top
	return &Cloud{weather: weather, box: bounds, base: base, top: top}, nil
}

// Bounds is the deck's region, from its base to its top.
func (cloud *Cloud) Bounds() maths.Box { return cloud.box }

// Asymmetry is the Henyey-Greenstein asymmetry the cloud scatters with.
func (cloud *Cloud) Asymmetry() float64 { return cloud.weather.Asymmetry }

// Tick is the deck at a time: the same bodies, carried by the wind.
func (cloud *Cloud) Tick(at time.Duration) Medium {
	ticked := *cloud
	ticked.seconds = at.Seconds()
	return &ticked
}

// Density is the extinction inside a body and nought between them, out of
// the region, below the base and above the top.
func (cloud *Cloud) Density(at maths.Vec) float64 {
	box := cloud.box
	if !weatherFinite(at.Len()) ||
		at.X < box.Min.X || at.X > box.Max.X || at.Z < box.Min.Z ||
		at.Z > box.Max.Z {
		return weatherZero
	}
	height := (at.Y - cloud.base) / cloud.weather.Depth
	if height <= weatherZero || height >= weatherOne {
		return weatherZero
	}
	share := cloud.weather.Coverage * cloud.thin.share(at) *
		(weatherOne - height*height)
	if share <= weatherZero {
		return weatherZero
	}
	threshold := weatherOne - share
	patch := cloud.patch(at)
	return cloud.weather.Extinction * maths.Smooth(
		threshold-mistTuning.shoulder, threshold+mistTuning.shoulder, patch)
}

// patch is the deck's noise at a point, carried by the wind, from nought
// to one, in cells of the weather's patch scale or the deck's depth.
func (cloud *Cloud) patch(at maths.Vec) float64 {
	scale := cloud.weather.Depth
	if cloud.weather.PatchScale > weatherZero {
		scale = cloud.weather.PatchScale
	}
	carried := at.Sub(cloud.weather.Wind.Scale(cloud.seconds))
	return noiseHalf + noiseHalf*noise.Fractal(cloud.weather.Seed,
		carried.X/scale, carried.Y/scale, carried.Z/scale,
		mistTuning.octaves)
}
