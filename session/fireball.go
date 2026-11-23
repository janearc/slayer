package session

import (
	"github.com/janearc/celestial/measure"
	"github.com/janearc/slayer/flame"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// An ignition's fireball, as -fireball lets one go from a recipe fire:
// its gas at fireballHeat in air at fireballAir, its soot fireballSoot a
// metre at 550 nm, and its stem a share fireballStem of that. All four
// are picked: 1,800 K is about a wood fire's hottest, the soot is
// half the crown fire's (crownSoot, in alight.go), and the air is a
// summer night's.
const (
	fireballHeat = 1800.0
	fireballAir  = 290.0
	fireballSoot = 0.5
	fireballStem = 0.3
)

// letGoFireball adds a fireball of -fireball's radius at a recipe fire's
// place, sitting on it, let go at -fireball-at: a thermal that rises,
// cools and rolls into a mushroom's cap. It glows by its own heat and
// lights nothing else: it is no lamp.
func (stage *staging) letGoFireball(seed uint64, at maths.Subspace) error {
	radius := stage.given.fireballRadius
	ball, err := flame.NewFireball(flame.Thermal{Radius: radius,
		Height: radius, Hottest: measure.Kelvin(fireballHeat),
		Air: measure.Kelvin(fireballAir), Soot: fireballSoot,
		Stem: fireballStem, Start: stage.given.fireballAt, Seed: seed},
		stage.given.exposure)
	if err != nil {
		return err
	}
	stage.world.AddVolume(thing.Volume{Medium: ball,
		Material: &thing.Material{Name: "fireball", Shine: ball.Shine()},
		Motion:   thing.Still(at)})
	return nil
}
