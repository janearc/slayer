package hillside

import (
	"fmt"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/city"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
)

// halved adds the set's world half to the world: its ocean sheet, boxes,
// spheres, lamps, spots, floods, suns, bodies, wisps and fizzes, built as the
// stage builds them, in the frame of the set's things.
//
// the world half says its places as a map does, east, north and up in
// metres, which the set reader turns to the world's space as x east, y up
// and z south: (east, up, minus north).
//
// a thing the set stands at an east and a north is at (east minus the
// window's west, the ground's height minus its lowest point, the window's
// north minus north). so a place of the world half lands in the same
// frame when moved by (minus the window's west, minus the lowest point,
// the window's north). a set on no terrain has no window, and its world
// half is not moved.
//
// a world half has its own lamps, which are added to the set's light and
// to the tally. its sky is not hung over the sky the set's light made.
func (making *staging) halved(read set.Set) error {
	said, has := read.World()
	if !has {
		return nil
	}
	if _, hasOcean := said.OceanSurface(); hasOcean && said.Unit() != 1 {
		return fmt.Errorf("%s: a terrain-backed ocean surface requires world unit 1 (metres); got %g",
			making.given.set, said.Unit())
	}
	before := len(making.world.Lamps)
	if err := stage.Add(making.world, said, making.rebasing(read)); err != nil {
		return fmt.Errorf("%s: %w", making.given.set, err)
	}
	if basin, has := said.Griffith(); has {
		at := basin.At()
		position := maths.V(at.East(), at.Up(), -at.North()).Add(making.rebasing(read))
		options := city.ShotOptions{
			CityForward: basin.CityForward(), RidgeForward: basin.RidgeForward(), CityDetail: basin.CityDetail(),
			TerrainSamples: basin.TerrainSamples(), LightScale: basin.LightScale(), LightFootprint: basin.LightFootprint(),
		}
		if err := city.AddGriffithBasinWithOptions(making.world, position, basin.Heading(), basin.Scale(), basin.ReliefScale(), options); err != nil {
			return fmt.Errorf("%s: Griffith basin: %w", making.given.set, err)
		}
	}
	if ocean := making.world.OceanSurface; ocean.MeanPlaneSource() {
		tag := making.given.oceanSunTag
		if tag == nil || ocean.Boundary == nil || tag.sun < 0 ||
			tag.sun >= len(making.world.Lamps) {
			return fmt.Errorf("%s: ocean direct-sun mean-plane mode has no validated direct sun tag", making.given.set)
		}
		switch tag.route {
		case thing.OceanSunRouteGround:
			if tag.fill < 0 || tag.fill >= len(making.world.Lamps) ||
				tag.fill == tag.sun {
				return fmt.Errorf("%s: ground ocean direct-sun mode has no distinct validated fill tag", making.given.set)
			}
		case thing.OceanSunRouteGroundlessAir:
			if tag.fill != -1 {
				return fmt.Errorf("%s: groundless-air ocean direct-sun mode must not tag a synthetic fill", making.given.set)
			}
			if making.airedSky == nil || making.world.Sky != making.airedSky {
				return fmt.Errorf("%s: world half replaced the exact air-computed sky while staging the groundless-air ocean", making.given.set)
			}
		case thing.OceanSunRouteMoon:
			if !ocean.DirectMoonMeanPlane || tag.fill != -1 {
				return fmt.Errorf("%s: ocean Moon source has inconsistent tags", making.given.set)
			}
			if err := making.checkLunarSource(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: ocean direct-sun mode has an unsupported staging route", making.given.set)
		}
		ocean.SunRoute, ocean.SunLampIndex, ocean.FillLampIndex = tag.route,
			tag.sun, tag.fill
	}
	making.made.lamps += len(making.world.Lamps) - before
	return nil
}

// rebasing is how far the world half's places move to stand in the
// frame of the things on the ground: the window's west and lowest point
// taken off, its north put on. a set with no ground has no window, and its
// places stay where they are.
func (making *staging) rebasing(read set.Set) maths.Vec {
	if _, grounded := read.Ground(); !grounded {
		return unshifted
	}
	return maths.V(-making.west, -making.lowest, making.north)
}
