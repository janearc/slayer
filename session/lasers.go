package session

import (
	"fmt"

	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
)

// lasers adds static launches after object placement, before retaining the
// scene. Named origins use world offsets; they do not inherit a random yaw.
func (stage *staging) lasers(read set.Set) error {
	for _, laser := range read.Lasers() {
		object, east, up, south := laser.Origin()
		at := maths.V(east, up, south)
		if object != "" {
			origin, found := stage.where(object)
			if !found {
				return fmt.Errorf("laser %s: object %q "+
					"has no placed origin",
					laser.Name(), object)
			}
			at = at.Add(origin)
		}
		azimuth, altitude := laser.Pointing()
		wavelength, strength := laser.Spectrum()
		spectrum := bands.Lines(map[float64]float64{
			wavelength: strength,
		})
		direction := skyward(azimuth, altitude)
		if laser.Launcher() {
			if err := stage.launcher(at, direction); err != nil {
				return fmt.Errorf("laser %s: %w", laser.Name(), err)
			}
		}
		// the beam is the manufactory's laser, as every object of a set is
		// the manufactory's; what it carries and the lamp it becomes are
		// staged here, since the manufactory never calls emit.
		radius, reach, divergence := laser.Envelope()
		made := manufacture.Laser{At: at, Toward: direction,
			Radius: measure.Metres(radius), Reach: measure.Metres(reach),
			Divergence: measure.Degrees(divergence)}
		point := emit.NewPoint(made.At, made.Toward, spectrum)
		beam, err := emit.NewBeam(point.When(stage.given.at), made.Radius,
			made.Reach, made.Divergence)
		if err != nil {
			return fmt.Errorf("laser %s: %w", laser.Name(), err)
		}
		stage.world.Lamp(beam)
	}
	stage.made.lamps = len(stage.world.Lamps)
	return nil
}
