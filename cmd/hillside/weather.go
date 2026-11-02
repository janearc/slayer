package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/weather"
)

// walls is how far from the equator the weather's grid reaches, north
// and south, in degrees: past it the cells crowd too close to keep.
const walls = 80.0

// weathered is the body with the set's weather as its flow, if the set
// gives it any: the weather reached at the set's moment by way of the
// cache, on the gpu where there is one.
func weathered(body planet.Body, made set.Planet,
	gases map[string]planet.Gas) (planet.Body, error) {
	body, err := lit(body, made, gases)
	if err != nil {
		return planet.Body{}, err
	}
	written, has := made.Weather()
	if !has {
		return body, nil
	}
	cells := written.Grid()
	grid, err := weather.NewGrid(cells, cells/2, walls*math.Pi/180,
		body.Radius())
	if err != nil {
		return planet.Body{}, err
	}
	rate, wide, strength := written.Storms()
	stirring, err := weather.NewStirring(rate, wide, strength)
	if err != nil {
		return planet.Body{}, err
	}
	day, drag, relax := written.Turning()
	sky, err := weather.New(grid, day, drag, relax, stirring, written.Seed())
	if err != nil {
		return planet.Body{}, err
	}
	dir, err := weatherCache()
	if err != nil {
		return planet.Body{}, err
	}
	moment, every := written.Moment()
	steps, err := sky.Reach(dir, moment, every, runtime.GOOS == "darwin")
	if err != nil {
		return planet.Body{}, err
	}
	fmt.Fprintf(os.Stderr, "weather: %.0f s reached, %d steps run\n",
		sky.Clock(), steps)
	return body.Flowing(sky), nil
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

// weatherCache is where weather is kept between shots: SLAYER_WEATHER,
// or the system's cache for this user.
func weatherCache() (string, error) {
	if dir := os.Getenv("SLAYER_WEATHER"); dir != "" {
		return dir, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "slayer", "weather"), nil
}
