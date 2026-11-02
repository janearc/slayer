package bodies

import (
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// a body placed at a scale that is no size, with a gas of no depth, an
// aurora upside down, a hole spinning at its mass, or a tail at its own
// sun, is refused by name, not put into the world as infinity or nil.
func TestABodyThatCannotBeMadeIsRefused(t *testing.T) {
	for name, c := range map[string]struct {
		b     Body
		scale float64
		want  string
	}{
		"no scale":      {Body{Name: "a", Kind: Star, Radius: 1}, 0, "scale"},
		"NaN scale":     {Body{Name: "a", Kind: Star, Radius: 1}, math.NaN(), "scale"},
		"endless scale": {Body{Name: "a", Kind: Star, Radius: 1}, math.Inf(1), "scale"},
		"shallow gas": {Body{Name: "a", Kind: Giant, Radius: 1,
			Gas: &Gas{Depth: 0, Opacity: 1}}, 1, "depth"},
		"upside-down aurora": {Body{Name: "a", Kind: Star, Radius: 1,
			Aurora: &Aurora{Low: 5, High: 5}}, 1, "aurora"},
		"spun out": {Body{Name: "a", Kind: Hole, Mass: Sun, Spin: 1}, 1, "horizon"},
		"tail at its sun": {Body{Name: "a", Kind: Star, Radius: 1,
			Tails: []Tail{{Length: 1, Base: 1, Sun: maths.V(0, 0, 0)}}}, 1, "tail"},
	} {
		err := c.b.Place(thing.NewWorld(), maths.Vec{}, c.scale, 2)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// an orbit past what a Duration holds is refused as an error, before
// thing.Period panics: by hand, a moon 1e10 km round the earth, or two
// suns 1e10 km apart, take hundreds of years and more.
func TestAnOrbitTooLongIsAnError(t *testing.T) {
	earth := Body{Name: "earth", Kind: Star, Radius: 6371, Mass: 5.972e24,
		Moons: []Moon{{Body: Body{Name: "far", Kind: Star, Radius: 1}, Orbit: 1e10}}}
	if err := earth.Place(thing.NewWorld(), maths.Vec{}, 1e-3, 2); err == nil ||
		!strings.Contains(err.Error(), "292") {
		t.Errorf("a moon at 1e10 km: %v", err)
	}
	sun := Body{Name: "sun", Kind: Rock, Radius: 1, Mass: Sun}
	if err := Binary(thing.NewWorld(), sun, sun, maths.Vec{}, 1e10, 1e-3, 2); err == nil ||
		!strings.Contains(err.Error(), "292") {
		t.Errorf("two suns 1e10 km apart: %v", err)
	}
	none := Body{Name: "none", Kind: Rock, Radius: 1}
	if err := Binary(thing.NewWorld(), none, none, maths.Vec{}, 10, 1, 2); err == nil ||
		!strings.Contains(err.Error(), "no mass") {
		t.Errorf("two bodies of no mass: %v", err)
	}
}
