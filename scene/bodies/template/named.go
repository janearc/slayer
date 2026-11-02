package template

import (
	"maps"
	"slices"

	"github.com/janearc/slayer/scene/bodies"
)

// named is every template by the name a file calls it: lower case, a
// dash between words.
var named = map[string]func() bodies.Body{
	"sol": Sol, "mercury": Mercury, "venus": Venus, "earth": Earth,
	"mars": Mars, "jupiter": Jupiter, "saturn": Saturn, "uranus": Uranus,
	"neptune": Neptune, "pluto": Pluto, "charon": Charon,
	"hale-bopp": HaleBopp, "hyakutake": Hyakutake, "magnetar": Magnetar,
	"pulsar": Pulsar, "quasar": Quasar, "hot-jupiter": HotJupiter,
}

// Named is the template of a name, its own copy to change, and whether
// there is one by that name.
func Named(name string) (bodies.Body, bool) {
	made, found := named[name]
	if !found {
		return bodies.Body{}, false
	}
	return made(), true
}

// Names is every template's name, in order.
func Names() []string { return slices.Sorted(maps.Keys(named)) }
