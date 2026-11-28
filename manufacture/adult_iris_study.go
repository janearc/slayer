package manufacture

import (
	"github.com/janearc/slayer/bioforms/ocular"
	"github.com/janearc/slayer/manufacture/animals/vertebrates/mammals/primates"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/noise"
)

type adultIrisStudy struct{ colour ocular.Colour }

func adultIrisRows() map[string]maker {
	return map[string]maker{
		"adult-human-iris-brown": adultIrisStudy{colour: ocular.Brown},
		"adult-human-iris-hazel": adultIrisStudy{colour: ocular.Hazel},
		"adult-human-iris-blue":  adultIrisStudy{colour: ocular.Blue},
	}
}

func (row adultIrisStudy) extent(from seed) (maths.Box, error) {
	return (adultHumanStudy{bareScalp: true}).extent(from)
}

func (row adultIrisStudy) made(from seed) (good, error) {
	if !from.made() {
		return good{}, errGood
	}
	spec := ocular.Human(ocular.HumanIris{Colour: row.colour, Seed: from.number})
	left, err := ocular.NewSurface(spec)
	if err != nil {
		return good{}, err
	}
	// The other eye keeps the same palette and shares the scene seed only
	// as a source of its own procedural marks.
	spec.Seed = noise.Mix(from.number + noise.Golden)
	right, err := ocular.NewSurface(spec)
	if err != nil {
		return good{}, err
	}
	gaze := primates.Gaze{
		Left:  primates.EyeAim{Appearance: left},
		Right: primates.EyeAim{Appearance: right},
	}
	return (adultHumanStudy{bareScalp: true}).withGaze(from, gaze)
}
