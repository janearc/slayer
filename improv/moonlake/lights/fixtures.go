package lights

import (
	"fmt"
	"math"
)

type FixtureConfig struct {
	FloodPasses         [2]string
	FloodGain, SpotGain float64
}

type FixtureCues struct {
	FloodLeft, FloodRight, Spots, SpotAim float64
	HasColour                             bool
	Colour                                [3]float64
}

type FixtureLayer struct {
	Name   string
	Amount float64
	Colour [3]float64
}

// PlanFixtures lists the fixture layers to draw: a white left flood and a
// magenta right flood, named by FloodPasses, and two spot layers for each side
// that share the spots by SpotAim, held between 0 and 2. Spots take a tint
// halfway to Colour when HasColour is set. Layers under 0.001 are left out.
// It refuses a SpotAim, amount or colour that is not finite.
func PlanFixtures(config FixtureConfig, cue FixtureCues) ([]FixtureLayer, error) {
	if !finite(cue.SpotAim) {
		return nil, fmt.Errorf("invalid fixture spot aim")
	}
	white := [3]float64{1, 1, 1}
	tint := white
	if cue.HasColour {
		for c := range tint {
			tint[c] = 0.5 + 0.5*cue.Colour[c]
		}
	}
	layers := []FixtureLayer{
		{config.FloodPasses[0], config.FloodGain * cue.FloodLeft, white},
		{config.FloodPasses[1], config.FloodGain * cue.FloodRight, [3]float64{1, 0.25, 0.85}},
	}
	aim := math.Max(0, math.Min(2, cue.SpotAim))
	near := math.Min(1, math.Floor(aim))
	f := aim - near
	for _, side := range []string{"left", "right"} {
		a, b := int(near), int(near)+1
		if side == "right" {
			a, b = 2-a, 2-b
		}
		layers = append(layers,
			FixtureLayer{fmt.Sprintf("spot-%s-%d", side, a), config.SpotGain * cue.Spots * (1 - f), tint},
			FixtureLayer{fmt.Sprintf("spot-%s-%d", side, b), config.SpotGain * cue.Spots * f, tint})
	}
	var active []FixtureLayer
	for _, layer := range layers {
		if err := validateFixtureLayer(layer); err != nil {
			return nil, err
		}
		if layer.Amount < 1e-3 {
			continue
		}
		active = append(active, layer)
	}
	return active, nil
}

// validateFixtureLayer refuses a layer whose amount or any colour channel is
// not finite, naming the layer in the error.
func validateFixtureLayer(layer FixtureLayer) error {
	if !finite(layer.Amount) {
		return fmt.Errorf("invalid fixture %s intensity", layer.Name)
	}
	for _, channel := range layer.Colour {
		if !finite(channel) {
			return fmt.Errorf("invalid fixture %s colour", layer.Name)
		}
	}
	return nil
}

// AddFixtures adds each layer's pass image, times its amount and colour, to
// sky. All layers and passes are checked first, so on an error sky is not
// changed. Layers under 0.001 are skipped. It refuses a missing pass or one
// not the size of sky.
func AddFixtures(sky Image, layers []FixtureLayer, passes map[string]Image) error {
	if err := sky.validate(); err != nil {
		return err
	}
	for _, layer := range layers {
		if err := validateFixtureLayer(layer); err != nil {
			return err
		}
		if layer.Amount < 1e-3 {
			continue
		}
		pass, found := passes[layer.Name]
		if !found {
			return fmt.Errorf("missing fixture pass %s", layer.Name)
		}
		if err := pass.validate(); err != nil {
			return err
		}
		if pass.Width != sky.Width || pass.Height != sky.Height {
			return fmt.Errorf("%s is %dx%d, the frame %dx%d", layer.Name,
				pass.Width, pass.Height, sky.Width, sky.Height)
		}
	}
	for _, layer := range layers {
		if layer.Amount < 1e-3 {
			continue
		}
		pass := passes[layer.Name]
		for i := 0; i < len(sky.Light); i += 3 {
			for c := range 3 {
				sky.Light[i+c] += layer.Amount * layer.Colour[c] * pass.Light[i+c]
			}
		}
	}
	return nil
}
