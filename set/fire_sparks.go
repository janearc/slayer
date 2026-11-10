package set

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/embers"
	"github.com/janearc/slayer/manufacture"
	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
	"time"
)

// fireSparks reads prescribed transport independently of fuel combustion.
func (reading reader) fireSparks(spec *manufacture.BurnSpec) step {
	return func(_, node *yaml.Node) error {
		var rate, life, radius, height, launch, spread, updraft, drag, gravity, hot, cool float64
		err := reading.keys(node, map[string]step{
			"rate":            reading.quantity(&rate, "s^-1", 0.001, 1e6),
			"lifetime":        reading.quantity(&life, "s", 0.001, 3600),
			"radius":          reading.quantity(&radius, "m", 0.0001, 1),
			"release-height":  reading.quantity(&height, "m", 0, 1e4),
			"launch-speed":    reading.quantity(&launch, "m/s", 0, 1000),
			"spread":          reading.quantity(&spread, "m/s", 0, 1000),
			"updraft":         reading.quantity(&updraft, "m/s", 0, 1000),
			"drag-time":       reading.quantity(&drag, "s", 0.001, 1000),
			"gravity":         reading.quantity(&gravity, "m/s^2", 0, 100),
			"temperature":     reading.quantity(&hot, "K", 1, 1e4),
			"end-temperature": reading.quantity(&cool, "K", 1, 1e4),
		}, nil)
		if err != nil {
			return err
		}
		if err = reading.all(node, "sparks", "rate", "lifetime", "radius", "release-height",
			"launch-speed", "spread", "updraft", "drag-time", "gravity", "temperature", "end-temperature"); err != nil {
			return err
		}
		sparks := embers.Spec{Rate: rate, Life: time.Duration(life * float64(time.Second)),
			Drag: time.Duration(drag * float64(time.Second)), Radius: measure.Metres(radius),
			Height: measure.Metres(height), Velocity: maths.V(0, launch, 0),
			Jitter: maths.V(spread, spread, spread), Air: maths.V(0, updraft, 0), Gravity: gravity,
			Hot: measure.Kelvin(hot), Cool: measure.Kelvin(cool)}
		if _, err = embers.New(sparks); err != nil {
			return reading.refuse(node, "%v", err)
		}
		spec.Sparks = &sparks
		return nil
	}
}
