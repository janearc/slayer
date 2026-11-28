package session

import (
	"fmt"

	"github.com/janearc/slayer/globe"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/planet"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/generate"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/set"
)

func hasFlowingBody(read set.Set) bool {
	world, has := read.World()
	if !has {
		return false
	}
	for _, body := range world.Bodies() {
		if _, has := body.Flowing(); has {
			return true
		}
	}
	return false
}

// flowingBodies stages generated spectral giants once the camera is known.
// The first bridge supports fixed bodies in metre units and one solar lamp.
func (making *staging) flowingBodies(read set.Set, eye maths.Vec) error {
	world, has := read.World()
	if !has {
		return nil
	}
	for _, each := range world.Bodies() {
		appearance, flowing := each.Flowing()
		if !flowing {
			continue
		}
		if world.Unit() != 1 || len(world.Binaries()) != 0 || len(world.Suns()) != 0 || len(world.Lamps()) != 0 || len(world.Spots()) != 0 || len(world.Floods()) != 0 || len(making.world.Lamps) != 1 {
			return fmt.Errorf("%s: flowing giant requires metre units and one scene sun, without other lights", each.Name())
		}
		if !read.InSpace() || read.Light() != "sun" {
			return fmt.Errorf("%s: flowing giant requires a scene in space lit by the sun", each.Name())
		}
		if (appearance.ShadowFill() > 0 || appearance.SunDiscDegrees() > 0) && len(world.Bodies()) < 2 {
			return fmt.Errorf("%s: flowing shadow fill or sun disc requires multiple bodies", each.Name())
		}
		if _, legacy := read.Planet(); legacy {
			return fmt.Errorf("%s: flowing giant cannot yet share a scene with a legacy planet", each.Name())
		}
		input, _ := each.Generated()
		if input.GiantLook != nil || input.GroundLook != nil || input.Clouds != nil || input.Sun.Luminosity != 1 {
			return fmt.Errorf("%s: flowing giant requires a solar-luminosity sun and no other appearance override", each.Name())
		}
		derived, err := generate.Derive(input)
		if err != nil {
			return err
		}
		if derived.Kind != bodies.Giant {
			return fmt.Errorf("%s: flowing appearance requires a giant", each.Name())
		}
		length, err := input.Distance()
		distance := length.InMetres()
		if err != nil || distance <= 0 {
			return fmt.Errorf("%s: flowing giant needs a positive distance to its sun", each.Name())
		}
		radius := derived.Radius.InKilometres() * 1000
		appearance = appearance.GeneratedSize(each.Name(), radius, derived.Gravity, distance/generate.AstronomicalUnit)
		written, _ := appearance.Weather()
		pace, turning := written.Film()
		if pace != 0 || turning != 0 {
			return fmt.Errorf("%s: flowing giant film weather is not supported yet", each.Name())
		}
		body, _, err := making.body(appearance)
		if err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
		middle := each.Middle().Add(making.rebasing(read))
		from := eye.Sub(middle)
		body, err = body.Seen(from, rayWide(read, making.given))
		if err != nil {
			return fmt.Errorf("%s: %w", each.Name(), err)
		}
		sun, hasSun := read.Sunlit()
		if !hasSun {
			return fmt.Errorf("%s: flowing giant needs a sun direction", each.Name())
		}
		tilt := input.Tilt.InRadians()
		if override, hasTilt := each.Tilt(); hasTilt {
			tilt = override * maths.Degree
		}
		turn := maths.Rotation(maths.XAxis, tilt)
		pole, prime := turn.Out(maths.Up), turn.Out(maths.ZAxis)
		face := planet.NewFaceFixed(body, pole, prime, skyward(sun.Where()), from,
			starlight(distance/generate.AstronomicalUnit), making.given.exposure)
		ball, err := globe.New(radius)
		if err != nil {
			return err
		}
		pixel := rayWide(read, making.given) * float64(making.given.samples)
		shine := charted(face, appearance.Chart(), radius, from.Len(), pixel, making.given.onGPU)
		material := &thing.Material{Name: each.Name(), Shine: shine}
		if len(world.Bodies()) > 1 {
			direction := skyward(sun.Where())
			material.DirectSun = &direction
			material.DirectSunFill = appearance.ShadowFill()
			material.DirectSunSize = appearance.SunDiscDegrees() * maths.Degree
			material.DirectSunSamples = appearance.SunDiscSamples()
		}
		making.world.Add(thing.Thing{Shape: ball, Material: material, Motion: thing.Still(maths.Subspace{Origin: middle})})
		making.made.things++
		day, drag, relax := written.Turning()
		rate, wide, strength := written.Storms()
		moment, every := written.Moment()
		making.given.departures = append(making.given.departures, fmt.Sprintf("made up: %s flowing atmosphere and weather: cells %d day %g drag %g relax %g storms rate %g width %g strength %g seed %d moment %g checkpoint %g; authored columns and zones", each.Name(), written.Grid(), day, drag, relax, rate, wide, strength, written.Seed(), moment, every))
		making.given.departures = append(making.given.departures, fmt.Sprintf("made up: %s flowing optics: %s", each.Name(), appearance.FlowingDescription()))
		if material.DirectSun == nil {
			making.given.departures = append(making.given.departures, fmt.Sprintf("%s: flowing face uses one precomputed sun and does not receive shadows from scene objects", each.Name()))
		} else {
			if material.DirectSunSize > 0 {
				making.given.departures = append(making.given.departures, fmt.Sprintf("%s: flowing face uses a central-direction prelit face with %d fixed distant uniform-disc visibility samples, diameter %g degrees; this approximates angular lighting and omits volumetric shadows", each.Name(), appearance.SunDiscSamples(), appearance.SunDiscDegrees()))
			} else {
				making.given.departures = append(making.given.departures, fmt.Sprintf("%s: flowing face uses one directional precomputed sun with opaque hard shadows; no penumbra or volumetric shadows", each.Name()))
			}
			if material.DirectSunFill > 0 {
				making.given.departures = append(making.given.departures, fmt.Sprintf("made up: %s shadow fill %g retains prelit face in blocked areas; not scattered light or penumbra", each.Name(), material.DirectSunFill))
			}
		}
	}
	return nil
}
