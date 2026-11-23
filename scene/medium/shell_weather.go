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
	if !weatherFinite(radius) || radius <= weatherZero || !weatherFinite(low) ||
		low < weatherZero || !weatherFinite(high) || high <= low ||
		!weatherFinite(radius+high) || weather.PatchScale <= weatherZero ||
		!weather.validMist() {
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
	return cube(radius)
}

// Tick shares checked controls and samples an absolute scene time.
func (shell *ShellWeather) Tick(at time.Duration) Medium {
	ticked := *shell
	ticked.parameters.Seconds = at.Seconds()
	return &ticked
}

// shellLegs is the most legs a shell crossing has: the outer ball's crossing
// less the inner ball's leaves a leg either side of the clear interior.
const shellLegs = 2 // defined: a shell has a near leg and a far leg

// shellParabola is the four of the parabola 4*h*(1-h), which is one at its
// peak halfway through the shell.
const shellParabola = 4 // defined: the parabola's peak is one

// Density combines advected patches with 4*h*(1-h), a parabola whose peak is
// one halfway through the shell and whose two radial boundaries are clear.
func (shell *ShellWeather) Density(point maths.Vec) float64 {
	parameters := shell.parameters
	height := point.Len() - parameters.Radius
	if !weatherFinite(height) || height <= parameters.Low ||
		height >= parameters.High {
		return weatherZero
	}
	share := (height - parameters.Low) / (parameters.High - parameters.Low)
	return parameters.Weather.mistDensity(point, weatherZero,
		parameters.Seconds, shellParabola*share*(weatherOne-share))
}

// Spans subtracts the inner ball from the outer crossing, retaining either
// shell leg when a camera or lamp lies in the intervening clear interior.
func (shell *ShellWeather) Spans(ray maths.Ray, near,
	far float64) [shellLegs]Interval {
	parameters := shell.parameters
	enter, leave, hit := sphere(parameters.Radius+parameters.High, ray, near, far)
	if !hit {
		return [shellLegs]Interval{}
	}
	into, out, inside := sphere(parameters.Radius+parameters.Low, ray, enter,
		leave)
	if !inside {
		return [shellLegs]Interval{{Enter: enter, Leave: leave}}
	}
	return [shellLegs]Interval{{Enter: enter, Leave: into},
		{Enter: out, Leave: leave}}
}
