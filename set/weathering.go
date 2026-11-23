package set

import (
	"slices"

	"github.com/janearc/slayer/maths"
	yaml "gopkg.in/yaml.v3"
)

// weatheringMost is the largest any number of a weathering layer may be:
// a settle in years, a reach in metres, a seed. picked: past the size of
// any place a set holds.
const weatheringMost = 1e6

// weatheringKinds are the agents a layer may hold, by the word a set file
// writes for each.
var weatheringKinds = []string{"rain", "soot", "dust", "moss", "ivy",
	"graffiti", "rust", "spall", "vacancy", "mineral", "glitter", "nano",
	"festival", "spout", "wet"}

// weatheringNumbers are the keys of a weathering agent that take one
// number each, in metres, years or shares as the agent's kind says.
var weatheringNumbers = []string{"strength", "settle", "grain", "share",
	"tilt", "level", "shelter", "reach", "length", "spacing", "climb", "low",
	"high", "shade", "north", "catch", "spread", "upwind", "rate", "turn",
	"fleck", "cell", "storey", "boarded", "broken", "proud", "plank", "slant",
	"crack", "spalled", "heaven", "behind", "sought", "shunned"}

// Weathering is what happens to the world's places over time, said as
// agents the world holds and not as a look of any one building: how old
// the place is, the wind that carries what is carried, the ground's
// height, and each cause with the numbers its rule takes.
type Weathering struct {
	seed, age, ground, probe, block float64
	wind                            [3]float64
	agents                          []WeatheringAgent
	ageFollows                      ageFollowing
}

// WeatheringAgent is one cause in a weathering layer.
type WeatheringAgent struct {
	name, kind, mark    string
	on, seek, shun      []string
	colour, other, line *Colour
	at, facing          [3]float64
	numbers             map[string]float64
}

// Weathering is the world's weathering, and whether the world says one.
func (made World) Weathering() (Weathering, bool) {
	if made.weathering == nil {
		return Weathering{}, false
	}
	return *made.weathering, true
}

// Seed makes every patch, cell and fleck of the weathering.
func (made Weathering) Seed() uint64 { return uint64(made.seed) }

// Age is how old the place is, in years, where the set says a number.
// Where the age follows a superblock's accretion it is nought here, and
// AgeFollows says how it is found.
func (made Weathering) Age() float64 { return made.age }

// Ground is the height of the ground, in metres.
func (made Weathering) Ground() float64 { return made.ground }

// Probe is how far off a surface a look overhead starts, in metres.
func (made Weathering) Probe() float64 { return made.probe }

// Block is how wide the cells are that what stands over a place is kept
// by, in metres.
func (made Weathering) Block() float64 { return made.block }

// Wind is the way the wind blows, in the world's space.
func (made Weathering) Wind() maths.Vec { return fromMap(made.wind) }

// Agents is every agent, in the order written.
func (made Weathering) Agents() []WeatheringAgent {
	return slices.Clone(made.agents)
}

// Name is the agent's name.
func (made WeatheringAgent) Name() string { return made.name }

// Kind is the agent's kind, one of the words a set file writes.
func (made WeatheringAgent) Kind() string { return made.kind }

// On is the words the material of what the agent lands on must hold.
func (made WeatheringAgent) On() []string { return slices.Clone(made.on) }

// Seek is the words of the materials graffiti goes for.
func (made WeatheringAgent) Seek() []string { return slices.Clone(made.seek) }

// Shun is the words of the materials graffiti leaves alone.
func (made WeatheringAgent) Shun() []string { return slices.Clone(made.shun) }

// Mark is the one kind of mark a graffiti agent writes, tag, throw-up or
// piece, or nothing where it writes all three.
func (made WeatheringAgent) Mark() string { return made.mark }

// Colour is what the agent lays, and whether it says.
func (made WeatheringAgent) Colour() (Colour, bool) {
	return colourSaid(made.colour)
}

// Other is the agent's second colour, and whether it says.
func (made WeatheringAgent) Other() (Colour, bool) {
	return colourSaid(made.other)
}

