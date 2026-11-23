package session

import (
	"fmt"
	"math"
	"os"
	"runtime"

	"github.com/janearc/slayer/diskcache"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/weather"
)

// walls is how far from the equator the weather's grid reaches, north
// and south, in degrees: past it the cells crowd too close to keep.
const walls = 80.0

// weathered is the body with the set's weather as its flow, if the set
// gives it any: the weather reached at the set's moment by way of the
// cache, on the gpu where there is one. The weather is given back too, or
// nothing where the set has none, so a film can carry it on.
func weathered(body planet.Body, made set.Planet,
	gases map[string]planet.Gas) (planet.Body, *weather.Weather, error) {
	body, err := lit(body, made, gases)
	if err != nil {
		return planet.Body{}, nil, err
	}
	written, has := made.Weather()
	if !has {
		return body, nil, nil
	}
	sky, err := stillAir(written, body.Radius())
	if err != nil {
		return planet.Body{}, nil, err
	}
	moment, _ := written.Moment()
	if err := reachedBy(sky, written, moment); err != nil {
		return planet.Body{}, nil, err
	}
	return body.Flowing(sky), sky, nil
}

// stillAir is a set's weather before it has run, on a planet of a radius
// in metres.
func stillAir(written set.Weather, radius float64) (*weather.Weather,
	error) {
	cells := written.Grid()
	grid, err := weather.NewGrid(cells, cells/2, walls*math.Pi/180, radius)
	if err != nil {
		return nil, err
	}
	rate, wide, strength := written.Storms()
	stirring, err := weather.NewStirring(rate, wide, strength)
	if err != nil {
		return nil, err
	}
	day, drag, relax := written.Turning()
	return weather.New(grid, day, drag, relax, stirring, written.Seed())
}

// reachedBy carries a set's weather to a moment, in seconds since it
// began, by way of the cache, on the gpu where there is one, and says so.
func reachedBy(sky *weather.Weather, written set.Weather,
	moment float64) error {
	dir, err := weatherCache(os.LookupEnv)
	if err != nil {
		return err
	}
	_, every := written.Moment()
	steps, err := sky.Reach(dir, moment, every, runtime.GOOS == "darwin")
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "weather: %.0f s reached, %d steps run\n",
		sky.Clock(), steps)
	return nil
}

// lit is the body with the heat, the cooking and the airglow the set
// gives it, each if it says.
func lit(body planet.Body, made set.Planet,
	gases map[string]planet.Gas) (planet.Body, error) {
	var err error
	if top, hottest, has := made.Heat(); has {
		if body, err = body.Heated(top, hottest); err != nil {
			return planet.Body{}, err
		}
	}
	if name, seconds, has := made.Cooking(); has {
		column, err := columnOf(made.Column(name), gases)
		if err != nil {
			return planet.Body{}, err
		}
		if body, err = body.Cooked(column, seconds); err != nil {
			return planet.Body{}, err
		}
	}
	if radiance, band, fronts, has := made.Airglow(); has {
		if body, err = body.Airglowing(radiance, band, fronts); err != nil {
			return planet.Body{}, err
		}
	}
	return body, nil
}

// weatherCache is where weather is kept between shots: the weather
// folder under the cache home, which the lookup says by SLAYER_CACHE.
func weatherCache(lookup diskcache.Lookup) (string, error) {
	return weather.CacheDir(lookup)
}
