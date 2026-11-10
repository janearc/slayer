package set

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/firefront"
	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
	"math"
	"slices"
	"time"
)

// FireFront is a checked prescribed ignition plan over an existing stand.
type FireFront struct {
	name, stand, recipe, from                            string
	bearing, speed, start, duration, cell, height, smoke float64
	fade                                                 float64
}

// FireFronts returns copied front configurations in YAML order.
func (made Set) FireFronts() []FireFront { return slices.Clone(made.fronts) }

// Name identifies this ignition plan.
func (front FireFront) Name() string { return front.name }

// Stand names the placed fuel targets.
func (front FireFront) Stand() string { return front.stand }

// Recipe names a fire recipe with one flame patch per target.
func (front FireFront) Recipe() string { return front.recipe }

// From names the object on the front's starting plane.
func (front FireFront) From() string { return front.from }

// Motion supplies a horizontal plane in the scene's Y-up coordinates.
func (front FireFront) Motion(origin maths.Vec) firefront.Spec {
	angle := front.bearing * maths.Degree
	return firefront.Spec{Origin: origin, Direction: maths.V(math.Sin(angle), 0, -math.Cos(angle)),
		Speed: front.speed, Cell: front.cell, Start: time.Duration(front.start * float64(time.Second)),
		Duration: time.Duration(front.duration * float64(time.Second))}
}

// IgnitionHeight is an authored offset above each target's ground location.
func (front FireFront) IgnitionHeight() measure.Length { return measure.Metres(front.height) }

// SmokeRadius is the authored footprint of a shared aggregate smoke column.
func (front FireFront) SmokeRadius() measure.Length { return measure.Metres(front.smoke) }

// fireFronts reads unit-bearing authored motion over named fuel stands.
func (reading reader) fireFronts(_, node *yaml.Node) error {
	return reading.each(node, func(name, block *yaml.Node) error {
		front := FireFront{name: name.Value}
		err := reading.keys(block, map[string]step{
			"stand": reading.text(&front.stand), "recipe": reading.text(&front.recipe), "from": reading.text(&front.from),
			"bearing":         reading.quantity(&front.bearing, "deg", 0, 360),
			"speed":           reading.quantity(&front.speed, "m/s", .000001, 1000),
			"starts":          reading.quantity(&front.start, "s", 0, 86400),
			"fade-time":       reading.quantity(&front.fade, "s", 0, 43200),
			"burn-duration":   reading.quantity(&front.duration, "s", .001, 86400),
			"group-width":     reading.quantity(&front.cell, "m", .01, 1e4),
			"ignition-height": reading.quantity(&front.height, "m", 0, 1e4),
			"smoke-radius":    reading.quantity(&front.smoke, "m", .001, 1e4),
		}, nil)
		if err != nil {
			return err
		}
		if err = reading.all(block, "fire front", "stand", "recipe", "from", "bearing", "speed", "starts", "burn-duration", "fade-time", "group-width", "ignition-height"); err != nil {
			return err
		}
		reading.set.fronts = append(reading.set.fronts, front)
		return nil
	})
}

// foundFronts resolves references and model choices after all keys are read.
func (reading reader) foundFronts() error {
	if len(reading.set.fronts) > 8 {
		return reading.refuse(&yaml.Node{}, "at most eight fire fronts are supported")
	}
	for _, front := range reading.set.fronts {
		if front.fade > front.duration/2 {
			return reading.refuse(&yaml.Node{}, "front %s fade exceeds half its burn duration", front.name)
		}
		recipe, ok := reading.set.Recipe(front.recipe)
		spec, fire := recipe.Fire()
		if !ok || !fire {
			return reading.refuse(&yaml.Node{}, "front %s needs a known fire recipe", front.name)
		}
		if spec.Layout.Pattern != "" {
			return reading.refuse(&yaml.Node{}, "front %s uses one flame patch per target; omit recipe layout", front.name)
		}
		if (spec.Smoke != nil) != (front.smoke > 0) {
			return reading.refuse(&yaml.Node{}, "front %s needs smoke-radius exactly when its recipe has smoke", front.name)
		}
		fuel := false
		for _, stand := range reading.set.stands {
			if stand.Name() == front.stand {
				fuel = stand.kind != "fire" && stand.alight == 0 && stand.count > 0
			}
		}
		if !fuel {
			return reading.refuse(&yaml.Node{}, "front %s needs an existing unlit non-fire stand", front.name)
		}
		if !reading.set.named(front.from) {
			return reading.refuse(&yaml.Node{}, "front %s starts from unknown object %s", front.name, front.from)
		}
	}
	return nil
}

// FadeTime smooths occupancy over the authored ignition and burnout intervals.
func (front FireFront) FadeTime() time.Duration {
	return time.Duration(front.fade * float64(time.Second))
}
