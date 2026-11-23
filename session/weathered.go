package session

import (
	"fmt"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/set"
	"github.com/janearc/slayer/stage"
	"github.com/janearc/slayer/weathering"
)

// weathered lays the world's weathering over everything staged, the
// set's things and its world half alike, once both are in. A world with
// no weathering is left as it is.
//
// the agents' places are the world half's, moved by rebasing as every
// place of the world half is, so that a mine or a plaza stands where the
// set says on the ground the things stand on.
func (making *staging) weathered(read set.Set) error {
	said, has := read.World()
	if !has {
		return nil
	}
	laid, has := said.Weathering()
	if !has {
		return nil
	}
	age, err := making.weatheredAge(read, laid)
	if err != nil {
		return err
	}
	layer := weatheringLayer(laid, said.Suns(), making.rebasing(read))
	layer.Setting.Age = age
	if err := weathering.Apply(making.world, layer); err != nil {
		return fmt.Errorf("%s: %w", making.given.set, err)
	}
	return nil
}

// northward is the way north runs in the world's space, which has x
// east, y up and z south.
var northward = maths.V(0, 0, -1)

// weatheringLayer is the layer a set's weathering asks for, its places
// moved by shift. The sun is the first the world says, and there is none
// where it says none.
func weatheringLayer(laid set.Weathering, suns []set.Sunlight,
	shift maths.Vec) *weathering.Layer {
	setting := weathering.Setting{Seed: laid.Seed(), Age: laid.Age(),
		Up: maths.YAxis, North: northward, Wind: laid.Wind(),
		Ground: laid.Ground() + shift.Y, Probe: laid.Probe(),
		Block: laid.Block()}
	if len(suns) > 0 {
		setting.Sun = suns[0].Toward().Unit()
	}
	layer := &weathering.Layer{Setting: setting}
	for _, each := range laid.Agents() {
		layer.Agents = append(layer.Agents, weatheringAgent(each, shift))
	}
	return layer
}

// weatheringAgent is the agent a set's agent asks for, its place moved by
// shift.
func weatheringAgent(each set.WeatheringAgent,
	shift maths.Vec) weathering.Agent {
	agent := weathering.Agent{Kind: weathering.Kind(each.Kind()),
		Name: each.Name(), On: each.On(), Seek: each.Seek(),
		Shun: each.Shun(), Mark: each.Mark(), At: each.At().Add(shift),
		Facing: each.Facing()}
	if colour, said := each.Colour(); said {
		agent.Colour = stage.Colour(colour)
	}
	if colour, said := each.Other(); said {
		agent.Other = stage.Colour(colour)
	}
	if colour, said := each.Line(); said {
		agent.Line = stage.Colour(colour)
	}
	numbers := map[string]*float64{"strength": &agent.Strength,
		"settle": &agent.Settle, "grain": &agent.Grain,
		"share": &agent.Share, "tilt": &agent.Tilt, "level": &agent.Level,
		"shelter": &agent.Shelter, "reach": &agent.Reach,
		"length": &agent.Length, "spacing": &agent.Spacing,
		"climb": &agent.Climb, "low": &agent.Low, "high": &agent.High,
		"shade": &agent.Shade, "north": &agent.North, "catch": &agent.Catch,
		"spread": &agent.Spread, "upwind": &agent.Upwind,
		"rate": &agent.Rate, "turn": &agent.Turn, "fleck": &agent.Fleck,
		"cell": &agent.Cell, "storey": &agent.Storey,
		"boarded": &agent.Boarded, "broken": &agent.Broken,
		"proud": &agent.Proud, "plank": &agent.Plank, "slant": &agent.Slant,
		"heaven": &agent.Heaven, "behind": &agent.Behind,
		"sought": &agent.Sought, "shunned": &agent.Shunned,
		"crack": &agent.Crack, "spalled": &agent.Spalled}
	for key, target := range numbers {
		*target = each.Number(key)
	}
	return agent
}
