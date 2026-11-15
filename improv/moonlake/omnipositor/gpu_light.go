package omnipositor

import (
	"fmt"

	"github.com/janearc/slayer/improv/moonlake/lights"
	"github.com/janearc/slayer/improv/moonlake/omnipositor/gpulights"
)

type gpuLightBase struct {
	renderer *gpulights.Renderer
}

// Base draws the plate on the GPU, scaled by master. If moon is not nil it
// also adds the moon, using open, shift and level.
func (base gpuLightBase) Base(plate lights.Image, moon *lights.Image, open []bool, master, shift, level float64) (lights.Image, error) {
	request := gpulights.Request{Plate: plate, Master: master}
	if moon != nil {
		request.Effects = gpulights.EffectMoon
		request.Moon, request.Open = moon, open
		request.MoonShift, request.MoonLevel = shift, level
	}
	result, err := base.renderer.Render(request)
	return result.Sky, err
}

// Fixtures draws the fixture layers over sky on the GPU, using passes, at a
// master level of 1.
func (base gpuLightBase) Fixtures(sky lights.Image, layers []lights.FixtureLayer, passes map[string]lights.Image) (lights.Image, error) {
	fixtures := make([]gpulights.FixtureLayer, len(layers))
	for index, layer := range layers {
		fixtures[index] = gpulights.FixtureLayer{Name: layer.Name, Amount: layer.Amount, Colour: layer.Colour}
	}
	result, err := base.renderer.Render(gpulights.Request{Plate: sky, Master: 1, Effects: gpulights.EffectFixtures, FixtureLayers: fixtures, Passes: passes})
	return result.Sky, err
}

// Fire draws the fire image for config and input on the GPU.
func (base gpuLightBase) Fire(config lights.FireConfig, input lights.FireInput) (lights.Image, error) {
	return gpulights.Fire(config, input)
}

// SkyGlow adds the sky glow described by input to sky on the GPU.
func (base gpuLightBase) SkyGlow(sky lights.Image, input lights.SkyGlowInput) (lights.Image, error) {
	return gpulights.SkyGlow(sky, input)
}

// Strobes adds the strobes described by input to sky on the GPU, and returns
// the number that gpulights.Strobes gives besides the image.
func (base gpuLightBase) Strobes(sky lights.Image, input lights.StrobeInput) (lights.Image, float64, error) {
	return gpulights.Strobes(sky, input)
}

// Searchlights plans the searchlight beams on the CPU, then draws them into
// sky and front on the GPU as splats.
func (base gpuLightBase) Searchlights(sky, front lights.Image, config lights.SearchConfig, input lights.SearchInput) (lights.Image, lights.Image, error) {
	planned, err := lights.PlanSearchlights(sky.Width, sky.Height, config, input)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	splats := make([]gpulights.Splat, len(planned))
	for index, item := range planned {
		splats[index] = gpulights.Splat{X: item.X, Y: item.Y, Sigma: item.Sigma, Amount: item.Amount, Colour: item.Colour, SkyOnly: item.SkyOnly}
	}
	return gpulights.Splats(sky, front, input.Open, input.ShoreRow, splats)
}

// Lasers plans the laser beams on the CPU, then draws them into sky and
// front on the GPU as splats.
func (base gpuLightBase) Lasers(sky, front lights.Image, config lights.LaserConfig, input lights.LaserInput) (lights.Image, lights.Image, error) {
	planned, err := lights.PlanLasers(sky.Width, sky.Height, config, input)
	if err != nil {
		return lights.Image{}, lights.Image{}, err
	}
	splats := make([]gpulights.Splat, len(planned))
	for index, item := range planned {
		splats[index] = gpulights.Splat{X: item.X, Y: item.Y, Sigma: item.Sigma, Amount: item.Amount, Colour: item.Colour, SkyOnly: item.SkyOnly}
	}
	return gpulights.Splats(sky, front, input.Open, input.ShoreRow, splats)
}

// BackHaze adds the back haze described by input to front on the GPU, using
// the trees mask.
func (base gpuLightBase) BackHaze(front lights.Image, trees []bool, input lights.BackHazeInput) (lights.Image, error) {
	return gpulights.BackHaze(front, trees, input)
}

// Beam draws the UFO beam into sky and front on the GPU, using the trees
// and water masks.
func (base gpuLightBase) Beam(sky, front lights.Image, config lights.UFOConfig, input lights.UFOInput, trees, water []bool) (lights.Image, lights.Image, error) {
	return gpulights.Beam(sky, front, config, input, trees, water)
}

// EnableExperimentalGPU makes the stage draw the base, fixtures and the other
// effects in gpuLightBase on the GPU, at the stage's size. It refuses a stage
// that has no renderer.
func (stage *RendererLightStage) EnableExperimentalGPU() error {
	if stage == nil || stage.renderer == nil {
		return fmt.Errorf("lighting: uninitialized stage")
	}
	renderer, err := gpulights.New(stage.width, stage.height)
	if err != nil {
		return err
	}
	stage.accelerator = gpuLightBase{renderer: renderer}
	return nil
}
