package stage

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/moonlight"
	"github.com/janearc/slayer/set"
)

// Moonward resolves a moon above the horizon to its direction and altitude.
func Moonward(moon set.Moon) (maths.Vec, measure.Angle, bool) {
	azimuth, altitude := moon.Where()
	if !(altitude > 0) {
		return maths.Vec{}, measure.Radians(0), false
	}
	return Skyward(azimuth, altitude), measure.Radians(altitude * maths.Degree), true
}

// DirectMoon constructs the distant lunar lamp through one air column.
// The sky table and optional disc are staged separately.
func DirectMoon(way maths.Vec, height measure.Length, rise measure.Angle, exposure float64) (emit.Distant, error) {
	arriving, err := moonlight.Full().Through(height, rise)
	if err != nil {
		return emit.Distant{}, err
	}
	return emit.NewDistant(way, arriving.Picture(exposure))
}
