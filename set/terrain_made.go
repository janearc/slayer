package set

import yaml "gopkg.in/yaml.v3"

// the bounds of made ground's rules
const (
	// leastMade and mostMade bound a made depth, width, spacing or
	// standout, in metres: a millimetre, and a hundred metres, past any
	// survey a set reads. Picked, so that a typo is refused.
	leastMade = 0.001
	mostMade  = 100
)

// Made is ground made finer than a window's survey, wherever the set's
// camera resolves finer than it: a seed, and the processes that make it
// besides the roughness every made ground has, read from the survey. A
// process the set does not say is not made.
type Made struct {
	seed float64
	// steep is the rise over run past which ground is a face that flutes
	// and beds work on, and steeped whether the set says it.
	steep   float64
	steeped bool
	// flutes are channels cut down a face, as deep and as far apart as
	// said; beds are ledges standing out of it, as far apart in height
	// and standing out as far as said.
	fluteDepth, fluteWidth  float64
	bedSpacing, bedStandout float64
	fluted, bedded          bool
}

// Made is the ground the set asks made finer than its survey, and
// whether it asks any.
func (ground Terrain) Made() (Made, bool) {
	if ground.made == nil {
		return Made{}, false
	}
	return *ground.made, true
}

// Seed makes the same ground each time.
func (made Made) Seed() uint64 { return uint64(made.seed) }

// Steep is the rise over run past which ground is a face for flutes and
// beds, and whether the set says it.
func (made Made) Steep() (float64, bool) { return made.steep, made.steeped }

// Flutes are how deep a channel is cut and how far apart channels are, in
// metres, and whether the set asks flutes.
func (made Made) Flutes() (depth, width float64, said bool) {
	return made.fluteDepth, made.fluteWidth, made.fluted
}

// Beds are how far apart in height ledges are and how far each stands
// out, in metres, and whether the set asks beds.
func (made Made) Beds() (spacing, standout float64, said bool) {
	return made.bedSpacing, made.bedStandout, made.bedded
}

// made is a step that reads the ground a terrain asks made: its seed,
// always said, and its steepness, flutes and beds, each if said.
func (reading reader) made(ground *Terrain) step {
	return func(_, value *yaml.Node) error {
		made := &Made{}
		ground.made = made
		wrote := map[string]bool{}
		err := reading.keys(value, map[string]step{
			"seed": reading.whole(&made.seed, leastDetail, mostSeed),
			"steep": reading.noted(wrote,
				reading.number(&made.steep, flattest, steepest)),
			"flutes": reading.noted(wrote, reading.pair("a flutes",
				"depth", "width", &made.fluteDepth, &made.fluteWidth)),
			"beds": reading.noted(wrote, reading.pair("a beds",
				"spacing", "standout", &made.bedSpacing,
				&made.bedStandout)),
		}, nil)
		if err == nil {
			err = reading.all(value, "made ground", "seed")
		}
		made.steeped, made.fluted, made.bedded = wrote["steep"],
			wrote["flutes"], wrote["beds"]
		return err
	}
}

// pair is a step that reads a block of two lengths in metres, both said.
func (reading reader) pair(what, first, second string,
	one, other *float64) step {
	return func(_, value *yaml.Node) error {
		err := reading.keys(value, map[string]step{
			first:  reading.number(one, leastMade, mostMade),
			second: reading.number(other, leastMade, mostMade),
		}, nil)
		if err == nil {
			err = reading.all(value, what, first, second)
		}
		return err
	}
}

// terrainMade refuses made ground with a detail, which makes ground below
// the map another way, and with a near window, which it would lie over.
func (reading reader) terrainMade(at *yaml.Node, ground *Terrain) error {
	switch {
	case ground.made == nil:
		return nil
	case ground.detailed:
		return reading.refuse(at, "made ground and detail each make the "+
			"ground below the map; a terrain says one")
	case ground.near != nil:
		return reading.refuse(at, "made ground is laid over the whole "+
			"window, so a terrain with a near window does not say it")
	}
	return nil
}
