package cities

import (
	"bytes"
	_ "embed" // the creatures are data, read from the file beside this one
	"fmt"
	"slices"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

//go:embed gargoyle-creatures.yaml
var creatureFile []byte

// Creature is how a creature of the manufactory is carved as a gargoyle:
// the study it is taken from, how large it is carved and how it is posed
// on its block. Its numbers are the data's, in gargoyle-creatures.yaml,
// which says what each means; this package knows nothing of the shape of
// the creature, only the name of its study.
type Creature struct {
	Name  string `yaml:"name"`
	Says  string `yaml:"says"`
	Study string `yaml:"study"`
	// Scale is how large the creature is carved, as a share of its own
	// size.
	Scale float64 `yaml:"scale"`
	// Roll, Pitch, Yaw and Lean are the turns, in degrees, that take the
	// creature from its own frame to the block's.
	Roll  float64 `yaml:"roll"`
	Pitch float64 `yaml:"pitch"`
	Yaw   float64 `yaml:"yaw"`
	Lean  float64 `yaml:"lean"`
	// Held is the point of the creature, in its own frame, held in place
	// on the block, and Up and Out how far over the slab and out of the
	// wall it is held.
	Held [3]float64 `yaml:"held"`
	Up   float64    `yaml:"up"`
	Out  float64    `yaml:"out"`
	// Mouth is the creature's spout, in its own frame.
	Mouth [3]float64 `yaml:"mouth"`
	Kind  string     `yaml:"kind"`
	By    string     `yaml:"by"`
}

// HeldAt is the held point as a vector, in the creature's own frame.
func (creature Creature) HeldAt() maths.Vec {
	return maths.V(creature.Held[0], creature.Held[1], creature.Held[2])
}

// MouthAt is the spout as a vector, in the creature's own frame.
func (creature Creature) MouthAt() maths.Vec {
	return maths.V(creature.Mouth[0], creature.Mouth[1], creature.Mouth[2])
}

// creatureData is the data file's shape.
type creatureData struct {
	Creatures []Creature `yaml:"creatures"`
}

// GargoyleCreatureRows are the creatures a gargoyle may be carved as, in
// the order the data writes them. An entry that names no study, is named
// twice, has no size, or is not picked by someone is refused, and so is a
// key that nobody knows.
func GargoyleCreatureRows() ([]Creature, error) {
	var data creatureData
	decoder := yaml.NewDecoder(bytes.NewReader(creatureFile))
	decoder.KnownFields(true)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("cities: gargoyle-creatures.yaml: %w", err)
	}
	var seen []string
	for _, each := range data.Creatures {
		if each.Name == "" || each.Study == "" || !(each.Scale > 0) ||
			each.Kind != pickedKind || each.By == "" {
			return nil, fmt.Errorf("cities: gargoyle creature %q needs a "+
				"name, a study, a size over nought, kind %s and who "+
				"picked it", each.Name, pickedKind)
		}
		if slices.Contains(seen, each.Name) {
			return nil, fmt.Errorf("cities: gargoyle-creatures.yaml says "+
				"%s twice", each.Name)
		}
		seen = append(seen, each.Name)
	}
	if len(data.Creatures) == 0 {
		return nil, fmt.Errorf("cities: gargoyle-creatures.yaml has no " +
			"creatures")
	}
	return data.Creatures, nil
}

// CreatureNamed is the creature of a name, or a refusal naming the
// creatures there are.
func CreatureNamed(name string) (Creature, error) {
	rows, err := GargoyleCreatureRows()
	if err != nil {
		return Creature{}, err
	}
	for _, each := range rows {
		if each.Name == name {
			return each, nil
		}
	}
	names := make([]string, len(rows))
	for index, each := range rows {
		names[index] = each.Name
	}
	return Creature{}, fmt.Errorf("cities: no gargoyle creature named %q, "+
		"only %v", name, names)
}
