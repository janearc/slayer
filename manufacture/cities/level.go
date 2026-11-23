package cities

import (
	_ "embed"
	"errors"
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/maths/strategies"
	yaml "gopkg.in/yaml.v3"
)

// A leveller is how much of a thing is drawn, chosen from how close the
// eye is: a thing is drawn finer the more of the film it spans. It is
// built once from the camera and asked of each thing where it stands and
// how far it reaches, so a tree beside the eye is drawn to its needles
// and the forest behind it as crowns.

// Leveller is the level a thing is drawn at, from the place of its middle
// and how far it reaches across, in metres, corner to corner.
type Leveller func(at maths.Subspace, extent float64) Detail

// Constant is the leveller that draws everything at one level, as a
// design was drawn before there were levellers.
func Constant(level Detail) Leveller {
	return func(maths.Subspace, float64) Detail { return level }
}

// Rung is one step of a ladder: the level a thing is drawn at once it
// spans so many pixels of the film.
type Rung struct {
	Level  Detail  `yaml:"level"`
	Pixels float64 `yaml:"pixels"`
}

// NewLeveller is the leveller of an eye and a ladder: a thing out of view
// is drawn at the first rung, and any other at the finest rung whose
// pixels its span reaches. It refuses an eye that sees nothing, and a
// ladder that is empty, does not start at nought or does not rise.
func NewLeveller(eye strategies.Eye, ladder []Rung) (Leveller, error) {
	if err := eye.Check(); err != nil {
		return nil, err
	}
	if err := checkLadder(ladder); err != nil {
		return nil, err
	}
	return func(at maths.Subspace, extent float64) Detail {
		if !eye.SeesBall(at.Origin, extent/2) {
			return ladder[0].Level
		}
		span := eye.Span(at.Origin, extent)
		level := ladder[0].Level
		for _, rung := range ladder[1:] {
			if span >= rung.Pixels {
				level = rung.Level
			}
		}
		return level
	}, nil
}

// checkLadder refuses a ladder that is empty, whose first rung is not at
// nought pixels, or whose rungs do not rise.
func checkLadder(ladder []Rung) error {
	if len(ladder) == 0 || ladder[0].Pixels != 0 {
		return errors.New("cities: a ladder starts at nought pixels")
	}
	for index := 1; index < len(ladder); index++ {
		if !(ladder[index].Pixels > ladder[index-1].Pixels) {
			return fmt.Errorf("cities: the rungs of a ladder rise, and %s "+
				"is not above %s", ladder[index].Level,
				ladder[index-1].Level)
		}
	}
	return nil
}

//go:embed levels.yaml
var levelsFile []byte

// levelsData is the levels file: the ladders by name, and the bands.
type levelsData struct {
	Ladders map[string][]Rung `yaml:"ladders"`
	Bands   map[string]int    `yaml:"bands"`
}

// levels is the levels file read.
func levels() (levelsData, error) {
	var data levelsData
	if err := yaml.Unmarshal(levelsFile, &data); err != nil {
		return levelsData{}, fmt.Errorf("cities: levels.yaml: %w", err)
	}
	return data, nil
}

// Ladder is the ladder of a name in the levels file: trees, buildings or
// manufactory. It refuses a name the file has not, and a ladder that is
// not one.
func Ladder(name string) ([]Rung, error) {
	data, err := levels()
	if err != nil {
		return nil, err
	}
	ladder, found := data.Ladders[name]
	if !found {
		return nil, fmt.Errorf("cities: levels.yaml has no ladder %q", name)
	}
	return ladder, checkLadder(ladder)
}

// band is how many storeys of the levels file's band of a name, or why
// there is none.
func band(name string) (int, error) {
	data, err := levels()
	if err != nil {
		return 0, err
	}
	storeys, found := data.Bands[name]
	if !found || storeys < 1 {
		return 0, fmt.Errorf("cities: levels.yaml has no band %q", name)
	}
	return storeys, nil
}
