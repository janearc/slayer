package main

import (
	"github.com/janearc/celestial/bands"
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
)

// lasers shares one world-space boresight across every launch. The original
// Gaussian beam and ordinary housing paths remain the existing CPU/GPU ones.
func (site *siteBuilder) lasers() error {
	lasers := site.plan.Lasers
	if lasers == nil {
		return nil
	}
	direction, err := site.frame.Compass(site.space.Origin, measure.Degrees(lasers.Azimuth), measure.Degrees(lasers.Altitude))
	if err != nil {
		return err
	}
	spectrum := bands.Lines(map[float64]float64{lasers.Wavelength: lasers.Strength})
	for _, offset := range lasers.Origins {
		world := site.space.Out(maths.V(offset[0], offset[1], offset[2]))
		origin, err := site.relative.Point(world)
		if err != nil {
			return err
		}
		point := emit.NewPoint(origin, direction, spectrum)
		beam, err := emit.NewBeam(point.When(0), measure.Metres(lasers.Radius), measure.Metres(lasers.Reach), measure.Degrees(lasers.Divergence))
		if err != nil {
			return err
		}
		site.world.Lamp(beam)
		if lasers.Launchers {
			value, err := site.load(prototypeKey{"guide-star-launcher", "quick", 0})
			if err != nil {
				return err
			}
			base := world.Sub(direction.Scale(value.bounds.Max.Y))
			if err := site.add(value, maths.NewSubspace(base, maths.Upright(direction), 1)); err != nil {
				return err
			}
		}
	}
	return nil
}
