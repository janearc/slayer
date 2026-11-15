package medium

import (
	"fmt"
	"time"

	"github.com/janearc/slayer/maths"
)

// ShellParameters is a snapshot of prescribed weather in planet-centred space.
// Low and High are radial heights; the clock and wind advect authored patches.
type ShellParameters struct {
	Radius, Low, High, Seconds float64
	Weather                    GroveWeather
	NoiseOctaves               int
	ThresholdShoulder          float64
}

// ShellWeather follows a spherical planet rather than a site's tangent plane.
// Solid terrain hides intersecting clouds through ordinary visibility; this
// field does not infer condensation or a forecast from the terrain.
type ShellWeather struct{ parameters ShellParameters }

// NewShellWeather requires explicit patch size and a finite shell above sea.
// The vertical profile uses the shell depth, independently of patch size.
func NewShellWeather(radius, low, high float64, weather GroveWeather) (*ShellWeather, error) {
	weather.Depth = high - low
	if !weatherFinite(radius) || radius <= 0 || !weatherFinite(low) || low < 0 ||
		!weatherFinite(high) || high <= low || !weatherFinite(radius+high) ||
		weather.PatchScale <= 0 || !weather.validMist() {
		return nil, fmt.Errorf("medium: invalid spherical weather shell")
	}
	return &ShellWeather{parameters: ShellParameters{Radius: radius, Low: low, High: high,
		Weather: weather, NoiseOctaves: mistTuning.octaves, ThresholdShoulder: mistTuning.shoulder}}, nil
}

// Parameters returns values rather than mutable field ownership to an encoder.
func (shell *ShellWeather) Parameters() ShellParameters { return shell.parameters }

// Bounds encloses the entire outer sphere without a geographic crop boundary.
func (shell *ShellWeather) Bounds() maths.Box {
	radius := shell.parameters.Radius + shell.parameters.High
	return maths.Box{Min: maths.V(-radius, -radius, -radius), Max: maths.V(radius, radius, radius)}
}

// Tick shares checked controls and samples an absolute scene time.
func (shell *ShellWeather) Tick(at time.Duration) Medium {
	copy := *shell
	copy.parameters.Seconds = at.Seconds()
	return &copy
}

// Density combines advected patches with 4*h*(1-h), a parabola whose peak is
// one halfway through the shell and whose two radial boundaries are clear.
func (shell *ShellWeather) Density(at maths.Vec) float64 {
	parameters := shell.parameters
	height := at.Len() - parameters.Radius
	if !weatherFinite(height) || height <= parameters.Low || height >= parameters.High {
		return 0
	}
	h := (height - parameters.Low) / (parameters.High - parameters.Low)
	return parameters.Weather.mistDensity(at, 0, parameters.Seconds, 4*h*(1-h))
}

// Spans subtracts the inner ball from the outer crossing, retaining either
// shell leg when a camera or lamp lies in the intervening clear interior.
func (shell *ShellWeather) Spans(ray maths.Ray, near, far float64) [2]Interval {
	parameters := shell.parameters
	enter, leave, hit := sphere(parameters.Radius+parameters.High, ray, near, far)
	if !hit {
		return [2]Interval{}
	}
	a, b, inside := sphere(parameters.Radius+parameters.Low, ray, enter, leave)
	if !inside {
		return [2]Interval{{Enter: enter, Leave: leave}}
	}
	return [2]Interval{{Enter: enter, Leave: a}, {Enter: b, Leave: leave}}
}
