package stage

import (
	"math"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// SpaceSun reads the directional source used by the producer's space lighting.
// The producer's space route accepts the sun at any altitude.
func SpaceSun(read set.Set) (set.Sun, maths.Vec, bool) {
	sun, has := read.Sunlit()
	if !has || !read.InSpace() || read.Light() != "sun" {
		return set.Sun{}, maths.Vec{}, false
	}
	return sun, Skyward(sun.Where()), true
}

// Skyward converts degrees clockwise from north and up from the horizon to
// the world's east, up and south axes.
func Skyward(azimuth, altitude float64) maths.Vec {
	round, up := azimuth*maths.Degree, altitude*maths.Degree
	return maths.V(math.Sin(round)*math.Cos(up), math.Sin(up),
		-math.Cos(round)*math.Cos(up))
}
