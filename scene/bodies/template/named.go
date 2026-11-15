package template

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/janearc/slayer/scene/bodies"
)

// named is every template by the name a file calls it: lower case, a
// dash between words.
var named = map[string]func() (bodies.Body, error){
	"sol": Sol, "mercury": Mercury, "venus": Venus, "earth": Earth,
	"mars": Mars, "jupiter": Jupiter, "saturn": Saturn, "uranus": Uranus,
	"neptune": Neptune, "pluto": Pluto, "charon": Charon,
	"hale-bopp": HaleBopp, "hyakutake": Hyakutake, "magnetar": Magnetar,
	"pulsar": Pulsar, "quasar": Quasar, "hot-jupiter": HotJupiter,
}

// Named is the template of a name, its own copy to change. It refuses
// a name no template carries, saying which there are, and surfaces
// what a template's own making refused.
func Named(name string) (bodies.Body, error) {
	made, found := named[name]
	if !found {
		return bodies.Body{}, fmt.Errorf("template: no body is "+
			"named %q; there are %s", name,
			strings.Join(Names(), ", "))
	}
	return made()
}

// Names is every template's name, in order.
func Names() []string { return slices.Sorted(maps.Keys(named)) }
