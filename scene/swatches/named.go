package swatches

import (
	"maps"
	"slices"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// named is every paint and lamp by the name a file calls it: the paints,
// the lamps and the gas giant's paints, each its own name in lower case.
var named = map[string]swatch.Swatch{
	"chalk": Chalk, "clay": Clay, "paint": Paint, "slate": Slate,
	"sea": Sea, "moss": Moss, "daylight": Daylight, "red": Red,
	"green": Green, "blue": Blue, "horizon": Horizon, "zenith": Zenith,
	"tungsten": Tungsten, "cream": Cream, "rust": Rust, "ochre": Ochre,
	"storm": Storm, "ice": Ice,
}

// Named is the swatch of a name, and whether there is one by that name.
func Named(name string) (swatch.Swatch, bool) {
	found, ok := named[name]
	return found, ok
}

// Names is every swatch's name, in order.
func Names() []string { return slices.Sorted(maps.Keys(named)) }
