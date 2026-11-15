package set

import yaml "gopkg.in/yaml.v3"

// A scene's flags say that its picture departs from the exact tracer's,
// on purpose, beside its renderer, as its fantasy block says the scene
// departs from the physics:
//
//	scene:
//	  vibes: on
//	  fastlight: on
//
// They are a job's flags (job/README.md, "flags and shortcuts"), the
// same names and words, and a job's stand over them. A set says no
// shortcuts. What a flag does is the renderer's; this package reads it.

// The words a flag takes. The flags' own names, vibes and fastlight,
// are written out where scene reads them, as every key of a set is.
const (
	flagOn  = "on"
	flagOff = "off"
)

// Flag is a flag's word in the set, whether it is on, and whether the
// set says it at all. A name that is no flag is unsaid.
func (made Set) Flag(name string) (on, said bool) {
	word, said := made.flags[name]
	return word == flagOn, said
}

// flag is a step that reads a flag of a name as on or off.
func (reading reader) flag(name string) step {
	return func(key, value *yaml.Node) error {
		var word string
		err := reading.word(&word, flagOn, flagOff)(key, value)
		if err != nil {
			return err
		}
		if reading.set.flags == nil {
			reading.set.flags = map[string]string{}
		}
		reading.set.flags[name] = word
		return nil
	}
}
