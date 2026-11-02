package bodies_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/bodies"
	"github.com/janearc/slayer/scene/shape"
	"github.com/janearc/slayer/scene/thing"
	"github.com/janearc/slayer/scene/world"
)

// a pair of three quarters and a quarter of the sun's mass, an
// astronomical unit apart, go round their common centre in the time
// the sun's whole mass gives that distance: a sidereal year, to 5e-4.
//
// the heavier is the nearer to the centre, by the share of the other:
// a quarter of the distance, 0.25 AU, and the lighter three quarters.
// at a scale of 1e-8 the radii are 0.25 x 1.495978707e8 x 1e-8 and
// three times that.
//
// the lighter starts on the far side, half a turn round. each ball is
// as wide as its body's radius in units.
func TestAPairGoRoundTheirCentreByTheirMasses(t *testing.T) {
	heavy := bodies.Body{Name: "heavy", Kind: bodies.Star, Radius: 8, Mass: 0.75 * bodies.Sun}
	light := bodies.Body{Name: "light", Kind: bodies.Star, Radius: 4, Mass: 0.25 * bodies.Sun}
	at := maths.V(1, 2, 3)
	w := thing.NewWorld()
	if err := bodies.Binary(w, heavy, light, at, auKm, 1e-8, 2); err != nil {
		t.Fatal(err)
	}
	if len(w.Things) != 2 {
		t.Fatalf("a pair places %d things", len(w.Things))
	}
	h, l := w.Things[0].Motion.(thing.Orbit), w.Things[1].Motion.(thing.Orbit)
	if h.Centre != at || l.Centre != at {
		t.Errorf("they go round %v and %v, not %v", h.Centre, l.Centre, at)
	}
	near(t, "the heavy one's orbit", h.Radius, 0.25*auKm*1e-8, 1e-9)
	near(t, "the light one's orbit", l.Radius, 0.75*auKm*1e-8, 1e-9)
	if h.Phase != 0 || l.Phase != math.Pi {
		t.Errorf("they start at %v and %v", h.Phase, l.Phase)
	}
	if h.Period != l.Period {
		t.Errorf("they go round in %v and %v", h.Period, l.Period)
	}
	near(t, "their period", h.Period.Seconds(), siderealYear, 5e-4)
	near(t, "the heavy one's width", w.Things[0].Shape.(shape.Sphere).Radius, 8e-8, 1e-12)
	near(t, "the light one's width", w.Things[1].Shape.(shape.Sphere).Radius, 4e-8, 1e-12)
}

// the period is kepler's for the two masses together at their distance,
// T = 2 pi sqrt(r^3 / G(M1 + M2)), whatever the sizes.
//
// two bodies of a quarter of a kilogram each, a metre apart, go round in
// 2 pi sqrt(1 / (G / 2)), which is 12.6 days, and are not refused for
// weighing under a kilogram together.
func TestAPairOfUnderAKilogramGoRoundByKepler(t *testing.T) {
	small := bodies.Body{Name: "s", Kind: bodies.Star, Radius: 1, Mass: 0.25}
	w := thing.NewWorld()
	if err := bodies.Binary(w, small, small, maths.Vec{}, 1e-3, 1, 2); err != nil {
		t.Fatal(err)
	}
	period := w.Things[0].Motion.(thing.Orbit).Period.Seconds()
	near(t, "their period", period, 2*math.Pi/omega(0.5, 1e-3), 1e-9)
	if days := period / 86400; days < 12.5 || days > 12.7 {
		t.Errorf("they go round in %v days", days)
	}
}

// each of a pair is a grey ball whatever it is, and is made as a ground
// is: one whose feature has nowhere to go refuses the pair, by name, and
// names the body.
func TestAPairIsGreyBallsAndRefusesAGroundThatCannotBeMade(t *testing.T) {
	star := bodies.Body{Name: "star", Kind: bodies.Star, Radius: 1, Mass: bodies.Sun,
		Glow: maths.Grey(5)}
	w := thing.NewWorld()
	if err := bodies.Binary(w, star, star, maths.Vec{}, 1e6, 1e-6, 2); err != nil {
		t.Fatal(err)
	}
	for i, th := range w.Things {
		if th.Material.Colour != maths.Grey(0.5) || th.Material.Glow != (maths.Light{}) {
			t.Errorf("star %d is %v glowing %v", i, th.Material.Colour, th.Material.Glow)
		}
	}
	bad := bodies.Body{Name: "bad", Kind: bodies.Rock, Radius: 1, Mass: bodies.Sun,
		Ground: &bodies.Ground{Terrain: world.Terrain{Seed: 1, Ocean: 0.5},
			Features: func(*world.World) ([]world.Feature, error) {
				return nil, fmt.Errorf("no land for maxwell")
			}}}
	for name, c := range map[string][2]bodies.Body{"first": {bad, star}, "second": {star, bad}} {
		err := bodies.Binary(thing.NewWorld(), c[0], c[1], maths.Vec{}, 1e6, 1e-6, 2)
		if err == nil || !strings.Contains(err.Error(), "bad: no land for maxwell") {
			t.Errorf("the %s is bad: %v", name, err)
		}
	}
}
