package manufacture

import (
	"math"
	"strconv"
	"time"

	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
)

// The Larson's lights, each by a name a scene can address it by:
//
//   - cabin, the red lamp under the dome;
//   - belly, a floodlight down from the belly disc, which landed lights
//     the ground, the legs and the stairs, and in flight whatever the
//     saucer hangs over;
//   - port-0 to port-11, the running lights: the portholes round the
//     upper slope, each one a lamp, numbered the way they go round. The
//     porthole's own glass glows as well, in saucerLooks, so a porthole
//     is seen lit and not only what it lights.
//
// Every number is picked, in saucer-choices.yaml.

// The names a scene addresses the saucer's lights by. cabinName is in
// saucer.go, with the cabin light.
const (
	bellyName = "belly"
	rimName   = "port-"
)

// rimLightName is the name of one running light, by its porthole's number
// round the hull.
func rimLightName(each int) string { return rimName + strconv.Itoa(each) }

// bellyLight is the floodlight under the belly disc: a cool white, in a
// wide cone.
func bellyLight() (light, error) {
	colour, err := newThermal(measure.Kelvin(saucerValue("belly-heat")))
	if err != nil {
		return light{}, err
	}
	over, err := newCone(measure.Degrees(saucerValue("belly-half")),
		measure.Degrees(saucerValue("belly-soft")))
	if err != nil {
		return light{}, err
	}
	return newLight(colour, measure.Watts(saucerValue("belly-power")), over)
}

// rimLight is one running light: a warm white, every way.
func rimLight() (light, error) {
	colour, err := newThermal(measure.Kelvin(saucerValue("rim-light-heat")))
	if err != nil {
		return light{}, err
	}
	return newLight(colour, measure.Watts(saucerValue("rim-light-power")),
		everyway{})
}

// lit is the saucer carrying every light it has: the cabin's, the
// belly's, and the running lights round the rim.
func (size saucerSize) lit(object good) (good, error) {
	beyond := measure.Metres(saucerValue("cabin-beyond"))
	cabin, err := cabinLight()
	if err != nil {
		return good{}, err
	}
	object = object.litAs(cabinName, cabin, maths.V(spaceZero,
		size.deck()+size.dome*saucerValue("cabin-up"), spaceZero),
		maths.Up, beyond)

	belly, err := bellyLight()
	if err != nil {
		return good{}, err
	}
	// the belly lamp keeps the programme too: lit when all are lit, and
	// low while the running lights chase
	belly, err = belly.signalling(programme(spaceZero, spaceOne, spaceZero))
	if err != nil {
		return good{}, err
	}
	object = object.litAs(bellyName, belly, maths.V(spaceZero,
		size.raised(-size.drop-saucerValue("belly-below")), spaceZero),
		maths.Up.Neg(), beyond)

	running, err := rimLight()
	if err != nil {
		return good{}, err
	}
	// each porthole is a running light: its lamp stands just proud of
	// the porthole's face, square to the slope, where ringPortholes put
	// the porthole itself
	across := size.radius + size.up*(size.dome-size.radius)
	height := size.raised(size.bowl + size.band + size.up*size.rise)
	normal := maths.V(size.rise, size.radius-size.dome, spaceZero).Unit()
	proud := size.depth + saucerValue("rim-light-out")
	for each := range size.portholes {
		angle := fullTurn * float64(each) / float64(size.portholes)
		out := maths.V(normal.X*math.Cos(angle), normal.Y,
			-normal.X*math.Sin(angle))
		centre := maths.V(across*math.Cos(angle), height,
			-across*math.Sin(angle))
		chasing, err := running.signalling(programme(each, size.portholes,
			int(saucerValue("lights-lit"))))
		if err != nil {
			return good{}, err
		}
		object = object.litAs(rimLightName(each), chasing,
			centre.Add(out.Scale(proud)), out, beyond)
	}
	return object, nil
}

// programme is a light's part in the Larson's programme, which every
// light of it but the cabin's keeps: for the first of each cycle they
// are all lit; then the running lights chase round the rim, so many at
// once, a lap at a time, and whatever is not lit is low. A running
// light has its place among the portholes. The belly lamp has the one
// place and is never one of the lit, so it is low for the whole chase.
func programme(place, places, lit int) signal {
	seconds := func(name string) time.Duration {
		return time.Duration(saucerValue(name) * float64(time.Second))
	}
	return signal{cycle: seconds("lights-cycle"),
		steady: seconds("lights-steady"), lap: seconds("lights-lap"),
		place: place, places: places, lit: lit,
		low: saucerValue("lights-low")}
}