// Line is graffiti's outline and the ink of its tags, and whether it says.
func (made WeatheringAgent) Line() (Colour, bool) {
	return colourSaid(made.line)
}

// colourSaid is a colour an agent may say, and whether it says it.
func colourSaid(colour *Colour) (Colour, bool) {
	if colour == nil {
		return Colour{}, false
	}
	return *colour, true
}

// At is the place the agent acts from, in the world's space.
func (made WeatheringAgent) At() maths.Vec { return fromMap(made.at) }

// Facing is the way the agent's seed or its row of dwellings faces, in
// the world's space.
func (made WeatheringAgent) Facing() maths.Vec { return fromMap(made.facing) }

// Number is a number the agent says, or nought where it says none.
func (made WeatheringAgent) Number(key string) float64 {
	return made.numbers[key]
}

// weathering reads the world's weathering.
func (reading reader) weathering(_, value *yaml.Node) error {
	made := &Weathering{}
	var source string
	err := reading.keys(value, map[string]step{
		"seed":   reading.number(&made.seed, 0, weatheringMost),
		"age":    reading.age(made),
		"ground": reading.number(&made.ground, -weatheringMost, weatheringMost),
		"probe":  reading.number(&made.probe, 0, weatheringMost),
		"block":  reading.number(&made.block, 0, weatheringMost),
		"wind":   reading.three(&made.wind, -weatheringMost, weatheringMost),
		"agents": reading.weatheringAgents(made),
		"source": reading.source(&source),
	}, nil)
	if err == nil {
		err = reading.all(value, "a weathering layer", "age", "block",
			"probe", "agents", "source")
	}
	if err == nil && made.block <= 0 {
		err = reading.refuse(value, "block is how wide the cells are, "+
			"and takes a number above nought")
	}
	reading.set.world.weathering = made
	return err
}

// weatheringAgents reads each agent of a layer, by its name.
func (reading reader) weatheringAgents(into *Weathering) step {
	return func(_, value *yaml.Node) error {
		return reading.each(value, func(name, block *yaml.Node) error {
			made := WeatheringAgent{name: name.Value,
				numbers: map[string]float64{}}
			steps := map[string]step{
				"kind":   reading.word(&made.kind, weatheringKinds...),
				"on":     reading.words(&made.on),
				"seek":   reading.words(&made.seek),
				"shun":   reading.words(&made.shun),
				"mark":   reading.word(&made.mark, "tag", "throw-up", "piece"),
				"colour": reading.agentColour(&made.colour),
				"other":  reading.agentColour(&made.other),
				"line":   reading.agentColour(&made.line),
				"at":     reading.three(&made.at, -weatheringMost, weatheringMost),
				"facing": reading.three(&made.facing, -1, 1),
			}
			for _, key := range weatheringNumbers {
				var number float64
				steps[key] = func(at, each *yaml.Node) error {
					err := reading.number(&number, 0, weatheringMost)(at, each)
					made.numbers[at.Value] = number
					return err
				}
			}
			err := reading.keys(block, steps, nil)
			if err == nil {
				err = reading.all(block, "a weathering agent", "kind")
			}
			into.agents = append(into.agents, made)
			return err
		})
	}
}

// agentColour reads a colour into a pointer, so that an agent that says
// none has none.
func (reading reader) agentColour(into **Colour) step {
	return func(name, value *yaml.Node) error {
		var made Colour
		err := reading.colour(&made)(name, value)
		*into = &made
		return err
	}
}

// words is a step that reads a list of words into target, at least one.
func (reading reader) words(target *[]string) step {
	return func(name, value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode || len(value.Content) == 0 {
			return reading.refuse(value, "%s takes a list of words", name.Value)
		}
		for _, each := range value.Content {
			if each.Kind != yaml.ScalarNode || each.Value == "" {
				return reading.refuse(each, "%s takes words", name.Value)
			}
			*target = append(*target, each.Value)
		}
		return nil
	}
}
