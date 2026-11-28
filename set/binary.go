package set

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/generate"
	yaml "gopkg.in/yaml.v3"
)

// A Pair is two stars generated from a binary's numbers, going round
// their centre of mass, which stands at its middle, at a time after the
// stars' nearest approach. The set says each star's numbers and the
// orbit; the stage draws both stars as glowing balls, each a lamp at its
// place.
//
// A pair with a lava drop may say where the drop stands, in place of its
// own middle: the middle is then worked back from the drop's place at the
// pair's time, so a world can be made about the drop, which is where a
// camera of fine detail wants its numbers small.
type Pair struct {
	name      string
	generated *generate.Binary
	middle    [3]float64
	dropAt    *[3]float64
	time      float64
}

// Name is the pair's name.
func (made Pair) Name() string { return made.name }

// Generated is the binary the pair is generated from.
func (made Pair) Generated() generate.Binary { return *made.generated }

// Middle is the place of the pair's centre of mass, in the world's unit,
// as the set says it. A pair that says where its drop stands has none
// said: see Placed.
func (made Pair) Middle() maths.Vec { return fromMap(made.middle) }

// Placed is the pair as it stands at its time, and the place of its
// centre of mass in the world's unit: the one the set says, or, for a pair
// that says where its drop stands, the one that puts the drop there. It
// refuses what the generator refuses.
func (made Pair) Placed(unit float64) (generate.Pair, maths.Vec, error) {
	placed, err := made.generated.Place(generate.Years(made.time))
	if err != nil || made.dropAt == nil {
		return placed, made.Middle(), err
	}
	return placed, fromMap(*made.dropAt).Sub(placed.Drop.At.Scale(1 / unit)),
		nil
}

// Time is how long after periapsis the stars are drawn, in years.
func (made Pair) Time() generate.Years { return generate.Years(made.time) }

// Binaries is every pair of stars, in the order written.
func (made World) Binaries() []Pair { return append([]Pair(nil), made.pairs...) }

// The most a set may say of a generated binary.
const (
	// heaviestStar is in the Sun's masses, past the heaviest stars known
	heaviestStar = heaviestSun
	// widestPair is in astronomical units, past any bound pair's orbit
	widestPair = 1e6
	// hottestStar is in kelvin, past any white dwarf's surface
	hottestStar = 2e5
	// largestStar is in the Sun's radii, past the largest red supergiant
	largestStar = 2000
	// mostEccentric is the eccentricity the generator takes at most
	mostEccentric = 0.99
	// longestPairTime is in years, past any pair's age
	longestPairTime = oldest
)

// pairSaid is a binary's numbers as a set says them, in the units a
// person says them in: masses in the Sun's, ages in years, temperatures
// in kelvin, radii in the Sun's, the separation in astronomical units,
// angles in degrees.
type pairSaid struct {
	primaryMass, primaryAge                       float64
	companionMass, companionHeat, companionRadius float64
	separation, eccentricity, inclination         float64
	drop                                          dropSaid
}

// dropSaid is a lava drop's numbers as a set says them: its mass in the
// Sun's, its temperature in kelvin, its age in years, its distance from
// the pair's centre of mass in astronomical units and its phase in
// degrees, and whether it says that what is made up in them is.
type dropSaid struct {
	mass, heat, age, distance, phase float64
	madeUp, saidMadeUp               bool
}

// pairs reads each pair of stars, by its name.
func (reading reader) pairs(_, value *yaml.Node) error {
	return reading.each(value, func(name, block *yaml.Node) error {
		made := Pair{name: name.Value}
		var dropAt [3]float64
		err := reading.keys(block, map[string]step{
			"generate": reading.generatedPair(&made.generated, name.Value),
			"middle":   reading.three(&made.middle, -farthest, farthest),
			"drop_at":  reading.three(&dropAt, -farthest, farthest),
			"time":     reading.number(&made.time, 0, longestPairTime),
		}, nil)
		if err == nil {
			err = reading.placedPair(block, &made, dropAt)
		}
		reading.set.world.pairs = append(reading.set.world.pairs, made)
		return err
	})
}

// placedPair checks where a pair stands: it says its generate and its
// time, and either its middle, or where its drop is, which it must have.
func (reading reader) placedPair(block *yaml.Node, made *Pair,
	dropAt [3]float64) error {
	if err := reading.all(block, "a pair of stars", "generate",
		"time"); err != nil {
		return err
	}
	if says(block, "middle") == says(block, "drop_at") {
		return reading.refuse(block, "a pair of stars says its middle or "+
			"where its drop is, one of the two")
	}
	if says(block, "drop_at") {
		if made.generated == nil || made.generated.Drop == nil {
			return reading.refuse(block, "a pair says where its drop is "+
				"only if it has a drop")
		}
		made.dropAt = &dropAt
	}
	return nil
}

