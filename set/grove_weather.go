package set

import yaml "gopkg.in/yaml.v3"

// Mist is a prescribed appearance field, not a transpiration model. A mist
// of kind prescribed-cloud is a deck of separate clouds between a base and
// a top in place of a layer over the ground.
type Mist struct {
	extinction, depth, coverage, seed float64
	asymmetry                         float64
	// angstrom is the particles' Angstrom exponent, where said; nitrogen
	// is the nitrogen dioxide mixed through it, parts in a thousand
	// million; edge is how far under its top it holds its density.
	angstrom, nitrogen, edge float64
	saysAngstrom             bool
	// size is the size of its patches, nought for its depth; a cloud's
	// base and top are scene heights.
	size, base, top float64
	cloud           bool
}

// Extinction is the prescribed peak extinction per metre.
func (mist Mist) Extinction() float64 { return mist.extinction }

// Depth is the prescribed layer thickness above ground, in metres.
func (mist Mist) Depth() float64 { return mist.depth }

// Coverage controls noise thresholds, not measured cloud cover.
func (mist Mist) Coverage() float64 { return mist.coverage }

// Asymmetry is the Henyey-Greenstein asymmetry of the mist's scattering:
// nought, or unsaid, scatters evenly every way, as before it could be
// said; toward one sends the light on forward, as droplets do.
func (mist Mist) Asymmetry() float64 { return mist.asymmetry }

// Angstrom is the particles' Angstrom exponent, and whether the mist says
// one: unsaid, it is as thick to every colour.
func (mist Mist) Angstrom() (float64, bool) {
	return mist.angstrom, mist.saysAngstrom
}

// NitrogenDioxide is the nitrogen dioxide mixed through the mist where it
// is thickest, in parts in a thousand million: nought for none.
func (mist Mist) NitrogenDioxide() float64 { return mist.nitrogen }

// Edge is how far under its top the mist holds its whole density before it
// thins to nothing, as air under a lid does: nought thins it from the
// bottom up.
func (mist Mist) Edge() float64 { return mist.edge }

// Size is the size of the mist's patches in metres, nought for its depth.
func (mist Mist) Size() float64 { return mist.size }

// Cloud is a cloud deck's base and top, heights in scene metres, and
// whether the mist is a cloud deck.
func (mist Mist) Cloud() (base, top float64, cloud bool) {
	return mist.base, mist.top, mist.cloud
}

// mistLobe is how near one either way a mist's asymmetry may come. A
// Henyey-Greenstein lobe stays strictly inside one by its definition;
// picked short of it, fable-team 2026-11-12, where the lobe is a spike
// no march resolves.
const mistLobe = 0.99

// mostAngstrom is the greatest Angstrom exponent a mist takes, the most
// scene/medium's aerosol accepts.
const mostAngstrom = 2.5

// mostNitrogen is the most nitrogen dioxide a mist takes, in parts in a
// thousand million: a hundredth of the air. Picked, as a bound.
const mostNitrogen = 1e7

// mostCloudHeight is the highest a cloud deck's top may be, in metres:
// the top of the troposphere's weather, near enough. Picked, as a bound.
const mostCloudHeight = 20000

// Seed keeps weather independent of mesh construction draws.
func (mist Mist) Seed() uint64 { return uint64(mist.seed) }

// Mist finds a named prescribed field.
func (made Set) Mist(name string) (Mist, bool) { m, ok := made.weather[name]; return m, ok }

// Wind gives its source bearing in degrees and speed in m/s.
func (made Set) Wind() (from, speed float64) { return made.windFrom, made.windSpeed }

// weather reads named fields and the wind that moves them.
func (reading reader) weather(_, node *yaml.Node) error {
	return reading.each(node, func(name, block *yaml.Node) error {
		if name.Value == "wind" {
			return reading.keys(block, map[string]step{
				"from":  reading.quantity(&reading.set.windFrom, "deg", 0, 360),
				"speed": reading.quantity(&reading.set.windSpeed, "m/s", 0, 100)}, nil)
		}
		m := Mist{}
		var kind string
		err := reading.keys(block, reading.mistSteps(&m, &kind), nil)
		if err != nil {
			return err
		}
		if err = reading.mistSaid(block, kind, &m); err != nil {
			return err
		}
		reading.set.weather[name.Value] = m
		return nil
	})
}

// mistSteps are the keys a mist or a cloud deck may say.
func (reading reader) mistSteps(m *Mist, kind *string) map[string]step {
	return map[string]step{
		"kind": reading.word(kind, "prescribed-mist",
			"prescribed-cloud"),
		"extinction": reading.quantity(&m.extinction, "m^-1", 0, 100),
		"depth":      reading.quantity(&m.depth, "m", 0.1, 1000),
		"coverage":   reading.number(&m.coverage, 0, 1),
		"seed":       reading.whole(&m.seed, 0, mostSeed),
		"asymmetry": reading.number(&m.asymmetry, -mistLobe,
			mistLobe),
		"angstrom": reading.number(&m.angstrom, 0, mostAngstrom),
		"nitrogen-dioxide": reading.quantity(&m.nitrogen, "ppb", 0,
			mostNitrogen),
		"edge": reading.quantity(&m.edge, "m", 0, 1000),
		"size": reading.quantity(&m.size, "m", 1, 1e6),
		"base": reading.quantity(&m.base, "m", 0, mostCloudHeight),
		"top":  reading.quantity(&m.top, "m", 0, mostCloudHeight),
	}
}

// mistSaid checks what a mist or a cloud deck must say and may not.
func (reading reader) mistSaid(block *yaml.Node, kind string,
	m *Mist) error {
	m.saysAngstrom = says(block, "angstrom")
	if kind == "prescribed-cloud" {
		m.cloud = true
		if says(block, "depth") || says(block, "edge") {
			return reading.refuse(block, "a cloud deck says its base "+
				"and top, not a depth or an edge")
		}
		if err := reading.all(block, "cloud deck", "kind", "extinction",
			"base", "top", "coverage", "seed"); err != nil {
			return err
		}
		if !(m.top > m.base) || m.coverage == 0 {
			return reading.refuse(block, "a cloud deck's top is above "+
				"its base, and it covers some of the sky")
		}
		m.depth = m.top - m.base
		return nil
	}
	if says(block, "base") || says(block, "top") {
		return reading.refuse(block, "a mist says its depth; a base "+
			"and a top are a cloud deck's")
	}
	if err := reading.all(block, "mist", "kind", "extinction", "depth",
		"coverage", "seed"); err != nil {
		return err
	}
	if m.edge > m.depth {
		return reading.refuse(block, "a mist's edge is within its depth")
	}
	return nil
}
