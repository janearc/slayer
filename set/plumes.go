package set

import (
	"fmt"
	"slices"

	yaml "gopkg.in/yaml.v3"
)

// Plume is smoke or steam from a mouth carried off by the set's wind: a
// stack on a boat, a cooking fire's flue, a vent in a street. It leaves
// from an offset of a named thing's origin, or of the scene's, and may be
// many mouths laid by a seed round that point. What it leaves as and into
// is said: the mouth's radius, the gas's speed and heat and the air's, and
// what its particles are and how fast they leave.
type Plume struct {
	name, object  string
	x, y, z       float64
	count, spread float64
	seed          float64
	// the mouth and the air it leaves into
	mouth, exit, hot, air float64
	// the particles: grams a second, square metres a gram, the share
	// scattered, the lobe, and the Angstrom exponent
	rate, extinction, albedo, asymmetry, angstrom float64
	// how far down the wind it is drawn, and the lid's scene height
	reach, lid float64
	saysLid    bool
	// source is where the plume's numbers come from, in the set's words
	source string
}

// Plumes is the set's plumes in the order the file gives them. The slice
// belongs to the caller.
func (made Set) Plumes() []Plume { return slices.Clone(made.plumes) }

// Name is the plume's name in the set.
func (plume Plume) Name() string { return plume.name }

// Origin is an optional thing and the offset east, up and south of it, in
// metres, of the plume's mouth, or of the middle its mouths are laid round.
func (plume Plume) Origin() (string, float64, float64, float64) {
	return plume.object, plume.x, plume.y, plume.z
}

// Mouths is how many mouths the plume has, the radius in metres they are
// laid within round its origin, and the seed that lays them.
func (plume Plume) Mouths() (int, float64, uint64) {
	return int(plume.count), plume.spread, uint64(plume.seed)
}

// Stack is a mouth's radius in metres, the speed the gas leaves it in
// metres a second, the gas's temperature there and the air's, in kelvin.
func (plume Plume) Stack() (mouth, exit, hot, air float64) {
	return plume.mouth, plume.exit, plume.hot, plume.air
}

// Particles are the grams a second a mouth gives off, their mass
// extinction in square metres a gram at 550 nm, the share of it they
// scatter, their Henyey-Greenstein asymmetry and their Angstrom exponent.
func (plume Plume) Particles() (rate, extinction, albedo, asymmetry,
	angstrom float64) {
	return plume.rate, plume.extinction, plume.albedo, plume.asymmetry,
		plume.angstrom
}

// Reach is how far down the wind the plume is drawn, in metres.
func (plume Plume) Reach() float64 { return plume.reach }

// Lid is the scene height in metres the plume cannot rise past, and
// whether the plume says one.
func (plume Plume) Lid() (float64, bool) { return plume.lid, plume.saysLid }

// mostMouths is the most mouths one plume lays. Picked, as a bound.
const mostMouths = 512

// plumes reads the named plumes in the file's order.
func (reading reader) plumes(_, node *yaml.Node) error {
	return reading.each(node, reading.plume)
}

// plume reads one plume, every part of what leaves said.
func (reading reader) plume(name, block *yaml.Node) error {
	plume := Plume{name: name.Value, count: 1}
	err := reading.keys(block, reading.plumeSteps(&plume), nil)
	if err != nil {
		return err
	}
	if err = reading.all(block, "plume", "from", "mouth", "exit",
		"temperature", "air-temperature", "rate", "mass-extinction",
		"albedo", "asymmetry", "reach", "source"); err != nil {
		return err
	}
	if err = reading.all(valueNode(block, "from"), "a plume's from",
		"east", "up", "south"); err != nil {
		return err
	}
	if plume.count > 1 && !says(block, "spread") {
		return reading.refuse(block, "a plume of many mouths says the "+
			"spread they are laid in")
	}
	if !(plume.hot > plume.air) {
		return reading.refuse(block, "a plume's gas leaves hotter than "+
			"the air it leaves into")
	}
	plume.saysLid = says(block, "lid")
	reading.set.plumes = append(reading.set.plumes, plume)
	return nil
}

// plumeSteps are the keys a plume says.
func (reading reader) plumeSteps(plume *Plume) map[string]step {
	return map[string]step{
		"from": reading.block(map[string]step{
			"object": reading.text(&plume.object),
			"east":   reading.quantity(&plume.x, "m", -farthest, farthest),
			"up":     reading.quantity(&plume.y, "m", -farthest, farthest),
			"south":  reading.quantity(&plume.z, "m", -farthest, farthest),
		}),
		"count":  reading.whole(&plume.count, 1, mostMouths),
		"spread": reading.quantity(&plume.spread, "m", 0, 1e5),
		"seed":   reading.whole(&plume.seed, 0, mostSeed),
		"mouth":  reading.quantity(&plume.mouth, "m", 0.001, 100),
		"exit":   reading.quantity(&plume.exit, "m/s", 0, 1000),
		"temperature": reading.quantity(&plume.hot, "K", 1,
			1e4),
		"air-temperature": reading.quantity(&plume.air, "K", 1, 1e4),
		"rate":            reading.quantity(&plume.rate, "g/s", 1e-9, 1e9),
		"mass-extinction": reading.quantity(&plume.extinction,
			"m^2/g", 0, 1e6),
		"albedo":    reading.number(&plume.albedo, 0, 1),
		"asymmetry": reading.number(&plume.asymmetry, -mistLobe, mistLobe),
		"angstrom":  reading.number(&plume.angstrom, 0, mostAngstrom),
		"reach":     reading.quantity(&plume.reach, "m", 1, 1e5),
		"lid":       reading.quantity(&plume.lid, "m", 0, mostCloudHeight),
		"source":    reading.text(&plume.source),
	}
}

// foundPlumes checks each plume's thing is in the set and that the set
// says a wind to carry them.
func (reading reader) foundPlumes() error {
	for _, plume := range reading.set.plumes {
		if plume.object != "" && !reading.set.named(plume.object) {
			return fmt.Errorf("%s: plume %q names unknown object %q",
				reading.file, plume.name, plume.object)
		}
		if !(reading.set.windSpeed > 0) {
			return fmt.Errorf("%s: plume %q needs the weather's wind, "+
				"above nought, to carry it", reading.file, plume.name)
		}
	}
	return nil
}
