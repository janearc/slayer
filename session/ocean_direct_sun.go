package session

import (
	"fmt"
	"math"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

// oceanSunTag records the supported route's exact direct sun and, for ordinary
// ground only, its separate legacy directional fill.
type oceanSunTag struct {
	sun, fill int
	route     thing.OceanSunRoute
}

// tagOceanDirectSun binds the requested mean-plane mode to one supported,
// validated staging path. The direct source is captured by its insertion
// index; its actual type and direction are checked so a fill or custom lamp
// cannot be mistaken for the sun.
func (stage *staging) tagOceanDirectSun(read set.Set,
	firstLamp int) error {
	world, has := read.World()
	if !has {
		return nil
	}
	surface, has := world.OceanSurface()
	if !has || surface.DirectSunMode() != "mean-plane" {
		return nil
	}
	if stage.given.onGPU {
		return fmt.Errorf("%s: ocean direct-sun mean-plane mode is CPU-only; select renderer: cpu", stage.given.set)
	}
	if read.Light() != "sun" {
		return fmt.Errorf("%s: ocean direct-sun mean-plane mode requires light: sun", stage.given.set)
	}
	_, hasSun := read.Sunlit()
	if !hasSun {
		return fmt.Errorf("%s: ocean direct-sun mean-plane mode requires an authored sun direction", stage.given.set)
	}
	if firstLamp < 0 || firstLamp >= len(stage.world.Lamps) {
		return fmt.Errorf("%s: direct sun staging did not produce the expected sun lamp", stage.given.set)
	}
	_, grounded := read.Ground()
	_, groundlessAir := read.GroundlessAir()
	if read.InSpace() {
		return fmt.Errorf("%s: ocean direct-sun mean-plane mode does not support space lighting", stage.given.set)
	}
	if groundlessAir {
		if grounded || read.Air() != "earth" || !stage.groundlessAir {
			return fmt.Errorf("%s: ocean direct-sun mean-plane air route requires the staged groundless-air Earth path", stage.given.set)
		}
		if stage.airedSun == nil || stage.airedSun.index != firstLamp {
			return fmt.Errorf("%s: groundless-air direct sun was not recorded at its exact aired insertion index", stage.given.set)
		}
		if len(stage.world.Lamps) != firstLamp+1 {
			return fmt.Errorf("%s: groundless-air ocean path requires exactly its one aired direct sun; found %d lamps after insertion", stage.given.set, len(stage.world.Lamps)-firstLamp)
		}
		if stage.airedSky == nil || stage.world.Sky != stage.airedSky {
			return fmt.Errorf("%s: groundless-air ocean path requires the exact sky instance installed by aired", stage.given.set)
		}
		inserted, ok := stage.world.Lamps[firstLamp].(emit.Distant)
		if !ok || inserted != stage.airedSun.lamp {
			return fmt.Errorf("%s: groundless-air lamp %d is not the exact emit.Distant recorded by aired", stage.given.set, firstLamp)
		}
		if err := stage.expectedOceanSun(read, firstLamp); err != nil {
			return err
		}
		stage.given.oceanSunTag = &oceanSunTag{sun: firstLamp,
			fill: -1, route: thing.OceanSunRouteGroundlessAir}
		return nil
	}
	if !grounded || read.Air() != "" {
		return fmt.Errorf("%s: ocean direct-sun mean-plane mode requires the ordinary ground route or explicit groundless-air Earth route", stage.given.set)
	}
	if len(stage.world.Lamps) != firstLamp+2 {
		return fmt.Errorf("%s: ground ocean direct-sun path requires exactly its sun and separate legacy fill lamps; found %d lamps after insertion", stage.given.set, len(stage.world.Lamps)-firstLamp)
	}
	if err := stage.expectedOceanSun(read, firstLamp); err != nil {
		return err
	}
	fill, ok := stage.world.Lamps[firstLamp+1].(emit.Distant)
	if !ok {
		return fmt.Errorf("%s: direct sun staging expected the separate legacy fill at lamp %d, got %T", stage.given.set, firstLamp+1, stage.world.Lamps[firstLamp+1])
	}
	fillToward, fillFar, _ := fill.Light(maths.Vec{}, maths.Vec{})
	if !math.IsInf(fillFar, 1) || !fillToward.Near(fillWay.Unit(), 1e-14) {
		return fmt.Errorf("%s: lamp %d is not the expected separate legacy fill", stage.given.set, firstLamp+1)
	}
	stage.given.oceanSunTag = &oceanSunTag{sun: firstLamp,
		fill: firstLamp + 1, route: thing.OceanSunRouteGround}
	return nil
}

// expectedOceanSun checks that the lamp at index is the distant sun.
func (stage *staging) expectedOceanSun(read set.Set, index int) error {
	direct, ok := stage.world.Lamps[index].(emit.Distant)
	if !ok {
		return fmt.Errorf("%s: direct sun staging expected emit.Distant at lamp %d, got %T", stage.given.set, index, stage.world.Lamps[index])
	}
	toward, far, _ := direct.Light(maths.Vec{}, maths.Vec{})
	sun, _ := read.Sunlit()
	azimuth, altitude := sun.Where()
	expected := skyward(azimuth, altitude)
	if !math.IsInf(far, 1) || !toward.Near(expected, 1e-14) ||
		toward.Y <= 0 {
		return fmt.Errorf("%s: lamp %d is not the expected above-horizon set sun", stage.given.set, index)
	}
	return nil
}

// refuseWorldOnlyOceanSun keeps the mode out of world-only sets until their
// different `film.Sun` staging path has an equally explicit source tag.
func refuseWorldOnlyOceanSun(read set.Set, file string) error {
	world, has := read.World()
	if !has {
		return nil
	}
	surface, has := world.OceanSurface()
	if has && surface.DirectSunMode() == "mean-plane" {
		return fmt.Errorf("%s: ocean direct-sun mean-plane mode currently requires the ordinary ground sunlight path", file)
	}
	return nil
}
