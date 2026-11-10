package set

import yaml "gopkg.in/yaml.v3"

// Mist is a prescribed appearance field, not a transpiration model.
type Mist struct{ extinction, depth, coverage, seed float64 }

// Extinction is the prescribed peak extinction per metre.
func (mist Mist) Extinction() float64 { return mist.extinction }

// Depth is the prescribed layer thickness above ground, in metres.
func (mist Mist) Depth() float64 { return mist.depth }

// Coverage controls noise thresholds, not measured cloud cover.
func (mist Mist) Coverage() float64 { return mist.coverage }

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
		err := reading.keys(block, map[string]step{
			"kind":       reading.only("prescribed-mist"),
			"extinction": reading.quantity(&m.extinction, "m^-1", 0, 100),
			"depth":      reading.quantity(&m.depth, "m", 0.1, 1000),
			"coverage":   reading.number(&m.coverage, 0, 1),
			"seed":       reading.whole(&m.seed, 0, mostSeed)}, nil)
		if err != nil {
			return err
		}
		if err = reading.all(block, "mist", "kind", "extinction", "depth", "coverage", "seed"); err != nil {
			return err
		}
		reading.set.weather[name.Value] = m
		return nil
	})
}
