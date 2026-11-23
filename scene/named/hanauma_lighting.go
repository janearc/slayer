package named

import (
	"fmt"
	"math"
	"time"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/daylight"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/scene/thing"
)

// hanaumaSolarLighting uses Hānauma's fixed modern Hawaii time zone.
// The sky and fill are deliberately simple authored lighting; the sun's
// direction comes from the supplied date and time.
func hanaumaSolarLighting(world *thing.World, held values) (bool, error) {
	year, month, day := held.whole("year"), held.whole("month"), held.whole("day")
	hawaii := time.FixedZone("HST", -10*60*60)
	at := time.Date(year, time.Month(month), day, held.whole("hour"), held.whole("minute"), 0, 0, hawaii)
	if at.Year() != year || int(at.Month()) != month || at.Day() != day {
		return false, fmt.Errorf("hanauma-bay: invalid calendar date")
	}
	position, err := daylight.Sun(21.2711, -157.6945, at)
	if err != nil {
		return false, err
	}
	if !position.Daylight {
		// This is a dim authored night sky and fill, not a calculated moon.
		world.Sky = sky.Gradient{
			Horizon: maths.RGB(.002, .003, .008),
			Zenith:  maths.RGB(.0005, .001, .004),
			Ground:  maths.RGB(.0002, .0005, .001),
		}
		fill, err := emit.NewDistant(maths.V(.28, .92, .27), maths.RGB(.003, .004, .008))
		if err != nil {
			return false, err
		}
		world.Lamp(fill)
		return true, nil
	}
	world.Sky = sky.Gradient{
		Horizon: maths.RGB(.17, .39, .70),
		Zenith:  maths.RGB(.026, .156, .49),
		Ground:  maths.RGB(.03, .11, .22),
	}
	azimuth, altitude := position.Azimuth*math.Pi/180, position.Altitude*math.Pi/180
	// Hānauma's X points east and Z points south.
	direction := maths.V(math.Sin(azimuth)*math.Cos(altitude),
		math.Sin(altitude), -math.Cos(azimuth)*math.Cos(altitude))
	sun, err := emit.NewDistant(direction, maths.RGB(1.7, 1.61, 1.47).Scale(maths.Smooth(0, 5, position.Altitude)))
	if err != nil {
		return false, err
	}
	fill, err := emit.NewDistant(maths.V(.28, .92, .27), maths.RGB(.19, .24, .36))
	if err != nil {
		return false, err
	}
	world.Lamp(sun)
	world.Lamp(fill)
	return false, nil
}

// The existing wet scene uses an emissive seabed approximation to represent
// daylight returning from shallow water. It must not glow after sunset.
func darkenHanaumaSeabed(world *thing.World) {
	for _, object := range world.Things {
		if object.Material != nil && object.Material.Name == "Hanauma sand, reef pavement and tuff" {
			material := *object.Material
			material.Shine = nil
			object.Material = &material
		}
	}
}