// generatedPair is a step that reads a generate block, a binary's
// numbers, into target: a primary by its mass and age, a companion by its
// mass, temperature and radius, and the orbit by its separation,
// eccentricity and inclination.
func (reading reader) generatedPair(target **generate.Binary,
	name string) step {
	return func(_, value *yaml.Node) error {
		var said pairSaid
		err := reading.keys(value, reading.pairKeys(&said), nil)
		if err == nil {
			err = reading.all(value, "a generated pair", "primary",
				"companion", "separation", "eccentricity")
		}
		if err == nil {
			err = reading.all(valueNode(value, "primary"), "a primary",
				"mass", "age")
		}
		if err == nil {
			err = reading.all(valueNode(value, "companion"), "a companion",
				"mass", "temperature", "radius")
		}
		if err == nil && says(value, "drop") {
			err = reading.droppable(value, said.drop)
		}
		*target = said.binary(name, says(value, "drop"))
		return err
	}
}

// droppable checks a lava drop a pair says: it says all its numbers, and
// where any of them is made up, as the generator finds them, it says
// made_up: true, so that a drop is never drawn with a number made up and
// not said.
func (reading reader) droppable(value *yaml.Node, said dropSaid) error {
	block := valueNode(value, "drop")
	err := reading.all(block, "a lava drop", "mass", "temperature", "age",
		"distance")
	if err != nil {
		return err
	}
	if lines := said.generated().MadeUp(); len(lines) > 0 &&
		!(said.saidMadeUp && said.madeUp) {
		return reading.refuse(block, "a lava drop says made_up: true "+
			"where its numbers are made up, and these are: %s", lines[0])
	}
	return nil
}

// pairKeys is the steps that read a pair's generate block into said.
func (reading reader) pairKeys(said *pairSaid) map[string]step {
	return map[string]step{
		"primary": reading.block(map[string]step{
			"mass": reading.number(&said.primaryMass, 0, heaviestStar),
			"age":  reading.number(&said.primaryAge, 0, oldest)}),
		"companion": reading.block(map[string]step{
			"mass": reading.number(&said.companionMass, 0, heaviestStar),
			"temperature": reading.number(&said.companionHeat, 0,
				hottestStar),
			"radius": reading.number(&said.companionRadius, 0,
				largestStar)}),
		"separation":   reading.number(&said.separation, 0, widestPair),
		"eccentricity": reading.number(&said.eccentricity, 0, mostEccentric),
		"inclination":  reading.number(&said.inclination, -180, 180),
		"drop": reading.block(map[string]step{
			"mass": reading.number(&said.drop.mass, 0, heaviestStar),
			"temperature": reading.number(&said.drop.heat, 0,
				hottestStar),
			"age":      reading.number(&said.drop.age, 0, oldest),
			"distance": reading.number(&said.drop.distance, 0, widestPair),
			"phase":    reading.number(&said.drop.phase, -360, 360),
			"made_up": reading.said(&said.drop.madeUp,
				&said.drop.saidMadeUp)}),
	}
}

// generated is the Drop a set's numbers say, in the units it is made in.
func (said dropSaid) generated() *generate.Drop {
	return &generate.Drop{Name: "the lava drop",
		Mass:        measure.SolarMasses(said.mass),
		Temperature: measure.Kelvin(said.heat), Age: generate.Years(said.age),
		Distance: measure.Metres(said.distance * generate.AstronomicalUnit),
		Phase:    measure.Degrees(said.phase)}
}

// binary is the Binary a set's numbers say, in the units it is made in,
// with the lava drop where the set says one.
func (said pairSaid) binary(name string, dropped bool) *generate.Binary {
	var drop *generate.Drop
	if dropped {
		drop = said.drop.generated()
	}
	return &generate.Binary{
		Drop: drop,
		Name: name,
		Primary: generate.Star{Name: name + " primary",
			Mass: measure.SolarMasses(said.primaryMass),
			Age:  generate.Years(said.primaryAge)},
		Companion: generate.Remnant{Name: name + " companion",
			Mass:        measure.SolarMasses(said.companionMass),
			Temperature: measure.Kelvin(said.companionHeat),
			Radius:      measure.SolarRadii(said.companionRadius)},
		Orbit: generate.Orbit{Distance: measure.Metres(said.separation *
			generate.AstronomicalUnit), Eccentricity: said.eccentricity,
			Inclination: measure.Degrees(said.inclination)},
	}
}
