package oceantex

import (
	"errors"
	"math"
)

// FoamField is one already-prepared surface at one instant. Foam is the
// instantaneous fraction from zero to one, not an accumulated history.
// Wet excludes ground at or above the surface's still-water level.
type FoamField interface {
	Sample(east, north float64) (height, foam float64, ok bool)
	Wet(east, north float64) bool
}

// FoamFrame gives a prepared earlier surface and its age in seconds relative
// to the current surface. Frames are ordered from youngest to oldest.
type FoamFrame struct {
	Age   float64
	Field FoamField
}

// FoamTrace follows prior instantaneous foam forward at a chosen constant
// horizontal drift, decaying exponentially with age. It takes the largest
// contribution rather than summing, so the number of history samples cannot
// create energy or push the foam fraction above one. These are authored
// rendering controls, not a simulation of advection or breaking.
//
// Only wet starting and ending points contribute. The straight path between
// them is not checked for intermediate land; swash and foam stranded on dry
// sand are not represented. Frames are prepared once, not per pixel.
type FoamTrace struct {
	current               FoamField
	earlier               []FoamFrame
	lifetime, east, north float64
}

// NewFoamTrace creates a causal fading envelope over prepared surfaces.
// Lifetime is the e-folding time in seconds; zero disables the history and
// returns the current instantaneous field. Drift is in metres per second.
func NewFoamTrace(current FoamField, earlier []FoamFrame, lifetime, driftEast, driftNorth float64) (*FoamTrace, error) {
	if current == nil || !finite(lifetime) || lifetime < 0 || !finite(driftEast) || !finite(driftNorth) || len(earlier) > 64 {
		return nil, errors.New("oceantex: invalid foam history, lifetime or drift")
	}
	last := 0.0
	for _, frame := range earlier {
		if frame.Field == nil || !finite(frame.Age) || frame.Age <= last {
			return nil, errors.New("oceantex: foam history needs surfaces with strictly increasing positive ages")
		}
		last = frame.Age
	}
	return &FoamTrace{current: current, earlier: append([]FoamFrame(nil), earlier...),
		lifetime: lifetime, east: driftEast, north: driftNorth}, nil
}

// Sample returns current wave height and the current or surviving prior foam
// fraction. The boolean is false for an invalid or unavailable current point.
func (trace *FoamTrace) Sample(east, north float64) (height, foam float64, ok bool) {
	if trace == nil || trace.current == nil || !finite(east) || !finite(north) {
		return 0, 0, false
	}
	height, foam, ok = trace.current.Sample(east, north)
	if !ok || !finite(height) || !finite(foam) || foam < 0 || foam > 1 {
		return 0, 0, false
	}
	if !trace.current.Wet(east, north) {
		return height, 0, true
	}
	if trace.lifetime == 0 {
		return height, foam, true
	}
	for _, frame := range trace.earlier {
		x := east - trace.east*frame.Age
		y := north - trace.north*frame.Age
		if !finite(x) || !finite(y) || !frame.Field.Wet(x, y) {
			continue
		}
		_, prior, found := frame.Field.Sample(x, y)
		if !found {
			continue
		}
		if !finite(prior) || prior < 0 || prior > 1 {
			return 0, 0, false
		}
		foam = math.Max(foam, prior*math.Exp(-frame.Age/trace.lifetime))
	}
	return height, foam, true
}
