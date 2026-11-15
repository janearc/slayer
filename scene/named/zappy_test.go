package named_test

import (
	"errors"
	"testing"

	"github.com/janearc/slayer/lantern"
	"github.com/janearc/slayer/scene/named"
)

// Zappy's defaults are zappy as package lantern defines it: no second
// copy of its numbers.
func TestZappysDefaultsAreZappyAsDefined(t *testing.T) {
	scene := mustGet(t, "zappy")
	defined := lantern.Zappy()
	for name, want := range map[string]any{"radius": defined.Radius,
		"kelvin": defined.Kelvin, "orbits": defined.Orbits,
		"glare": defined.Glare, "beam-cone": defined.Beams.Cone,
		"beam-spin": defined.Beams.Spin,
		"tumble":    defined.Tumble} {
		if got := mustParameter(t, scene, name).Default(); got != want {
			t.Errorf("zappy's %s defaults to %v, want %v", name, got,
				want)
		}
	}
}

// anti-tests: near misses of zappy's own parameters are refused by name:
// no star, no rings, more than full wear, an eye past straight up.
func TestZappyRefusesNearMisses(t *testing.T) {
	scene := mustGet(t, "zappy")
	for _, with := range []map[string]any{{"radius": float64(none)},
		{"orbits": none}, {"wear": fraction}, {"elevation": 90.5},
		{"haze": -fraction}, {"distance": float64(none)}} {
		if err := scene.Check(with); !errors.Is(err, named.ErrBounds) {
			t.Errorf("%v: %v; want ErrBounds", with, err)
		}
	}
}
