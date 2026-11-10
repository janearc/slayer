package medium

import (
	_ "embed"
	"math"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

// weatherZero is the defined additive identity.
// weatherOne is the defined multiplicative identity.
const weatherZero, weatherOne = 0, 1

// noiseHalf is defined by mapping a signed unit range to a positive unit range.
const noiseHalf = 0.5

//go:embed mist-weather.yaml
var mistWeatherRaw []byte

// mistChoices holds immutable model choices after Puzzlemouth reads them.
// Density queries do not parse data or search the numeric catalogue.
type mistChoices struct {
	octaves           int
	shoulder, samples float64
}

var mistTuning = readMistChoices()

// readMistChoices treats invalid embedded data as a code fault.
func readMistChoices() mistChoices {
	read, err := numbers.Read("mist-weather.yaml", mistWeatherRaw)
	if err != nil {
		panic(err)
	}
	return mistChoices{
		octaves:  int(mistChoice(read, "octaves")),
		shoulder: mistChoice(read, "threshold-shoulder"),
		samples:  mistChoice(read, "samples-per-depth"),
	}
}

// mistChoice reads an immutable choice. A missing name is a code fault.
func mistChoice(read numbers.Numbers, name string) float64 {
	choice, err := read.Get(name)
	if err != nil {
		panic(err)
	}
	return choice.Value
}

// NewWeather describes prescribed mist before a canopy or layer checks it.
// GroundFloor and Fade remain available for canopy-bound fields.
func NewWeather(extinction, depth, coverage float64, seed uint64,
	wind maths.Vec,
) GroveWeather {
	return GroveWeather{Extinction: extinction, Depth: depth,
		Coverage: coverage, Seed: seed, Wind: wind}
}

// weatherFinite refuses NaN and either infinity before field construction.
func weatherFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, weatherZero)
}

// validMist checks inputs shared by canopy and regional prescribed weather.
func (weather GroveWeather) validMist() bool {
	return weatherFinite(weather.Extinction) &&
		weather.Extinction >= weatherZero &&
		weatherFinite(weather.Depth) && weather.Depth > weatherZero &&
		weatherFinite(weather.Coverage) &&
		weather.Coverage >= weatherZero &&
		weather.Coverage <= weatherOne &&
		weatherFinite(weather.Wind.Len()) &&
		weatherFinite(weather.GroundFloor)
}

// mistDensity shares the existing height fade and advected noise field.
// Weight is a spatial mask, such as canopy coverage; it is not a water rate.
func (weather GroveWeather) mistDensity(at maths.Vec, below, seconds,
	weight float64,
) float64 {
	if weight == weatherZero || weather.Coverage == weatherZero {
		return weatherZero
	}
	if weather.Coverage < weatherOne {
		advected := at.Sub(weather.Wind.Scale(seconds))
		patch := noiseHalf + noiseHalf*noise.Fractal(weather.Seed,
			advected.X/weather.Depth, advected.Y/weather.Depth,
			advected.Z/weather.Depth, mistTuning.octaves)
		threshold := weatherOne - weather.Coverage
		weight *= maths.Smooth(threshold-mistTuning.shoulder,
			threshold+mistTuning.shoulder, patch)
	}
	return weather.Extinction * weight *
		(weatherOne - maths.Smooth(weatherZero, weather.Depth, below))
}
