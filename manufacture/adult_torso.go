package manufacture

import (
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/scene/thing"
)

// AdultTorso constructs a native anatomical study extending from pelvis
// through the base of the neck. Its pose and morphology are explicit inputs.
func AdultTorso(spec primates.Torso) (Kind, error) {
	if err := spec.Check(); err != nil {
		return kind{}, err
	}
	return kind{name: "adult-torso-study", how: torsoMaker{spec}}, nil
}

type torsoMaker struct{ spec primates.Torso }

func (m torsoMaker) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	surface, err := m.spec.Surface(from.number, primates.Around, primates.Rows)
	if err != nil {
		return good{}, err
	}
	return good{native: []thing.Thing{surface}}, nil
}
